package game

import "sort"

// frontLine returns the forward column of a faction's front line and the column
// of its battalion staffs behind it. Two empty columns separate the two lines.
func frontLine(f Faction) (front, back int) {
	half := MapCols / 2
	if f == Wehrmacht {
		return half - 2, half - 3
	}
	return half + 1, half + 2
}

// deployFront moves the starting battalions from their cities to the front
// line. Neither side has to march across its own half in the opening turns.
func (s *State) deployFront() {
	used := map[Coord]bool{}
	for _, f := range []Faction{Wehrmacht, SovietUnion} {
		s.deployFaction(f, used)
	}
}

// deployFaction spreads one faction's battalions along its front line. Each
// battalion holds a north-to-south section of the line, ordered by its home
// city, so staff and companies stay together.
func (s *State) deployFaction(f Faction, used map[Coord]bool) {
	var battalions []*Element
	for _, e := range s.elements {
		if e.Faction == f && e.Kind == Battalion && e.Token != nil {
			battalions = append(battalions, e)
		}
	}
	if len(battalions) == 0 {
		return
	}
	sort.SliceStable(battalions, func(i, j int) bool {
		return s.homeRow(battalions[i]) < s.homeRow(battalions[j])
	})

	front, back := frontLine(f)
	reserve := back + (back - front)
	for i, b := range battalions {
		lo := i * MapRows / len(battalions)
		hi := (i + 1) * MapRows / len(battalions)
		center := (lo + hi) / 2

		staff := s.deployInBand(back, center, lo, hi, 1, used)
		if len(staff) == 0 {
			staff = s.deployColumn(back, center, 1, used)
		}
		if len(staff) > 0 {
			b.Token.Pos = staff[0]
			center = staff[0].Row
		}

		want := min(len(b.Children), MaxBattalionCompanies)
		spots := s.deployInBand(front, center, lo, hi, want, used)
		if len(spots) < want {
			spots = append(spots, s.deployInBand(reserve, center, lo, hi, want-len(spots), used)...)
		}
		if len(spots) < want {
			spots = append(spots, s.deployColumn(front, center, want-len(spots), used)...)
		}
		if len(spots) < want {
			spots = append(spots, s.deployColumn(reserve, center, want-len(spots), used)...)
		}
		for j, c := range spots {
			if j >= len(b.Children) {
				break
			}
			if tok := b.Children[j].Token; tok != nil {
				tok.Pos = c
			}
		}
	}
}

// deployInBand returns up to n free hexes in a column between rows lo and hi,
// nearest to center first.
func (s *State) deployInBand(col, center, lo, hi, n int, used map[Coord]bool) []Coord {
	var out []Coord
	for d := 0; d < hi-lo && len(out) < n; d++ {
		rows := []int{center - d}
		if d > 0 {
			rows = append(rows, center+d)
		}
		for _, row := range rows {
			if len(out) >= n {
				break
			}
			if row < lo || row >= hi {
				continue
			}
			c := Coord{col, row}
			if s.ObjectAt(c) != nil || used[c] {
				continue
			}
			used[c] = true
			out = append(out, c)
		}
	}
	return out
}

// deployColumn returns up to n free hexes in a column, nearest to center first.
// It is the fallback when a battalion's own section is blocked.
func (s *State) deployColumn(col, center, n int, used map[Coord]bool) []Coord {
	var out []Coord
	for d := 0; d < MapRows && len(out) < n; d++ {
		rows := []int{center - d}
		if d > 0 {
			rows = append(rows, center+d)
		}
		for _, row := range rows {
			if len(out) >= n {
				break
			}
			if row < 0 || row >= MapRows {
				continue
			}
			c := Coord{col, row}
			if s.ObjectAt(c) != nil || used[c] {
				continue
			}
			used[c] = true
			out = append(out, c)
		}
	}
	return out
}

// homeRow is the row of a battalion's home city, used to order the front.
func (s *State) homeRow(b *Element) int {
	if b.Home != nil {
		return b.Home.Pos.Row
	}
	if b.Token != nil {
		return b.Token.Pos.Row
	}
	return 0
}
