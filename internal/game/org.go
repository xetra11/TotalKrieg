package game

import "errors"

// CompanyRole is the type of a company.
type CompanyRole int

const (
	RoleSchuetzen CompanyRole = iota
	RoleMG
	RoleGranatwerfer
)

func (r CompanyRole) String() string {
	switch r {
	case RoleMG:
		return "MG-Kompanie"
	case RoleGranatwerfer:
		return "Granatwerfer-Kompanie"
	}
	return "Schützenkompanie"
}

// companyStrength holds the strength of a full company. A battalion's strength
// is the sum of its companies, so the unit table in types.go follows from it.
var companyStrength = [2][3]int{
	Wehrmacht:   {RoleSchuetzen: 3, RoleMG: 3},
	Sowjetunion: {RoleSchuetzen: 2, RoleMG: 2, RoleGranatwerfer: 1},
}

// Element is a node of the order of battle. It keeps its place in the tree for
// good: a detached company still belongs to its battalion and division.
type Element struct {
	Faction  Faction
	Kind     UnitKind
	Name     string
	Short    string
	Parent   *Element
	Children []*Element

	// Company data.
	Role     CompanyRole
	Callsign string
	Leader   string
	Current  int
	Full     int

	// Token is set when the element forms its own unit on the map.
	Token *Unit
}

// Root returns the top of the element's formation.
func (e *Element) Root() *Element {
	for e.Parent != nil {
		e = e.Parent
	}
	return e
}

// Holder returns the map unit the element currently marches with.
func (e *Element) Holder() *Unit {
	for x := e; x != nil; x = x.Parent {
		if x.Token != nil {
			return x.Token
		}
	}
	return nil
}

// Detached reports whether the element is split off from its parent.
func (e *Element) Detached() bool {
	return e.Parent != nil && e.Token != nil
}

// Strength sums the element's companies, detached ones included.
func (e *Element) Strength() int {
	sum := 0
	for _, c := range e.companies(false) {
		sum += c.Current
	}
	return sum
}

// FullStrength is the establishment value of Strength.
func (e *Element) FullStrength() int {
	sum := 0
	for _, c := range e.companies(false) {
		sum += c.Full
	}
	return sum
}

// companies lists the companies below an element. With presentOnly set,
// branches that left as their own unit are skipped.
func (e *Element) companies(presentOnly bool) []*Element {
	if e.Kind == Kompanie {
		return []*Element{e}
	}
	var out []*Element
	for _, c := range e.Children {
		if presentOnly && c.Token != nil {
			continue
		}
		out = append(out, c.companies(presentOnly)...)
	}
	return out
}

// Tokens lists the map units of a formation, the element's own included.
func (e *Element) Tokens() []*Unit {
	var out []*Unit
	if e.Token != nil {
		out = append(out, e.Token)
	}
	for _, c := range e.Children {
		out = append(out, c.Tokens()...)
	}
	return out
}

// Strength sums the companies marching with the unit.
func (u *Unit) Strength() int {
	sum := 0
	for _, c := range u.Element.companies(true) {
		sum += c.Current
	}
	return sum
}

// FullStrength is the establishment value of Strength.
func (u *Unit) FullStrength() int {
	sum := 0
	for _, c := range u.Element.companies(true) {
		sum += c.Full
	}
	return sum
}

// takeLoss spreads a loss over the companies, strongest first.
func (u *Unit) takeLoss(n int) {
	companies := u.Element.companies(true)
	for ; n > 0; n-- {
		var worst *Element
		for _, c := range companies {
			if c.Current > 0 && (worst == nil || c.Current > worst.Current) {
				worst = c
			}
		}
		if worst == nil {
			return
		}
		worst.Current--
	}
}

// DetachSpots returns the free hexes next to a unit.
func (s *State) DetachSpots(u *Unit) []Coord {
	var out []Coord
	for _, c := range u.Pos.Neighbors() {
		if OnMap(c) && s.UnitAt(c) == nil {
			out = append(out, c)
		}
	}
	return out
}

// CanDetach explains why an element can't leave its unit, or returns nil.
func (s *State) CanDetach(e *Element) error {
	u := e.Holder()
	switch {
	case s.Over:
		return errors.New("Die Partie ist beendet.")
	case s.Phase != PhaseBewegung:
		return errors.New("Ausgliedern geht nur in der Bewegungsphase.")
	case u == nil || u.Faction != s.Active:
		return errors.New("Dieser Verband gehört dir nicht.")
	case e.Token != nil:
		return errors.New("Diese Einheit ist schon ausgegliedert.")
	case e.Kind != Kompanie && e.Kind != Bataillon:
		return errors.New("Nur Kompanien und Bataillone lassen sich ausgliedern.")
	case u.Moved || u.Fresh:
		return errors.New("Ausgliedern geht nur, bevor der Verband zieht.")
	case e.Strength() <= 0:
		return errors.New("Diese Einheit ist aufgerieben.")
	case u.Strength()-e.Strength() <= 0:
		return errors.New("Der Verband braucht mindestens eine eigene Kompanie.")
	case len(s.DetachSpots(u)) == 0:
		return errors.New("Neben dem Verband ist kein Feld frei.")
	}
	return nil
}

// Detach splits an element off as its own unit on a neighboring hex. It stays
// part of its formation and can still move this turn.
func (s *State) Detach(e *Element, to Coord) (*Unit, error) {
	if err := s.CanDetach(e); err != nil {
		return nil, err
	}
	u := e.Holder()
	if Distance(u.Pos, to) != 1 || !OnMap(to) || s.UnitAt(to) != nil {
		return nil, errors.New("Dieses Feld ist nicht frei.")
	}
	t := s.placeElement(e, to)
	t.Attacked = u.Attacked
	return t, nil
}

// reattachTarget finds the nearest superior unit still on the map.
func (u *Unit) reattachTarget() *Unit {
	if u.Element.Parent == nil {
		return nil
	}
	return u.Element.Parent.Holder()
}

// CanReattach returns the unit a detachment would rejoin, or why it can't.
func (s *State) CanReattach(u *Unit) (*Unit, error) {
	if !u.Element.Detached() {
		return nil, errors.New("Diese Einheit ist nicht ausgegliedert.")
	}
	if s.Phase != PhaseBewegung || u.Faction != s.Active {
		return nil, errors.New("Eingliedern geht nur in der eigenen Bewegungsphase.")
	}
	t := u.reattachTarget()
	if t == nil {
		return nil, errors.New("Der Stammverband ist nicht mehr auf der Karte.")
	}
	if Distance(u.Pos, t.Pos) != 1 {
		return nil, errors.New("Der Stammverband muss direkt benachbart sein.")
	}
	return t, nil
}

// Reattach returns a detachment to its parent unit.
func (s *State) Reattach(u *Unit) (*Unit, error) {
	t, err := s.CanReattach(u)
	if err != nil {
		return nil, err
	}
	s.remove(u)
	return t, nil
}
