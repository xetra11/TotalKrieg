package game

const (
	// CommandRange is how far a company may stand from its battalion staff and
	// keep its link.
	CommandRange = 4
	// UnlinkedMove is a company's movement without a link.
	UnlinkedMove = 1
	// SupplyRange is how far a unit may stand from an own city and still be
	// reinforced each round.
	SupplyRange = 3
	// MaxBattalionCompanies is the most companies a battalion may field.
	MaxBattalionCompanies = 4

	staffStrength = 1
)

// CompanyRole is the type of a company.
type CompanyRole int

const (
	RoleRifle CompanyRole = iota
	RoleMG
	RoleMortar
)

func (r CompanyRole) String() string {
	switch r {
	case RoleMG:
		return "MG company"
	case RoleMortar:
		return "Mortar company"
	}
	return "Rifle company"
}

var companyStrength = [2][3]int{
	Wehrmacht:   {RoleRifle: 3, RoleMG: 3},
	SovietUnion: {RoleRifle: 2, RoleMG: 2, RoleMortar: 1},
}

// companyRoles lists the companies of a battalion. Both sides stay at four so
// a battalion never exceeds MaxBattalionCompanies.
var companyRoles = [2][]CompanyRole{
	Wehrmacht:   {RoleRifle, RoleRifle, RoleRifle, RoleMG},
	SovietUnion: {RoleRifle, RoleRifle, RoleRifle, RoleMG},
}

// BattalionSize is the number of companies in a battalion.
func BattalionSize(f Faction) int {
	return len(companyRoles[f])
}

// Element is a node of the order of battle. It keeps its place when its unit
// is destroyed, so losses stay visible.
type Element struct {
	Faction  Faction
	Kind     UnitKind
	Name     string
	Short    string
	Parent   *Element
	Children []*Element

	Role     CompanyRole
	Callsign string
	Leader   string
	Current  int
	Full     int

	// Home is the city a battalion was raised from, nil for independent
	// companies and formations placed without a city.
	Home *Object

	// Token is the element's unit on the map, nil once destroyed.
	Token *Unit
}

// Root returns the top of the element's formation.
func (e *Element) Root() *Element {
	for e.Parent != nil {
		e = e.Parent
	}
	return e
}

// Strength is a company's value, or the sum of a battalion's companies.
func (e *Element) Strength() int {
	if e.Kind == Company {
		return e.Current
	}
	sum := 0
	for _, c := range e.Children {
		sum += c.Current
	}
	return sum
}

// FullStrength is the establishment value of Strength.
func (e *Element) FullStrength() int {
	if e.Kind == Company {
		return e.Full
	}
	sum := 0
	for _, c := range e.Children {
		sum += c.Full
	}
	return sum
}

// Tokens lists the map units of a formation.
func (e *Element) Tokens() []*Unit {
	var out []*Unit
	if e.Token != nil {
		out = append(out, e.Token)
	}
	for _, c := range e.Children {
		out = append(out, c.Tokens()...)
	}
	return out
}

// alive reports whether the element still has its unit on the map. It also
// catches elements whose token was dropped without clearing the pointer, as
// test helpers and hand-built states may do.
func (s *State) alive(e *Element) bool {
	return e != nil && e.Token != nil && s.UnitAt(e.Token.Pos) == e.Token
}

// LivingCompanies counts a battalion's companies that are still on the map.
func (s *State) LivingCompanies(b *Element) int {
	n := 0
	for _, c := range b.Children {
		if s.alive(c) {
			n++
		}
	}
	return n
}

// CityBattalion returns the living battalion raised from a city, or nil.
func (s *State) CityBattalion(o *Object) *Element {
	if o == nil || !o.Kind.IsCity() {
		return nil
	}
	for _, e := range s.elements {
		if e.Kind == Battalion && e.Home == o && e.Faction == o.Owner && s.alive(e) {
			return e
		}
	}
	return nil
}

// CanAssignCompany reports whether a company raised in the city would join its
// battalion. That needs a living battalion with a free slot.
func (s *State) CanAssignCompany(o *Object) bool {
	if o == nil || !o.Kind.IsCity() || o.Owner != s.Active {
		return false
	}
	b := s.CityBattalion(o)
	if b == nil {
		return false
	}
	revive, add := s.companySlot(b)
	return revive != nil || add
}

// companySlot finds room in a battalion. It returns a destroyed company to
// revive, or add=true when the battalion may take a new company element.
func (s *State) companySlot(b *Element) (revive *Element, add bool) {
	for _, c := range b.Children {
		if !s.alive(c) {
			return c, false
		}
	}
	return nil, len(b.Children) < MaxBattalionCompanies
}

// inSupply reports whether an own city is within SupplyRange of the unit.
func (s *State) inSupply(u *Unit) bool {
	for _, o := range s.Objects {
		if o.Owner == u.Faction && o.Kind.IsCity() && Distance(u.Pos, o.Pos) <= SupplyRange {
			return true
		}
	}
	return false
}

// Strength is the unit's current fighting value.
func (u *Unit) Strength() int {
	return u.Element.Current
}

// FullStrength is the unit's establishment value.
func (u *Unit) FullStrength() int {
	return u.Element.Full
}

// Staff returns the company's battalion staff on the map, or nil.
func (u *Unit) Staff() *Unit {
	if u.Element.Parent == nil {
		return nil
	}
	return u.Element.Parent.Token
}

// Linked reports whether a company stands within reach of its battalion staff.
func (u *Unit) Linked() bool {
	staff := u.Staff()
	return staff != nil && Distance(u.Pos, staff.Pos) <= CommandRange
}

// Allowance is the unit's movement this turn. A company's depends on its link,
// so moving the staff up restores it at once.
func (u *Unit) Allowance() int {
	if u.Kind == Company && !u.Linked() {
		return UnlinkedMove
	}
	return u.Stats().Move
}

// MovesLeft is the movement the unit has left this turn.
func (u *Unit) MovesLeft() int {
	if u.Spent {
		return 0
	}
	return max(0, u.Allowance()-u.Used)
}

// takeLosses spreads a loss over units, strongest first.
func takeLosses(units []*Unit, n int) {
	for ; n > 0; n-- {
		var strongest *Unit
		for _, u := range units {
			if u.Strength() > 0 && (strongest == nil || u.Strength() > strongest.Strength()) {
				strongest = u
			}
		}
		if strongest == nil {
			return
		}
		strongest.Element.Current--
	}
}

func totalStrength(units []*Unit) int {
	sum := 0
	for _, u := range units {
		sum += u.Strength()
	}
	return sum
}
