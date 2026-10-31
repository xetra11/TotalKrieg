package mission

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/xetra11/TotalKrieg/internal/game"
)

// staffs returns the battalions of companies.
func staffs(companies ...*game.Element) []*game.Element {
	var out []*game.Element
	for _, c := range companies {
		out = append(out, c.Parent)
	}
	return out
}

// battalion returns the companies of a new battalion of a faction.
func battalion(s *game.State, f game.Faction, pos game.Coord) []*game.Element {
	return s.Place(f, game.Battalion, pos).Element.Children
}

func count(units []Platoon, want string) int {
	n := 0
	for _, p := range units {
		if p.Unit == want {
			n++
		}
	}
	return n
}

func emptyState() *game.State {
	s := game.New(3, game.Wehrmacht)
	s.Units = nil
	s.Objects = s.Objects[:0]
	return s
}

func TestCompanyCountsCarryOver(t *testing.T) {
	s := emptyState()
	own := battalion(s, game.Wehrmacht, game.Coord{Col: 3, Row: 3})[:2]
	foe := battalion(s, game.SovietUnion, game.Coord{Col: 15, Row: 3})[:3]
	b := Build(Battle{Round: 1, Faction: game.Wehrmacht, Player: own, Enemy: foe})

	p, e := b.Sides.Player.Platoons, b.Sides.Enemy.Platoons
	if got := count(p, "wer_mibn_1141_comcoy"); got != 2 {
		t.Errorf("%d German company HQs, want 2", got)
	}
	if got := count(p, "wer_mibn_1141_plat"); got != 6 {
		t.Errorf("%d German rifle platoons, want 6", got)
	}
	if got := count(e, "ra_rbn_1242_comcoy"); got != 3 {
		t.Errorf("%d Soviet company HQs, want 3", got)
	}
	if got := count(e, "ra_rbn_1242_plat2"); got != 9 {
		t.Errorf("%d Soviet rifle platoons, want 9", got)
	}
}

func TestEngagementDecidesTheForces(t *testing.T) {
	s := emptyState()
	target := game.Coord{Col: 12, Row: 8}
	n := target.Neighbors()
	own := s.Place(game.Wehrmacht, game.Battalion, game.Coord{Col: 4, Row: 2}).Element.Children
	foe := s.Place(game.SovietUnion, game.Battalion, game.Coord{Col: 20, Row: 2}).Element.Children
	own[0].Token.Pos = n[0]
	own[1].Token.Pos = n[1]
	foe[0].Token.Pos = target
	foe[1].Token.Pos = n[3]
	foe[2].Token.Pos = n[4]

	b := NewBattle(s, own[0].Token, target)
	if len(b.Player) != 2 || len(b.Enemy) != 3 {
		t.Fatalf("%d vs %d companies, want 2 vs 3", len(b.Player), len(b.Enemy))
	}
	if len(b.PlayerStaffs) != 0 || len(b.EnemyStaffs) != 0 {
		t.Errorf("staffs out of range joined: %d, %d", len(b.PlayerStaffs), len(b.EnemyStaffs))
	}
}

func TestLinkedStaffJoinsTheMission(t *testing.T) {
	s := emptyState()
	staff := s.Place(game.Wehrmacht, game.Battalion, game.Coord{Col: 5, Row: 5})
	c := staff.Element.Children[0].Token
	target := c.Pos.Neighbors()[0]
	if s.UnitAt(target) != nil {
		target = c.Pos.Neighbors()[3]
	}
	if s.UnitAt(target) != nil {
		t.Skip("no free neighbor in this layout")
	}
	s.Place(game.SovietUnion, game.Company, target)
	b := NewBattle(s, c, target)
	if len(b.PlayerStaffs) != 1 {
		t.Fatalf("%d staffs, want 1", len(b.PlayerStaffs))
	}
	if got := count(Build(b).Sides.Player.Platoons, "wer_mibn_1141_combat"); got != 1 {
		t.Errorf("%d battalion HQ platoons, want 1", got)
	}
}

func TestStaffAsTargetIsFieldedAsHQ(t *testing.T) {
	s := emptyState()
	u := s.Place(game.Wehrmacht, game.Company, game.Coord{Col: 5, Row: 5})
	staff := s.Place(game.SovietUnion, game.Battalion, game.Coord{Col: 15, Row: 5})
	staff.Pos = game.Coord{Col: 6, Row: 5}
	for _, c := range staff.Element.Children {
		c.Token.Pos = game.Coord{Col: 20, Row: 2}
	}
	b := NewBattle(s, u, staff.Pos)
	if len(b.Enemy) != 0 || len(b.EnemyStaffs) != 1 {
		t.Fatalf("%d companies, %d staffs", len(b.Enemy), len(b.EnemyStaffs))
	}
	if got := count(Build(b).Sides.Enemy.Platoons, "ra_rbn_1242_combat"); got != 1 {
		t.Errorf("%d battalion HQ platoons, want 1", got)
	}
}

func TestRolesMapToTheirPlatoons(t *testing.T) {
	s := emptyState()
	soviet := battalion(s, game.SovietUnion, game.Coord{Col: 15, Row: 5})
	german := battalion(s, game.Wehrmacht, game.Coord{Col: 5, Row: 5})
	b := Build(Battle{Faction: game.SovietUnion, Player: soviet[3:], Enemy: german[3:]})
	own := b.Sides.Player.Platoons
	if count(own, "ra_rbn_1242_mg_plat") != 1 || count(own, "ra_rbn_1242_mt82_plat") != 1 {
		t.Errorf("Soviet special companies: %+v", own)
	}
	if count(b.Sides.Enemy.Platoons, "wer_mibn_1141_mg_plat1") != 1 {
		t.Errorf("German MG company: %+v", b.Sides.Enemy.Platoons)
	}
}

func TestWeakCompanyFieldsFewerPlatoons(t *testing.T) {
	s := emptyState()
	c := battalion(s, game.Wehrmacht, game.Coord{Col: 5, Row: 5})[0]
	c.Current = 1
	b := Build(Battle{Faction: game.Wehrmacht, Player: []*game.Element{c}, Enemy: battalion(s, game.SovietUnion, game.Coord{Col: 15, Row: 5})[:1]})
	if got := count(b.Sides.Player.Platoons, "wer_mibn_1141_plat"); got != 1 {
		t.Errorf("%d rifle platoons at strength 1 of 3, want 1", got)
	}
	if got := Describe(c); got != "Company HQ, 1 rifle platoon" {
		t.Errorf("Describe = %q", got)
	}
}

func TestBaseMatchesPlayerNation(t *testing.T) {
	s := emptyState()
	g := battalion(s, game.Wehrmacht, game.Coord{Col: 5, Row: 5})[:1]
	r := battalion(s, game.SovietUnion, game.Coord{Col: 15, Row: 5})[:1]
	if got := Build(Battle{Faction: game.Wehrmacht, Player: g, Enemy: r}).Base; got != "tr06" {
		t.Errorf("German base %s", got)
	}
	if got := Build(Battle{Faction: game.SovietUnion, Player: r, Enemy: g}).Base; got != "tr02" {
		t.Errorf("Soviet base %s", got)
	}
}

func TestLimitsHoldAndRatioSurvives(t *testing.T) {
	s := emptyState()
	g := battalion(s, game.Wehrmacht, game.Coord{Col: 5, Row: 5})
	g = append(g, battalion(s, game.Wehrmacht, game.Coord{Col: 5, Row: 12})...)
	r := battalion(s, game.SovietUnion, game.Coord{Col: 15, Row: 5})
	r = append(r, battalion(s, game.SovietUnion, game.Coord{Col: 15, Row: 12})...)
	for own := 1; own <= 6; own++ {
		for foe := 1; foe <= 6; foe++ {
			b := Build(Battle{Faction: game.Wehrmacht, Player: g[:own], Enemy: r[:foe], PlayerStaffs: staffs(g[0], g[5]), EnemyStaffs: staffs(r[0], r[5])})
			p, e := len(b.Sides.Player.Platoons), len(b.Sides.Enemy.Platoons)
			if p < 1 || e < 1 || p > maxPlatoonsPerSide || e > maxPlatoonsPerSide {
				t.Errorf("%d vs %d companies: %d vs %d platoons", own, foe, p, e)
			}
		}
	}
}

func TestCallsignsAndTextsAreValid(t *testing.T) {
	s := emptyState()
	r := battalion(s, game.SovietUnion, game.Coord{Col: 15, Row: 5})
	g := battalion(s, game.Wehrmacht, game.Coord{Col: 5, Row: 5})
	b := Build(Battle{Round: 7, Hex: game.Coord{Col: 3, Row: 4}, Faction: game.SovietUnion, Player: r, Enemy: g, PlayerStaffs: staffs(r[0]), EnemyStaffs: staffs(g[0])})
	seen := map[string]bool{}
	for _, p := range append(b.Sides.Player.Platoons, b.Sides.Enemy.Platoons...) {
		if seen[p.Callsign] {
			t.Errorf("callsign %s used twice", p.Callsign)
		}
		seen[p.Callsign] = true
	}
	data, err := json.Marshal(b)
	if err != nil {
		t.Fatal(err)
	}
	for _, c := range string(data) {
		if c > 127 {
			t.Fatalf("non-ASCII character %q in the scenario", c)
		}
	}
	for _, c := range b.ID {
		if !strings.ContainsRune("abcdefghijklmnopqrstuvwxyz0123456789_", c) {
			t.Fatalf("invalid id %q", b.ID)
		}
	}
}

// TestGeneratorAcceptsScenarios runs the real MiusMissionMaker without starting
// the game. It needs the tools folder and an installed Mius-Front.
func TestGeneratorAcceptsScenarios(t *testing.T) {
	cfg := Config{ToolDir: os.Getenv("TK_MISSIONMAKER"), GameDir: os.Getenv("TK_GAME"), OutDir: t.TempDir()}
	if !cfg.Available() {
		t.Skip("set TK_MISSIONMAKER to run this test")
	}
	s := emptyState()
	g := battalion(s, game.Wehrmacht, game.Coord{Col: 5, Row: 5})
	g = append(g, battalion(s, game.Wehrmacht, game.Coord{Col: 5, Row: 12})...)
	r := battalion(s, game.SovietUnion, game.Coord{Col: 15, Row: 5})
	r = append(r, battalion(s, game.SovietUnion, game.Coord{Col: 15, Row: 12})...)
	sizes := [][2]int{{1, 1}, {2, 3}, {3, 2}, {4, 5}, {5, 4}, {6, 6}, {1, 6}, {6, 1}}
	for _, f := range []game.Faction{game.Wehrmacht, game.SovietUnion} {
		for _, n := range sizes {
			b := Battle{Round: 1, Faction: f, Player: g[:n[0]], Enemy: r[:n[1]], PlayerStaffs: staffs(g[0]), EnemyStaffs: staffs(r[0]), Fortified: n[0] > n[1]}
			if f == game.SovietUnion {
				b.Player, b.Enemy = r[:n[0]], g[:n[1]]
			}
			sc := Build(b)
			sc.ID += fmt.Sprintf("_%d_%d_%d", f, n[0], n[1])
			res, err := Prepare(cfg, sc)
			if err != nil {
				t.Fatalf("%s: %v\n%s", sc.ID, err, res.Output)
			}
			if _, err := os.Stat(res.MissionFile); err != nil {
				t.Fatalf("%s: no mission file", sc.ID)
			}
			if strings.Contains(res.Output, "note:") {
				t.Logf("%s: %s", sc.ID, res.Output)
			}
		}
	}
}

func TestRosterNamesThePlatoons(t *testing.T) {
	s := game.New(3, game.Wehrmacht)
	staff := s.Place(game.Wehrmacht, game.Battalion, game.Coord{Col: 5, Row: 5})
	var rifle, mg *game.Element
	for _, c := range staff.Element.Children {
		switch c.Role {
		case game.RoleMG:
			mg = c
		default:
			rifle = c
		}
	}
	got := Build(Battle{
		Faction:      game.Wehrmacht,
		Player:       []*game.Element{rifle, mg},
		PlayerStaffs: []*game.Element{staff.Element},
		Enemy:        []*game.Element{rifle},
	}).Roster
	want := []string{"1st Rifle Platoon", "MG Platoon", "2nd Rifle Platoon", "3rd Rifle Platoon", "Company HQ", "Company HQ", "Battalion HQ"}
	for i, label := range want {
		if got[i].Label != label {
			t.Errorf("platoon %d is %q, want %q", i, got[i].Label, label)
		}
	}
	if got[1].Owner != mg || got[6].Owner != staff.Element {
		t.Errorf("owners %v, %v", got[1].Owner, got[6].Owner)
	}
}

func TestInstalledLayout(t *testing.T) {
	dir := t.TempDir()
	c := Installed(dir, "")
	if c.Available() {
		t.Fatal("found tools in an empty folder")
	}
	tools := filepath.Join(dir, "MiusMissionMaker", "tools")
	if err := os.MkdirAll(tools, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(tools, "scenario.js"), nil, 0o644); err != nil {
		t.Fatal(err)
	}
	if !c.Available() {
		t.Fatal("installed tools not found")
	}
	if c.Node != filepath.Join(dir, "node", "node.exe") {
		t.Errorf("node at %s", c.Node)
	}
	if strings.HasPrefix(c.OutDir, dir) {
		t.Errorf("missions go to the program folder: %s", c.OutDir)
	}
}

func TestMissingNodeIsReported(t *testing.T) {
	dir := t.TempDir()
	c := Config{ToolDir: dir, OutDir: dir, Node: filepath.Join(dir, "no-node.exe")}
	if _, err := Prepare(c, Scenario{ID: "test"}); err == nil {
		t.Error("ran without Node.js")
	}
}
