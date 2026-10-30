package game

import "testing"

// setStrength puts a unit at an exact strength.
func setStrength(u *Unit, n int) {
	for _, c := range u.Element.companies(true) {
		c.Current = c.Full
	}
	u.takeLoss(u.FullStrength() - n)
}

// openField returns a match without objects or units, in the movement phase.
func openField() *State {
	s := New(5, Wehrmacht)
	clearField(s)
	s.Objects = nil
	s.Phase = PhaseBewegung
	return s
}

func firstCompany(u *Unit) *Element {
	return u.Element.companies(true)[0]
}

func TestBattalionStructure(t *testing.T) {
	s := openField()
	cases := []struct {
		faction Faction
		count   int
	}{{Wehrmacht, 4}, {Sowjetunion, 5}}
	for i, c := range cases {
		u := s.Place(c.faction, Bataillon, Coord{i, 0})
		if got := len(u.Element.Children); got != c.count {
			t.Errorf("%v battalion has %d companies, want %d", c.faction, got, c.count)
		}
		if got, want := u.Strength(), Stats(c.faction, Bataillon).Strength; got != want {
			t.Errorf("%v battalion strength %d, want %d", c.faction, got, want)
		}
	}
}

func TestDivisionStructure(t *testing.T) {
	s := openField()
	cases := []struct {
		faction   Faction
		companies int
	}{{Wehrmacht, 36}, {Sowjetunion, 45}}
	for i, c := range cases {
		u := s.Place(c.faction, Division, Coord{i, 0})
		if got := len(u.Element.Children); got != 3 {
			t.Fatalf("%d regiments, want 3", got)
		}
		for _, reg := range u.Element.Children {
			if reg.Kind != Regiment || len(reg.Children) != 3 {
				t.Errorf("regiment %s has %d battalions", reg.Short, len(reg.Children))
			}
		}
		if got := len(u.Element.companies(true)); got != c.companies {
			t.Errorf("%v division has %d companies, want %d", c.faction, got, c.companies)
		}
		if got, want := u.Strength(), Stats(c.faction, Division).Strength; got != want {
			t.Errorf("%v division strength %d, want %d", c.faction, got, want)
		}
	}
}

func TestCompaniesHaveCallsignAndLeader(t *testing.T) {
	s := openField()
	seen := map[string]bool{}
	for _, c := range s.Place(Wehrmacht, Division, Coord{0, 0}).Element.companies(true) {
		if c.Callsign == "" || c.Leader == "" {
			t.Fatalf("company %s lacks callsign or leader", c.Short)
		}
		if seen[c.Callsign] {
			t.Errorf("callsign %s used twice", c.Callsign)
		}
		seen[c.Callsign] = true
		if seen[c.Short] {
			t.Errorf("designation %s used twice", c.Short)
		}
		seen[c.Short] = true
	}
}

func TestDetachKeepsMembership(t *testing.T) {
	s := openField()
	u := s.Place(Wehrmacht, Bataillon, Coord{5, 5})
	c := firstCompany(u)
	to := Coord{6, 5}
	k, err := s.Detach(c, to)
	if err != nil {
		t.Fatal(err)
	}
	if k.Pos != to || k.Kind != Kompanie || k.Strength() != c.Full {
		t.Fatalf("detachment = %+v", k)
	}
	if u.Strength() != 9 {
		t.Errorf("parent strength %d, want 9", u.Strength())
	}
	if c.Parent != u.Element || c.Root() != u.Element || len(u.Element.Children) != 4 {
		t.Error("company lost its place in the battalion")
	}
	if !c.Detached() {
		t.Error("company not marked as detached")
	}
	if len(u.Element.Tokens()) != 2 {
		t.Errorf("formation has %d tokens, want 2", len(u.Element.Tokens()))
	}
}

func TestPincerBothSidesMoveInOneTurn(t *testing.T) {
	s := openField()
	u := s.Place(Wehrmacht, Bataillon, Coord{10, 8})
	k, err := s.Detach(firstCompany(u), Coord{10, 7})
	if err != nil {
		t.Fatal(err)
	}
	if err := s.Move(k, Coord{11, 5}); err != nil {
		t.Errorf("detachment can't move: %v", err)
	}
	if err := s.Move(u, Coord{10, 10}); err != nil {
		t.Errorf("parent can't move: %v", err)
	}
}

func TestDetachRules(t *testing.T) {
	s := openField()
	u := s.Place(Wehrmacht, Bataillon, Coord{5, 5})
	c := firstCompany(u)

	s.Phase = PhaseAngriff
	if _, err := s.Detach(c, Coord{6, 5}); err == nil {
		t.Error("detached outside the movement phase")
	}
	s.Phase = PhaseBewegung
	if _, err := s.Detach(c, Coord{8, 5}); err == nil {
		t.Error("detached onto a far hex")
	}
	s.Place(Sowjetunion, Kompanie, Coord{6, 5})
	if _, err := s.Detach(c, Coord{6, 5}); err == nil {
		t.Error("detached onto an occupied hex")
	}
	u.Moved = true
	if err := s.CanDetach(c); err == nil {
		t.Error("detached after the parent moved")
	}
}

func TestLastCompanyStays(t *testing.T) {
	s := openField()
	u := s.Place(Wehrmacht, Bataillon, Coord{10, 8})
	spots := s.DetachSpots(u)
	children := u.Element.Children
	for i := 0; i < 3; i++ {
		if _, err := s.Detach(children[i], spots[i]); err != nil {
			t.Fatal(err)
		}
	}
	if err := s.CanDetach(children[3]); err == nil {
		t.Error("detached the last company")
	}
}

func TestEnemyUnitsCantBeDetached(t *testing.T) {
	s := openField()
	e := s.Place(Sowjetunion, Bataillon, Coord{5, 5})
	if err := s.CanDetach(firstCompany(e)); err == nil {
		t.Error("detached an enemy company")
	}
}

func TestReattach(t *testing.T) {
	s := openField()
	u := s.Place(Wehrmacht, Bataillon, Coord{5, 5})
	c := firstCompany(u)
	k, err := s.Detach(c, Coord{6, 5})
	if err != nil {
		t.Fatal(err)
	}
	setStrength(k, 2)
	k.Pos = Coord{9, 5}
	if _, err := s.Reattach(k); err == nil {
		t.Error("reattached from afar")
	}
	k.Pos = Coord{6, 5}
	got, err := s.Reattach(k)
	if err != nil {
		t.Fatal(err)
	}
	if got != u {
		t.Errorf("rejoined %v", got)
	}
	if s.UnitAt(Coord{6, 5}) != nil {
		t.Error("detachment still on the map")
	}
	if c.Detached() {
		t.Error("company still marked as detached")
	}
	if u.Strength() != 11 {
		t.Errorf("strength %d, want 11: the loss must stay", u.Strength())
	}
}

func TestLossesStayWithTheHitUnit(t *testing.T) {
	s := openField()
	u := s.Place(Wehrmacht, Bataillon, Coord{5, 5})
	c := firstCompany(u)
	k, err := s.Detach(c, Coord{6, 5})
	if err != nil {
		t.Fatal(err)
	}
	u.takeLoss(4)
	if k.Strength() != c.Full {
		t.Errorf("detachment lost strength with its parent: %d", k.Strength())
	}
	if u.Strength() != 5 {
		t.Errorf("parent strength %d, want 5", u.Strength())
	}
	for _, x := range u.Element.companies(true) {
		if x.Current < 1 {
			t.Errorf("loss not spread evenly: %s at %d", x.Short, x.Current)
		}
	}
}

func TestDetachmentSurvivesItsParent(t *testing.T) {
	s := openField()
	u := s.Place(Wehrmacht, Bataillon, Coord{5, 5})
	k, err := s.Detach(firstCompany(u), Coord{6, 5})
	if err != nil {
		t.Fatal(err)
	}
	s.remove(u)
	if s.UnitAt(Coord{6, 5}) != k {
		t.Fatal("detachment left the map with its parent")
	}
	if k.Element.Parent != u.Element {
		t.Error("detachment lost its membership")
	}
	if _, err := s.CanReattach(k); err == nil {
		t.Error("reattached to a destroyed battalion")
	}
}

func TestBattalionLeavesDivision(t *testing.T) {
	s := openField()
	d := s.Place(Wehrmacht, Division, Coord{10, 8})
	b := d.Element.Children[0].Children[0]
	u, err := s.Detach(b, Coord{11, 8})
	if err != nil {
		t.Fatal(err)
	}
	if u.Kind != Bataillon || u.Strength() != 12 || d.Strength() != 96 {
		t.Fatalf("battalion %d, division %d", u.Strength(), d.Strength())
	}
	k, err := s.Detach(u.Element.Children[0], Coord{12, 8})
	if err != nil {
		t.Fatal(err)
	}
	if k.Element.Root() != d.Element {
		t.Error("company lost its division")
	}
	if len(d.Element.Tokens()) != 3 {
		t.Errorf("division has %d tokens, want 3", len(d.Element.Tokens()))
	}
}

func TestCompanyRejoinsDivisionWhenBattalionIsGone(t *testing.T) {
	s := openField()
	d := s.Place(Wehrmacht, Division, Coord{10, 8})
	b, err := s.Detach(d.Element.Children[0].Children[0], Coord{11, 8})
	if err != nil {
		t.Fatal(err)
	}
	k, err := s.Detach(b.Element.Children[0], Coord{10, 9})
	if err != nil {
		t.Fatal(err)
	}
	s.remove(b)
	if got, err := s.CanReattach(k); err != nil || got != d {
		t.Fatalf("reattach target %v, %v", got, err)
	}
}
