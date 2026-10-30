package game

import "sort"

const (
	threatRange = 2
	// attackRatioTenths is the 1.2 strength ratio from the GDD, in tenths.
	attackRatioTenths = 12
)

// objectWeight ranks capture targets: capital, city, factory, village.
var objectWeight = map[ObjectKind]float64{
	Hauptstadt: 8,
	Stadt:      5,
	Fabrik:     3,
	Dorf:       1,
}

// PlayTurn plays the active faction's whole turn with the AI priorities from
// the GDD: defend, attack, capture, raise units.
func (s *State) PlayTurn() {
	if s.Over {
		return
	}
	s.aiRecruit()
	s.Advance()
	s.aiMove()
	s.Advance()
	s.aiAttack()
	s.Advance()
}

// threatenedCities returns own cities with an enemy within threat range.
func (s *State) threatenedCities() []*Object {
	var out []*Object
	for _, o := range s.Objects {
		if o.Owner == s.Active && o.Kind.IsCity() && s.enemyNear(o.Pos) {
			out = append(out, o)
		}
	}
	return out
}

func (s *State) enemyNear(c Coord) bool {
	for _, u := range s.Units {
		if u.Faction != s.Active && Distance(u.Pos, c) <= threatRange {
			return true
		}
	}
	return false
}

// aiRecruit raises companies in threatened cities. Every other city saves for
// a division, so one occupied city can't freeze all recruiting.
func (s *State) aiRecruit() {
	for _, o := range s.threatenedCities() {
		s.Recruit(o, Kompanie)
	}
	for _, o := range s.ownCitiesByFront() {
		if _, err := s.Recruit(o, Division); err == nil {
			return
		}
	}
}

// ownCitiesByFront sorts own cities, the ones nearest an enemy object first.
func (s *State) ownCitiesByFront() []*Object {
	var out []*Object
	for _, o := range s.Objects {
		if o.Owner == s.Active && o.Kind.IsCity() {
			out = append(out, o)
		}
	}
	sort.SliceStable(out, func(i, j int) bool {
		return s.distanceToEnemyObject(out[i].Pos) < s.distanceToEnemyObject(out[j].Pos)
	})
	return out
}

func (s *State) distanceToEnemyObject(c Coord) int {
	best := MapCols + MapRows
	for _, o := range s.Objects {
		if o.Owner != s.Active {
			best = min(best, Distance(c, o.Pos))
		}
	}
	return best
}

func (s *State) aiMove() {
	busy := map[*Unit]bool{}
	for _, o := range s.threatenedCities() {
		if u := s.nearestFreeUnit(o.Pos, busy); u != nil {
			busy[u] = true
			s.moveToward(u, o.Pos)
		}
	}
	for _, u := range append([]*Unit(nil), s.Units...) {
		if u.Faction != s.Active || busy[u] || u.Moved || u.Fresh {
			continue
		}
		if s.hasWorthwhileTarget(u) {
			continue
		}
		if target, ok := s.bestObjective(u); ok {
			s.moveToward(u, target)
		}
	}
}

func (s *State) nearestFreeUnit(c Coord, busy map[*Unit]bool) *Unit {
	var best *Unit
	for _, u := range s.Units {
		if u.Faction != s.Active || busy[u] || u.Fresh || u.Moved {
			continue
		}
		if best == nil || Distance(u.Pos, c) < Distance(best.Pos, c) {
			best = u
		}
	}
	return best
}

// moveToward moves a unit to the reachable hex nearest the target.
func (s *State) moveToward(u *Unit, target Coord) {
	bestPos, bestDist := u.Pos, Distance(u.Pos, target)
	for _, c := range sortedCoords(s.Reachable(u)) {
		if d := Distance(c, target); d < bestDist {
			bestPos, bestDist = c, d
		}
	}
	if bestPos != u.Pos {
		s.Move(u, bestPos)
	}
}

// sortedCoords keeps AI moves deterministic for a given seed.
func sortedCoords(m map[Coord]int) []Coord {
	out := make([]Coord, 0, len(m))
	for c := range m {
		out = append(out, c)
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].Row != out[j].Row {
			return out[i].Row < out[j].Row
		}
		return out[i].Col < out[j].Col
	})
	return out
}

func (s *State) hasWorthwhileTarget(u *Unit) bool {
	return s.bestAttackTarget(u) != nil
}

// bestAttackTarget returns the weakest adjacent enemy worth attacking.
func (s *State) bestAttackTarget(u *Unit) *Unit {
	var best *Unit
	for _, n := range u.Pos.Neighbors() {
		e := s.UnitAt(n)
		if e == nil || e.Faction == u.Faction {
			continue
		}
		if u.Strength()*10 < e.Strength()*attackRatioTenths {
			continue
		}
		if best == nil || e.Strength() < best.Strength() {
			best = e
		}
	}
	return best
}

// bestObjective picks the enemy object with the best weight for its distance.
func (s *State) bestObjective(u *Unit) (Coord, bool) {
	var best Coord
	bestScore := 0.0
	for _, o := range s.Objects {
		if o.Owner == s.Active {
			continue
		}
		score := objectWeight[o.Kind] / float64(Distance(u.Pos, o.Pos)+1)
		if score > bestScore {
			best, bestScore = o.Pos, score
		}
	}
	return best, bestScore > 0
}

func (s *State) aiAttack() {
	for _, u := range append([]*Unit(nil), s.Units...) {
		if u.Faction != s.Active || u.Strength() <= 0 {
			continue
		}
		if t := s.bestAttackTarget(u); t != nil {
			s.Attack(u, t.Pos)
		}
	}
}
