package game

import (
	"iter"
	"sort"
)

const (
	threatRange = 2
	// attackRatioTenths is the 1.2 strength ratio from the GDD, in tenths.
	attackRatioTenths = 12
)

// objectWeight ranks capture targets: capital, city, factory, village.
var objectWeight = map[ObjectKind]float64{
	Capital: 8,
	City:    5,
	Factory: 3,
	Village: 1,
}

// PlayTurn plays the active faction's whole turn and resolves every fight
// automatically.
func (s *State) PlayTurn() {
	for range s.AITurn() {
	}
}

// AITurn plays the active faction's turn with the AI priorities from the GDD:
// defend, attack, capture, raise units. It stops at every fight.
func (s *State) AITurn() iter.Seq[*Fight] {
	return func(yield func(*Fight) bool) {
		if s.Over {
			return
		}
		s.fleeAll()
		s.aiRecruit()
		s.aiMoveStaffs()
		s.aiDefend()
		if !s.aiAttack(yield) {
			return
		}
		s.aiAdvance()
		if !s.aiAttack(yield) {
			return
		}
		s.aiMoveStaffs()
		s.EndTurn()
	}
}

// aiMoveStaffs keeps every staff in the middle of its companies.
func (s *State) aiMoveStaffs() {
	for _, u := range s.Units {
		if u.Faction != s.Active || u.Kind != Battalion {
			continue
		}
		col, row, n := 0, 0, 0
		for _, t := range u.Element.Tokens() {
			if t != u {
				col, row, n = col+t.Pos.Col, row+t.Pos.Row, n+1
			}
		}
		if n > 0 {
			s.moveToward(u, Coord{col / n, row / n})
		}
	}
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
// a battalion, so one occupied city can't freeze all recruiting.
func (s *State) aiRecruit() {
	for _, o := range s.threatenedCities() {
		s.Recruit(o, Company)
	}
	for _, o := range s.ownCitiesByFront() {
		if _, err := s.Recruit(o, Battalion); err == nil {
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

// aiDefend sends the nearest unit to every threatened city.
func (s *State) aiDefend() {
	busy := map[*Unit]bool{}
	for _, o := range s.threatenedCities() {
		if u := s.nearestFreeUnit(o.Pos, busy); u != nil {
			busy[u] = true
			s.moveToward(u, o.Pos)
		}
	}
}

// aiAdvance moves the remaining companies toward the best objective.
func (s *State) aiAdvance() {
	for _, u := range append([]*Unit(nil), s.Units...) {
		if u.Faction != s.Active || u.Kind != Company || u.MovesLeft() == 0 {
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
		if u.Faction != s.Active || u.Kind != Company || busy[u] || u.MovesLeft() == 0 {
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

// bestAttackTarget returns the weakest enemy in reach worth attacking.
func (s *State) bestAttackTarget(u *Unit) *Unit {
	var best *Unit
	targets := s.AttackTargets(u)
	for _, c := range sortedCoords(attackCoords(targets)) {
		e := s.UnitAt(c)
		if u.Strength()*10 < e.Strength()*attackRatioTenths {
			continue
		}
		if best == nil || e.Strength() < best.Strength() {
			best = e
		}
	}
	return best
}

func attackCoords(targets map[Coord]Coord) map[Coord]int {
	out := make(map[Coord]int, len(targets))
	for c := range targets {
		out[c] = 0
	}
	return out
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

func (s *State) aiAttack(yield func(*Fight) bool) bool {
	for _, u := range append([]*Unit(nil), s.Units...) {
		if u.Faction != s.Active || u.Strength() <= 0 {
			continue
		}
		if t := s.bestAttackTarget(u); t != nil && !s.fight(u, t.Pos, yield) {
			return false
		}
	}
	return true
}
