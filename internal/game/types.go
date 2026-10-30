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
	Regiment // appears in the order of battle only, never as a map unit
)

// UnitKinds lists the kinds a city can raise, smallest first.
var UnitKinds = []UnitKind{Kompanie, Bataillon, Division}

func (k UnitKind) String() string {
	switch k {
	case Kompanie:
		return "Kompanie"
	case Bataillon:
		return "Bataillon"
	case Regiment:
		return "Regiment"
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
// Strengths are the sums of the companies, see companyStrength.
var statsTable = [2][3]UnitStats{
	Wehrmacht: {
		Kompanie:  {Cost: 1, Strength: 3, Move: 3, Men: "187", Note: "Schützenkompanie"},
		Bataillon: {Cost: 3, Strength: 12, Move: 2, Men: "838", Note: "3 Schützenkompanien, 1 MG-Kompanie"},
		Division:  {Cost: 27, Strength: 108, Move: 1, Men: "16.977", Note: "3 Regimenter zu je 3 Bataillonen"},
	},
	Sowjetunion: {
		Kompanie:  {Cost: 1, Strength: 2, Move: 3, Men: "143", Note: "Schützenkompanie"},
		Bataillon: {Cost: 2, Strength: 9, Move: 2, Men: "619", Note: "3 Schützenkompanien, MG- und Granatwerfer-Kompanie"},
		Division:  {Cost: 18, Strength: 81, Move: 1, Men: "ca. 9.500", Note: "3 Regimenter zu je 3 Bataillonen"},
	},
}

// Stats returns the values of a unit type for a faction.
func Stats(f Faction, k UnitKind) UnitStats {
	return statsTable[f][k]
}

// Unit is a unit on the map.
type Unit struct {
	ID        int
	Faction   Faction
	Kind      UnitKind
	Pos       Coord
	Element   *Element // the unit's place in the order of battle
	Name      string   // full designation
	Short     string   // map label
	Flag      string   // text inside the symbol, where the faction's signs carry one
	MovesLeft int      // fields left this turn, 0 after an attack or when just raised
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
