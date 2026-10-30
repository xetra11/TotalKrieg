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

// clearBoard empties the map so a test can set up its own position.
func clearBoard(a *App) {
	a.st.Units = nil
	a.st.Objects = a.st.Objects[:0]
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

	var city *game.Object
	for _, o := range st.Objects {
		if o.Owner == game.Wehrmacht && o.Kind == game.Stadt {
			city = o
		}
	}

	// Move the garrison out, then raise a company in the same turn.
	blocker := st.UnitAt(city.Pos)
	clickHexAt(a, city.Pos)
	if a.selUnit != blocker || len(a.reach) == 0 {
		t.Fatal("garrison not selected for a move")
	}
	var to game.Coord
	for c, d := range a.reach {
		if d == 1 {
			to = c
			break
		}
	}
	clickHexAt(a, to)
	if blocker.Pos != to {
		t.Fatalf("unit did not move: %v, message %q", blocker.Pos, a.message)
	}
	if blocker.MovesLeft != 1 || a.selUnit != blocker {
		t.Errorf("after one field: moves %d, selected %v", blocker.MovesLeft, a.selUnit)
	}

	a.cancel()
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

// startAttack selects a company and clicks an enemy two fields away.
func startAttack(t *testing.T, a *App) (*game.Unit, *game.Unit) {
	t.Helper()
	st := a.st
	clearBoard(a)
	u := st.Place(game.Wehrmacht, game.Kompanie, game.Coord{Col: 5, Row: 5})
	e := st.Place(game.Sowjetunion, game.Kompanie, game.Coord{Col: 7, Row: 5})
	clickHexAt(a, u.Pos)
	clickHexAt(a, e.Pos)
	if a.attack == nil || a.attack.unit != u || a.attack.target != e.Pos {
		t.Fatalf("no attack prompt, message %q", a.message)
	}
	return u, e
}

func TestMoveOntoEnemyOpensPromptAndAutoResolves(t *testing.T) {
	a := newApp(t, game.Wehrmacht)
	u, _ := startAttack(t, a)
	if len(a.st.Reports) != 0 {
		t.Fatal("fight resolved before the choice")
	}
	clickRect(a, autoResolveButton())
	if a.attack != nil {
		t.Error("prompt still open")
	}
	if len(a.st.Reports) != 1 {
		t.Fatalf("no fight, message %q", a.message)
	}
	if u.MovesLeft != 0 {
		t.Errorf("moves left %d", u.MovesLeft)
	}
}

func TestCancelledAttackChangesNothing(t *testing.T) {
	a := newApp(t, game.Wehrmacht)
	u, _ := startAttack(t, a)
	pos := u.Pos
	clickRect(a, cancelAttackButton())
	if a.attack != nil || len(a.st.Reports) != 0 {
		t.Fatalf("attack %v, reports %d", a.attack, len(a.st.Reports))
	}
	if u.Pos != pos || u.MovesLeft != 3 {
		t.Errorf("unit changed: %v, moves %d", u.Pos, u.MovesLeft)
	}
}

func TestTacticalOptionsDoNothingYet(t *testing.T) {
	a := newApp(t, game.Wehrmacht)
	startAttack(t, a)
	for i := range tacticalOptions {
		clickRect(a, tacticalRow(i))
	}
	if a.attack == nil || len(a.st.Reports) != 0 {
		t.Fatalf("attack %v, reports %d", a.attack, len(a.st.Reports))
	}
}

func TestEnterResolvesPromptNotTurn(t *testing.T) {
	a := newApp(t, game.Wehrmacht)
	startAttack(t, a)
	a.confirm()
	if a.st.Active != game.Wehrmacht || len(a.st.Reports) != 1 {
		t.Fatalf("active %v, reports %d", a.st.Active, len(a.st.Reports))
	}
}

func TestPromptBlocksTheMap(t *testing.T) {
	a := newApp(t, game.Wehrmacht)
	u, _ := startAttack(t, a)
	clickRect(a, nextButton())
	if a.st.Active != game.Wehrmacht {
		t.Error("turn ended behind the prompt")
	}
	a.cancel()
	if a.attack != nil || a.selUnit != u {
		t.Errorf("attack %v, selection %v", a.attack, a.selUnit)
	}
}

// openBattalion leaves one selected battalion on an empty map.
func openBattalion(t *testing.T, a *App) *game.Unit {
	t.Helper()
	clearBoard(a)
	u := a.st.Place(game.Wehrmacht, game.Bataillon, game.Coord{Col: 10, Row: 8})
	clickHexAt(a, u.Pos)
	if a.selUnit != u {
		t.Fatal("battalion not selected")
	}
	return u
}

func TestOrgRowsCoverEveryElement(t *testing.T) {
	a := newApp(t, game.Wehrmacht)
	st := a.st
	st.Units = nil
	for _, f := range []game.Faction{game.Wehrmacht, game.Sowjetunion} {
		d := st.Place(f, game.Division, game.Coord{Col: int(f), Row: 0})
		rows := orgRows(d.Element)
		want := map[game.Faction]int{game.Wehrmacht: 3 + 9 + 36, game.Sowjetunion: 3 + 9 + 45}[f]
		if len(rows) != want {
			t.Errorf("%v: %d rows, want %d", f, len(rows), want)
		}
		for _, r := range rows {
			if r.r.y+r.r.h > orgPanel.y+orgPanel.h || r.r.x+r.r.w > orgPanel.x+orgPanel.w {
				t.Fatalf("%v: row %s leaves the window", f, r.e.Short)
			}
		}
	}
}

func detachRow(u *game.Unit, e *game.Element) orgRow {
	for _, r := range orgRows(u.Element.Root()) {
		if r.e == e {
			return r
		}
	}
	return orgRow{}
}

func TestDetachAndReattachByClicks(t *testing.T) {
	a := newApp(t, game.Wehrmacht)
	st := a.st
	u := openBattalion(t, a)
	company := u.Element.Children[0]

	clickRect(a, orgButton())
	if !a.showOrg {
		t.Fatal("order of battle did not open")
	}
	clickRect(a, detachRow(u, company).detachButton())
	if a.pendingDetach != company || a.showOrg {
		t.Fatalf("detach not pending, message %q", a.message)
	}
	to := game.Coord{Col: 11, Row: 8}
	clickHexAt(a, to)
	k := st.UnitAt(to)
	if k == nil || k.Element != company {
		t.Fatalf("company not placed, message %q", a.message)
	}
	if a.selUnit != k || len(a.reach) == 0 {
		t.Fatal("detachment not selected for its move")
	}

	clickRect(a, reattachButton())
	if st.UnitAt(to) != nil || a.selUnit != u {
		t.Fatalf("company did not rejoin, message %q", a.message)
	}
	if u.Strength() != 12 {
		t.Errorf("strength %d", u.Strength())
	}
}

func TestDetachCancelsOnOtherHex(t *testing.T) {
	a := newApp(t, game.Wehrmacht)
	u := openBattalion(t, a)
	clickRect(a, orgButton())
	clickRect(a, detachRow(u, u.Element.Children[0]).detachButton())
	clickHexAt(a, game.Coord{Col: 2, Row: 2})
	if a.pendingDetach != nil || len(a.st.Units) != 1 {
		t.Fatalf("pending %v, units %d", a.pendingDetach, len(a.st.Units))
	}
}

func TestDetachedRowSelectsItsUnit(t *testing.T) {
	a := newApp(t, game.Wehrmacht)
	st := a.st
	u := openBattalion(t, a)
	k, err := st.Detach(u.Element.Children[0], game.Coord{Col: 11, Row: 8})
	if err != nil {
		t.Fatal(err)
	}
	a.selectUnit(u)
	a.openOrg()
	clickRect(a, detachRow(u, k.Element).r)
	if a.selUnit != k || a.showOrg {
		t.Fatalf("selected %v, window open %v", a.selUnit, a.showOrg)
	}
}

func TestNoActionsDuringAITurn(t *testing.T) {
	a := newApp(t, game.Wehrmacht)
	st := a.st
	st.EndTurn()
	clickRect(a, nextButton())
	if st.Active != game.Sowjetunion {
		t.Errorf("turn passed during the AI turn: %v", st.Active)
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
