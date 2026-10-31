package game

import (
	"errors"
	"slices"
)

// Rout is a beaten unit fleeing one hex from the enemy. AI units flee at once,
// the human player picks the hex.
type Rout struct {
	Unit   *Unit
	From   Coord // the enemy it flees from
	Victor *Unit // takes the vacated hex, nil after a lost attack
}

// startRout lets a beaten unit flee. A human unit keeps its hex until the player
// has picked one.
func (s *State) startRout(u *Unit, from Coord, victor *Unit) {
	r := &Rout{Unit: u, From: from, Victor: victor}
	if u.Faction == s.First {
		s.routs = slices.DeleteFunc(s.routs, func(x *Rout) bool { return x.Unit == u })
		s.routs = append(s.routs, r)
		return
	}
	if options := s.RoutOptions(r); len(options) > 0 {
		s.escape(r, options[0])
	}
}

// RoutOptions lists the free neighboring hexes farther from the enemy.
func (s *State) RoutOptions(r *Rout) []Coord {
	var out []Coord
	for _, c := range r.Unit.Pos.Neighbors() {
		if OnMap(c) && s.UnitAt(c) == nil && Distance(c, r.From) > Distance(r.Unit.Pos, r.From) {
			out = append(out, c)
		}
	}
	return out
}

// NextRout returns the rout waiting for the human player. A unit that was
// destroyed or has nowhere to go drops out and stays where it is.
func (s *State) NextRout() *Rout {
	for len(s.routs) > 0 && !s.Over {
		r := s.routs[0]
		if r.Unit.Element.Token == r.Unit && len(s.RoutOptions(r)) > 0 {
			return r
		}
		s.routs = s.routs[1:]
	}
	return nil
}

// Flee carries out a waiting rout.
func (s *State) Flee(r *Rout, to Coord) error {
	if !slices.Contains(s.RoutOptions(r), to) {
		return errors.New("The unit can't flee there.")
	}
	s.routs = slices.DeleteFunc(s.routs, func(x *Rout) bool { return x == r })
	s.escape(r, to)
	return nil
}

func (s *State) escape(r *Rout, to Coord) {
	vacated := r.Unit.Pos
	r.Unit.Pos = to
	if v := r.Victor; v != nil && v.Element.Token == v {
		v.Pos = vacated
	}
}

// fleeAll settles the routs nobody chose a hex for, when the AI plays both sides.
func (s *State) fleeAll() {
	for r := s.NextRout(); r != nil; r = s.NextRout() {
		s.Flee(r, s.RoutOptions(r)[0])
	}
}

// afterFight moves the units once the losses are booked. The lead company takes
// an emptied hex, the beaten side's lead unit flees, and all of that side sits
// out its next turn.
func (s *State) afterFight(u *Unit, attackers, defenders []*Unit, target Coord, o Outcome) {
	d := defenders[0]
	switch o {
	case AttackerWins:
		markBeaten(defenders)
	case DefenderWins:
		markBeaten(attackers)
	}
	switch {
	case u.Strength() <= 0:
	case d.Strength() <= 0:
		u.Pos = target
	case o == AttackerWins:
		s.startRout(d, u.Pos, u)
	case o == DefenderWins:
		s.startRout(u, target, nil)
	}
}

func markBeaten(units []*Unit) {
	for _, u := range units {
		u.Beaten = true
	}
}

// autoOutcome calls the side that lost the larger share of its strength beaten.
func autoOutcome(attackerLoss, attackers, defenderLoss, defenders int) Outcome {
	a, d := attackerLoss*defenders, defenderLoss*attackers
	switch {
	case a < d:
		return AttackerWins
	case a > d:
		return DefenderWins
	}
	return Draw
}
