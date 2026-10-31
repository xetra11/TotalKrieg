package game

// Outcome is the result of a fight fought outside the game.
type Outcome int

const (
	AttackerWins Outcome = iota
	Draw
	DefenderWins
)

func (o Outcome) String() string {
	switch o {
	case AttackerWins:
		return "Victory"
	case Draw:
		return "Draw"
	}
	return "Defeat"
}

// outcomeLosses holds the attacker's and defender's losses in percent. A beaten
// defender is out of the fight, so the attacker takes the hex.
var outcomeLosses = map[Outcome][2]int{
	AttackerWins: {25, 100},
	Draw:         {30, 30},
	DefenderWins: {50, 20},
}

// FieldResult is what a tactical game reports back.
type FieldResult struct {
	Text      string
	Outcome   Outcome
	Remaining map[*Element]float64 // share of strength each element kept
	Platoons  []PlatoonLoss
}

// PlatoonLoss is what one platoon of a tactical game lost.
type PlatoonLoss struct {
	Owner     *Element
	Name      string
	Squads    int
	Lost      int
	Condition float64 // of the surviving squads, 0 to 1
}

// ScaledStrength applies a kept share to a strength. A unit with survivors
// keeps at least 1 point.
func ScaledStrength(current int, share float64) int {
	if share <= 0 || current <= 0 {
		return 0
	}
	return max(1, int(float64(current)*share+0.5))
}

// ShareOf returns an element's kept share. Companies left out of a scaled-down
// mission share the fate of their side.
func (r *FieldResult) ShareOf(e *Element, side []*Element) float64 {
	if share, ok := r.Remaining[e]; ok {
		return share
	}
	sum, n := 0.0, 0
	for _, x := range side {
		if share, ok := r.Remaining[x]; ok {
			sum += share
			n++
		}
	}
	if n == 0 {
		return 1
	}
	return sum / float64(n)
}

// ApplyFieldResult books a fight fought in a tactical game with its real losses.
func (s *State) ApplyFieldResult(u *Unit, target Coord, res *FieldResult) (Report, error) {
	if err := s.CanAttack(u, target); err != nil {
		return Report{}, err
	}
	attackers, defenders := s.Engagement(u, target)
	u.Pos = s.AttackTargets(u)[target]
	d := defenders[0]
	r := Report{
		Round:           s.Round,
		Attacker:        fightLabel(attackers),
		Defender:        fightLabel(defenders),
		AttackerFaction: u.Faction,
		Pos:             target,
	}
	r.AttackerLoss = s.applyShares(attackers, res)
	r.DefenderLoss = s.applyShares(defenders, res)
	for e, share := range res.Remaining {
		if e.Kind == Battalion && e.Token != nil && share <= 0 {
			e.Current = 0
			s.remove(e.Token)
		}
	}
	for _, x := range attackers {
		x.Spent = true
	}
	r.AttackerDestroyed = u.Strength() <= 0
	r.DefenderDestroyed = d.Strength() <= 0
	s.addReport(r)
	s.afterFight(u, d, target, res.Outcome)
	return r, nil
}

// applyShares sets every unit's new strength and returns the side's loss.
func (s *State) applyShares(units []*Unit, res *FieldResult) int {
	side := make([]*Element, len(units))
	for i, x := range units {
		side[i] = x.Element
	}
	lost := 0
	for _, x := range units {
		before := x.Strength()
		x.Element.Current = ScaledStrength(before, res.ShareOf(x.Element, side))
		lost += before - x.Strength()
		if x.Strength() <= 0 {
			s.remove(x)
		}
	}
	return lost
}

// Engagement lists who fights over a hex. The attacking company brings every
// own company next to the target that hasn't fought yet. The target's unit
// gets support from the companies of its side next to it. Staffs don't join.
func (s *State) Engagement(u *Unit, target Coord) (attackers, defenders []*Unit) {
	attackers = []*Unit{u}
	if d := s.UnitAt(target); d != nil {
		defenders = []*Unit{d}
	}
	for _, x := range s.Units {
		if x == u || x.Kind != Company || x.Pos == target || Distance(x.Pos, target) != 1 {
			continue
		}
		switch {
		case x.Faction == u.Faction && !x.Spent:
			attackers = append(attackers, x)
		case x.Faction != u.Faction:
			defenders = append(defenders, x)
		}
	}
	return attackers, defenders
}

// ResolveBattle applies the result of a fight fought in a tactical game.
func (s *State) ResolveBattle(u *Unit, target Coord, o Outcome) (Report, error) {
	if err := s.CanAttack(u, target); err != nil {
		return Report{}, err
	}
	attackers, defenders := s.Engagement(u, target)
	u.Pos = s.AttackTargets(u)[target]
	losses := outcomeLosses[o]
	return s.settle(u, attackers, defenders, target, o,
		ceilPercent(totalStrength(attackers), losses[0]),
		ceilPercent(totalStrength(defenders), losses[1])), nil
}
