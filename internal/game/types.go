package game

// Faction identifies one of the two sides.
type Faction int

const (
	Wehrmacht Faction = iota
	Sowjetunion
	NoFaction
)

// Other returns the opposing faction.
func (f Faction) Other() Faction {
	if f == Wehrmacht {
		return Sowjetunion
	}
	return Wehrmacht
}

func (f Faction) String() string {
	switch f {
	case Wehrmacht:
		return "Wehrmacht"
	case Sowjetunion:
		return "Sowjetunion"
	}
	return "Keine"
}

// ObjectKind is a map object type.
type ObjectKind int

const (
	Hauptstadt ObjectKind = iota
	Stadt
	Dorf
	Fabrik
)

func (k ObjectKind) String() string {
	switch k {
	case Hauptstadt:
		return "Hauptstadt"
	case Stadt:
		return "Stadt"
	case Dorf:
		return "Dorf"
	}
	return "Fabrik"
}

// IsCity reports whether the object earns manpower and raises units.
func (k ObjectKind) IsCity() bool {
	return k == Hauptstadt || k == Stadt
}

// Object is a capturable map object.
type Object struct {
	Kind      ObjectKind
	Pos       Coord
	Owner     Faction
	Name      string
	Recruited bool // raised a unit this round
}

// UnitKind is a unit size.
type UnitKind int

const (
	Kompanie UnitKind = iota
	Bataillon
	Division
)

// UnitKinds lists all kinds, smallest first.
var UnitKinds = []UnitKind{Kompanie, Bataillon, Division}

func (k UnitKind) String() string {
	switch k {
	case Kompanie:
		return "Kompanie"
	case Bataillon:
		return "Bataillon"
	}
	return "Division"
}

// UnitStats holds the game values and the historical figures of a unit type.
type UnitStats struct {
	Cost     int
	Strength int
	Move     int
	Men      string // historical establishment strength
	Note     string // historical structure
}

// statsTable holds the start values from docs/GDD.md, section Einheiten.
var statsTable = [2][3]UnitStats{
	Wehrmacht: {
		Kompanie:  {Cost: 1, Strength: 4, Move: 3, Men: "187", Note: "Schützenkompanie"},
		Bataillon: {Cost: 3, Strength: 12, Move: 2, Men: "838", Note: "3 Schützenkompanien, 1 MG-Kompanie"},
		Division:  {Cost: 12, Strength: 48, Move: 1, Men: "16.977", Note: "Infanterie-Division, 3 Regimenter"},
	},
	Sowjetunion: {
		Kompanie:  {Cost: 1, Strength: 3, Move: 3, Men: "143", Note: "Schützenkompanie"},
		Bataillon: {Cost: 2, Strength: 9, Move: 2, Men: "619", Note: "3 Schützenkompanien, MG- und Granatwerfer-Kompanie"},
		Division:  {Cost: 9, Strength: 36, Move: 1, Men: "ca. 9.500", Note: "Schützendivision, 3 Regimenter"},
	},
}

// Stats returns the values of a unit type for a faction.
func Stats(f Faction, k UnitKind) UnitStats {
	return statsTable[f][k]
}

// Unit is a unit on the map.
type Unit struct {
	ID       int
	Faction  Faction
	Kind     UnitKind
	Pos      Coord
	Strength int
	Name     string // full designation
	Short    string // map label
	Flag     string // text inside the symbol, where the faction's signs carry one
	Moved    bool
	Attacked bool
	Fresh    bool // raised this round, may not move yet
}

// Stats returns the values of the unit's type.
func (u *Unit) Stats() UnitStats {
	return Stats(u.Faction, u.Kind)
}

// Stock is a faction's pool.
type Stock struct {
	Manpower  int
	Resources int
}

// Phase is a step of a faction's turn. Income runs when the turn starts,
// conquest when it ends.
type Phase int

const (
	PhaseAufstellung Phase = iota
	PhaseBewegung
	PhaseAngriff
)

func (p Phase) String() string {
	switch p {
	case PhaseAufstellung:
		return "Aufstellung"
	case PhaseBewegung:
		return "Bewegung"
	}
	return "Angriff"
}

// Report records a finished fight.
type Report struct {
	Round             int
	Attacker          string
	Defender          string
	AttackerFaction   Faction
	AttackerLoss      int
	DefenderLoss      int
	AttackerDestroyed bool
	DefenderDestroyed bool
	Pos               Coord
}
