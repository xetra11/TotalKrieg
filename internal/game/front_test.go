package game

import "testing"

func TestControllerFollowsObjectsAndUnits(t *testing.T) {
	s := openField()
	s.Objects = []*Object{
		{Kind: City, Pos: Coord{2, 8}, Owner: Wehrmacht},
		{Kind: City, Pos: Coord{21, 8}, Owner: SovietUnion},
	}

	if got := s.Controller(Coord{2, 8}); got != Wehrmacht {
		t.Errorf("Wehrmacht city hex controlled by %v", got)
	}
	if got := s.Controller(Coord{21, 8}); got != SovietUnion {
		t.Errorf("Soviet city hex controlled by %v", got)
	}
	if got := s.Controller(Coord{0, 8}); got != Wehrmacht {
		t.Errorf("western rear controlled by %v", got)
	}
	if got := s.Controller(Coord{23, 8}); got != SovietUnion {
		t.Errorf("eastern rear controlled by %v", got)
	}

	// A unit controls the hex it stands on, even deep in enemy ground.
	raider := s.Place(Wehrmacht, Company, Coord{19, 8})
	if got := s.Controller(raider.Pos); got != Wehrmacht {
		t.Errorf("raider hex controlled by %v", got)
	}

	// Two opposing companies meet; moving one shifts the boundary.
	soviet := s.Place(SovietUnion, Company, Coord{10, 8})
	german := s.Place(Wehrmacht, Company, Coord{11, 8})
	if got := s.Controller(german.Pos); got != Wehrmacht {
		t.Errorf("German company hex controlled by %v", got)
	}
	if got := s.Controller(soviet.Pos); got != SovietUnion {
		t.Errorf("Soviet company hex controlled by %v", got)
	}

	german.Pos = Coord{13, 8}
	if got := s.Controller(Coord{12, 8}); got != Wehrmacht {
		t.Errorf("hex 12 controlled by %v after the advance", got)
	}
	if got := s.Controller(Coord{11, 8}); got != SovietUnion {
		t.Errorf("hex 11 controlled by %v after the advance", got)
	}
}

func TestInitialFrontSplitsTheMap(t *testing.T) {
	s := New(7, Wehrmacht)
	for row := 0; row < MapRows; row++ {
		if got := s.Controller(Coord{0, row}); got != Wehrmacht {
			t.Errorf("row %d: western edge controlled by %v", row, got)
		}
		if got := s.Controller(Coord{MapCols - 1, row}); got != SovietUnion {
			t.Errorf("row %d: eastern edge controlled by %v", row, got)
		}
	}
}
