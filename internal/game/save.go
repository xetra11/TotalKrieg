package game

import (
	"errors"
	"fmt"
	"math/rand/v2"
)

// snapshotVersion is the layout of Snapshot. It changes when the fields do, so
// an old save is rejected instead of being read into a wrong match.
const snapshotVersion = 2

// Snapshot is a match in a form that can be written to disk. Units, elements,
// orders and routs refer to each other by their index in the lists, because the
// live state uses pointers.
type Snapshot struct {
	Version int     `json:"version"`
	Seed    uint64  `json:"seed"`
	First   Faction `json:"first"`
	Round   int     `json:"round"`
	Active  Faction `json:"active"`

	Objects  []Object          `json:"objects"`
	Units    []UnitSnapshot    `json:"units"`
	Elements []ElementSnapshot `json:"elements"`
	Stocks   [2]Stock          `json:"stocks"`
	Reports  []Report          `json:"reports"`

	Over   bool    `json:"over"`
	Winner Faction `json:"winner"`
	Reason string  `json:"reason"`

	Orders []OrderSnapshot `json:"orders,omitempty"`
	Routs  []RoutSnapshot  `json:"routs,omitempty"`

	RNG           []byte      `json:"rng"`
	NextID        int         `json:"next_id"`
	Counts        [2][2]int   `json:"counts"`
	CallsignsUsed [2]int      `json:"callsigns_used"`
	SurnamePool   [2][]string `json:"surname_pool"`
}

// UnitSnapshot is a unit without its pointer to the order of battle.
type UnitSnapshot struct {
	ID      int      `json:"id"`
	Faction Faction  `json:"faction"`
	Kind    UnitKind `json:"kind"`
	Pos     Coord    `json:"pos"`
	Element int      `json:"element"`

	Name  string `json:"name"`
	Short string `json:"short"`
	Flag  string `json:"flag,omitempty"`

	Used    int  `json:"used"`
	Spent   bool `json:"spent,omitempty"`
	Beaten  bool `json:"beaten,omitempty"`
	Resting bool `json:"resting,omitempty"`
}

// ElementSnapshot is one node of the order of battle. Parent and Token are
// indices, -1 when there is none.
type ElementSnapshot struct {
	Faction Faction  `json:"faction"`
	Kind    UnitKind `json:"kind"`
	Name    string   `json:"name"`
	Short   string   `json:"short"`

	Parent   int   `json:"parent"`
	Children []int `json:"children,omitempty"`

	Role     CompanyRole `json:"role"`
	Callsign string      `json:"callsign,omitempty"`
	Leader   string      `json:"leader,omitempty"`
	Current  int         `json:"current"`
	Full     int         `json:"full"`

	Home  int `json:"home"` // index of the home city, -1 when none
	Token int `json:"token"`
}

// OrderSnapshot is a planned order, referring to a unit by index.
type OrderSnapshot struct {
	Unit   int   `json:"unit"`
	To     Coord `json:"to"`
	Attack bool  `json:"attack,omitempty"`
}

// RoutSnapshot is a waiting rout, referring to units by index. Victor is -1
// when the fleeing unit lost its attack.
type RoutSnapshot struct {
	Unit   int   `json:"unit"`
	From   Coord `json:"from"`
	Victor int   `json:"victor"`
}

// Snapshot returns the match in a form that survives a restart.
func (s *State) Snapshot() (Snapshot, error) {
	if s == nil {
		return Snapshot{}, errors.New("no match to save")
	}
	snap := Snapshot{
		Version:       snapshotVersion,
		Seed:          s.Seed,
		First:         s.First,
		Round:         s.Round,
		Active:        s.Active,
		Stocks:        s.Stocks,
		Reports:       append([]Report(nil), s.Reports...),
		Over:          s.Over,
		Winner:        s.Winner,
		Reason:        s.Reason,
		NextID:        s.nextID,
		Counts:        s.counts,
		CallsignsUsed: s.callsignsUsed,
		SurnamePool:   s.surnamePool,
	}
	objectIndex := make(map[*Object]int, len(s.Objects))
	for _, o := range s.Objects {
		if o == nil {
			continue
		}
		objectIndex[o] = len(snap.Objects)
		snap.Objects = append(snap.Objects, *o)
	}

	if s.pcg != nil {
		data, err := s.pcg.MarshalBinary()
		if err != nil {
			return Snapshot{}, fmt.Errorf("save random generator: %w", err)
		}
		snap.RNG = data
	}

	elements, index := s.snapshotElements()
	for _, e := range elements {
		es := ElementSnapshot{
			Faction:  e.Faction,
			Kind:     e.Kind,
			Name:     e.Name,
			Short:    e.Short,
			Parent:   -1,
			Role:     e.Role,
			Callsign: e.Callsign,
			Leader:   e.Leader,
			Current:  e.Current,
			Full:     e.Full,
			Home:     -1,
			Token:    -1,
		}
		if e.Parent != nil {
			es.Parent = index[e.Parent]
		}
		if e.Home != nil {
			if idx, ok := objectIndex[e.Home]; ok {
				es.Home = idx
			}
		}
		for _, c := range e.Children {
			es.Children = append(es.Children, index[c])
		}
		snap.Elements = append(snap.Elements, es)
	}

	unitIndex := make(map[*Unit]int, len(s.Units))
	for i, u := range s.Units {
		unitIndex[u] = i
	}
	for i, e := range elements {
		if e.Token != nil {
			snap.Elements[i].Token = unitIndex[e.Token]
		}
	}

	for _, u := range s.Units {
		if u == nil {
			continue
		}
		snap.Units = append(snap.Units, UnitSnapshot{
			ID:      u.ID,
			Faction: u.Faction,
			Kind:    u.Kind,
			Pos:     u.Pos,
			Element: index[u.Element],
			Name:    u.Name,
			Short:   u.Short,
			Flag:    u.Flag,
			Used:    u.Used,
			Spent:   u.Spent,
			Beaten:  u.Beaten,
			Resting: u.Resting,
		})
	}

	for _, o := range s.orders {
		snap.Orders = append(snap.Orders, OrderSnapshot{Unit: unitIndex[o.Unit], To: o.To, Attack: o.Attack})
	}
	for _, r := range s.routs {
		rs := RoutSnapshot{Unit: unitIndex[r.Unit], From: r.From, Victor: -1}
		if r.Victor != nil {
			rs.Victor = unitIndex[r.Victor]
		}
		snap.Routs = append(snap.Routs, rs)
	}
	return snap, nil
}

// snapshotElements lists every element, including formations whose units are
// all destroyed. It also returns the index of each element.
func (s *State) snapshotElements() ([]*Element, map[*Element]int) {
	var elements []*Element
	seen := map[*Element]bool{}
	var add func(*Element)
	add = func(e *Element) {
		if e == nil || seen[e] {
			return
		}
		seen[e] = true
		elements = append(elements, e)
		add(e.Parent)
		for _, c := range e.Children {
			add(c)
		}
	}
	for _, e := range s.elements {
		add(e)
	}
	for _, u := range s.Units {
		if u != nil {
			add(u.Element)
		}
	}
	index := make(map[*Element]int, len(elements))
	for i, e := range elements {
		index[e] = i
	}
	return elements, index
}

// Restore rebuilds a match from a snapshot.
func Restore(snap Snapshot) (*State, error) {
	if snap.Version != snapshotVersion {
		return nil, fmt.Errorf("unsupported save version %d", snap.Version)
	}
	s := &State{
		Seed:          snap.Seed,
		First:         snap.First,
		Round:         snap.Round,
		Active:        snap.Active,
		Stocks:        snap.Stocks,
		Reports:       append([]Report(nil), snap.Reports...),
		Over:          snap.Over,
		Winner:        snap.Winner,
		Reason:        snap.Reason,
		nextID:        snap.NextID,
		counts:        snap.Counts,
		callsignsUsed: snap.CallsignsUsed,
		surnamePool:   snap.SurnamePool,
	}
	for i := range snap.Objects {
		o := snap.Objects[i]
		s.Objects = append(s.Objects, &o)
	}

	s.elements = make([]*Element, len(snap.Elements))
	for i, es := range snap.Elements {
		s.elements[i] = &Element{
			Faction:  es.Faction,
			Kind:     es.Kind,
			Name:     es.Name,
			Short:    es.Short,
			Role:     es.Role,
			Callsign: es.Callsign,
			Leader:   es.Leader,
			Current:  es.Current,
			Full:     es.Full,
		}
	}
	for i, es := range snap.Elements {
		e := s.elements[i]
		if es.Parent >= 0 {
			if !validIndex(es.Parent, len(s.elements)) {
				return nil, fmt.Errorf("element %d has parent %d", i, es.Parent)
			}
			e.Parent = s.elements[es.Parent]
		}
		if es.Home >= 0 {
			if !validIndex(es.Home, len(s.Objects)) {
				return nil, fmt.Errorf("element %d has home %d", i, es.Home)
			}
			e.Home = s.Objects[es.Home]
		}
		for _, c := range es.Children {
			if !validIndex(c, len(s.elements)) {
				return nil, fmt.Errorf("element %d has child %d", i, c)
			}
			e.Children = append(e.Children, s.elements[c])
		}
	}

	s.Units = make([]*Unit, len(snap.Units))
	for i, us := range snap.Units {
		if !validIndex(us.Element, len(s.elements)) {
			return nil, fmt.Errorf("unit %d has element %d", i, us.Element)
		}
		s.Units[i] = &Unit{
			ID:      us.ID,
			Faction: us.Faction,
			Kind:    us.Kind,
			Pos:     us.Pos,
			Element: s.elements[us.Element],
			Name:    us.Name,
			Short:   us.Short,
			Flag:    us.Flag,
			Used:    us.Used,
			Spent:   us.Spent,
			Beaten:  us.Beaten,
			Resting: us.Resting,
		}
	}
	for i, es := range snap.Elements {
		if es.Token < 0 {
			continue
		}
		if !validIndex(es.Token, len(s.Units)) {
			return nil, fmt.Errorf("element %d has token %d", i, es.Token)
		}
		s.elements[i].Token = s.Units[es.Token]
	}

	for _, os := range snap.Orders {
		if !validIndex(os.Unit, len(s.Units)) {
			return nil, fmt.Errorf("order refers to unit %d", os.Unit)
		}
		s.orders = append(s.orders, &Order{Unit: s.Units[os.Unit], To: os.To, Attack: os.Attack})
	}
	for _, rs := range snap.Routs {
		if !validIndex(rs.Unit, len(s.Units)) {
			return nil, fmt.Errorf("rout refers to unit %d", rs.Unit)
		}
		r := &Rout{Unit: s.Units[rs.Unit], From: rs.From}
		if rs.Victor >= 0 {
			if !validIndex(rs.Victor, len(s.Units)) {
				return nil, fmt.Errorf("rout refers to victor %d", rs.Victor)
			}
			r.Victor = s.Units[rs.Victor]
		}
		s.routs = append(s.routs, r)
	}

	pcg := rand.NewPCG(snap.Seed, snap.Seed^0x9e3779b97f4a7c15)
	if len(snap.RNG) > 0 {
		if err := pcg.UnmarshalBinary(snap.RNG); err != nil {
			return nil, fmt.Errorf("restore random generator: %w", err)
		}
	}
	s.pcg = pcg
	s.rng = rand.New(pcg)
	return s, nil
}

func validIndex(i, n int) bool {
	return i >= 0 && i < n
}
