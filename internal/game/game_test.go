package game

import "testing"

func TestDistanceAndNeighbors(t *testing.T) {
	c := Coord{5, 5}
	for _, n := range c.Neighbors() {
		if got := Distance(c, n); got != 1 {
			t.Errorf("Distance(%v, %v) = %d, want 1", c, n, got)
		}
	}
	if got := Distance(Coord{0, 0}, Coord{3, 0}); got != 3 {
		t.Errorf("row distance = %d, want 3", got)
	}
	if got := Distance(Coord{0, 0}, Coord{0, 4}); got != 4 {
		t.Errorf("column distance = %d, want 4", got)
	}
}

func TestMapGeneration(t *testing.T) {
	for seed := uint64(0); seed < 300; seed++ {
		s := New(seed, Wehrmacht)
		if len(s.Objects) != 20 {
			t.Fatalf("seed %d: %d objects, want 20", seed, len(s.Objects))
		}
		for _, f := range []Faction{Wehrmacht, SovietUnion} {
			counts := map[ObjectKind]int{}
			for _, o := range s.Objects {
				if o.Owner != f {
					continue
				}
				counts[o.Kind]++
				west := o.Pos.Col < MapCols/2
				if west != (f == Wehrmacht) {
					t.Errorf("seed %d: %v on the wrong half: %v", seed, o.Kind, o.Pos)
				}
				if o.Kind == Capital {
					rear := o.Pos.Col < MapCols/4 || o.Pos.Col >= MapCols*3/4
					if !rear {
						t.Errorf("seed %d: capital not in the rear: %v", seed, o.Pos)
					}
				}
			}
			want := map[ObjectKind]int{Capital: 1, City: 2, Village: 5, Factory: 2}
			for k, n := range want {
				if counts[k] != n {
					t.Errorf("seed %d: %v has %d %v, want %d", seed, f, counts[k], k, n)
				}
			}
		}
		for i, a := range s.Objects {
			for _, b := range s.Objects[i+1:] {
				if Distance(a.Pos, b.Pos) < minObjectGap {
					t.Errorf("seed %d: objects too close: %v %v", seed, a.Pos, b.Pos)
				}
			}
		}
		if !balanced(s.Objects) {
			t.Errorf("seed %d: unbalanced cities", seed)
		}
	}
}

func TestSeedRepeatsMap(t *testing.T) {
	a, b := New(42, Wehrmacht), New(42, Wehrmacht)
	for i := range a.Objects {
		if a.Objects[i].Pos != b.Objects[i].Pos {
			t.Fatalf("object %d differs", i)
		}
	}
}

func TestStartState(t *testing.T) {
	s := New(1, Wehrmacht)
	staffs, companies := map[Faction]int{}, map[Faction]int{}
	for _, u := range s.Units {
		switch u.Kind {
		case Battalion:
			staffs[u.Faction]++
			if o := s.ObjectAt(u.Pos); o == nil || !o.Kind.IsCity() {
				t.Errorf("staff %s not in a city", u.Name)
			}
		case Company:
			companies[u.Faction]++
			if !u.Linked() {
				t.Errorf("company %s starts without a link", u.Name)
			}
		}
	}
	if staffs[Wehrmacht] != 3 || staffs[SovietUnion] != 3 {
		t.Errorf("staffs = %v, want 3 each", staffs)
	}
	if companies[Wehrmacht] != 12 || companies[SovietUnion] != 15 {
		t.Errorf("companies = %v, want 12 and 15", companies)
	}
	if got := s.Stocks[Wehrmacht]; got != (Stock{Manpower: 3, Resources: 2}) {
		t.Errorf("first turn income = %+v", got)
	}
	if got := s.Stocks[SovietUnion]; got != (Stock{}) {
		t.Errorf("second faction has income before its turn: %+v", got)
	}
}

// clearField removes all units so a test can set up its own position.
func clearField(s *State) {
	s.Units = nil
}

// openField returns a match without objects or units.
func openField() *State {
	s := New(5, Wehrmacht)
	clearField(s)
	s.Objects = nil
	return s
}

// linked places one company at pos with its staff next to it.
func linked(s *State, f Faction, pos Coord) *Unit {
	staff := s.Place(f, Battalion, Coord{pos.Col, pos.Row + 1})
	tokens := staff.Element.Tokens()
	c := tokens[1]
	for _, x := range tokens[2:] {
		s.remove(x)
	}
	c.Pos = pos
	return c
}

func cityOf(s *State, f Faction) *Object {
	for _, o := range s.Objects {
		if o.Owner == f && o.Kind == City {
			return o
		}
	}
	return nil
}

func TestRecruitRules(t *testing.T) {
	s := New(1, Wehrmacht)
	o := cityOf(s, Wehrmacht)
	if _, err := s.Recruit(o, Company); err == nil {
		t.Error("recruited on an occupied city")
	}
	clearField(s)
	s.Stocks[Wehrmacht].Manpower = 2
	if _, err := s.Recruit(o, Battalion); err == nil {
		t.Error("recruited without enough manpower")
	}
	u, err := s.Recruit(o, Company)
	if err != nil {
		t.Fatal(err)
	}
	if s.Stocks[Wehrmacht].Manpower != 1 {
		t.Errorf("manpower = %d", s.Stocks[Wehrmacht].Manpower)
	}
	if u.MovesLeft() != 0 {
		t.Error("new unit can move")
	}
	if err := s.Move(u, u.Pos.Neighbors()[0]); err == nil {
		t.Error("fresh unit moved")
	}
}

func TestRecruitBattalionFieldsStaffAndCompanies(t *testing.T) {
	s := New(1, Wehrmacht)
	clearField(s)
	s.Stocks[Wehrmacht].Manpower = 3
	o := cityOf(s, Wehrmacht)
	u, err := s.Recruit(o, Battalion)
	if err != nil {
		t.Fatal(err)
	}
	if u.Pos != o.Pos || u.Kind != Battalion {
		t.Fatalf("staff %+v", u)
	}
	if got := len(s.Units); got != 5 {
		t.Fatalf("%d units, want staff and 4 companies", got)
	}
	for _, x := range s.Units {
		if x.MovesLeft() != 0 {
			t.Errorf("%s can move in its first round", x.Short)
		}
	}
}

func TestOneRecruitPerCity(t *testing.T) {
	s := New(1, Wehrmacht)
	clearField(s)
	s.Stocks[Wehrmacht].Manpower = 50
	o := cityOf(s, Wehrmacht)
	u, err := s.Recruit(o, Company)
	if err != nil {
		t.Fatal(err)
	}
	s.remove(u)
	if _, err := s.Recruit(o, Company); err == nil {
		t.Error("city recruited twice in a round")
	}
}

func TestEnemyBlocksOwnUnitsPass(t *testing.T) {
	s := openField()
	u := linked(s, Wehrmacht, Coord{5, 5})
	east := Coord{6, 5}
	s.Place(SovietUnion, Company, east)
	if _, ok := s.Reachable(u)[east]; ok {
		t.Error("enemy hex is reachable")
	}
	s.remove(s.UnitAt(east))
	s.Place(Wehrmacht, Company, east)
	reach := s.Reachable(u)
	if _, ok := reach[east]; ok {
		t.Error("own unit's hex is reachable")
	}
	if _, ok := reach[Coord{7, 5}]; !ok {
		t.Error("can't pass an own unit")
	}
}

func TestMovementBudget(t *testing.T) {
	s := openField()
	u := linked(s, Wehrmacht, Coord{5, 5})
	if err := s.Move(u, Coord{6, 5}); err != nil {
		t.Fatal(err)
	}
	if u.MovesLeft() != 2 {
		t.Fatalf("moves left %d, want 2", u.MovesLeft())
	}
	if err := s.Move(u, Coord{8, 5}); err != nil {
		t.Fatal(err)
	}
	if err := s.Move(u, Coord{9, 5}); err == nil {
		t.Error("moved past the budget")
	}
	s.EndTurn()
	s.EndTurn()
	if u.MovesLeft() != 3 {
		t.Errorf("moves after the round %d, want 3", u.MovesLeft())
	}
}

func TestMoveOntoEnemyIsNoPlainMove(t *testing.T) {
	s := openField()
	u := linked(s, Wehrmacht, Coord{5, 5})
	s.Place(SovietUnion, Company, Coord{6, 5})
	if err := s.Move(u, Coord{6, 5}); err == nil {
		t.Error("moved onto an enemy without a fight")
	}
}

func TestAttackFromDistance(t *testing.T) {
	s := openField()
	u := linked(s, Wehrmacht, Coord{5, 5})
	target := Coord{8, 5}
	d := s.Place(SovietUnion, Company, target)
	setStrength(d, 2)
	from, ok := s.AttackTargets(u)[target]
	if !ok {
		t.Fatal("enemy 3 fields away is no target")
	}
	if Distance(from, target) != 1 {
		t.Fatalf("attacks from %v", from)
	}
	r, err := s.Attack(u, target)
	if err != nil {
		t.Fatal(err)
	}
	if !r.AttackerDestroyed && u.Pos != from && u.Pos != target {
		t.Errorf("attacker at %v, want %v or %v", u.Pos, from, target)
	}
	if u.MovesLeft() != 0 {
		t.Errorf("moves left after attack: %d", u.MovesLeft())
	}
}

func TestCombatLossesStayInRange(t *testing.T) {
	for seed := uint64(0); seed < 200; seed++ {
		s := New(seed, Wehrmacht)
		clearField(s)
		s.Objects = nil
		u := s.Place(Wehrmacht, Company, Coord{5, 5})
		for i := 0; i < 2; i++ {
			s.Place(SovietUnion, Company, []Coord{{6, 5}, {7, 5}}[i])
		}
		u.Element.Current, u.Element.Full = 100, 100
		d := s.UnitAt(Coord{6, 5})
		d.Element.Current, d.Element.Full = 100, 100
		r, err := s.Attack(u, d.Pos)
		if err != nil {
			t.Fatal(err)
		}
		// The second enemy stands next to the target and supports it.
		if r.AttackerLoss < 21 || r.AttackerLoss > 41 {
			t.Errorf("attacker loss %d outside 20-40%% of 102", r.AttackerLoss)
		}
		if r.DefenderLoss < 20 || r.DefenderLoss > 40 {
			t.Errorf("defender loss %d outside 20-40%% of 100", r.DefenderLoss)
		}
		if _, err := s.Attack(u, d.Pos); err == nil {
			t.Error("attacked twice")
		}
	}
}

func TestCityBonusRaisesAttackerLoss(t *testing.T) {
	losses := func(inCity bool) int {
		total := 0
		for seed := uint64(0); seed < 200; seed++ {
			s := New(seed, Wehrmacht)
			clearField(s)
			s.Objects = nil
			if inCity {
				s.Objects = []*Object{{Kind: City, Pos: Coord{6, 5}, Owner: SovietUnion}}
			}
			u := s.Place(Wehrmacht, Company, Coord{5, 5})
			d := s.Place(SovietUnion, Company, Coord{6, 5})
			u.Element.Current, d.Element.Current = 100, 100
			r, err := s.Attack(u, Coord{6, 5})
			if err != nil {
				t.Fatal(err)
			}
			total += r.AttackerLoss
		}
		return total
	}
	open, city := losses(false), losses(true)
	ratio := float64(city) / float64(open)
	if ratio < 1.2 || ratio > 1.3 {
		t.Errorf("city loss ratio = %.2f, want about 1.25", ratio)
	}
}

func TestAttackerAdvancesAfterKill(t *testing.T) {
	s := openField()
	u := linked(s, Wehrmacht, Coord{5, 5})
	d := s.Place(SovietUnion, Company, Coord{6, 5})
	setStrength(d, 1)
	r, err := s.Attack(u, Coord{6, 5})
	if err != nil {
		t.Fatal(err)
	}
	if !r.DefenderDestroyed {
		t.Fatal("defender survived")
	}
	if u.Pos != (Coord{6, 5}) {
		t.Errorf("attacker at %v", u.Pos)
	}
}

func TestLossesStay(t *testing.T) {
	s := openField()
	u := linked(s, Wehrmacht, Coord{5, 5})
	setStrength(u, 1)
	for i := 0; i < 4; i++ {
		s.EndTurn()
	}
	if u.Strength() != 1 {
		t.Errorf("strength = %d, want 1", u.Strength())
	}
}

func TestConquestAndIncome(t *testing.T) {
	s := New(1, Wehrmacht)
	clearField(s)
	o := cityOf(s, SovietUnion)
	s.Place(Wehrmacht, Company, o.Pos)
	s.EndTurn()
	if o.Owner != Wehrmacht {
		t.Fatal("city not conquered at end of turn")
	}
	if s.Active != SovietUnion {
		t.Fatalf("active = %v", s.Active)
	}
	if got := s.Stocks[SovietUnion].Manpower; got != 2 {
		t.Errorf("soviet manpower = %d, want 2", got)
	}
}

func TestStaffsDontCapture(t *testing.T) {
	s := New(1, Wehrmacht)
	clearField(s)
	o := cityOf(s, SovietUnion)
	u := s.Place(Wehrmacht, Battalion, Coord{0, 0})
	u.Pos = o.Pos
	s.EndTurn()
	if o.Owner != SovietUnion {
		t.Error("staff captured a city")
	}
}

func TestCapitalWinsImmediately(t *testing.T) {
	s := New(1, Wehrmacht)
	clearField(s)
	for _, o := range s.Objects {
		if o.Kind == Capital && o.Owner == SovietUnion {
			s.Place(Wehrmacht, Company, o.Pos)
		}
	}
	s.EndTurn()
	if !s.Over || s.Winner != Wehrmacht {
		t.Fatalf("over=%v winner=%v", s.Over, s.Winner)
	}
}

func TestRoundLimitCountsCities(t *testing.T) {
	s := New(1, Wehrmacht)
	clearField(s)
	cityOf(s, SovietUnion).Owner = Wehrmacht
	s.Round = MaxRounds
	s.Active = SovietUnion
	s.EndTurn()
	if !s.Over || s.Winner != Wehrmacht {
		t.Fatalf("over=%v winner=%v reason=%q", s.Over, s.Winner, s.Reason)
	}
}

func TestRoundLimitDraw(t *testing.T) {
	s := New(1, Wehrmacht)
	clearField(s)
	s.Round = MaxRounds
	s.Active = SovietUnion
	s.EndTurn()
	if !s.Over || s.Winner != NoFaction {
		t.Fatalf("over=%v winner=%v", s.Over, s.Winner)
	}
}

func TestAIDefendsAndRaisesCompanies(t *testing.T) {
	s := New(1, Wehrmacht)
	clearField(s)
	o := cityOf(s, Wehrmacht)
	near := Coord{o.Pos.Col, o.Pos.Row + 2}
	if !OnMap(near) {
		near = Coord{o.Pos.Col, o.Pos.Row - 2}
	}
	s.Place(SovietUnion, Company, near)
	s.aiRecruit()
	u := s.UnitAt(o.Pos)
	if u == nil || u.Kind != Company {
		t.Fatalf("threatened city raised %+v", u)
	}
}

func TestAISavesForBattalion(t *testing.T) {
	s := New(1, Wehrmacht)
	clearField(s)
	s.Stocks[Wehrmacht].Manpower = 2
	s.aiRecruit()
	if len(s.Units) != 0 {
		t.Fatalf("raised %s below battalion cost", s.Units[0].Name)
	}
	s.Stocks[Wehrmacht].Manpower = 3
	s.aiRecruit()
	if len(s.Units) != 5 {
		t.Fatalf("units = %d", len(s.Units))
	}
}

func TestAIRespectsAttackRatio(t *testing.T) {
	s := openField()
	u := linked(s, Wehrmacht, Coord{5, 5})
	u.Element.Current, u.Element.Full = 10, 10
	e := s.Place(SovietUnion, Company, Coord{6, 5})
	e.Element.Current, e.Element.Full = 9, 9
	if s.bestAttackTarget(u) != nil {
		t.Error("attacks at 10 against 9")
	}
	u.Element.Current = 11
	if s.bestAttackTarget(u) == nil {
		t.Error("skips attack at 11 against 9")
	}
}

// TestFullMatches lets the AI play both sides. It checks that matches end and
// that no rule is broken along the way.
func TestFullMatches(t *testing.T) {
	wins := map[Faction]int{}
	rounds := 0
	for seed := uint64(0); seed < 400; seed++ {
		s := New(seed, Faction(seed%2))
		for steps := 0; !s.Over; steps++ {
			if steps > 2*MaxRounds+2 {
				t.Fatalf("seed %d: match did not end", seed)
			}
			s.PlayTurn()
			seen := map[Coord]bool{}
			for _, u := range s.Units {
				if seen[u.Pos] {
					t.Fatalf("seed %d: two units on %v", seed, u.Pos)
				}
				seen[u.Pos] = true
				if u.Strength() <= 0 {
					t.Fatalf("seed %d: dead unit on the map", seed)
				}
				if !OnMap(u.Pos) {
					t.Fatalf("seed %d: unit off the map", seed)
				}
			}
			for f, st := range s.Stocks {
				if st.Manpower < 0 {
					t.Fatalf("seed %d: faction %d has negative manpower", seed, f)
				}
			}
		}
		wins[s.Winner]++
		rounds += s.Round
	}
	t.Logf("Wehrmacht %d, Soviet Union %d, draws %d, mean rounds %.1f",
		wins[Wehrmacht], wins[SovietUnion], wins[NoFaction], float64(rounds)/400)
}
