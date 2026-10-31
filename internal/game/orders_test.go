package game

import "testing"

func commit(s *State) {
	for range s.CommitTurn() {
	}
}

func TestPlanMovesNothing(t *testing.T) {
	s := openField()
	u := linked(s, Wehrmacht, Coord{5, 5})
	if err := s.Plan(u, Coord{8, 5}); err != nil {
		t.Fatal(err)
	}
	if u.Pos != (Coord{5, 5}) || u.MovesLeft() != 3 {
		t.Errorf("unit at %v, moves %d", u.Pos, u.MovesLeft())
	}
	if err := s.Plan(u, Coord{5, 7}); err != nil {
		t.Fatal(err)
	}
	if len(s.Orders()) != 1 || s.OrderOf(u).To != (Coord{5, 7}) {
		t.Fatalf("orders %+v", s.Orders())
	}
	s.CancelOrder(u)
	if s.OrderOf(u) != nil {
		t.Error("order survived")
	}
}

func TestPlanChecksReach(t *testing.T) {
	s := openField()
	u := linked(s, Wehrmacht, Coord{5, 5})
	if err := s.Plan(u, Coord{9, 5}); err == nil {
		t.Error("planned beyond the movement")
	}
	spent := s.Place(Wehrmacht, Company, Coord{12, 5})
	spent.Spent = true
	if err := s.Plan(spent, Coord{12, 6}); err == nil {
		t.Error("planned for a spent unit")
	}
	foe := s.Place(SovietUnion, Company, Coord{20, 5})
	if err := s.Plan(foe, Coord{20, 6}); err == nil {
		t.Error("planned for the enemy")
	}
}

func TestOrdersClaimTheirHex(t *testing.T) {
	s := openField()
	a := linked(s, Wehrmacht, Coord{5, 5})
	b := s.Place(Wehrmacht, Company, Coord{7, 5})
	to := Coord{6, 5}
	if err := s.Plan(a, to); err != nil {
		t.Fatal(err)
	}
	if err := s.Plan(b, to); err == nil {
		t.Error("two units ordered onto one hex")
	}
	if _, ok := s.OrderTargets(a)[to]; !ok {
		t.Error("a unit lost its own hex")
	}
}

func TestCommitCarriesOutOrdersAndEndsTheTurn(t *testing.T) {
	s := openField()
	u := linked(s, Wehrmacht, Coord{5, 5})
	if err := s.Plan(u, Coord{8, 5}); err != nil {
		t.Fatal(err)
	}
	commit(s)
	if u.Pos != (Coord{8, 5}) || s.Active != SovietUnion || len(s.Orders()) != 0 {
		t.Errorf("unit at %v, active %v, orders %d", u.Pos, s.Active, len(s.Orders()))
	}
}

func TestMovedCompanySupportsTheAttack(t *testing.T) {
	s := openField()
	target := Coord{10, 8}
	s.Place(SovietUnion, Company, target)
	n := target.Neighbors()
	attacker := linked(s, Wehrmacht, n[0])
	support := s.Place(Wehrmacht, Company, n[1].Neighbors()[0])
	if Distance(support.Pos, target) != 2 {
		t.Fatalf("support at %v", support.Pos)
	}
	if err := s.Plan(attacker, target); err != nil {
		t.Fatal(err)
	}
	if err := s.Plan(support, n[1]); err != nil {
		t.Fatal(err)
	}
	var fights int
	for f := range s.CommitTurn() {
		fights++
		attackers, _ := s.Engagement(f.Unit, f.Target)
		if len(attackers) != 2 {
			t.Errorf("%d attackers, want 2", len(attackers))
		}
	}
	if fights != 1 || len(s.Reports) != 1 {
		t.Errorf("%d fights, %d reports", fights, len(s.Reports))
	}
}

func TestStaffMovesAfterTheAttacks(t *testing.T) {
	s := openField()
	u := linked(s, Wehrmacht, Coord{5, 5})
	staff := u.Staff()
	target := Coord{8, 5}
	s.Place(SovietUnion, Company, target)
	if err := s.Plan(staff, staff.Pos.Neighbors()[0]); err != nil {
		t.Fatal(err)
	}
	if err := s.Plan(u, target); err != nil {
		t.Fatal(err)
	}
	moved := staff.Pos
	for f := range s.CommitTurn() {
		if staff.Pos != moved || f.Unit != u {
			t.Errorf("staff at %v during the fight", staff.Pos)
		}
	}
	if staff.Pos == moved {
		t.Error("staff stayed")
	}
}

func TestAttackOnAGoneTargetLapses(t *testing.T) {
	s := openField()
	u := linked(s, Wehrmacht, Coord{5, 5})
	foe := s.Place(SovietUnion, Company, Coord{7, 5})
	if err := s.Plan(u, foe.Pos); err != nil {
		t.Fatal(err)
	}
	s.remove(foe)
	commit(s)
	if len(s.Reports) != 0 || u.Pos != (Coord{5, 5}) || s.Active != SovietUnion {
		t.Errorf("reports %d, unit at %v, active %v", len(s.Reports), u.Pos, s.Active)
	}
}

func TestResolvedFightIsNotFoughtAgain(t *testing.T) {
	s := openField()
	u := linked(s, Wehrmacht, Coord{5, 5})
	foe := s.Place(SovietUnion, Company, Coord{7, 5})
	if err := s.Plan(u, foe.Pos); err != nil {
		t.Fatal(err)
	}
	for f := range s.CommitTurn() {
		f.Resolved = true
	}
	if len(s.Reports) != 0 {
		t.Errorf("%d reports", len(s.Reports))
	}
}

func TestAITurnStopsAtItsFights(t *testing.T) {
	s := openField()
	own := s.Place(Wehrmacht, Company, Coord{5, 5})
	s.EndTurn()
	foe := s.Place(SovietUnion, Company, Coord{6, 5})
	setStrength(foe, 6)
	fights := 0
	for f := range s.AITurn() {
		fights++
		if f.Unit != foe || f.Target != own.Pos || s.Active != SovietUnion {
			t.Errorf("fight %+v", f)
		}
		if _, err := s.Attack(f.Unit, f.Target); err != nil {
			t.Fatal(err)
		}
		f.Resolved = true
	}
	if fights != 1 || len(s.Reports) != 1 || s.Active != Wehrmacht {
		t.Errorf("%d fights, %d reports, active %v", fights, len(s.Reports), s.Active)
	}
}

func TestBeatenAttackerSitsOutItsNextTurn(t *testing.T) {
	s := openField()
	u := linked(s, Wehrmacht, Coord{5, 5})
	d := s.Place(SovietUnion, Company, Coord{7, 5})
	if _, err := s.ResolveBattle(u, d.Pos, DefenderWins); err != nil {
		t.Fatal(err)
	}
	if !u.Beaten || !u.OutOfAction() || d.OutOfAction() {
		t.Fatalf("beaten %v, defender %v", u.Beaten, d.OutOfAction())
	}
	s.EndTurn()
	if !u.OutOfAction() {
		t.Error("mark gone before the turn it sits out")
	}
	s.EndTurn()
	if !u.Resting || u.MovesLeft() != 0 {
		t.Fatalf("resting %v, moves %d", u.Resting, u.MovesLeft())
	}
	if len(s.AttackTargets(u)) != 0 || len(s.Reachable(u)) != 0 {
		t.Error("resting unit can act")
	}
	s.EndTurn()
	if u.OutOfAction() {
		t.Error("mark stayed after the turn")
	}
	s.EndTurn()
	if u.MovesLeft() == 0 {
		t.Error("unit still can't move")
	}
}

func TestBeatenDefendersSitOutTheirNextTurn(t *testing.T) {
	s := openField()
	target := Coord{10, 8}
	d := s.Place(SovietUnion, Company, target)
	n := target.Neighbors()
	support := s.Place(SovietUnion, Company, n[3])
	u := linked(s, Wehrmacht, n[0])
	res := &FieldResult{Outcome: AttackerWins, Remaining: map[*Element]float64{u.Element: 1, d.Element: 0.5, support.Element: 1}}
	if _, err := s.ApplyFieldResult(u, target, res); err != nil {
		t.Fatal(err)
	}
	s.EndTurn()
	if !d.Resting || !support.Resting || d.MovesLeft() != 0 {
		t.Fatalf("resting %v and %v", d.Resting, support.Resting)
	}
}

func TestRestingUnitCanBeAttacked(t *testing.T) {
	s := openField()
	u := linked(s, Wehrmacht, Coord{5, 5})
	d := s.Place(SovietUnion, Company, Coord{7, 5})
	d.Resting = true
	if _, ok := s.AttackTargets(u)[d.Pos]; !ok {
		t.Error("resting unit is no target")
	}
}

func TestDrawBeatsNobody(t *testing.T) {
	s := openField()
	u := linked(s, Wehrmacht, Coord{5, 5})
	d := s.Place(SovietUnion, Company, Coord{7, 5})
	if _, err := s.ResolveBattle(u, d.Pos, Draw); err != nil {
		t.Fatal(err)
	}
	if u.OutOfAction() || d.OutOfAction() {
		t.Error("a draw beat a side")
	}
}
