package game

import (
	"strings"
	"testing"
)

func TestOneBattalionPerCity(t *testing.T) {
	s := New(1, Wehrmacht)
	clearField(s)
	s.Stocks[Wehrmacht].Manpower = 20
	o := cityOf(s, Wehrmacht)

	staff, err := s.Recruit(o, Battalion)
	if err != nil {
		t.Fatal(err)
	}

	// The staff blocks the city hex, so move it aside to test the city cap.
	staff.Pos = Coord{o.Pos.Col + 1, o.Pos.Row}
	o.Recruited = false
	_, err = s.Recruit(o, Battalion)
	if err == nil || !strings.Contains(err.Error(), "already has a battalion") {
		t.Fatalf("second battalion error = %v", err)
	}

	// A destroyed battalion frees the city for a new one.
	s.remove(staff)
	clearField(s)
	o.Recruited = false
	if _, err := s.Recruit(o, Battalion); err != nil {
		t.Errorf("no new battalion after the first was destroyed: %v", err)
	}
}

func TestBattalionCompanyCapAndAssignment(t *testing.T) {
	s := openField()
	s.Stocks[Wehrmacht].Manpower = 20
	o := &Object{Kind: City, Pos: Coord{5, 5}, Owner: Wehrmacht}
	s.Objects = []*Object{o}

	staff, err := s.Recruit(o, Battalion)
	if err != nil {
		t.Fatal(err)
	}
	b := staff.Element
	if got := len(b.Children); got != MaxBattalionCompanies {
		t.Fatalf("%d companies, want %d", got, MaxBattalionCompanies)
	}

	// A full battalion cannot take another company, so it stays independent.
	o.Recruited = false
	indep, err := s.Recruit(o, Company)
	if err != nil {
		t.Fatal(err)
	}
	if indep.Element.Parent != nil {
		t.Error("company joined a full battalion")
	}
	if indep.Linked() {
		t.Error("independent company is linked")
	}
	if indep.Pos == o.Pos {
		t.Error("company took the staff's city hex")
	}
	if got := Distance(indep.Pos, o.Pos); got != 1 {
		t.Errorf("company at distance %d, want next to the city", got)
	}

	// Losing a company frees a slot; the next company replaces it.
	victim := b.Children[0].Token
	s.remove(victim)
	o.Recruited = false
	repl, err := s.Recruit(o, Company)
	if err != nil {
		t.Fatal(err)
	}
	if repl.Element.Parent != b {
		t.Error("replacement did not join the battalion")
	}
	if got := len(b.Tokens()) - 1; got != MaxBattalionCompanies {
		t.Errorf("%d living companies, want %d", got, MaxBattalionCompanies)
	}
	if !repl.Linked() {
		t.Error("replacement company is not linked to its staff")
	}
	if repl.MovesLeft() != 0 {
		t.Error("fresh company can move")
	}
}

func TestCompanyWithoutBattalionIsIndependent(t *testing.T) {
	s := New(1, Wehrmacht)
	clearField(s)
	s.Stocks[Wehrmacht].Manpower = 10
	o := cityOf(s, Wehrmacht)

	u, err := s.Recruit(o, Company)
	if err != nil {
		t.Fatal(err)
	}
	if u.Element.Parent != nil {
		t.Error("company joined a battalion that does not exist")
	}
	if u.Linked() {
		t.Error("independent company is linked")
	}
}

func TestBattalionSizeWithinCap(t *testing.T) {
	for _, f := range []Faction{Wehrmacht, SovietUnion} {
		if got := BattalionSize(f); got > MaxBattalionCompanies {
			t.Errorf("%v battalion has %d companies, max %d", f, got, MaxBattalionCompanies)
		}
	}
}

func TestSnapshotKeepsCityBattalion(t *testing.T) {
	s := New(42, Wehrmacht)
	o := cityOf(s, Wehrmacht)
	if s.CityBattalion(o) == nil {
		t.Fatal("city has no starting battalion")
	}

	snap, err := s.Snapshot()
	if err != nil {
		t.Fatal(err)
	}
	r, err := Restore(snap)
	if err != nil {
		t.Fatal(err)
	}
	ro := r.ObjectAt(o.Pos)
	if ro == nil {
		t.Fatal("city not restored")
	}
	if r.CityBattalion(ro) == nil {
		t.Error("restored city lost its battalion")
	}
}
