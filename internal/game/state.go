// Package game holds the TotalKrieg rules. It has no graphics dependency, so
// the rules, the AI and whole matches can run in tests.
package game

import (
	"errors"
	"fmt"
	"math/rand/v2"
	"strings"
)

const unitKindCount = 2

const (
	// MaxRounds ends the match when no capital fell.
	MaxRounds = 40

	cityDefenseBonusPercent = 125
	minRollPercent          = 20
	maxRollPercent          = 40
	reportLimit             = 50
)

// State is a running match.
type State struct {
	Seed    uint64
	First   Faction // moves first every round, the human player
	Round   int
	Active  Faction
	Objects []*Object
	Units   []*Unit
	Stocks  [2]Stock
	Reports []Report

	Over   bool
	Winner Faction // NoFaction on a draw
	Reason string

	orders        []*Order
	routs         []*Rout
	rng           *rand.Rand
	nextID        int
	counts        [2][unitKindCount]int
	callsignsUsed [2]int
	surnamePool   [2][]string
}

// New starts a match. The first faction opens every round.
func New(seed uint64, first Faction) *State {
	s := &State{
		Seed:   seed,
		First:  first,
		Round:  1,
		Active: first,
		Winner: NoFaction,
		rng:    rand.New(rand.NewPCG(seed, seed^0x9e3779b97f4a7c15)),
	}
	s.Objects = generateObjects(s.rng)
	for _, o := range s.Objects {
		if o.Kind.IsCity() {
			s.Place(o.Owner, Battalion, o.Pos)
		}
	}
	s.beginTurn()
	return s
}

// Place puts a new, full-strength formation on the map. A battalion's staff
// takes the hex, its companies the free hexes around it.
func (s *State) Place(f Faction, k UnitKind, pos Coord) *Unit {
	e := s.buildFormation(f, k)
	u := s.placeElement(e, pos)
	spots := s.freeSpots(pos, len(e.Children))
	for i, c := range e.Children {
		s.placeElement(c, spots[i])
	}
	return u
}

// placeElement gives an element its unit on the map.
func (s *State) placeElement(e *Element, pos Coord) *Unit {
	u := &Unit{ID: s.nextID, Faction: e.Faction, Kind: e.Kind, Pos: pos, Element: e, Name: e.Name, Short: e.Short}
	if e.Faction == SovietUnion && e.Kind == Battalion {
		u.Flag = strings.SplitN(e.Short, "/", 2)[0]
	}
	e.Token = u
	s.nextID++
	s.Units = append(s.Units, u)
	return u
}

// freeSpots returns the n free hexes nearest a position. Objects stay clear, so
// a company doesn't block the city a staff starts in.
func (s *State) freeSpots(pos Coord, n int) []Coord {
	var out []Coord
	for ring := 1; len(out) < n; ring++ {
		dist := map[Coord]int{}
		for row := 0; row < MapRows; row++ {
			for col := 0; col < MapCols; col++ {
				c := Coord{col, row}
				if Distance(pos, c) == ring && s.UnitAt(c) == nil && s.ObjectAt(c) == nil {
					dist[c] = ring
				}
			}
		}
		out = append(out, sortedCoords(dist)...)
	}
	return out[:n]
}

// UnitAt returns the unit on a hex, or nil.
func (s *State) UnitAt(c Coord) *Unit {
	for _, u := range s.Units {
		if u.Pos == c {
			return u
		}
	}
	return nil
}

// ObjectAt returns the object on a hex, or nil.
func (s *State) ObjectAt(c Coord) *Object {
	for _, o := range s.Objects {
		if o.Pos == c {
			return o
		}
	}
	return nil
}

// OnMap reports whether a coordinate lies on the map.
func OnMap(c Coord) bool {
	return c.Col >= 0 && c.Col < MapCols && c.Row >= 0 && c.Row < MapRows
}

func (s *State) beginTurn() {
	f := s.Active
	for _, u := range s.Units {
		if u.Faction == f {
			u.Used, u.Spent = 0, u.Beaten
			u.Resting, u.Beaten = u.Beaten, false
		}
	}
	for _, o := range s.Objects {
		if o.Owner != f {
			continue
		}
		o.Recruited = false
		switch {
		case o.Kind.IsCity():
			s.Stocks[f].Manpower++
		case o.Kind == Factory:
			s.Stocks[f].Resources++
		}
	}
}

// CanRecruit explains why a unit can't be raised in a city, or returns nil.
func (s *State) CanRecruit(o *Object, k UnitKind) error {
	switch {
	case s.Over:
		return errors.New("The match is over.")
	case o == nil || !o.Kind.IsCity():
		return errors.New("Only cities raise units.")
	case o.Owner != s.Active:
		return errors.New("This city is not yours.")
	case o.Recruited:
		return errors.New("This city already raised a unit this round.")
	case s.UnitAt(o.Pos) != nil:
		return errors.New("The city hex is occupied.")
	case s.Stocks[s.Active].Manpower < Stats(s.Active, k).Cost:
		return errors.New("Not enough manpower.")
	}
	return nil
}

// Recruit raises a unit in a city.
func (s *State) Recruit(o *Object, k UnitKind) (*Unit, error) {
	if err := s.CanRecruit(o, k); err != nil {
		return nil, err
	}
	s.Stocks[s.Active].Manpower -= Stats(s.Active, k).Cost
	o.Recruited = true
	u := s.Place(s.Active, k, o.Pos)
	for _, t := range u.Element.Tokens() {
		t.Spent = true
	}
	return u, nil
}

// Reachable returns the hexes a unit can end its move on, with their distance.
// Enemy units block, own units can be passed but not joined.
func (s *State) Reachable(u *Unit) map[Coord]int {
	out := map[Coord]int{}
	budget := u.MovesLeft()
	dist := map[Coord]int{u.Pos: 0}
	frontier := []Coord{u.Pos}
	for len(frontier) > 0 {
		c := frontier[0]
		frontier = frontier[1:]
		d := dist[c]
		if d == budget {
			continue
		}
		for _, n := range c.Neighbors() {
			if !OnMap(n) {
				continue
			}
			if _, seen := dist[n]; seen {
				continue
			}
			other := s.UnitAt(n)
			if other != nil && other.Faction != u.Faction {
				continue
			}
			dist[n] = d + 1
			frontier = append(frontier, n)
			if other == nil {
				out[n] = d + 1
			}
		}
	}
	return out
}

// Move relocates a unit and spends its movement. Moving onto an enemy is an
// attack, see AttackTargets.
func (s *State) Move(u *Unit, to Coord) error {
	switch {
	case s.Over:
		return errors.New("The match is over.")
	case u.Faction != s.Active:
		return errors.New("This unit is not yours.")
	case u.MovesLeft() == 0:
		return errors.New("This unit can't move any more this round.")
	}
	d, ok := s.Reachable(u)[to]
	if !ok {
		return errors.New("This hex is out of reach.")
	}
	u.Pos = to
	u.Used += d
	return nil
}

// AttackTargets maps each enemy hex the unit can move onto to the hex it
// attacks from. That is the nearest hex next to the enemy. Only companies
// attack, staffs never do.
func (s *State) AttackTargets(u *Unit) map[Coord]Coord {
	out := map[Coord]Coord{}
	budget := u.MovesLeft()
	if u.Kind != Company || budget == 0 {
		return out
	}
	reach := s.Reachable(u)
	reach[u.Pos] = 0
	best := map[Coord]int{}
	for _, c := range sortedCoords(reach) {
		d := reach[c]
		if d >= budget {
			continue
		}
		for _, n := range c.Neighbors() {
			e := s.UnitAt(n)
			if e == nil || e.Faction == u.Faction {
				continue
			}
			if old, seen := best[n]; !seen || d < old {
				best[n], out[n] = d, c
			}
		}
	}
	return out
}

// CanAttack explains why a unit can't attack a hex, or returns nil.
func (s *State) CanAttack(u *Unit, target Coord) error {
	switch {
	case s.Over:
		return errors.New("The match is over.")
	case u.Faction != s.Active:
		return errors.New("This unit is not yours.")
	}
	if _, ok := s.AttackTargets(u)[target]; !ok {
		return errors.New("This target is out of reach.")
	}
	return nil
}

// Attack moves a company next to the target and resolves the fight
// automatically. Both sides roll, and each loses the opposing strength times
// the opposing roll, rounded up. Every attacker is done for this turn.
func (s *State) Attack(u *Unit, target Coord) (Report, error) {
	if err := s.CanAttack(u, target); err != nil {
		return Report{}, err
	}
	attackers, defenders := s.Engagement(u, target)
	u.Pos = s.AttackTargets(u)[target]

	defenderPercent := s.rollPercent()
	if o := s.ObjectAt(target); o != nil && o.Kind.IsCity() {
		defenderPercent = defenderPercent * cityDefenseBonusPercent / 100
	}
	attackerLoss := ceilPercent(totalStrength(defenders), defenderPercent)
	defenderLoss := ceilPercent(totalStrength(attackers), s.rollPercent())
	o := autoOutcome(attackerLoss, totalStrength(attackers), defenderLoss, totalStrength(defenders))
	return s.settle(u, attackers, defenders, target, o, attackerLoss, defenderLoss), nil
}

// settle books the losses of a fight.
func (s *State) settle(u *Unit, attackers, defenders []*Unit, target Coord, o Outcome, attackerLoss, defenderLoss int) Report {
	d := defenders[0]
	r := Report{
		Round:           s.Round,
		Attacker:        fightLabel(attackers),
		Defender:        fightLabel(defenders),
		AttackerFaction: u.Faction,
		AttackerLoss:    attackerLoss,
		DefenderLoss:    defenderLoss,
		Pos:             target,
	}
	takeLosses(attackers, attackerLoss)
	takeLosses(defenders, defenderLoss)
	for _, x := range attackers {
		x.Spent = true
		if x.Strength() <= 0 {
			s.remove(x)
		}
	}
	for _, x := range defenders {
		if x.Strength() <= 0 {
			s.remove(x)
		}
	}
	r.DefenderDestroyed = d.Strength() <= 0
	r.AttackerDestroyed = u.Strength() <= 0
	s.addReport(r)
	s.afterFight(u, attackers, defenders, target, o)
	return r
}

func (s *State) addReport(r Report) {
	s.Reports = append(s.Reports, r)
	if len(s.Reports) > reportLimit {
		s.Reports = s.Reports[len(s.Reports)-reportLimit:]
	}
}

// fightLabel names the lead unit and how many others fought with it.
func fightLabel(units []*Unit) string {
	if len(units) == 1 {
		return units[0].Short
	}
	return fmt.Sprintf("%s +%d", units[0].Short, len(units)-1)
}

func (s *State) rollPercent() int {
	return minRollPercent + s.rng.IntN(maxRollPercent-minRollPercent+1)
}

func ceilPercent(strength, percent int) int {
	return (strength*percent + 99) / 100
}

// remove takes a unit off the map. Its element stays in the order of battle.
func (s *State) remove(u *Unit) {
	u.Element.Token = nil
	for i, x := range s.Units {
		if x == u {
			s.Units = append(s.Units[:i], s.Units[i+1:]...)
			return
		}
	}
}

// EndTurn takes objects under the active faction's companies and passes the
// turn. Staffs don't capture.
func (s *State) EndTurn() {
	if s.Over {
		return
	}
	f := s.Active
	s.orders = nil
	for _, u := range s.Units {
		if u.Faction != f {
			continue
		}
		u.Resting = false
		if u.Kind != Company {
			continue
		}
		o := s.ObjectAt(u.Pos)
		if o == nil || o.Owner == f {
			continue
		}
		o.Owner = f
		if o.Kind == Capital {
			s.finish(f, fmt.Sprintf("%s captured the enemy capital %s.", f, o.Name))
			return
		}
	}
	s.Active = f.Other()
	if s.Active == s.First {
		s.Round++
		if s.Round > MaxRounds {
			s.decideByCount()
			return
		}
	}
	s.beginTurn()
}

func (s *State) finish(winner Faction, reason string) {
	s.Over, s.Winner, s.Reason = true, winner, reason
}

func (s *State) decideByCount() {
	cities, all := s.Holdings(Wehrmacht)
	cityS, allS := s.Holdings(SovietUnion)
	switch {
	case cities > cityS:
		s.finish(Wehrmacht, fmt.Sprintf("After %d rounds the Wehrmacht holds more cities.", MaxRounds))
	case cityS > cities:
		s.finish(SovietUnion, fmt.Sprintf("After %d rounds the Soviet Union holds more cities.", MaxRounds))
	case all > allS:
		s.finish(Wehrmacht, "Cities are even, the Wehrmacht holds more objects.")
	case allS > all:
		s.finish(SovietUnion, "Cities are even, the Soviet Union holds more objects.")
	default:
		s.finish(NoFaction, "Draw: equal cities and objects.")
	}
}

// Holdings counts a faction's cities, capital included, and all its objects.
func (s *State) Holdings(f Faction) (cities, objects int) {
	for _, o := range s.Objects {
		if o.Owner != f {
			continue
		}
		objects++
		if o.Kind.IsCity() {
			cities++
		}
	}
	return cities, objects
}
