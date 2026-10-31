package game

import "testing"

// setStrength puts a unit at an exact strength.
func setStrength(u *Unit, n int) {
	u.Element.Current = n
}

func TestBattalionStructure(t *testing.T) {
	s := openField()
	cases := []struct {
		faction Faction
		count   int
	}{{Wehrmacht, 4}, {SovietUnion, 5}}
	for i, c := range cases {
		u := s.Place(c.faction, Battalion, Coord{5 + i*10, 5})
		if got := len(u.Element.Children); got != c.count {
			t.Errorf("%v battalion has %d companies, want %d", c.faction, got, c.count)
		}
		if got := len(u.Element.Tokens()); got != c.count+1 {
			t.Errorf("%v battalion has %d map units, want %d", c.faction, got, c.count+1)
		}
		if got, want := u.Element.Strength(), Stats(c.faction, Battalion).Strength; got != want {
			t.Errorf("%v battalion strength %d, want %d", c.faction, got, want)
		}
	}
}

func TestCompaniesHaveCallsignAndLeader(t *testing.T) {
	s := openField()
	seen := map[string]bool{}
	for i := 0; i < 3; i++ {
		for _, c := range s.Place(Wehrmacht, Battalion, Coord{3 + 6*i, 8}).Element.Children {
			if c.Callsign == "" || c.Leader == "" {
				t.Fatalf("company %s lacks callsign or leader", c.Short)
			}
			if seen[c.Callsign] || seen[c.Short] {
				t.Errorf("%s or %s used twice", c.Callsign, c.Short)
			}
			seen[c.Callsign], seen[c.Short] = true, true
		}
	}
}

func TestStaffsNeverAttack(t *testing.T) {
	s := openField()
	u := s.Place(Wehrmacht, Battalion, Coord{5, 5})
	s.Place(SovietUnion, Company, Coord{5, 6})
	if len(s.AttackTargets(u)) != 0 {
		t.Error("staff has attack targets")
	}
}

func TestLinkSetsCompanyMovement(t *testing.T) {
	s := openField()
	staff := s.Place(Wehrmacht, Battalion, Coord{5, 5})
	c := staff.Element.Children[0].Token
	if !c.Linked() || c.MovesLeft() != 3 {
		t.Fatalf("linked %v, moves %d", c.Linked(), c.MovesLeft())
	}
	c.Pos = Coord{5 + CommandRange + 1, 5}
	if c.Linked() {
		t.Fatal("linked beyond the command range")
	}
	if c.MovesLeft() != UnlinkedMove {
		t.Errorf("moves %d without link, want %d", c.MovesLeft(), UnlinkedMove)
	}
	if n := len(s.Reachable(c)); n != 6 {
		t.Errorf("reaches %d hexes without link, want 6", n)
	}
}

func TestMovingTheStaffRestoresTheLink(t *testing.T) {
	s := openField()
	staff := s.Place(Wehrmacht, Battalion, Coord{5, 5})
	c := staff.Element.Children[0].Token
	for _, x := range staff.Element.Children[1:] {
		s.remove(x.Token)
	}
	c.Pos = Coord{10, 5}
	if err := s.Move(staff, Coord{6, 5}); err != nil {
		t.Fatal(err)
	}
	if !c.Linked() || c.MovesLeft() != 3 {
		t.Errorf("linked %v, moves %d", c.Linked(), c.MovesLeft())
	}
}

func TestDestroyedStaffCutsTheLink(t *testing.T) {
	s := openField()
	staff := s.Place(Wehrmacht, Battalion, Coord{5, 5})
	c := staff.Element.Children[0].Token
	s.remove(staff)
	if c.Linked() || c.MovesLeft() != UnlinkedMove {
		t.Errorf("linked %v, moves %d", c.Linked(), c.MovesLeft())
	}
	if c.Element.Parent != staff.Element {
		t.Error("company lost its battalion")
	}
}

func TestFreeCompanyHasNoLink(t *testing.T) {
	s := openField()
	c := s.Place(Wehrmacht, Company, Coord{5, 5})
	if c.Linked() || c.MovesLeft() != UnlinkedMove {
		t.Errorf("linked %v, moves %d", c.Linked(), c.MovesLeft())
	}
}

func TestStaffMoves(t *testing.T) {
	s := openField()
	staff := s.Place(Wehrmacht, Battalion, Coord{5, 5})
	if staff.MovesLeft() != 2 {
		t.Errorf("staff moves %d, want 2", staff.MovesLeft())
	}
}

func TestEngagementBringsNeighbors(t *testing.T) {
	s := openField()
	target := Coord{10, 8}
	d := s.Place(SovietUnion, Company, target)
	n := target.Neighbors()
	u := s.Place(Wehrmacht, Company, n[0])
	s.Place(Wehrmacht, Company, n[1])
	spent := s.Place(Wehrmacht, Company, n[2])
	spent.Spent = true
	s.Place(Wehrmacht, Battalion, Coord{2, 2})
	s.Place(SovietUnion, Company, n[3])
	s.Place(SovietUnion, Company, n[4])
	s.Place(SovietUnion, Company, Coord{20, 2})

	attackers, defenders := s.Engagement(u, target)
	if len(attackers) != 2 {
		t.Errorf("%d attackers, want 2", len(attackers))
	}
	if len(defenders) != 3 || defenders[0] != d {
		t.Errorf("%d defenders, want 3 led by the target", len(defenders))
	}
}

func TestAllAttackersAreSpent(t *testing.T) {
	s := openField()
	target := Coord{10, 8}
	s.Place(SovietUnion, Company, target)
	n := target.Neighbors()
	a := s.Place(Wehrmacht, Company, n[0])
	b := s.Place(Wehrmacht, Company, n[1])
	if _, err := s.Attack(a, target); err != nil {
		t.Fatal(err)
	}
	if a.Strength() > 0 && !a.Spent || b.Strength() > 0 && !b.Spent {
		t.Error("attacker can still act")
	}
}

func TestScaledStrength(t *testing.T) {
	cases := []struct {
		current int
		share   float64
		want    int
	}{{3, 1, 3}, {3, 0.98, 3}, {3, 0.5, 2}, {3, 0.1, 1}, {2, 0.4, 1}, {3, 0, 0}, {0, 1, 0}}
	for _, c := range cases {
		if got := ScaledStrength(c.current, c.share); got != c.want {
			t.Errorf("ScaledStrength(%d, %.2f) = %d, want %d", c.current, c.share, got, c.want)
		}
	}
}

func TestFieldResultVictoryPushesTheDefenderBack(t *testing.T) {
	s := openField()
	u := linked(s, Wehrmacht, Coord{5, 5})
	target := Coord{7, 5}
	d := s.Place(SovietUnion, Company, target)
	res := &FieldResult{Outcome: AttackerWins, Remaining: map[*Element]float64{u.Element: 0.7, d.Element: 0.5}}
	r, err := s.ApplyFieldResult(u, target, res)
	if err != nil {
		t.Fatal(err)
	}
	if u.Strength() != 2 || d.Strength() != 1 {
		t.Fatalf("strengths %d and %d", u.Strength(), d.Strength())
	}
	if u.Pos != target {
		t.Errorf("attacker at %v", u.Pos)
	}
	if d.Pos == target || Distance(d.Pos, target) != 1 {
		t.Errorf("defender at %v", d.Pos)
	}
	if r.AttackerLoss != 1 || r.DefenderLoss != 1 || len(s.Reports) != 1 {
		t.Errorf("report %+v", r)
	}
	if u.MovesLeft() != 0 {
		t.Errorf("moves left %d", u.MovesLeft())
	}
}

func TestFieldResultDefeatLeavesTheHex(t *testing.T) {
	s := openField()
	u := linked(s, Wehrmacht, Coord{5, 5})
	target := Coord{7, 5}
	d := s.Place(SovietUnion, Company, target)
	res := &FieldResult{Outcome: DefenderWins, Remaining: map[*Element]float64{u.Element: 0, d.Element: 1}}
	r, err := s.ApplyFieldResult(u, target, res)
	if err != nil {
		t.Fatal(err)
	}
	if !r.AttackerDestroyed || u.Element.Token != nil {
		t.Errorf("attacker survived: %+v", r)
	}
	if d.Pos != target || d.Strength() != 2 {
		t.Errorf("defender %v at %d", d.Pos, d.Strength())
	}
}

func TestFieldResultCanKillAStaff(t *testing.T) {
	s := openField()
	u := linked(s, Wehrmacht, Coord{5, 5})
	staff := u.Staff()
	d := s.Place(SovietUnion, Company, Coord{7, 5})
	res := &FieldResult{Outcome: Draw, Remaining: map[*Element]float64{u.Element: 1, d.Element: 1, staff.Element: 0}}
	if _, err := s.ApplyFieldResult(u, d.Pos, res); err != nil {
		t.Fatal(err)
	}
	if staff.Element.Token != nil || u.Linked() {
		t.Error("staff survived its loss")
	}
}

func TestLeftOutCompanyTakesTheSideAverage(t *testing.T) {
	s := openField()
	target := Coord{10, 8}
	d := s.Place(SovietUnion, Company, target)
	n := target.Neighbors()
	a := s.Place(Wehrmacht, Company, n[0])
	b := s.Place(Wehrmacht, Company, n[1])
	res := &FieldResult{Outcome: Draw, Remaining: map[*Element]float64{a.Element: 0.4, d.Element: 1}}
	if _, err := s.ApplyFieldResult(a, target, res); err != nil {
		t.Fatal(err)
	}
	if a.Strength() != 1 || b.Strength() != 1 {
		t.Errorf("strengths %d and %d", a.Strength(), b.Strength())
	}
}

func TestBattleOutcomes(t *testing.T) {
	cases := []struct {
		outcome      Outcome
		attackerLoss int
		defenderLoss int
		defenderDead bool
	}{
		{AttackerWins, 1, 2, true},
		{Draw, 1, 1, false},
		{DefenderWins, 2, 1, false},
	}
	for _, c := range cases {
		s := openField()
		u := linked(s, Wehrmacht, Coord{5, 5})
		d := s.Place(SovietUnion, Company, Coord{7, 5})
		r, err := s.ResolveBattle(u, d.Pos, c.outcome)
		if err != nil {
			t.Fatal(err)
		}
		if r.AttackerLoss != c.attackerLoss || r.DefenderLoss != c.defenderLoss {
			t.Errorf("%v: losses %d and %d, want %d and %d", c.outcome, r.AttackerLoss, r.DefenderLoss, c.attackerLoss, c.defenderLoss)
		}
		if r.DefenderDestroyed != c.defenderDead {
			t.Errorf("%v: defender destroyed %v", c.outcome, r.DefenderDestroyed)
		}
		if c.defenderDead && u.Pos != (Coord{7, 5}) {
			t.Errorf("%v: attacker at %v", c.outcome, u.Pos)
		}
		if u.MovesLeft() != 0 || len(s.Reports) != 1 {
			t.Errorf("%v: moves %d, reports %d", c.outcome, u.MovesLeft(), len(s.Reports))
		}
	}
}
