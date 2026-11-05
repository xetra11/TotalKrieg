package game

import (
	"testing"
)

func TestSnapshotAndRestore(t *testing.T) {
	s := New(42, Wehrmacht)
	clearField(s)
	s.Stocks[Wehrmacht].Manpower = 10
	o := cityOf(s, Wehrmacht)
	u, err := s.Recruit(o, Battalion)
	if err != nil {
		t.Fatal(err)
	}
	s.Move(u, Coord{o.Pos.Col + 1, o.Pos.Row})
	s.EndTurn()

	snap, err := s.Snapshot()
	if err != nil {
		t.Fatal(err)
	}
	if snap.Version != snapshotVersion {
		t.Errorf("version = %d", snap.Version)
	}
	if snap.Seed != 42 || snap.First != Wehrmacht || snap.Round != 1 {
		t.Errorf("snapshot = %+v", snap)
	}

	r, err := Restore(snap)
	if err != nil {
		t.Fatal(err)
	}
	if r.Seed != s.Seed || r.First != s.First || r.Round != s.Round || r.Active != s.Active {
		t.Errorf("state = %+v, want %+v", r, s)
	}
	if r.Stocks[Wehrmacht].Manpower != 7 {
		t.Errorf("manpower = %d", r.Stocks[Wehrmacht].Manpower)
	}
	if len(r.Units) != len(s.Units) {
		t.Errorf("%d units, want %d", len(r.Units), len(s.Units))
	}
	if len(r.Objects) != len(s.Objects) {
		t.Errorf("%d objects, want %d", len(r.Objects), len(s.Objects))
	}
}

func TestRestoreUnitsAndElements(t *testing.T) {
	s := New(7, Wehrmacht)
	clearField(s)
	staff := s.Place(Wehrmacht, Battalion, Coord{5, 5})
	c := staff.Element.Children[0].Token

	snap, err := s.Snapshot()
	if err != nil {
		t.Fatal(err)
	}
	r, err := Restore(snap)
	if err != nil {
		t.Fatal(err)
	}

	if len(r.Units) != 5 {
		t.Fatalf("%d units", len(r.Units))
	}
	rs := r.Units[0]
	if rs.Kind != Battalion || rs.Element == nil || rs.Element.Kind != Battalion {
		t.Errorf("staff = %+v, element = %+v", rs, rs.Element)
	}
	if len(rs.Element.Children) != 4 {
		t.Fatalf("staff has %d companies", len(rs.Element.Children))
	}
	rc := r.Units[1]
	if rc.Element.Parent != rs.Element {
		t.Error("company parent not linked")
	}
	if rc.Element.Token != rc {
		t.Error("element token not linked")
	}

	c.Element.Current = 1
	s.remove(c)
	snap, _ = s.Snapshot()
	r, err = Restore(snap)
	if err != nil {
		t.Fatal(err)
	}
	if len(r.Units) != 4 {
		t.Errorf("%d units after removal", len(r.Units))
	}
	if r.Units[0].Element.Children[0].Current != 1 || r.Units[0].Element.Children[0].Token != nil {
		t.Error("destroyed company element not restored")
	}
}

func TestRestoreWithdrawal(t *testing.T) {
	s := openField()
	u := s.Place(Wehrmacht, Company, Coord{5, 5})
	u.Withdrew = true
	snap, err := s.Snapshot()
	if err != nil {
		t.Fatal(err)
	}
	r, err := Restore(snap)
	if err != nil {
		t.Fatal(err)
	}
	if !r.Units[0].Withdrew {
		t.Error("withdrawal not restored")
	}
}

func TestRestoreOrders(t *testing.T) {
	s := openField()
	u := linked(s, Wehrmacht, Coord{5, 5})
	s.Plan(u, Coord{6, 5})
	s.Plan(u.Staff(), Coord{6, 6})

	snap, err := s.Snapshot()
	if err != nil {
		t.Fatal(err)
	}
	r, err := Restore(snap)
	if err != nil {
		t.Fatal(err)
	}

	if len(r.Orders()) != 2 {
		t.Fatalf("%d orders", len(r.Orders()))
	}
	if r.Orders()[0].Unit.Kind != Company || r.Orders()[0].To != (Coord{6, 5}) {
		t.Errorf("order 0 = %+v", r.Orders()[0])
	}
}

func TestRestoreRNG(t *testing.T) {
	s := New(42, Wehrmacht)
	for i := 0; i < 100; i++ {
		s.rng.IntN(100)
	}
	snap, err := s.Snapshot()
	if err != nil {
		t.Fatal(err)
	}
	r, err := Restore(snap)
	if err != nil {
		t.Fatal(err)
	}

	for i := 0; i < 50; i++ {
		want := s.rng.IntN(100)
		got := r.rng.IntN(100)
		if got != want {
			t.Fatalf("roll %d: got %d, want %d", i, got, want)
		}
	}
}

func TestRestoreRout(t *testing.T) {
	s := openField()
	u := linked(s, Wehrmacht, Coord{5, 5})
	targetPos := Coord{7, 5}
	d := s.Place(SovietUnion, Company, targetPos)
	if _, err := s.ResolveBattle(u, d.Pos, DefenderWins); err != nil {
		t.Fatal(err)
	}

	r := s.NextRout()
	if r == nil {
		t.Fatal("no rout after battle")
	}

	snap, err := s.Snapshot()
	if err != nil {
		t.Fatal(err)
	}
	restored, err := Restore(snap)
	if err != nil {
		t.Fatal(err)
	}

	rr := restored.NextRout()
	if rr == nil {
		t.Fatal("rout not restored")
	}
	if rr.From != targetPos {
		t.Errorf("rout from = %+v, want %+v", rr.From, targetPos)
	}
	if rr.Unit == nil || rr.Unit.Faction != Wehrmacht {
		t.Errorf("rout unit = %+v", rr.Unit)
	}
	if rr.Victor != nil {
		t.Error("lost attacker should have no victor")
	}
}

func TestSnapshotWithoutState(t *testing.T) {
	var s *State
	_, err := s.Snapshot()
	if err == nil {
		t.Error("snapshot of nil state succeeded")
	}
}

func TestRestoreBadVersion(t *testing.T) {
	snap := Snapshot{Version: 999}
	_, err := Restore(snap)
	if err == nil {
		t.Error("restore of bad version succeeded")
	}
}

func TestRestoreBadIndices(t *testing.T) {
	s := New(1, Wehrmacht)
	snap, _ := s.Snapshot()

	snap.Elements[0].Parent = 999
	if _, err := Restore(snap); err == nil {
		t.Error("bad parent index accepted")
	}

	snap, _ = s.Snapshot()
	snap.Elements[0].Children = []int{999}
	if _, err := Restore(snap); err == nil {
		t.Error("bad child index accepted")
	}

	snap, _ = s.Snapshot()
	snap.Units[0].Element = 999
	if _, err := Restore(snap); err == nil {
		t.Error("bad unit element accepted")
	}

	snap, _ = s.Snapshot()
	snap.Elements[0].Token = 999
	if _, err := Restore(snap); err == nil {
		t.Error("bad token index accepted")
	}
}
