package game

import (
	"math"
	"testing"
)

func TestStartingUnitsDeployToFront(t *testing.T) {
	center := float64(MapCols-1) / 2
	for seed := uint64(0); seed < 300; seed++ {
		s := New(seed, Wehrmacht)

		seen := map[Coord]bool{}
		for _, u := range s.Units {
			if seen[u.Pos] {
				t.Errorf("seed %d: two units share %v", seed, u.Pos)
			}
			seen[u.Pos] = true
			if math.Abs(float64(u.Pos.Col)-center) > 4 {
				t.Errorf("seed %d: %s starts in the rear at %v", seed, u.Name, u.Pos)
			}
			if s.ObjectAt(u.Pos) != nil {
				t.Errorf("seed %d: %s starts on an object at %v", seed, u.Name, u.Pos)
			}
			if u.Kind == Company && !u.Linked() {
				t.Errorf("seed %d: %s starts without a link", seed, u.Name)
			}
		}

		// The two lines keep a no man's land of at least two hexes.
		for _, a := range s.Units {
			for _, b := range s.Units {
				if a.Faction == b.Faction {
					continue
				}
				if d := Distance(a.Pos, b.Pos); d < 3 {
					t.Errorf("seed %d: front lines touch: %v and %v are %d apart", seed, a.Pos, b.Pos, d)
				}
			}
		}

		for _, o := range s.Objects {
			if !o.Kind.IsCity() {
				continue
			}
			if u := s.UnitAt(o.Pos); u != nil {
				t.Errorf("seed %d: city %s starts occupied by %s", seed, o.Name, u.Name)
			}
			if s.CityBattalion(o) == nil {
				t.Errorf("seed %d: city %s lost its battalion", seed, o.Name)
			}
		}
	}
}

func TestFrontLineIsSymmetric(t *testing.T) {
	west, westBack := frontLine(Wehrmacht)
	east, eastBack := frontLine(SovietUnion)

	if west >= MapCols/2 || east < MapCols/2 {
		t.Errorf("front lines not on their own halves: %d and %d", west, east)
	}
	if westBack != west-1 || eastBack != east+1 {
		t.Errorf("staff lines not behind the front: %d/%d, %d/%d", west, westBack, east, eastBack)
	}
	if west+east != MapCols-1 {
		t.Errorf("front lines not mirrored around the center: %d and %d", west, east)
	}
	if d := Distance(Coord{west, 8}, Coord{east, 8}); d < 3 {
		t.Errorf("front lines only %d hexes apart", d)
	}
}
