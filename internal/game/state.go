// Package game holds the TotalKrieg rules. It has no graphics dependency, so
// the rules, the AI and whole matches can run in tests.
package game

import (
	"errors"
	"fmt"
	"math/rand/v2"
	"strings"
)

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

	rng           *rand.Rand
	nextID        int
	counts        [2][3]int
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
			s.Place(o.Owner, Bataillon, o.Pos)
		}
	}
	s.beginTurn()
	return s
}

// Place puts a new, full-strength formation on the map.
func (s *State) Place(f Faction, k UnitKind, pos Coord) *Unit {
	return s.placeElement(s.buildFormation(f, k), pos)
}

// placeElement gives an element its own unit on the map.
func (s *State) placeElement(e *Element, pos Coord) *Unit {
	u := &Unit{ID: s.nextID, Faction: e.Faction, Kind: e.Kind, Pos: pos, Element: e, Name: e.Name, Short: e.Short}
	u.MovesLeft = u.Stats().Move
	if e.Faction == Sowjetunion {
		u.Flag = sovietFlag(e)
	}
	e.Token = u
	s.nextID++
	s.Units = append(s.Units, u)
	return u
}

// sovietFlag is the text inside the flag: the designation on a division flag,
// the battalion number on a pennant.
func sovietFlag(e *Element) string {
	switch e.Kind {
	case Division:
		return e.Short
	case Bataillon:
		return strings.SplitN(e.Short, "/", 2)[0]
	}
	return ""
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
			u.MovesLeft = u.Stats().Move
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
		case o.Kind == Fabrik:
			s.Stocks[f].Resources++
		}
	}
}

// CanRecruit explains why a unit can't be raised in a city, or returns nil.
func (s *State) CanRecruit(o *Object, k UnitKind) error {
	switch {
	case s.Over:
		return errors.New("Die Partie ist beendet.")
	case o == nil || !o.Kind.IsCity():
		return errors.New("Nur Städte stellen Einheiten auf.")
	case o.Owner != s.Active:
		return errors.New("Diese Stadt gehört dir nicht.")
	case o.Recruited:
		return errors.New("Diese Stadt hat in dieser Runde schon aufgestellt.")
	case s.UnitAt(o.Pos) != nil:
		return errors.New("Das Stadtfeld ist besetzt.")
	case s.Stocks[s.Active].Manpower < Stats(s.Active, k).Cost:
		return errors.New("Zu wenig Manpower.")
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
	u.MovesLeft = 0
	return u, nil
}

// Reachable returns the hexes a unit can end its move on, with their distance.
// Enemy units block, own units can be passed but not joined.
func (s *State) Reachable(u *Unit) map[Coord]int {
	out := map[Coord]int{}
	dist := map[Coord]int{u.Pos: 0}
	frontier := []Coord{u.Pos}
	for len(frontier) > 0 {
		c := frontier[0]
		frontier = frontier[1:]
		d := dist[c]
		if d == u.MovesLeft {
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
		return errors.New("Die Partie ist beendet.")
	case u.Faction != s.Active:
		return errors.New("Diese Einheit gehört dir nicht.")
	case u.MovesLeft == 0:
		return errors.New("Diese Einheit kann in dieser Runde nicht mehr ziehen.")
	}
	d, ok := s.Reachable(u)[to]
	if !ok {
		return errors.New("Dieses Feld ist nicht erreichbar.")
	}
	u.Pos = to
	u.MovesLeft -= d
	return nil
}

// AttackTargets maps each enemy hex the unit can move onto to the hex it
// attacks from. That is the nearest hex next to the enemy.
func (s *State) AttackTargets(u *Unit) map[Coord]Coord {
	out := map[Coord]Coord{}
	if u.MovesLeft == 0 {
		return out
	}
	reach := s.Reachable(u)
	reach[u.Pos] = 0
	best := map[Coord]int{}
	for _, c := range sortedCoords(reach) {
		d := reach[c]
		if d >= u.MovesLeft {
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
		return errors.New("Die Partie ist beendet.")
	case u.Faction != s.Active:
		return errors.New("Diese Einheit gehört dir nicht.")
	}
	if _, ok := s.AttackTargets(u)[target]; !ok {
		return errors.New("Dieses Ziel ist nicht erreichbar.")
	}
	return nil
}

// Attack moves a unit next to the target and resolves the fight automatically.
// Both sides roll, and each loses the opposing strength times the opposing
// roll, rounded up. An attack ends the unit's movement.
func (s *State) Attack(u *Unit, target Coord) (Report, error) {
	if err := s.CanAttack(u, target); err != nil {
		return Report{}, err
	}
	d := s.UnitAt(target)
	u.Pos = s.AttackTargets(u)[target]
	u.MovesLeft = 0

	defenderPercent := s.rollPercent()
	if o := s.ObjectAt(target); o != nil && o.Kind.IsCity() {
		defenderPercent = defenderPercent * cityDefenseBonusPercent / 100
	}
	attackerLoss := ceilPercent(d.Strength(), defenderPercent)
	defenderLoss := ceilPercent(u.Strength(), s.rollPercent())
	u.takeLoss(attackerLoss)
	d.takeLoss(defenderLoss)

	r := Report{
		Round:           s.Round,
		Attacker:        u.Short,
		Defender:        d.Short,
		AttackerFaction: u.Faction,
		AttackerLoss:    attackerLoss,
		DefenderLoss:    defenderLoss,
		Pos:             target,
	}
	if d.Strength() <= 0 {
		r.DefenderDestroyed = true
		s.remove(d)
	}
	if u.Strength() <= 0 {
		r.AttackerDestroyed = true
		s.remove(u)
	} else if r.DefenderDestroyed {
		u.Pos = target
	}
	s.Reports = append(s.Reports, r)
	if len(s.Reports) > reportLimit {
		s.Reports = s.Reports[len(s.Reports)-reportLimit:]
	}
	return r, nil
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

// EndTurn takes objects under the active faction's units and passes the turn.
func (s *State) EndTurn() {
	if s.Over {
		return
	}
	f := s.Active
	for _, u := range s.Units {
		if u.Faction != f {
			continue
		}
		o := s.ObjectAt(u.Pos)
		if o == nil || o.Owner == f {
			continue
		}
		o.Owner = f
		if o.Kind == Hauptstadt {
			s.finish(f, fmt.Sprintf("%s hat die gegnerische Hauptstadt %s erobert.", f, o.Name))
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
	cityS, allS := s.Holdings(Sowjetunion)
	switch {
	case cities > cityS:
		s.finish(Wehrmacht, fmt.Sprintf("Nach %d Runden hält die Wehrmacht mehr Städte.", MaxRounds))
	case cityS > cities:
		s.finish(Sowjetunion, fmt.Sprintf("Nach %d Runden hält die Sowjetunion mehr Städte.", MaxRounds))
	case all > allS:
		s.finish(Wehrmacht, "Städte gleich, die Wehrmacht hält mehr Objekte.")
	case allS > all:
		s.finish(Sowjetunion, "Städte gleich, die Sowjetunion hält mehr Objekte.")
	default:
		s.finish(NoFaction, "Unentschieden: gleich viele Städte und Objekte.")
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
