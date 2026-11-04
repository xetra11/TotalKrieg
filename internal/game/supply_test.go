package game

import "testing"

func TestSupplyReinforcesUnitsInRange(t *testing.T) {
	s := openField()
	s.Objects = []*Object{{Kind: City, Pos: Coord{5, 5}, Owner: Wehrmacht}}

	near := s.Place(Wehrmacht, Company, Coord{6, 5})
	near.Element.Current = 1
	far := s.Place(Wehrmacht, Company, Coord{5, 5 + SupplyRange + 1})
	far.Element.Current = 1

	s.beginTurn()

	if got := near.Strength(); got != 2 {
		t.Errorf("near strength = %d, want 2", got)
	}
	if got := far.Strength(); got != 1 {
		t.Errorf("far strength = %d, want 1", got)
	}

	// Supply tops a unit up to its establishment strength and never past it.
	near.Element.Current = near.FullStrength() - 1
	s.beginTurn()
	if got, want := near.Strength(), near.FullStrength(); got != want {
		t.Errorf("near strength = %d, want full %d", got, want)
	}
	s.beginTurn()
	if got, want := near.Strength(), near.FullStrength(); got != want {
		t.Errorf("supply raised past full: %d, want %d", got, want)
	}
}

func TestSupplyOnlyComesFromOwnCities(t *testing.T) {
	s := openField()
	s.Objects = []*Object{{Kind: City, Pos: Coord{5, 5}, Owner: SovietUnion}}

	u := s.Place(Wehrmacht, Company, Coord{6, 5})
	u.Element.Current = 1

	s.beginTurn()

	if got := u.Strength(); got != 1 {
		t.Errorf("enemy city reinforced the unit: %d", got)
	}
}
