package ui

import (
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/xetra11/TotalKrieg/internal/game"
	"github.com/xetra11/TotalKrieg/internal/mission"
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
	clickRect(a, factionButton(game.SovietUnion))
	if a.st == nil || a.st.First != game.SovietUnion {
		t.Fatalf("state = %+v", a.st)
	}
}

func TestPlayerTurnByClicks(t *testing.T) {
	a := newApp(t, game.Wehrmacht)
	st := a.st

	var city *game.Object
	for _, o := range st.Objects {
		if o.Owner == game.Wehrmacht && o.Kind == game.City {
			city = o
		}
	}

	// An order changes nothing until the turn is committed.
	staff := st.UnitAt(city.Pos)
	clickHexAt(a, city.Pos)
	if a.selUnit != staff || len(a.reach) == 0 {
		t.Fatal("staff not selected for a move")
	}
	var to game.Coord
	for c := range a.reach {
		to = c
		break
	}
	clickHexAt(a, to)
	if staff.Pos != city.Pos || st.OrderOf(staff) == nil || a.selUnit != staff {
		t.Fatalf("staff at %v, order %v, message %q", staff.Pos, st.OrderOf(staff), a.message)
	}
	if !strings.Contains(a.unitState(staff), "move to") {
		t.Errorf("state %q", a.unitState(staff))
	}
	clickHexAt(a, staff.Pos)
	if st.OrderOf(staff) != nil {
		t.Fatal("clicking the unit kept its order")
	}
	clickHexAt(a, to)

	clickRect(a, nextButton())
	if staff.Pos != to || st.Active != game.SovietUnion {
		t.Fatalf("staff at %v, active %v", staff.Pos, st.Active)
	}
	for i := 0; i < aiDelayFrames; i++ {
		a.tick()
	}
	if st.Active != game.Wehrmacht || st.Round != 2 {
		t.Fatalf("after AI: active %v, round %d", st.Active, st.Round)
	}

	clickHexAt(a, city.Pos)
	if a.selCity != city {
		t.Fatal("city not selected")
	}
	clickRect(a, recruitButton(0))
	u := st.UnitAt(city.Pos)
	if u == nil || u.Kind != game.Company {
		t.Fatalf("no company raised, message %q", a.message)
	}
	if st.Stocks[game.Wehrmacht].Manpower != 5 {
		t.Errorf("manpower = %d", st.Stocks[game.Wehrmacht].Manpower)
	}
}

// startAttack orders a linked company against an enemy two hexes away and
// commits the turn.
func startAttack(t *testing.T, a *App) (*game.Unit, *game.Unit) {
	t.Helper()
	st := a.st
	clearBoard(a)
	staff := st.Place(game.Wehrmacht, game.Battalion, game.Coord{Col: 5, Row: 7})
	for _, c := range staff.Element.Children[1:] {
		c.Token.Pos = game.Coord{Col: c.Token.Pos.Col - 4, Row: c.Token.Pos.Row}
	}
	u := staff.Element.Children[0].Token
	u.Pos = game.Coord{Col: 5, Row: 5}
	e := st.Place(game.SovietUnion, game.Company, game.Coord{Col: 7, Row: 5})
	clickHexAt(a, u.Pos)
	clickHexAt(a, e.Pos)
	if o := st.OrderOf(u); o == nil || !o.Attack || a.attack != nil {
		t.Fatalf("order %v, attack %v, message %q", o, a.attack, a.message)
	}
	clickRect(a, nextButton())
	if a.attack == nil || a.attack.Unit != u || a.attack.Target != e.Pos {
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
	if u.MovesLeft() != 0 {
		t.Errorf("moves left %d", u.MovesLeft())
	}
}

func TestStaffCantStartAnAttack(t *testing.T) {
	a := newApp(t, game.Wehrmacht)
	clearBoard(a)
	staff := a.st.Place(game.Wehrmacht, game.Battalion, game.Coord{Col: 5, Row: 5})
	e := a.st.Place(game.SovietUnion, game.Company, game.Coord{Col: 5, Row: 9})
	staff.Pos = game.Coord{Col: 5, Row: 8}
	clickHexAt(a, staff.Pos)
	clickHexAt(a, e.Pos)
	if a.st.OrderOf(staff) != nil {
		t.Error("staff got an attack order")
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
	if u.Pos != pos || u.MovesLeft() != 3 {
		t.Errorf("unit changed: %v, moves %d", u.Pos, u.MovesLeft())
	}
	a.tick()
	if a.st.Active != game.SovietUnion {
		t.Errorf("turn did not go on: %v", a.st.Active)
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

// waitForMission lets the generator goroutine finish.
func waitForMission(t *testing.T, a *App) {
	t.Helper()
	for i := 0; i < 2000; i++ {
		a.tick()
		if a.battle == nil || a.battle.done == nil {
			return
		}
		time.Sleep(time.Millisecond)
	}
	t.Fatal("mission never finished")
}

func TestPreviewComesBeforeTheGame(t *testing.T) {
	a := newApp(t, game.Wehrmacht)
	starts := 0
	a.runMission = func(mission.Scenario) error {
		starts++
		return nil
	}
	startAttack(t, a)
	clickRect(a, missionButton())
	if a.battle == nil || a.battle.running || a.battle.done != nil {
		t.Fatalf("no preview: %+v", a.battle)
	}
	if len(a.battle.own) != 2 || len(a.battle.foe) != 1 {
		t.Errorf("mapping rows %v, %v", a.battle.own, a.battle.foe)
	}

	if starts != 0 || len(a.st.Reports) != 0 {
		t.Fatalf("starts %d, reports %d during the preview", starts, len(a.st.Reports))
	}

	clickRect(a, backButton())
	if a.battle != nil || a.attack == nil {
		t.Fatalf("battle %v, attack %v", a.battle, a.attack)
	}
	clickRect(a, missionButton())
	a.cancel()
	if a.battle != nil || a.attack == nil {
		t.Fatal("Esc did not leave the preview")
	}
}

func TestMissionResolveByClicks(t *testing.T) {
	a := newApp(t, game.Wehrmacht)
	var got mission.Scenario
	a.runMission = func(s mission.Scenario) error {
		got = s
		return nil
	}
	u, e := startAttack(t, a)
	clickRect(a, missionButton())
	clickRect(a, startMissionButton())
	if a.battle == nil || a.battle.done == nil {
		t.Fatal("no battle started")
	}
	waitForMission(t, a)
	if !a.battle.running {
		t.Fatal("battle not running")
	}
	if got.Base != "tr06" || len(got.Sides.Player.Platoons) != 5 || len(got.Sides.Enemy.Platoons) != 4 {
		t.Fatalf("scenario %+v", got)
	}
	if len(a.st.Reports) != 0 {
		t.Fatal("fight resolved before the result")
	}

	a.confirm()
	a.cancel()
	clickRect(a, nextButton())
	if a.battle == nil || a.st.Active != game.Wehrmacht {
		t.Fatal("the running battle was left")
	}

	clickRect(a, outcomeButton(0))
	if a.battle != nil || a.attack != nil {
		t.Fatal("dialog still open")
	}
	if len(a.st.Reports) != 1 || a.st.UnitAt(e.Pos) != u {
		t.Fatalf("reports %d, on target %v", len(a.st.Reports), a.st.UnitAt(e.Pos))
	}
}

// runningBattle starts a mission and returns once the game would be open.
func runningBattle(t *testing.T, a *App) (*game.Unit, *game.Unit) {
	t.Helper()
	a.runMission = func(mission.Scenario) error { return nil }
	u, e := startAttack(t, a)
	clickRect(a, missionButton())
	clickRect(a, startMissionButton())
	waitForMission(t, a)
	return u, e
}

func letResultPoll(a *App) {
	for i := 0; i < resultCheckFrames; i++ {
		a.tick()
	}
}

func TestResultComesInByItself(t *testing.T) {
	a := newApp(t, game.Wehrmacht)
	var report *game.FieldResult
	a.readResult = func(mission.Scenario, time.Time) (*game.FieldResult, error) {
		return report, nil
	}
	u, e := runningBattle(t, a)
	target := e.Pos

	letResultPoll(a)
	if a.battle.result != nil {
		t.Fatal("result before the battle ended")
	}
	report = &game.FieldResult{
		Text:      "MINOR VICTORY",
		Outcome:   game.AttackerWins,
		Remaining: map[*game.Element]float64{u.Element: 0.7, e.Element: 0.5},
	}
	letResultPoll(a)
	if a.battle.result != report {
		t.Fatal("result not picked up")
	}
	s := a.battle.sides
	rows := lossRows(s.Player, s.PlayerStaffs, report)
	if len(rows) != 2 || !strings.Contains(rows[0].text, "3 → 2") {
		t.Errorf("rows %v", rows)
	}
	if len(a.st.Reports) != 0 {
		t.Fatal("result booked before the player saw it")
	}

	clickRect(a, applyResultButton())
	if a.battle != nil || a.attack != nil {
		t.Fatal("dialog still open")
	}
	if u.Strength() != 2 || e.Strength() != 1 || a.st.UnitAt(target) != u {
		t.Fatalf("strengths %d and %d, on target %v", u.Strength(), e.Strength(), a.st.UnitAt(target))
	}
	if len(a.st.Reports) != 1 {
		t.Fatalf("%d reports", len(a.st.Reports))
	}
}

func TestLossRowsListPlatoons(t *testing.T) {
	c := &game.Element{Kind: game.Company, Short: "1./514", Callsign: "Fuchs", Current: 3}
	left := &game.Element{Kind: game.Company, Short: "2./514", Callsign: "Dachs", Current: 3}
	staff := &game.Element{Kind: game.Battalion, Short: "I./514"}
	res := &game.FieldResult{
		Remaining: map[*game.Element]float64{c: 0.5, staff: 0},
		Platoons: []game.PlatoonLoss{
			{Owner: c, Name: "1st Rifle Platoon", Squads: 6, Lost: 6},
			{Owner: c, Name: "2nd Rifle Platoon", Squads: 6, Lost: 2, Condition: 0.9},
			{Owner: c, Name: "Company HQ", Squads: 5, Condition: 1},
			{Owner: staff, Name: "Battalion HQ", Squads: 3, Lost: 3},
		},
	}
	var got []string
	for _, r := range lossRows([]*game.Element{c, left}, []*game.Element{staff}, res) {
		got = append(got, r.text)
	}
	want := []string{
		"1./514 Fuchs: strength 3 → 2",
		"1st Rifle Platoon: all 6 squads lost",
		"2nd Rifle Platoon: 2 of 6 squads lost, the rest at 90% condition",
		"Company HQ: no squad lost, condition 100%",
		"2./514 Dachs: strength 3 → 2",
		"not in the mission, losses as the rest of its side",
		"I./514: staff lost",
		"Battalion HQ: all 3 squads lost",
	}
	if strings.Join(got, "\n") != strings.Join(want, "\n") {
		t.Errorf("rows\n%s", strings.Join(got, "\n"))
	}
	lost, squads := sideLoss([]*game.Element{c, left, staff}, res)
	if lost != 11 || squads != 20 {
		t.Errorf("side lost %d of %d", lost, squads)
	}
}

func TestLostAttackLetsThePlayerFlee(t *testing.T) {
	a := newApp(t, game.Wehrmacht)
	a.readResult = func(mission.Scenario, time.Time) (*game.FieldResult, error) {
		return nil, nil
	}
	u, e := runningBattle(t, a)
	from := a.st.AttackTargets(u)[e.Pos]
	target := e.Pos
	clickRect(a, outcomeButton(2))

	r := a.st.NextRout()
	if r == nil || r.Unit != u || a.selUnit != u || u.Pos != from {
		t.Fatalf("no rout: %+v, unit at %v", r, u.Pos)
	}
	if len(a.reach) != 3 {
		t.Fatalf("%d options", len(a.reach))
	}

	// Nothing else happens until the unit fled.
	a.confirm()
	a.cancel()
	a.toggleOrg()
	clickRect(a, nextButton())
	clickHexAt(a, target)
	if a.st.Active != game.Wehrmacht || a.showOrg || u.Pos != from || a.st.NextRout() == nil {
		t.Fatalf("rout left: active %v, unit at %v", a.st.Active, u.Pos)
	}

	var to game.Coord
	for c := range a.reach {
		to = c
		break
	}
	clickHexAt(a, to)
	if u.Pos != to || a.st.NextRout() != nil {
		t.Fatalf("unit at %v, want %v", u.Pos, to)
	}
	if e.Pos != target {
		t.Errorf("victorious defender moved to %v", e.Pos)
	}
}

// aiFight sets a strong AI company next to a player company and commits.
func aiFight(t *testing.T, seed uint64) (a *App, own, foe *game.Unit) {
	t.Helper()
	f := game.Wehrmacht
	a, err := New(Options{Seed: seed, Faction: &f})
	if err != nil {
		t.Fatal(err)
	}
	clearBoard(a)
	own = a.st.Place(game.Wehrmacht, game.Company, game.Coord{Col: 5, Row: 5})
	foe = a.st.Place(game.SovietUnion, game.Company, game.Coord{Col: 6, Row: 5})
	foe.Element.Current = 6
	clickRect(a, nextButton())
	return a, own, foe
}

func aiFrames(a *App) {
	for i := 0; i < aiDelayFrames+1; i++ {
		a.tick()
	}
}

func TestPlayerPicksTheHexDuringTheAITurn(t *testing.T) {
	// The fight is rolled, so look for a seed where the company survives it.
	for seed := uint64(1); seed < 100; seed++ {
		a, own, foe := aiFight(t, seed)
		hex, enemyHex := own.Pos, foe.Pos
		aiFrames(a)
		st := a.st
		if st.NextRout() == nil {
			continue
		}
		if st.Active != game.SovietUnion || own.Pos != hex || foe.Pos != enemyHex || a.selUnit != own {
			t.Fatalf("active %v, own %v, foe %v, selected %v", st.Active, own.Pos, foe.Pos, a.selUnit)
		}
		var to game.Coord
		for c := range a.reach {
			to = c
			break
		}
		clickHexAt(a, to)
		if own.Pos != to || foe.Pos != hex {
			t.Fatalf("own %v, foe %v", own.Pos, foe.Pos)
		}
		aiFrames(a)
		if st.Active != game.Wehrmacht || !own.Resting || own.MovesLeft() != 0 {
			t.Fatalf("active %v, resting %v, moves %d", st.Active, own.Resting, own.MovesLeft())
		}
		if !strings.Contains(a.unitState(own), "Beaten") {
			t.Errorf("state %q", a.unitState(own))
		}
		return
	}
	t.Fatal("no seed left a beaten company alive")
}

func TestAIAttackGoesToThePlayer(t *testing.T) {
	a, own, foe := aiFight(t, 7)
	a.runMission = func(mission.Scenario) error { return nil }
	aiFrames(a)
	if !a.incoming() || a.attack.Unit != foe || a.attack.Target != own.Pos {
		t.Fatalf("attack %+v", a.attack)
	}
	if !strings.Contains(a.promptTitle(), "enemy attacks") {
		t.Errorf("title %q", a.promptTitle())
	}

	// The AI's attack can't be refused, and the turn waits.
	a.cancel()
	clickRect(a, cancelAttackButton())
	aiFrames(a)
	if a.attack == nil || a.st.Active != game.SovietUnion || len(a.st.Reports) != 0 {
		t.Fatalf("attack %v, active %v, reports %d", a.attack, a.st.Active, len(a.st.Reports))
	}

	clickRect(a, missionButton())
	if s := a.battle.scenario; !s.Defending || s.Base != "tr04" || s.Sides.Player.Platoons[0].Callsign != "panther_01" {
		t.Fatalf("scenario %+v", s)
	}
	clickRect(a, startMissionButton())
	waitForMission(t, a)

	// Victory is the player's: the attacker fails and the defender stays.
	hex := own.Pos
	clickRect(a, outcomeButton(0))
	if a.attack != nil || a.st.UnitAt(hex) != own || !foe.Beaten {
		t.Fatalf("attack %v, on hex %v, beaten %v", a.attack, a.st.UnitAt(hex), foe.Beaten)
	}
	aiFrames(a)
	if a.st.Active != game.Wehrmacht {
		t.Fatalf("active %v", a.st.Active)
	}
}

func TestAIAttackIsAutomaticWithoutTheTools(t *testing.T) {
	a, _, _ := aiFight(t, 7)
	aiFrames(a)
	if a.attack != nil || len(a.st.Reports) != 1 {
		t.Fatalf("attack %v, reports %d", a.attack, len(a.st.Reports))
	}
}

func TestMissionFailureReturnsToThePreview(t *testing.T) {
	a := newApp(t, game.Wehrmacht)
	calls := 0
	a.runMission = func(mission.Scenario) error {
		calls++
		if calls == 1 {
			return errors.New("error: Mius-Front is still running")
		}
		return nil
	}
	startAttack(t, a)
	clickRect(a, missionButton())
	clickRect(a, startMissionButton())
	waitForMission(t, a)
	if a.battle == nil || a.battle.running {
		t.Fatalf("battle %+v", a.battle)
	}
	if !strings.Contains(a.battle.failure, "still running") {
		t.Errorf("failure %q", a.battle.failure)
	}

	clickRect(a, startMissionButton())
	waitForMission(t, a)
	if !a.battle.running || a.battle.failure != "" {
		t.Errorf("second start: %+v", a.battle)
	}
}

func TestMissionButtonNeedsTheTools(t *testing.T) {
	a := newApp(t, game.Wehrmacht)
	if a.runMission != nil {
		t.Fatal("tools found without a path")
	}
	startAttack(t, a)
	clickRect(a, missionButton())
	if a.battle != nil {
		t.Error("mission started without the tools")
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
	if a.attack != nil || u.MovesLeft() != 3 {
		t.Errorf("attack %v, moves %d", a.attack, u.MovesLeft())
	}
}

func TestOrgWindowListsBattalionAndSelectsCompany(t *testing.T) {
	a := newApp(t, game.Wehrmacht)
	clearBoard(a)
	staff := a.st.Place(game.Wehrmacht, game.Battalion, game.Coord{Col: 10, Row: 8})
	clickHexAt(a, staff.Pos)
	clickRect(a, orgButton())
	if !a.showOrg {
		t.Fatal("order of battle did not open")
	}
	rows := orgRows(staff.Element)
	if len(rows) != 5 {
		t.Fatalf("%d rows, want staff and 4 companies", len(rows))
	}
	for _, r := range rows {
		if r.r.y+r.r.h > orgPanel.y+orgPanel.h {
			t.Fatalf("row %s leaves the window", r.e.Short)
		}
	}
	clickRect(a, rows[2].r)
	if a.showOrg || a.selUnit != rows[2].e.Token {
		t.Fatalf("selected %v, window open %v", a.selUnit, a.showOrg)
	}
}

func TestMembershipShowsTheLink(t *testing.T) {
	a := newApp(t, game.Wehrmacht)
	clearBoard(a)
	staff := a.st.Place(game.Wehrmacht, game.Battalion, game.Coord{Col: 5, Row: 8})
	c := staff.Element.Children[0].Token
	if got := membership(c); !strings.Contains(got, "linked") {
		t.Errorf("membership %q", got)
	}
	c.Pos = game.Coord{Col: 20, Row: 8}
	if got := membership(c); !strings.Contains(got, "no staff link") {
		t.Errorf("membership %q", got)
	}
}

func TestNoActionsDuringAITurn(t *testing.T) {
	a := newApp(t, game.Wehrmacht)
	st := a.st
	st.EndTurn()
	clickRect(a, nextButton())
	if st.Active != game.SovietUnion {
		t.Errorf("turn passed during the AI turn: %v", st.Active)
	}
	for _, u := range st.Units {
		if u.Kind == game.Battalion && u.Faction == game.SovietUnion {
			st.Units = removeUnit(st.Units, u)
			break
		}
	}
	a.selCity = nil
	for _, o := range st.Objects {
		if o.Owner == game.SovietUnion && o.Kind.IsCity() {
			clickHexAt(a, o.Pos)
			clickRect(a, recruitButton(0))
		}
	}
	if st.Stocks[game.SovietUnion].Manpower != 3 {
		t.Errorf("player spent the AI's manpower: %d", st.Stocks[game.SovietUnion].Manpower)
	}
}

func removeUnit(units []*game.Unit, u *game.Unit) []*game.Unit {
	var out []*game.Unit
	for _, x := range units {
		if x != u {
			out = append(out, x)
		}
	}
	return out
}
