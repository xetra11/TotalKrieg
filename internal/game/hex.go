package game

// Coord addresses a hex by column and row on a pointy-top grid with odd rows
// shifted right.
type Coord struct {
	Col, Row int
}

// oddRowNeighbors holds the neighbor offsets per row parity.
var oddRowNeighbors = [2][6]Coord{
	{{1, 0}, {0, -1}, {-1, -1}, {-1, 0}, {-1, 1}, {0, 1}},
	{{1, 0}, {1, -1}, {0, -1}, {-1, 0}, {0, 1}, {1, 1}},
}

// Neighbors returns the six adjacent coordinates, on or off the map.
func (c Coord) Neighbors() [6]Coord {
	var out [6]Coord
	for i, d := range oddRowNeighbors[c.Row&1] {
		out[i] = Coord{c.Col + d.Col, c.Row + d.Row}
	}
	return out
}

// Distance returns the number of hex steps between two coordinates.
func Distance(a, b Coord) int {
	ax, az, ay := cube(a)
	bx, bz, by := cube(b)
	return (abs(ax-bx) + abs(ay-by) + abs(az-bz)) / 2
}

func cube(c Coord) (x, z, y int) {
	x = c.Col - (c.Row-(c.Row&1))/2
	z = c.Row
	y = -x - z
	return x, z, y
}

func abs(v int) int {
	if v < 0 {
		return -v
	}
	return v
}
