package game

// Faction identifies one of the two sides.
type Faction int

const (
	Wehrmacht Faction = iota
	SovietUnion
	NoFaction
)

// Other returns the opposing faction.
func (f Faction) Other() Faction {
	if f == Wehrmacht {
		return SovietUnion
	}
	return Wehrmacht
}

func (f Faction) String() string {
	switch f {
	case Wehrmacht:
		return "Wehrmacht"
	case SovietUnion:
		return "Soviet Union"
	}
	return "None"
}

// ObjectKind is a map object type.
type ObjectKind int

const (
	Capital ObjectKind = iota
	City
	Village
	Factory
)

func (k ObjectKind) String() string {
	switch k {
	case Capital:
		return "Capital"
	case City:
		return "City"
	case Village:
		return "Village"
	}
	return "Factory"
}

// IsCity reports whether the object earns manpower and raises units.
func (k ObjectKind) IsCity() bool {
	return k == Capital || k == City
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
	Company UnitKind = iota
	Battalion
)

// UnitKinds lists the kinds a city can raise, smallest first.
var UnitKinds = []UnitKind{Company, Battalion}

func (k UnitKind) String() string {
	if k == Company {
		return "Company"
	}
	return "Battalion"
}

// UnitStats holds the game values and the historical figures of a unit type.
type UnitStats struct {
	Cost     int
	Strength int
	Move     int
	Men      string // historical establishment strength
	Note     string // historical structure
}

// statsTable holds the start values from docs/GDD.md, section Units.
// A battalion's strength sums its companies, its map unit is only the staff.
// A company's Move needs the link to its staff, see Unit.Allowance.
var statsTable = [2][unitKindCount]UnitStats{
	Wehrmacht: {
		Company:   {Cost: 1, Strength: 3, Move: 3, Men: "187", Note: "Rifle company"},
		Battalion: {Cost: 3, Strength: 12, Move: 2, Men: "838", Note: "3 rifle companies, 1 MG company"},
	},
	SovietUnion: {
		Company:   {Cost: 1, Strength: 2, Move: 3, Men: "143", Note: "Rifle company"},
		Battalion: {Cost: 2, Strength: 9, Move: 2, Men: "619", Note: "3 rifle companies, MG and mortar company"},
	},
}

// Stats returns the values of a unit type for a faction.
func Stats(f Faction, k UnitKind) UnitStats {
	return statsTable[f][k]
}

// Unit is a unit on the map.
type Unit struct {
	ID      int
	Faction Faction
	Kind    UnitKind
	Pos     Coord
	Element *Element // the unit's place in the order of battle
	Name    string   // full designation
	Short   string   // map label
	Flag    string   // text inside the symbol, where the faction's signs carry one
	Used    int      // fields moved this turn
	Spent   bool     // fought this turn or just raised, no movement left
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
