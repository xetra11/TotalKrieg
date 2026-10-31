package mission

import (
	"math"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/xetra11/TotalKrieg/internal/game"
)

// testdata holds a real battle: mission tk_r02_j10, 8 German against 7 Soviet
// platoons, ended as MINOR VICTORY with three Soviet rifle platoons lost.
func j10(t *testing.T) ([]byte, string) {
	t.Helper()
	data, err := os.ReadFile("testdata/tacback_j10.cfgpack")
	if err != nil {
		t.Fatal(err)
	}
	log, err := os.ReadFile("testdata/engine_j10.log")
	if err != nil {
		t.Fatal(err)
	}
	return data, string(log)
}

func roster(n int) []RosterEntry {
	out := make([]RosterEntry, n)
	for i := range out {
		out[i] = RosterEntry{Owner: &game.Element{Kind: game.Company}, Label: "Platoon"}
	}
	return out
}

func TestParseRealResult(t *testing.T) {
	data, log := j10(t)
	r := roster(15)
	res, err := parseResult(data, log, r)
	if err != nil {
		t.Fatal(err)
	}
	if res.Text != "MINOR VICTORY" || res.Outcome != game.AttackerWins {
		t.Fatalf("result %q, outcome %v", res.Text, res.Outcome)
	}
	// pine_01, pine_03 and pine_04 are the lost rifle platoons.
	for _, i := range []int{8, 10, 11} {
		if got := res.Remaining[r[i].Owner]; got != 0 && i != 11 {
			t.Errorf("lost platoon %d keeps %.2f", i, got)
		}
	}
	if got := res.Remaining[r[11].Owner]; got > 0.2 {
		t.Errorf("pine_04 keeps %.2f, nearly all squads are lost", got)
	}
	if got := res.Remaining[r[0].Owner]; got < 0.9 || got > 1 {
		t.Errorf("panther_01 at %.2f", got)
	}
	if got := res.Remaining[r[14].Owner]; got < 0.9 {
		t.Errorf("Soviet battalion HQ at %.2f, it took no losses", got)
	}
}

func TestSharedOwnerAveragesItsPlatoons(t *testing.T) {
	data, log := j10(t)
	r := roster(15)
	r[10] = r[8]  // two lost rifle platoons of one company
	r[12] = r[14] // an HQ that lost one of two squads, with an unharmed staff
	res, err := parseResult(data, log, r)
	if err != nil {
		t.Fatal(err)
	}
	if got := res.Remaining[r[8].Owner]; got != 0 {
		t.Errorf("two lost platoons leave %.2f", got)
	}
	if got := res.Remaining[r[14].Owner]; got < 0.6 || got > 0.9 {
		t.Errorf("mixed owner at %.2f", got)
	}
}

func TestOtherMissionIsRefused(t *testing.T) {
	data, log := j10(t)
	if _, err := parseResult(data, log, roster(4)); err == nil {
		t.Error("accepted a result for another mission")
	}
}

func TestMissingLogLineIsAnError(t *testing.T) {
	data, _ := j10(t)
	if _, err := parseResult(data, "nothing here", roster(15)); err == nil {
		t.Error("accepted a log without a result")
	}
}

func TestLastResultInLogCounts(t *testing.T) {
	text, code, err := tacticalResult("Tactical result : MINOR DEFEAT [2]\nx\n [t] Tactical result : MINOR VICTORY [4]\n")
	if err != nil || text != "MINOR VICTORY" || code != 4 {
		t.Fatalf("%q %d %v", text, code, err)
	}
}

func TestOutcomeCodes(t *testing.T) {
	for code, want := range map[int]game.Outcome{2: game.DefenderWins, 3: game.Draw, 4: game.AttackerWins} {
		if got := outcomeOf(code); got != want {
			t.Errorf("code %d = %v", code, got)
		}
	}
}

func TestReadResultWaitsForTheBattle(t *testing.T) {
	data, log := j10(t)
	dir := t.TempDir()
	cfg := Config{GameDir: dir}
	s := Scenario{Roster: roster(15)}

	if res, err := ReadResult(cfg, s, time.Now()); res != nil || err != nil {
		t.Fatalf("no file yet: %v, %v", res, err)
	}
	path := filepath.Join(dir, resultFile)
	logPath := filepath.Join(dir, engineLog)
	for _, p := range []string{path, logPath} {
		if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
			t.Fatal(err)
		}
	}
	if err := os.WriteFile(path, data, 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(logPath, []byte(log), 0o644); err != nil {
		t.Fatal(err)
	}

	if res, err := ReadResult(cfg, s, time.Now().Add(time.Hour)); res != nil || err != nil {
		t.Fatalf("stale file: %v, %v", res, err)
	}
	res, err := ReadResult(cfg, s, time.Now().Add(-time.Hour))
	if err != nil || res == nil {
		t.Fatalf("finished battle: %v, %v", res, err)
	}
	if math.IsNaN(res.Remaining[s.Roster[0].Owner]) {
		t.Error("NaN strength")
	}
}

func TestPlatoonLossesOfARealBattle(t *testing.T) {
	data, log := j10(t)
	r := roster(15)
	res, err := parseResult(data, log, r)
	if err != nil {
		t.Fatal(err)
	}
	if len(res.Platoons) != 15 {
		t.Fatalf("%d platoons", len(res.Platoons))
	}
	lost, squads := 0, 0
	for _, p := range res.Platoons {
		lost, squads = lost+p.Lost, squads+p.Squads
		if p.Lost == p.Squads && p.Condition != 0 {
			t.Errorf("lost platoon in condition %.2f", p.Condition)
		}
	}
	if lost != 16 || squads != 62 {
		t.Errorf("%d of %d squads lost", lost, squads)
	}
	if p := res.Platoons[8]; p.Owner != r[8].Owner || p.Lost != 5 || p.Squads != 5 {
		t.Errorf("pine_01 %+v", p)
	}
	if p := res.Platoons[12]; p.Lost != 1 || p.Squads != 2 || p.Condition < 0.9 {
		t.Errorf("pine_05 %+v", p)
	}
}
