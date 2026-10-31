package game

import (
	"slices"
	"testing"
)

func TestLostAttackWaitsForThePlayer(t *testing.T) {
	s := openField()
	u := linked(s, Wehrmacht, Coord{5, 5})
	d := s.Place(SovietUnion, Company, Coord{7, 5})
	if _, err := s.ResolveBattle(u, d.Pos, DefenderWins); err != nil {
		t.Fatal(err)
	}
	r := s.NextRout()
	if r == nil || r.Unit != u || r.Victor != nil {
		t.Fatalf("rout %+v", r)
	}
	options := s.RoutOptions(r)
	if len(options) != 3 {
		t.Fatalf("options %v", options)
	}
	for _, c := range options {
		if Distance(c, d.Pos) <= Distance(u.Pos, d.Pos) {
			t.Errorf("%v leads toward the enemy", c)
		}
	}
	if err := s.Flee(r, d.Pos); err == nil {
		t.Error("fled onto the enemy")
	}
	if err := s.Flee(r, options[1]); err != nil {
		t.Fatal(err)
	}
	if u.Pos != options[1] || s.NextRout() != nil {
		t.Errorf("unit at %v, rout %v", u.Pos, s.NextRout())
	}
}

func TestDrawStartsNoRout(t *testing.T) {
	s := openField()
	u := linked(s, Wehrmacht, Coord{5, 5})
	d := s.Place(SovietUnion, Company, Coord{7, 5})
	if _, err := s.ResolveBattle(u, d.Pos, Draw); err != nil {
		t.Fatal(err)
	}
	if s.NextRout() != nil || d.Pos != (Coord{7, 5}) {
		t.Errorf("rout %v, defender at %v", s.NextRout(), d.Pos)
	}
}

// beaten leaves a human company beaten by an AI company next to it.
func beaten(t *testing.T) (s *State, own, foe *Unit, hex Coord) {
	t.Helper()
	s = openField()
	hex = Coord{5, 5}
	own = s.Place(Wehrmacht, Company, hex)
	s.EndTurn()
	foe = s.Place(SovietUnion, Company, Coord{6, 5})
	res := &FieldResult{Outcome: AttackerWins, Remaining: map[*Element]float64{foe.Element: 1, own.Element: 0.5}}
	if _, err := s.ApplyFieldResult(foe, hex, res); err != nil {
		t.Fatal(err)
	}
	if s.NextRout() == nil {
		t.Fatal("no rout")
	}
	return s, own, foe, hex
}

func TestBeatenPlayerUnitKeepsItsHexUntilItFlees(t *testing.T) {
	s, own, foe, hex := beaten(t)
	if own.Pos != hex || foe.Pos != (Coord{6, 5}) {
		t.Fatalf("own %v, foe %v", own.Pos, foe.Pos)
	}
	r := s.NextRout()
	to := s.RoutOptions(r)[0]
	if err := s.Flee(r, to); err != nil {
		t.Fatal(err)
	}
	if own.Pos != to || foe.Pos != hex {
		t.Errorf("own %v, foe %v", own.Pos, foe.Pos)
	}
}

func TestDestroyedUnitDropsItsRout(t *testing.T) {
	s, own, _, _ := beaten(t)
	s.remove(own)
	if s.NextRout() != nil {
		t.Error("rout of a destroyed unit")
	}
}

func TestSecondDefeatRoutsOnce(t *testing.T) {
	s := openField()
	own := s.Place(Wehrmacht, Company, Coord{5, 5})
	foe := s.Place(SovietUnion, Company, Coord{6, 5})
	s.startRout(own, foe.Pos, foe)
	s.startRout(own, foe.Pos, nil)
	if len(s.routs) != 1 || s.routs[0].Victor != nil {
		t.Errorf("routs %+v", s.routs)
	}
}

func TestSurroundedUnitStays(t *testing.T) {
	s := openField()
	own := s.Place(Wehrmacht, Company, Coord{5, 5})
	foe := s.Place(SovietUnion, Company, Coord{6, 5})
	for _, c := range own.Pos.Neighbors() {
		if s.UnitAt(c) == nil {
			s.Place(Wehrmacht, Company, c)
		}
	}
	s.startRout(own, foe.Pos, foe)
	if s.NextRout() != nil || own.Pos != (Coord{5, 5}) || foe.Pos != (Coord{6, 5}) {
		t.Errorf("rout %v, own %v, foe %v", s.NextRout(), own.Pos, foe.Pos)
	}
}

func TestAIPlayingBothSidesSettlesRouts(t *testing.T) {
	s := openField()
	own := s.Place(Wehrmacht, Company, Coord{5, 5})
	foe := s.Place(SovietUnion, Company, Coord{6, 5})
	s.startRout(own, foe.Pos, nil)
	s.PlayTurn()
	if s.NextRout() != nil || own.Pos == (Coord{5, 5}) {
		t.Errorf("rout %v, own %v", s.NextRout(), own.Pos)
	}
}

func TestAutoOutcomeBeatsTheLargerShare(t *testing.T) {
	cases := []struct {
		attackerLoss, attackers, defenderLoss, defenders int
		want                                             Outcome
	}{
		{1, 3, 1, 2, AttackerWins},
		{2, 3, 1, 3, DefenderWins},
		{1, 2, 1, 2, Draw},
		{1, 6, 2, 3, AttackerWins},
	}
	for _, c := range cases {
		got := autoOutcome(c.attackerLoss, c.attackers, c.defenderLoss, c.defenders)
		if got != c.want {
			t.Errorf("%+v: %v", c, got)
		}
	}
}

func TestEveryOptionIsFree(t *testing.T) {
	s := openField()
	own := s.Place(Wehrmacht, Company, Coord{5, 5})
	foe := s.Place(SovietUnion, Company, Coord{6, 5})
	s.startRout(own, foe.Pos, foe)
	options := s.RoutOptions(s.NextRout())
	blocked := options[0]
	s.Place(Wehrmacht, Company, blocked)
	if slices.Contains(s.RoutOptions(s.NextRout()), blocked) {
		t.Error("fled onto an occupied hex")
	}
}
