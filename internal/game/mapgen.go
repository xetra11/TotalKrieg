package game

import "math/rand/v2"

const (
	// MapCols and MapRows are the map size from the GDD.
	MapCols = 24
	MapRows = 16

	minObjectGap      = 2
	maxCityImbalance  = 2.0
	maxPlacementTries = 1000
)

// generateObjects places every object for both factions. Layouts whose cities
// sit unevenly far from the center line are rolled again.
func generateObjects(rng *rand.Rand) []*Object {
	for {
		if objs, ok := tryGenerate(rng); ok && balanced(objs) {
			return objs
		}
	}
}

func tryGenerate(rng *rand.Rand) ([]*Object, bool) {
	var objs []*Object
	for _, f := range []Faction{Wehrmacht, SovietUnion} {
		names := cityNames[f]
		plan := []struct {
			kind  ObjectKind
			count int
		}{{Capital, 1}, {City, 2}, {Factory, 2}, {Village, 5}}
		next := 0
		for _, p := range plan {
			for i := 0; i < p.count; i++ {
				pos, ok := placeObject(rng, f, p.kind, objs)
				if !ok {
					return nil, false
				}
				name := p.kind.String()
				if p.kind.IsCity() {
					name = names[next]
					next++
				}
				objs = append(objs, &Object{Kind: p.kind, Pos: pos, Owner: f, Name: name})
			}
		}
	}
	return objs, true
}

func placeObject(rng *rand.Rand, f Faction, kind ObjectKind, objs []*Object) (Coord, bool) {
	for try := 0; try < maxPlacementTries; try++ {
		c := randomSpot(rng, f, kind)
		if farEnough(c, objs) {
			return c, true
		}
	}
	return Coord{}, false
}

// randomSpot picks a hex on the faction's half. Capitals stay in the rear half
// of that side.
func randomSpot(rng *rand.Rand, f Faction, kind ObjectKind) Coord {
	half := MapCols / 2
	col := rng.IntN(half)
	if kind == Capital {
		col = rng.IntN(half / 2)
	}
	if f == SovietUnion {
		col = MapCols - 1 - col
	}
	return Coord{col, rng.IntN(MapRows)}
}

func farEnough(c Coord, objs []*Object) bool {
	for _, o := range objs {
		if Distance(c, o.Pos) < minObjectGap {
			return false
		}
	}
	return true
}

func balanced(objs []*Object) bool {
	diff := meanCityDistance(objs, Wehrmacht) - meanCityDistance(objs, SovietUnion)
	if diff < 0 {
		diff = -diff
	}
	return diff <= maxCityImbalance
}

// meanCityDistance averages how far a faction's cities sit from the center line.
func meanCityDistance(objs []*Object, f Faction) float64 {
	center := float64(MapCols-1) / 2
	sum, n := 0.0, 0
	for _, o := range objs {
		if o.Owner != f || !o.Kind.IsCity() {
			continue
		}
		d := float64(o.Pos.Col) - center
		if d < 0 {
			d = -d
		}
		sum += d
		n++
	}
	return sum / float64(n)
}
