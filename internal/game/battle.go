package game

import "errors"

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
	return s.ApplyFightResult(&Fight{Unit: u, Target: target}, res)
}

// ApplyFightResult books a fight fought in a tactical game with its real losses.
// It uses the exact forces of the fight, including companies that attacked the
// same hex and the defender's chosen support.
func (s *State) ApplyFightResult(f *Fight, res *FieldResult) (Report, error) {
	if err := s.CanAttack(f.Unit, f.Target); err != nil {
		return Report{}, err
	}
	s.stageAttackers(f)
	attackers, defenders := s.engagement(f.Unit, f.Target, f.Others, !f.NoSupport)
	if len(defenders) == 0 {
		return Report{}, errors.New("Nothing to attack.")
	}
	d := defenders[0]
	r := Report{
		Round:           s.Round,
		Attacker:        fightLabel(attackers),
		Defender:        fightLabel(defenders),
		AttackerFaction: f.Unit.Faction,
		Pos:             f.Target,
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
	r.AttackerDestroyed = f.Unit.Strength() <= 0
	r.DefenderDestroyed = d.Strength() <= 0
	s.addReport(r)
	s.afterFight(f.Unit, attackers, defenders, f.Target, res.Outcome)
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
	return s.engagement(u, target, nil, true)
}

// FightForces lists the exact units that would fight for a planned fight. It
// also counts the further companies ordered onto the same target, so the
// attacker's strength is combined even before they step up.
func (s *State) FightForces(f *Fight) (attackers, defenders []*Unit) {
	return s.engagement(f.Unit, f.Target, f.Others, !f.NoSupport)
}

// engagement collects the forces of a fight. Others are further companies
// whose order points at the target; support decides whether the defender's
// nearby companies join.
func (s *State) engagement(u *Unit, target Coord, others []*Unit, support bool) (attackers, defenders []*Unit) {
	attackers = []*Unit{u}
	seen := map[*Unit]bool{u: true}
	for _, x := range others {
		if x == nil || seen[x] || x.Faction != u.Faction || x.Kind != Company || x.Spent || x.Strength() <= 0 {
			continue
		}
		seen[x] = true
		attackers = append(attackers, x)
	}
	if d := s.UnitAt(target); d != nil && d.Faction != u.Faction {
		defenders = []*Unit{d}
	}
	for _, x := range s.Units {
		if x == u || seen[x] || x.Kind != Company || x.Pos == target || Distance(x.Pos, target) != 1 {
			continue
		}
		switch {
		case x.Faction == u.Faction:
			if !x.Spent && x.Strength() > 0 {
				seen[x] = true
				attackers = append(attackers, x)
			}
		case support:
			defenders = append(defenders, x)
		}
	}
	return attackers, defenders
}

// SupportCandidates lists the defender's companies next to a hex that could
// join the battle.
func (s *State) SupportCandidates(u *Unit, target Coord) []*Unit {
	d := s.UnitAt(target)
	if d == nil || d.Faction == u.Faction {
		return nil
	}
	var out []*Unit
	for _, x := range s.Units {
		if x.Kind == Company && x.Faction == d.Faction && x.Pos != target && Distance(x.Pos, target) == 1 {
			out = append(out, x)
		}
	}
	return out
}

// stageAttackers moves the ordered companies next to the target, each onto its
// own free hex, so a combined attack can be shown and fought as one battle.
func (s *State) stageAttackers(f *Fight) {
	taken := map[Coord]bool{}
	for _, u := range append([]*Unit{f.Unit}, f.Others...) {
		if u.Kind != Company || u.Faction != f.Unit.Faction || u.Spent || u.Strength() <= 0 {
			continue
		}
		if Distance(u.Pos, f.Target) == 1 && !taken[u.Pos] {
			taken[u.Pos] = true
			continue
		}
		if _, ok := s.AttackTargets(u)[f.Target]; !ok {
			continue
		}
		spot, ok := s.attackSpot(u, f.Target, taken)
		if !ok {
			continue
		}
		u.Pos = spot
		taken[spot] = true
	}
}

// attackSpot finds a free hex next to the target that the unit can reach.
func (s *State) attackSpot(u *Unit, target Coord, taken map[Coord]bool) (Coord, bool) {
	reach := s.Reachable(u)
	reach[u.Pos] = 0
	budget, bestDist := u.MovesLeft(), u.MovesLeft()
	var best Coord
	found := false
	for _, c := range sortedCoords(reach) {
		d := reach[c]
		if d >= budget || d >= bestDist || taken[c] || Distance(c, target) != 1 {
			continue
		}
		if s.UnitAt(c) != nil && c != u.Pos {
			continue
		}
		best, bestDist, found = c, d, true
	}
	return best, found
}

// ResolveBattle applies the result of a fight fought in a tactical game.
func (s *State) ResolveBattle(u *Unit, target Coord, o Outcome) (Report, error) {
	return s.ResolveFight(&Fight{Unit: u, Target: target}, o)
}

// ResolveFight applies the result of a fight fought in a tactical game.
func (s *State) ResolveFight(f *Fight, o Outcome) (Report, error) {
	if err := s.CanAttack(f.Unit, f.Target); err != nil {
		return Report{}, err
	}
	s.stageAttackers(f)
	attackers, defenders := s.engagement(f.Unit, f.Target, f.Others, !f.NoSupport)
	if len(defenders) == 0 {
		return Report{}, errors.New("Nothing to attack.")
	}
	losses := outcomeLosses[o]
	return s.settle(f.Unit, attackers, defenders, f.Target, o,
		ceilPercent(totalStrength(attackers), losses[0]),
		ceilPercent(totalStrength(defenders), losses[1])), nil
}

// CanWithdraw explains why the defender can't pull back from an attack, or
// returns nil. A unit may do this only once; a later attack must be fought.
func (s *State) CanWithdraw(f *Fight) error {
	if f == nil || f.Unit == nil {
		return errors.New("No attack.")
	}
	if err := s.CanAttack(f.Unit, f.Target); err != nil {
		return err
	}
	d := s.UnitAt(f.Target)
	switch {
	case d == nil || d.Faction == f.Unit.Faction:
		return errors.New("Nothing to withdraw from.")
	case d.Withdrew:
		return errors.New("This unit already withdrew once.")
	case len(s.WithdrawOptions(f)) == 0:
		return errors.New("No free hex to withdraw to.")
	}
	return nil
}

// WithdrawOptions lists the free neighboring hexes farther from the attacker.
func (s *State) WithdrawOptions(f *Fight) []Coord {
	d := s.UnitAt(f.Target)
	if d == nil || f.Unit == nil {
		return nil
	}
	var out []Coord
	for _, c := range d.Pos.Neighbors() {
		if OnMap(c) && s.UnitAt(c) == nil && Distance(c, f.Unit.Pos) > Distance(d.Pos, f.Unit.Pos) {
			out = append(out, c)
		}
	}
	return out
}

// WithdrawFight pulls the defender back instead of fighting. The attacker takes
// the vacated hex without losses. The defender can do this only once.
func (s *State) WithdrawFight(f *Fight) (Report, error) {
	if err := s.CanWithdraw(f); err != nil {
		return Report{}, err
	}
	d := s.UnitAt(f.Target)
	options := s.WithdrawOptions(f)
	to := options[0]
	for _, c := range options[1:] {
		if Distance(c, f.Unit.Pos) > Distance(to, f.Unit.Pos) {
			to = c
		}
	}
	vacated := d.Pos
	d.Pos = to
	d.Withdrew = true
	f.Unit.Pos = vacated
	f.Unit.Spent = true
	for _, x := range f.Others {
		x.Spent = true
	}
	r := Report{
		Round:           s.Round,
		Attacker:        fightLabel(append([]*Unit{f.Unit}, f.Others...)),
		Defender:        d.Short,
		AttackerFaction: f.Unit.Faction,
		Withdrew:        true,
		Pos:             vacated,
	}
	s.addReport(r)
	return r, nil
}
