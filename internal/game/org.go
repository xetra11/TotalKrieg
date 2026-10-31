package game

const (
	// CommandRange is how far a company may stand from its battalion staff and
	// keep its link.
	CommandRange = 4
	// UnlinkedMove is a company's movement without a link.
	UnlinkedMove = 1

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

// companyRoles lists the companies of a battalion.
var companyRoles = [2][]CompanyRole{
	Wehrmacht:   {RoleRifle, RoleRifle, RoleRifle, RoleMG},
	SovietUnion: {RoleRifle, RoleRifle, RoleRifle, RoleMG, RoleMortar},
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
