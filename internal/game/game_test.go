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
		for _, f := range []Faction{Wehrmacht, Sowjetunion} {
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
				if o.Kind == Hauptstadt {
					rear := o.Pos.Col < MapCols/4 || o.Pos.Col >= MapCols*3/4
					if !rear {
						t.Errorf("seed %d: capital not in the rear: %v", seed, o.Pos)
					}
				}
			}
			want := map[ObjectKind]int{Hauptstadt: 1, Stadt: 2, Dorf: 5, Fabrik: 2}
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
	if len(s.Units) != 6 {
		t.Fatalf("%d units, want 6", len(s.Units))
	}
	for _, u := range s.Units {
		if u.Kind != Bataillon {
			t.Errorf("unit %s is %v", u.Name, u.Kind)
		}
		if o := s.ObjectAt(u.Pos); o == nil || !o.Kind.IsCity() {
			t.Errorf("unit %s not in a city", u.Name)
		}
	}
	if got := s.Stocks[Wehrmacht]; got != (Stock{Manpower: 3, Resources: 2}) {
		t.Errorf("first turn income = %+v", got)
	}
	if got := s.Stocks[Sowjetunion]; got != (Stock{}) {
		t.Errorf("second faction has income before its turn: %+v", got)
	}
}

// clearField removes all units so a test can set up its own position.
func clearField(s *State) {
	s.Units = nil
}

func cityOf(s *State, f Faction) *Object {
	for _, o := range s.Objects {
		if o.Owner == f && o.Kind == Stadt {
			return o
		}
	}
	return nil
}

func TestRecruitRules(t *testing.T) {
	s := New(1, Wehrmacht)
	o := cityOf(s, Wehrmacht)
	if _, err := s.Recruit(o, Kompanie); err == nil {
		t.Error("recruited on an occupied city")
	}
	clearField(s)
	if _, err := s.Recruit(o, Division); err == nil {
		t.Error("recruited without enough manpower")
	}
	u, err := s.Recruit(o, Bataillon)
	if err != nil {
		t.Fatal(err)
	}
	if s.Stocks[Wehrmacht].Manpower != 0 {
		t.Errorf("manpower = %d", s.Stocks[Wehrmacht].Manpower)
	}
	if !u.Fresh {
		t.Error("new unit is not fresh")
	}
	s.Phase = PhaseBewegung
	if err := s.Move(u, u.Pos.Neighbors()[0]); err == nil {
		t.Error("fresh unit moved")
	}
	if len(s.Reachable(u)) != 0 {
		t.Error("fresh unit has reachable hexes")
	}
}

func TestOneRecruitPerCity(t *testing.T) {
	s := New(1, Wehrmacht)
	clearField(s)
	s.Stocks[Wehrmacht].Manpower = 50
	o := cityOf(s, Wehrmacht)
	u, err := s.Recruit(o, Kompanie)
	if err != nil {
		t.Fatal(err)
	}
	s.remove(u)
	if _, err := s.Recruit(o, Kompanie); err == nil {
		t.Error("city recruited twice in a round")
	}
}

func TestMovementRange(t *testing.T) {
	s := New(1, Wehrmacht)
	clearField(s)
	u := s.Place(Wehrmacht, Kompanie, Coord{10, 8})
	far := s.Reachable(u)
	for c, d := range far {
		if d > 3 || Distance(u.Pos, c) > 3 {
			t.Errorf("%v reachable at %d", c, d)
		}
	}
	if len(far) != 36 {
		t.Errorf("%d hexes in range 3, want 36", len(far))
	}
	div := s.Place(Wehrmacht, Division, Coord{2, 2})
	if n := len(s.Reachable(div)); n != 6 {
		t.Errorf("division reaches %d hexes, want 6", n)
	}
}

func TestEnemyBlocksOwnUnitsPass(t *testing.T) {
	s := New(1, Wehrmacht)
	clearField(s)
	u := s.Place(Wehrmacht, Bataillon, Coord{5, 5})
	east := Coord{6, 5}
	s.Place(Sowjetunion, Kompanie, east)
	reach := s.Reachable(u)
	if _, ok := reach[east]; ok {
		t.Error("enemy hex is reachable")
	}
	s.remove(s.UnitAt(east))
	s.Place(Wehrmacht, Kompanie, east)
	reach = s.Reachable(u)
	if _, ok := reach[east]; ok {
		t.Error("own unit's hex is reachable")
	}
	if _, ok := reach[Coord{7, 5}]; !ok {
		t.Error("can't pass an own unit")
	}
}

func TestMoveOnlyInMovePhaseAndOnce(t *testing.T) {
	s := New(1, Wehrmacht)
	clearField(s)
	u := s.Place(Wehrmacht, Kompanie, Coord{5, 5})
	if err := s.Move(u, Coord{6, 5}); err == nil {
		t.Error("moved in the recruit phase")
	}
	s.Phase = PhaseBewegung
	if err := s.Move(u, Coord{6, 5}); err != nil {
		t.Fatal(err)
	}
	if err := s.Move(u, Coord{7, 5}); err == nil {
		t.Error("moved twice")
	}
}

func TestCombatLossesStayInRange(t *testing.T) {
	for seed := uint64(0); seed < 200; seed++ {
		s := New(seed, Wehrmacht)
		clearField(s)
		s.Objects = nil
		u := s.Place(Wehrmacht, Division, Coord{5, 5})
		d := s.Place(Sowjetunion, Division, Coord{6, 5})
		s.Phase = PhaseAngriff
		r, err := s.Attack(u, d.Pos)
		if err != nil {
			t.Fatal(err)
		}
		if r.AttackerLoss < 17 || r.AttackerLoss > 33 {
			t.Errorf("attacker loss %d outside 20-40%% of 81", r.AttackerLoss)
		}
		if r.DefenderLoss < 22 || r.DefenderLoss > 44 {
			t.Errorf("defender loss %d outside 20-40%% of 108", r.DefenderLoss)
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
				s.Objects = []*Object{{Kind: Stadt, Pos: Coord{6, 5}, Owner: Sowjetunion}}
			}
			u := s.Place(Wehrmacht, Division, Coord{5, 5})
			s.Place(Sowjetunion, Division, Coord{6, 5})
			s.Phase = PhaseAngriff
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
	s := New(3, Wehrmacht)
	clearField(s)
	s.Objects = nil
	u := s.Place(Wehrmacht, Division, Coord{5, 5})
	d := s.Place(Sowjetunion, Kompanie, Coord{6, 5})
	setStrength(d, 1)
	s.Phase = PhaseAngriff
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
	s := New(3, Wehrmacht)
	clearField(s)
	u := s.Place(Wehrmacht, Bataillon, Coord{5, 5})
	setStrength(u, 5)
	for i := 0; i < 3; i++ {
		s.Advance()
	}
	for i := 0; i < 3; i++ {
		s.Advance()
	}
	if u.Strength() != 5 {
		t.Errorf("strength = %d, want 5", u.Strength())
	}
}

func TestConquestAndIncome(t *testing.T) {
	s := New(1, Wehrmacht)
	clearField(s)
	o := cityOf(s, Sowjetunion)
	s.Place(Wehrmacht, Kompanie, o.Pos)
	s.Phase = PhaseAngriff
	s.Advance()
	if o.Owner != Wehrmacht {
		t.Fatal("city not conquered at end of turn")
	}
	if s.Active != Sowjetunion {
		t.Fatalf("active = %v", s.Active)
	}
	if got := s.Stocks[Sowjetunion].Manpower; got != 2 {
		t.Errorf("soviet manpower = %d, want 2", got)
	}
}

func TestCapitalWinsImmediately(t *testing.T) {
	s := New(1, Wehrmacht)
	clearField(s)
	for _, o := range s.Objects {
		if o.Kind == Hauptstadt && o.Owner == Sowjetunion {
			s.Place(Wehrmacht, Kompanie, o.Pos)
		}
	}
	s.Phase = PhaseAngriff
	s.Advance()
	if !s.Over || s.Winner != Wehrmacht {
		t.Fatalf("over=%v winner=%v", s.Over, s.Winner)
	}
}

func TestRoundLimitCountsCities(t *testing.T) {
	s := New(1, Wehrmacht)
	clearField(s)
	cityOf(s, Sowjetunion).Owner = Wehrmacht
	s.Round = MaxRounds
	s.Active = Sowjetunion
	s.Phase = PhaseAngriff
	s.Advance()
	if !s.Over || s.Winner != Wehrmacht {
		t.Fatalf("over=%v winner=%v reason=%q", s.Over, s.Winner, s.Reason)
	}
}

func TestRoundLimitDraw(t *testing.T) {
	s := New(1, Wehrmacht)
	clearField(s)
	s.Round = MaxRounds
	s.Active = Sowjetunion
	s.Phase = PhaseAngriff
	s.Advance()
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
	s.Place(Sowjetunion, Kompanie, near)
	s.aiRecruit()
	u := s.UnitAt(o.Pos)
	if u == nil || u.Kind != Kompanie {
		t.Fatalf("threatened city raised %+v", u)
	}
}

func TestAISavesForDivision(t *testing.T) {
	s := New(1, Wehrmacht)
	clearField(s)
	s.Stocks[Wehrmacht].Manpower = 26
	s.aiRecruit()
	if len(s.Units) != 0 {
		t.Fatalf("raised %s below division cost", s.Units[0].Name)
	}
	s.Stocks[Wehrmacht].Manpower = 27
	s.aiRecruit()
	if len(s.Units) != 1 || s.Units[0].Kind != Division {
		t.Fatalf("units = %d", len(s.Units))
	}
}

func TestAIRecruitsDespiteOccupiedThreatenedCity(t *testing.T) {
	s := New(1, Wehrmacht)
	clearField(s)
	o := cityOf(s, Wehrmacht)
	s.Place(Wehrmacht, Bataillon, o.Pos)
	near := Coord{o.Pos.Col, o.Pos.Row + 2}
	if !OnMap(near) {
		near = Coord{o.Pos.Col, o.Pos.Row - 2}
	}
	s.Place(Sowjetunion, Kompanie, near)
	s.Stocks[Wehrmacht].Manpower = 40
	s.aiRecruit()
	for _, u := range s.Units {
		if u.Kind == Division {
			return
		}
	}
	t.Errorf("no division raised while one city was blocked, manpower left %d", s.Stocks[Wehrmacht].Manpower)
}

func TestAIRespectsAttackRatio(t *testing.T) {
	s := New(1, Wehrmacht)
	clearField(s)
	s.Objects = nil
	u := s.Place(Wehrmacht, Bataillon, Coord{5, 5})
	s.Place(Sowjetunion, Bataillon, Coord{6, 5})
	setStrength(u, 10)
	if s.bestAttackTarget(u) != nil {
		t.Error("attacks at 10 against 9")
	}
	setStrength(u, 11)
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
	t.Logf("Wehrmacht %d, Sowjetunion %d, Unentschieden %d, mittlere Runden %.1f",
		wins[Wehrmacht], wins[Sowjetunion], wins[NoFaction], float64(rounds)/400)
}
