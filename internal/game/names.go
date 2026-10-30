package game

import "fmt"

// All names here are placeholders until the list of real formations is
// verified, see the open decisions in docs/GDD.md. Company leaders are invented.
var divisionNumbers = [2][]int{
	Wehrmacht:   {294, 306, 336, 111, 17, 302, 335, 79, 370, 73},
	Sowjetunion: {127, 130, 315, 271, 387, 221, 216, 346, 118, 243},
}

var cityNames = [2][]string{
	Wehrmacht:   {"Stalino", "Mariupol", "Gorlowka"},
	Sowjetunion: {"Rostow", "Schachty", "Nowotscherkassk"},
}

var callsigns = [2][]string{
	Wehrmacht: {
		"Adler", "Falke", "Wolf", "Bär", "Luchs", "Habicht", "Dachs", "Iltis",
		"Marder", "Sperber", "Uhu", "Hirsch", "Fuchs", "Keiler", "Bussard", "Rabe",
		"Eule", "Otter", "Wiesel", "Milan", "Kranich", "Biber", "Elch", "Schwan",
	},
	Sowjetunion: {
		"Sokol", "Berkut", "Jastreb", "Wolga", "Don", "Ural", "Amur", "Dnepr",
		"Granit", "Kedr", "Taiga", "Burja", "Beresa", "Sarja", "Grom", "Strela",
		"Almas", "Kama", "Oka", "Pamir", "Altai", "Rubin", "Topas", "Wjuga",
	},
}

// ranks is weighted toward officers. By 1943 NCOs also led companies.
var ranks = [2][]string{
	Wehrmacht:   {"Hptm.", "Oblt.", "Oblt.", "Lt.", "Lt.", "Lt.", "Hfw.", "Fw."},
	Sowjetunion: {"Kpt.", "St.Lt.", "St.Lt.", "Lt.", "Lt.", "Lt.", "U.Lt.", "Stsch."},
}

var surnames = [2][]string{
	Wehrmacht: {
		"Keller", "Brandt", "Vogt", "Lorenz", "Hartmann", "Krause", "Seidel", "Winkler",
		"Pohl", "Engel", "Busch", "Kuhn", "Lange", "Franke", "Sommer", "Thiel",
		"Roth", "Berger", "Haas", "Schulte", "Dietrich", "Ernst", "Graf", "Ludwig",
		"Albrecht", "Baumann", "Conrad", "Dorn", "Ebert", "Fiedler", "Gerlach", "Hahn",
		"Jansen", "Kessler", "Lindner", "Maurer", "Nagel", "Ott", "Peters", "Reuter",
		"Sauer", "Timm", "Ulrich", "Voigt", "Weber", "Ziegler", "Arnold", "Bock",
	},
	Sowjetunion: {
		"Petrow", "Iwanow", "Sidorow", "Kusnezow", "Smirnow", "Popow", "Sokolow", "Morosow",
		"Wolkow", "Lebedew", "Orlow", "Nikitin", "Saizew", "Gusew", "Titow", "Below",
		"Komarow", "Karpow", "Frolow", "Romanow", "Tarasow", "Jakowlew", "Bykow", "Wlassow",
		"Antonow", "Baranow", "Danilow", "Jegorow", "Fomin", "Gromow", "Issajew", "Kirillow",
		"Lasarew", "Makarow", "Naumow", "Ossipow", "Panin", "Rodin", "Sawin", "Tichonow",
		"Uwarow", "Filippow", "Chochlow", "Zarew", "Tschernow", "Schubin", "Schtscherbakow", "Juditsch",
		"Alexejew", "Borissow", "Grigorjew", "Dmitrijew", "Stepanow", "Michailow", "Kowaljow", "Medwedew",
	},
}

func (s *State) newElement(f Faction, k UnitKind, parent *Element, name, short string) *Element {
	e := &Element{Faction: f, Kind: k, Name: name, Short: short, Parent: parent}
	if parent != nil {
		parent.Children = append(parent.Children, e)
	}
	return e
}

// buildFormation creates a new formation with its full order of battle.
func (s *State) buildFormation(f Faction, k UnitKind) *Element {
	n := s.counts[f][k]
	s.counts[f][k]++
	switch k {
	case Division:
		return s.buildDivision(f, n)
	case Bataillon:
		reg := regimentBase[f] + n/3
		return s.buildBattalion(f, nil, reg, n%3)
	}
	return s.buildFreeCompany(f, n)
}

var (
	regimentBase         = [2]int{Wehrmacht: 513, Sowjetunion: 10}
	divisionRegimentBase = [2]int{Wehrmacht: 601, Sowjetunion: 101}
)

func (s *State) buildDivision(f Faction, n int) *Element {
	num := divisionNumbers[f][n%len(divisionNumbers[f])]
	name, short := fmt.Sprintf("%d. Infanterie-Division", num), fmt.Sprintf("%d. I.D.", num)
	if f == Sowjetunion {
		name, short = fmt.Sprintf("%d. Schützendivision", num), fmt.Sprintf("%d сд", num)
	}
	div := s.newElement(f, Division, nil, name, short)
	for r := 0; r < 3; r++ {
		reg := divisionRegimentBase[f] + 3*n + r
		rname, rshort := fmt.Sprintf("Infanterie-Regiment %d", reg), fmt.Sprintf("I.R. %d", reg)
		if f == Sowjetunion {
			rname, rshort = fmt.Sprintf("%d. Schützenregiment", reg), fmt.Sprintf("%d сп", reg)
		}
		regiment := s.newElement(f, Regiment, div, rname, rshort)
		for b := 0; b < 3; b++ {
			s.buildBattalion(f, regiment, reg, b)
		}
	}
	return div
}

// buildBattalion creates the idx-th battalion of a regiment, counted from 0.
func (s *State) buildBattalion(f Faction, parent *Element, reg, idx int) *Element {
	if f == Wehrmacht {
		b := s.newElement(f, Bataillon, parent,
			fmt.Sprintf("%s./Infanterie-Regiment %d", roman(idx+1), reg),
			fmt.Sprintf("%s./%d", roman(idx+1), reg))
		// Companies run through the regiment, every fourth is a machine gun company.
		for i := 0; i < 4; i++ {
			num := 4*idx + i + 1
			role := RoleSchuetzen
			if i == 3 {
				role = RoleMG
			}
			s.buildCompany(f, b, role, fmt.Sprintf("%d./Infanterie-Regiment %d", num, reg), fmt.Sprintf("%d./%d", num, reg))
		}
		return b
	}
	b := s.newElement(f, Bataillon, parent,
		fmt.Sprintf("%d. Bataillon, %d. Schützenregiment", idx+1, reg),
		fmt.Sprintf("%d/%d сп", idx+1, reg))
	for i := 0; i < 3; i++ {
		num := 3*idx + i + 1
		s.buildCompany(f, b, RoleSchuetzen, fmt.Sprintf("%d. Schützenkompanie, %d. Schützenregiment", num, reg), fmt.Sprintf("%d ср/%d", num, reg))
	}
	s.buildCompany(f, b, RoleMG, fmt.Sprintf("%d. MG-Kompanie, %d. Schützenregiment", idx+1, reg), fmt.Sprintf("%d пр/%d", idx+1, reg))
	s.buildCompany(f, b, RoleGranatwerfer, fmt.Sprintf("%d. Granatwerfer-Kompanie, %d. Schützenregiment", idx+1, reg), fmt.Sprintf("%d мр/%d", idx+1, reg))
	return b
}

func (s *State) buildFreeCompany(f Faction, n int) *Element {
	name, short := fmt.Sprintf("%d. selbständige Kompanie", n+1), fmt.Sprintf("%d. Kp.", n+1)
	if f == Sowjetunion {
		short = fmt.Sprintf("%d оср", n+1)
	}
	return s.buildCompany(f, nil, RoleSchuetzen, name, short)
}

func (s *State) buildCompany(f Faction, parent *Element, role CompanyRole, name, short string) *Element {
	e := s.newElement(f, Kompanie, parent, name, short)
	e.Role = role
	e.Full = companyStrength[f][role]
	e.Current = e.Full
	e.Callsign = s.nextCallsign(f)
	e.Leader = s.pick(ranks[f]) + " " + s.nextSurname(f)
	return e
}

// nextSurname deals from a shuffled pool, so names only repeat once it is empty.
func (s *State) nextSurname(f Faction) string {
	if len(s.surnamePool[f]) == 0 {
		pool := append([]string(nil), surnames[f]...)
		s.rng.Shuffle(len(pool), func(i, j int) { pool[i], pool[j] = pool[j], pool[i] })
		s.surnamePool[f] = pool
	}
	name := s.surnamePool[f][0]
	s.surnamePool[f] = s.surnamePool[f][1:]
	return name
}

// nextCallsign deals out the list, then starts over with a number.
func (s *State) nextCallsign(f Faction) string {
	list := callsigns[f]
	n := s.callsignsUsed[f]
	s.callsignsUsed[f]++
	name := list[n%len(list)]
	if round := n / len(list); round > 0 {
		return fmt.Sprintf("%s %d", name, round+1)
	}
	return name
}

func (s *State) pick(list []string) string {
	return list[s.rng.IntN(len(list))]
}

func roman(n int) string {
	switch n {
	case 1:
		return "I"
	case 2:
		return "II"
	}
	return "III"
}
