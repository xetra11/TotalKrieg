package ui

import (
	"testing"

	"github.com/xetra11/TotalKrieg/internal/game"
)

func newApp(t *testing.T, f game.Faction) *App {
	t.Helper()
	a, err := New(Options{Seed: 7, Faction: &f})
	if err != nil {
		t.Fatal(err)
	}
	return a
}

func clickHexAt(a *App, c game.Coord) {
	a.click(hexCenter(c))
}

func clickRect(a *App, r rect) {
	a.click(r.x+r.w/2, r.y+r.h/2)
}

func TestHexHitTest(t *testing.T) {
	for row := 0; row < game.MapRows; row++ {
		for col := 0; col < game.MapCols; col++ {
			c := game.Coord{Col: col, Row: row}
			x, y := hexCenter(c)
			got, ok := hexAt(x, y)
			if !ok || got != c {
				t.Fatalf("hexAt(center of %v) = %v, %v", c, got, ok)
			}
		}
	}
	if _, ok := hexAt(5, 5); ok {
		t.Error("corner of the screen hit a hex")
	}
}

func TestMapFitsMapArea(t *testing.T) {
	x, y := hexCenter(game.Coord{Col: game.MapCols - 1, Row: 1})
	if x+hexW/2 > mapW {
		t.Errorf("map ends at x=%.0f", x+hexW/2)
	}
	if _, y = hexCenter(game.Coord{Row: game.MapRows - 1}); y+hexSize > hintY {
		t.Errorf("map ends at y=%.0f", y+hexSize)
	}
}

func TestMenuStartsChosenFaction(t *testing.T) {
	a, err := New(Options{Seed: 7})
	if err != nil {
		t.Fatal(err)
	}
	clickRect(a, factionButton(game.Sowjetunion))
	if a.st == nil || a.st.First != game.Sowjetunion {
		t.Fatalf("state = %+v", a.st)
	}
}

func TestPlayerTurnByClicks(t *testing.T) {
	a := newApp(t, game.Wehrmacht)
	st := a.st

	// Free a city, then raise a company through the panel.
	var city *game.Object
	for _, o := range st.Objects {
		if o.Owner == game.Wehrmacht && o.Kind == game.Stadt {
			city = o
		}
	}
	st.Phase = game.PhaseBewegung
	blocker := st.UnitAt(city.Pos)
	a.selUnit = blocker
	a.reach = st.Reachable(blocker)
	var to game.Coord
	for c := range a.reach {
		to = c
		break
	}
	clickHexAt(a, to)
	if blocker.Pos != to {
		t.Fatalf("unit did not move: %v, message %q", blocker.Pos, a.message)
	}
	st.Phase = game.PhaseAufstellung

	clickHexAt(a, city.Pos)
	if a.selCity != city {
		t.Fatal("city not selected")
	}
	clickRect(a, recruitButton(0))
	u := st.UnitAt(city.Pos)
	if u == nil || u.Kind != game.Kompanie {
		t.Fatalf("no company raised, message %q", a.message)
	}
	if st.Stocks[game.Wehrmacht].Manpower != 2 {
		t.Errorf("manpower = %d", st.Stocks[game.Wehrmacht].Manpower)
	}

	clickRect(a, nextButton())
	if st.Phase != game.PhaseBewegung {
		t.Fatalf("phase = %v", st.Phase)
	}
	clickRect(a, nextButton())
	if st.Phase != game.PhaseAngriff {
		t.Fatalf("phase = %v", st.Phase)
	}
	clickRect(a, nextButton())
	if st.Active != game.Sowjetunion {
		t.Fatalf("active = %v", st.Active)
	}

	for i := 0; i < aiDelayFrames; i++ {
		a.tick()
	}
	if st.Active != game.Wehrmacht || st.Round != 2 {
		t.Fatalf("after AI: active %v, round %d", st.Active, st.Round)
	}
}

func TestAttackByClicks(t *testing.T) {
	a := newApp(t, game.Wehrmacht)
	st := a.st
	st.Units = nil
	st.Objects = st.Objects[:0]
	st.Phase = game.PhaseAngriff
	u := &game.Unit{Faction: game.Wehrmacht, Kind: game.Division, Pos: game.Coord{Col: 5, Row: 5}, Strength: 48, Name: "A", Short: "A"}
	e := &game.Unit{Faction: game.Sowjetunion, Kind: game.Kompanie, Pos: game.Coord{Col: 6, Row: 5}, Strength: 3, Name: "B", Short: "B"}
	st.Units = []*game.Unit{u, e}

	clickHexAt(a, u.Pos)
	if a.selUnit != u {
		t.Fatal("attacker not selected")
	}
	clickHexAt(a, e.Pos)
	if len(st.Reports) != 1 {
		t.Fatalf("no fight, message %q", a.message)
	}
}

func TestNoActionsDuringAITurn(t *testing.T) {
	a := newApp(t, game.Wehrmacht)
	st := a.st
	for st.Active == game.Wehrmacht {
		st.Advance()
	}
	clickRect(a, nextButton())
	if st.Phase != game.PhaseAufstellung {
		t.Errorf("phase moved during the AI turn: %v", st.Phase)
	}
	a.selCity = nil
	for _, o := range st.Objects {
		if o.Owner == game.Sowjetunion && o.Kind.IsCity() {
			clickHexAt(a, o.Pos)
			clickRect(a, recruitButton(0))
		}
	}
	if st.Stocks[game.Sowjetunion].Manpower != 3 {
		t.Errorf("player spent the AI's manpower: %d", st.Stocks[game.Sowjetunion].Manpower)
	}
}
