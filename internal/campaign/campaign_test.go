package campaign

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/xetra11/TotalKrieg/internal/game"
)

func testStore(t *testing.T) *Store {
	t.Helper()
	dir := filepath.Join(t.TempDir(), "campaigns")
	return NewStore(dir)
}

func TestNewAndSave(t *testing.T) {
	store := testStore(t)
	st := game.New(42, game.Wehrmacht)
	rec, err := store.New("Test Campaign", st)
	if err != nil {
		t.Fatal(err)
	}
	if rec.ID == "" || rec.Name != "Test Campaign" {
		t.Errorf("record = %+v", rec)
	}
	if rec.State.Seed != 42 || rec.State.First != game.Wehrmacht {
		t.Errorf("state = %+v", rec.State)
	}
	if rec.Created.IsZero() || rec.Updated.IsZero() {
		t.Error("timestamps not set")
	}

	entries, err := os.ReadDir(store.Dir)
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != 1 || !strings.HasSuffix(entries[0].Name(), ".json") {
		t.Fatalf("entries = %v", entries)
	}
	// ID should be based on sanitized name
	if !strings.HasPrefix(entries[0].Name(), "test-campaign") {
		t.Errorf("filename = %s", entries[0].Name())
	}
}

func TestLoadAndUpdate(t *testing.T) {
	store := testStore(t)
	st := game.New(7, game.SovietUnion)
	rec, err := store.New("Soviet Campaign", st)
	if err != nil {
		t.Fatal(err)
	}

	loaded, err := store.Load(rec.ID)
	if err != nil {
		t.Fatal(err)
	}
	if loaded.Name != rec.Name || loaded.State.Seed != 7 {
		t.Errorf("loaded = %+v", loaded)
	}

	st.EndTurn()
	st.EndTurn()
	snap, _ := st.Snapshot()
	rec.State = snap
	time.Sleep(time.Millisecond)
	if err := store.Save(&rec); err != nil {
		t.Fatal(err)
	}

	loaded, err = store.Load(rec.ID)
	if err != nil {
		t.Fatal(err)
	}
	if loaded.State.Round != 2 {
		t.Errorf("round = %d", loaded.State.Round)
	}
	if !loaded.Updated.After(rec.Created) {
		t.Error("updated not after created")
	}
}

func TestList(t *testing.T) {
	store := testStore(t)
	if list, err := store.List(); err != nil || len(list) != 0 {
		t.Fatalf("empty list = %v, %v", list, err)
	}

	st1 := game.New(1, game.Wehrmacht)
	rec1, _ := store.New("First Campaign", st1)
	time.Sleep(2 * time.Millisecond)
	st2 := game.New(2, game.SovietUnion)
	rec2, _ := store.New("Second Campaign", st2)

	list, err := store.List()
	if err != nil {
		t.Fatal(err)
	}
	if len(list) != 2 {
		t.Fatalf("%d campaigns", len(list))
	}
	// Most recent first
	if list[0].ID != rec2.ID || list[1].ID != rec1.ID {
		t.Errorf("order = %s, %s", list[0].ID, list[1].ID)
	}
	if list[0].Name != "Second Campaign" || list[0].Faction != game.SovietUnion {
		t.Errorf("summary = %+v", list[0])
	}
}

func TestListWithBrokenFile(t *testing.T) {
	store := testStore(t)
	st := game.New(1, game.Wehrmacht)
	store.New("Good Campaign", st)

	os.MkdirAll(store.Dir, 0o755)
	os.WriteFile(filepath.Join(store.Dir, "broken.json"), []byte("{bad json"), 0o644)

	list, err := store.List()
	if err != nil {
		t.Fatal(err)
	}
	if len(list) != 1 {
		t.Errorf("broken file not skipped: %d campaigns", len(list))
	}
}

func TestDelete(t *testing.T) {
	store := testStore(t)
	st := game.New(1, game.Wehrmacht)
	rec, _ := store.New("To Delete", st)

	if err := store.Delete(rec.ID); err != nil {
		t.Fatal(err)
	}
	if _, err := store.Load(rec.ID); !os.IsNotExist(err) {
		t.Errorf("load after delete: %v", err)
	}
	list, _ := store.List()
	if len(list) != 0 {
		t.Error("deleted campaign still in list")
	}
}

func TestSummary(t *testing.T) {
	store := testStore(t)
	st := game.New(5, game.Wehrmacht)
	for i := 0; i < 3; i++ {
		st.EndTurn()
	}
	st.Round = 10
	st.Over = true
	st.Winner = game.Wehrmacht
	st.Reason = "Test victory"

	rec, _ := store.New("Finished Campaign", st)
	s := rec.Summary()
	if s.ID != rec.ID || s.Name != "Finished Campaign" || s.Faction != game.Wehrmacht {
		t.Errorf("summary = %+v", s)
	}
	if s.Round != 10 || !s.Over || s.Winner != game.Wehrmacht {
		t.Errorf("state = round %d, over %v, winner %v", s.Round, s.Over, s.Winner)
	}
}

func TestInvalidID(t *testing.T) {
	store := testStore(t)
	for _, id := range []string{"", ".", "..", "/etc/passwd", "../other"} {
		if _, err := store.Load(id); err == nil {
			t.Errorf("load accepted bad id %q", id)
		}
		st := game.New(1, game.Wehrmacht)
		snap, _ := st.Snapshot()
		rec := Record{ID: id, State: snap}
		if err := store.Save(&rec); err == nil {
			t.Errorf("save accepted bad id %q", id)
		}
	}
}

func TestUniqueID(t *testing.T) {
	store := testStore(t)
	os.MkdirAll(store.Dir, 0o755)
	id1 := store.uniqueID("test")
	os.WriteFile(filepath.Join(store.Dir, id1+".json"), []byte("{}"), 0o644)
	id2 := store.uniqueID("test")
	if id1 == id2 {
		t.Error("uniqueID did not suffix a collision")
	}
	if !strings.HasPrefix(id2, id1) {
		t.Errorf("id2 %q does not start with %q", id2, id1)
	}
}

func TestStoreWithoutState(t *testing.T) {
	store := testStore(t)
	rec := Record{ID: "test", Name: "No state"}
	if err := store.Save(&rec); err == nil {
		t.Error("save without state succeeded")
	}
}

func TestDefaultDir(t *testing.T) {
	dir, err := DefaultDir()
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(dir, "TotalKrieg") || !strings.Contains(dir, "campaigns") {
		t.Errorf("dir = %q", dir)
	}
}

func TestSanitizeName(t *testing.T) {
	cases := []struct {
		input string
		want  string
	}{
		{"Test Campaign", "test-campaign"},
		{"Wehrmacht 1941", "wehrmacht-1941"},
		{"Test!!!Campaign", "test-campaign"},
		{"  Spaces  ", "spaces"},
		{"Multiple   Spaces", "multiple-spaces"},
		{"Special@#$Chars", "special-chars"},
		{"", ""},
	}
	for _, c := range cases {
		got := sanitizeName(c.input)
		if got != c.want {
			t.Errorf("sanitizeName(%q) = %q, want %q", c.input, got, c.want)
		}
	}
}

func TestSameNameCreatesUnique(t *testing.T) {
	store := testStore(t)
	st1 := game.New(1, game.Wehrmacht)
	rec1, _ := store.New("Same Name", st1)
	
	st2 := game.New(2, game.SovietUnion)
	rec2, _ := store.New("Same Name", st2)
	
	if rec1.ID == rec2.ID {
		t.Error("same name did not create unique IDs")
	}
	if !strings.HasPrefix(rec1.ID, "same-name") || !strings.HasPrefix(rec2.ID, "same-name") {
		t.Errorf("IDs = %q, %q", rec1.ID, rec2.ID)
	}
	
	list, _ := store.List()
	if len(list) != 2 {
		t.Errorf("expected 2 campaigns, got %d", len(list))
	}
}
