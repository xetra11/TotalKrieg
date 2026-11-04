package game

import "math"

// Controller returns the faction that controls a hex. Control goes to the
// nearest living unit or object, so moving a unit into enemy ground visibly
// pushes the front. Units win ties against objects, and ties between units go
// to the lower faction number so the line stays stable.
func (s *State) Controller(c Coord) Faction {
	best := math.MaxInt
	owner := NoFaction
	for _, u := range s.Units {
		if u == nil {
			continue
		}
		d := Distance(c, u.Pos)
		if d < best || (d == best && owner != NoFaction && u.Faction < owner) {
			best, owner = d, u.Faction
		}
	}
	for _, o := range s.Objects {
		if o == nil {
			continue
		}
		if d := Distance(c, o.Pos); d < best {
			best, owner = d, o.Owner
		}
	}
	return owner
}
