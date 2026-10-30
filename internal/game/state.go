// Package game holds the TotalKrieg rules. It has no graphics dependency, so
// the rules, the AI and whole matches can run in tests.
package game

import (
	"errors"
	"fmt"
	"math/rand/v2"
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
	Phase   Phase
	Objects []*Object
	Units   []*Unit
	Stocks  [2]Stock
	Reports []Report

	Over   bool
	Winner Faction // NoFaction on a draw
	Reason string

	rng    *rand.Rand
	nextID int
	counts [2][3]int
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
			s.addUnit(o.Owner, Bataillon, o.Pos)
		}
	}
	s.beginTurn()
	return s
}

func (s *State) addUnit(f Faction, k UnitKind, pos Coord) *Unit {
	u := &Unit{ID: s.nextID, Faction: f, Kind: k, Pos: pos, Strength: Stats(f, k).Strength}
	s.nextID++
	nameUnit(u, s.counts[f][k])
	s.counts[f][k]++
	s.Units = append(s.Units, u)
	return u
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
	s.Phase = PhaseAufstellung
	for _, u := range s.Units {
		if u.Faction == f {
			u.Moved, u.Attacked, u.Fresh = false, false, false
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
	case s.Phase != PhaseAufstellung:
		return errors.New("Aufstellen geht nur in der Aufstellungsphase.")
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
	u := s.addUnit(s.Active, k, o.Pos)
	u.Fresh = true
	return u, nil
}

// Reachable returns the hexes a unit can end its move on, with their distance.
// Enemy units block, own units can be passed but not joined.
func (s *State) Reachable(u *Unit) map[Coord]int {
	out := map[Coord]int{}
	if u.Moved || u.Fresh {
		return out
	}
	dist := map[Coord]int{u.Pos: 0}
	frontier := []Coord{u.Pos}
	for len(frontier) > 0 {
		c := frontier[0]
		frontier = frontier[1:]
		d := dist[c]
		if d == u.Stats().Move {
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

// Move relocates a unit during the movement phase.
func (s *State) Move(u *Unit, to Coord) error {
	switch {
	case s.Over:
		return errors.New("Die Partie ist beendet.")
	case s.Phase != PhaseBewegung:
		return errors.New("Bewegen geht nur in der Bewegungsphase.")
	case u.Faction != s.Active:
		return errors.New("Diese Einheit gehört dir nicht.")
	case u.Fresh:
		return errors.New("Neue Einheiten ziehen erst in der nächsten Runde.")
	case u.Moved:
		return errors.New("Diese Einheit ist schon gezogen.")
	}
	if _, ok := s.Reachable(u)[to]; !ok {
		return errors.New("Dieses Feld ist nicht erreichbar.")
	}
	u.Pos = to
	u.Moved = true
	return nil
}

// CanAttack explains why a unit can't attack a hex, or returns nil.
func (s *State) CanAttack(u *Unit, target Coord) error {
	switch {
	case s.Over:
		return errors.New("Die Partie ist beendet.")
	case s.Phase != PhaseAngriff:
		return errors.New("Angreifen geht nur in der Angriffsphase.")
	case u.Faction != s.Active:
		return errors.New("Diese Einheit gehört dir nicht.")
	case u.Attacked:
		return errors.New("Diese Einheit hat schon angegriffen.")
	case Distance(u.Pos, target) != 1:
		return errors.New("Das Ziel muss direkt benachbart sein.")
	}
	d := s.UnitAt(target)
	if d == nil || d.Faction == u.Faction {
		return errors.New("Dort steht keine feindliche Einheit.")
	}
	return nil
}

// Attack resolves a fight. Both sides roll, and each loses the opposing
// strength times the opposing roll, rounded up.
func (s *State) Attack(u *Unit, target Coord) (Report, error) {
	if err := s.CanAttack(u, target); err != nil {
		return Report{}, err
	}
	d := s.UnitAt(target)
	u.Attacked = true

	defenderPercent := s.rollPercent()
	if o := s.ObjectAt(target); o != nil && o.Kind.IsCity() {
		defenderPercent = defenderPercent * cityDefenseBonusPercent / 100
	}
	attackerLoss := ceilPercent(d.Strength, defenderPercent)
	defenderLoss := ceilPercent(u.Strength, s.rollPercent())
	u.Strength -= attackerLoss
	d.Strength -= defenderLoss

	r := Report{
		Round:           s.Round,
		Attacker:        u.Short,
		Defender:        d.Short,
		AttackerFaction: u.Faction,
		AttackerLoss:    attackerLoss,
		DefenderLoss:    defenderLoss,
		Pos:             target,
	}
	if d.Strength <= 0 {
		r.DefenderDestroyed = true
		s.remove(d)
	}
	if u.Strength <= 0 {
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

func (s *State) remove(u *Unit) {
	for i, x := range s.Units {
		if x == u {
			s.Units = append(s.Units[:i], s.Units[i+1:]...)
			return
		}
	}
}

// Advance moves to the next phase, or ends the turn after the attack phase.
func (s *State) Advance() {
	if s.Over {
		return
	}
	if s.Phase != PhaseAngriff {
		s.Phase++
		return
	}
	s.endTurn()
}

func (s *State) endTurn() {
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
