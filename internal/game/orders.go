package game

import (
	"errors"
	"iter"
	"slices"
)

// Order is a move or attack planned for the end of the turn.
type Order struct {
	Unit   *Unit
	To     Coord
	Attack bool
}

// Fight is an attack about to happen. Whoever steps through a turn may resolve
// it and set Resolved, otherwise the rules resolve it automatically.
type Fight struct {
	Unit     *Unit
	Target   Coord
	Resolved bool
}

// Orders lists the planned orders in the order they were given.
func (s *State) Orders() []*Order {
	return s.orders
}

// OrderOf returns a unit's planned order, or nil.
func (s *State) OrderOf(u *Unit) *Order {
	for _, o := range s.orders {
		if o.Unit == u {
			return o
		}
	}
	return nil
}

// OrderTargets returns the hexes a unit can be ordered to. Links and reach count
// from where the units stand now. Hexes another order claims are taken.
func (s *State) OrderTargets(u *Unit) map[Coord]int {
	reach := s.Reachable(u)
	for _, o := range s.orders {
		if o.Unit != u && !o.Attack {
			delete(reach, o.To)
		}
	}
	return reach
}

// Plan gives a unit its order for this turn and replaces an earlier one. Moving
// onto an enemy is an attack.
func (s *State) Plan(u *Unit, to Coord) error {
	switch {
	case s.Over:
		return errors.New("The match is over.")
	case u.Faction != s.Active:
		return errors.New("This unit is not yours.")
	case u.MovesLeft() == 0:
		return errors.New("This unit can't move any more this round.")
	}
	_, attack := s.AttackTargets(u)[to]
	if _, ok := s.OrderTargets(u)[to]; !ok && !attack {
		return errors.New("This hex is out of reach.")
	}
	s.CancelOrder(u)
	s.orders = append(s.orders, &Order{Unit: u, To: to, Attack: attack})
	return nil
}

// CancelOrder drops a unit's order.
func (s *State) CancelOrder(u *Unit) {
	s.orders = slices.DeleteFunc(s.orders, func(o *Order) bool { return o.Unit == u })
}

// CommitTurn carries out the orders and ends the turn. Companies move first, so
// they can support the attacks that follow. Staffs move last, so every attack
// still has the link it was planned with. Orders that became impossible lapse.
func (s *State) CommitTurn() iter.Seq[*Fight] {
	return func(yield func(*Fight) bool) {
		orders := s.orders
		s.orders = nil
		for _, o := range orders {
			if o.Unit.Kind == Company {
				s.carryOut(o)
			}
		}
		for _, o := range orders {
			if o.Attack && s.CanAttack(o.Unit, o.To) == nil && !s.fight(o.Unit, o.To, yield) {
				return
			}
		}
		for _, o := range orders {
			if o.Unit.Kind != Company {
				s.carryOut(o)
			}
		}
		s.EndTurn()
	}
}

func (s *State) carryOut(o *Order) {
	if !o.Attack && s.UnitAt(o.To) == nil {
		o.Unit.Pos = o.To
		o.Unit.Used = o.Unit.Allowance()
	}
}

// fight offers an attack to whoever steps through the turn, then resolves it
// unless they did. It reports false when the turn is abandoned.
func (s *State) fight(u *Unit, target Coord, yield func(*Fight) bool) bool {
	f := &Fight{Unit: u, Target: target}
	if !yield(f) {
		return false
	}
	if !f.Resolved {
		s.Attack(u, target)
	}
	return true
}
