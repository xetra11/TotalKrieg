package game

import "fmt"

// All names here are placeholders until the list of real formations is
// verified, see the open decisions in docs/GDD.md. Company leaders are invented.
var cityNames = [2][]string{
	Wehrmacht:   {"Stalino", "Mariupol", "Gorlovka"},
	SovietUnion: {"Rostov", "Shakhty", "Novocherkassk"},
}

var callsigns = [2][]string{
	Wehrmacht: {
		"Adler", "Falke", "Wolf", "Bär", "Luchs", "Habicht", "Dachs", "Iltis",
		"Marder", "Sperber", "Uhu", "Hirsch", "Fuchs", "Keiler", "Bussard", "Rabe",
		"Eule", "Otter", "Wiesel", "Milan", "Kranich", "Biber", "Elch", "Schwan",
	},
	SovietUnion: {
		"Sokol", "Berkut", "Yastreb", "Volga", "Don", "Ural", "Amur", "Dnepr",
		"Granit", "Kedr", "Taiga", "Burya", "Beryoza", "Zarya", "Grom", "Strela",
		"Almaz", "Kama", "Oka", "Pamir", "Altai", "Rubin", "Topaz", "Vyuga",
	},
}

// ranks is weighted toward officers. By 1943 NCOs also led companies.
var ranks = [2][]string{
	Wehrmacht:   {"Capt.", "1Lt.", "1Lt.", "2Lt.", "2Lt.", "2Lt.", "MSgt.", "Sgt."},
	SovietUnion: {"Capt.", "Sr.Lt.", "Sr.Lt.", "Lt.", "Lt.", "Lt.", "Jr.Lt.", "Sgt.Maj."},
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
	SovietUnion: {
		"Petrov", "Ivanov", "Sidorov", "Kuznetsov", "Smirnov", "Popov", "Sokolov", "Morozov",
		"Volkov", "Lebedev", "Orlov", "Nikitin", "Zaitsev", "Gusev", "Titov", "Belov",
		"Komarov", "Karpov", "Frolov", "Romanov", "Tarasov", "Yakovlev", "Bykov", "Vlasov",
		"Antonov", "Baranov", "Danilov", "Yegorov", "Fomin", "Gromov", "Isayev", "Kirillov",
		"Lazarev", "Makarov", "Naumov", "Osipov", "Panin", "Rodin", "Savin", "Tikhonov",
		"Uvarov", "Filippov", "Khokhlov", "Tsarev", "Chernov", "Shubin", "Shcherbakov", "Yudich",
		"Alexeyev", "Borisov", "Grigoryev", "Dmitriyev", "Stepanov", "Mikhailov", "Kovalyov", "Medvedev",
	},
}

var regimentBase = [2]int{Wehrmacht: 513, SovietUnion: 10}

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
	if k == Battalion {
		return s.buildBattalion(f, regimentBase[f]+n/3, n%3)
	}
	return s.buildFreeCompany(f, n)
}

// buildBattalion creates the idx-th battalion of a regiment, counted from 0.
func (s *State) buildBattalion(f Faction, reg, idx int) *Element {
	var b *Element
	if f == Wehrmacht {
		b = s.newElement(f, Battalion, nil,
			fmt.Sprintf("%s./Infantry Regiment %d", roman(idx+1), reg),
			fmt.Sprintf("%s./%d", roman(idx+1), reg))
	} else {
		b = s.newElement(f, Battalion, nil,
			fmt.Sprintf("%s Battalion, %s Rifle Regiment", Ordinal(idx+1), Ordinal(reg)),
			fmt.Sprintf("%d/%d RR", idx+1, reg))
	}
	b.Current, b.Full = staffStrength, staffStrength
	rifle := 0
	for i, role := range companyRoles[f] {
		name, short := companyName(f, role, idx, i, rifle, reg)
		if role == RoleRifle {
			rifle++
		}
		s.buildCompany(f, b, role, name, short)
	}
	return b
}

// companyName numbers companies through the regiment. German ones count on
// past the machine gun company, Soviet special companies count by battalion.
func companyName(f Faction, role CompanyRole, battalion, i, rifle, reg int) (name, short string) {
	if f == Wehrmacht {
		num := 4*battalion + i + 1
		return fmt.Sprintf("%d./Infantry Regiment %d", num, reg), fmt.Sprintf("%d./%d", num, reg)
	}
	switch role {
	case RoleMG:
		return fmt.Sprintf("%s MG Company, %s Rifle Regiment", Ordinal(battalion+1), Ordinal(reg)), fmt.Sprintf("%d MG/%d", battalion+1, reg)
	case RoleMortar:
		return fmt.Sprintf("%s Mortar Company, %s Rifle Regiment", Ordinal(battalion+1), Ordinal(reg)), fmt.Sprintf("%d Mtr/%d", battalion+1, reg)
	}
	num := 3*battalion + rifle + 1
	return fmt.Sprintf("%s Rifle Company, %s Rifle Regiment", Ordinal(num), Ordinal(reg)), fmt.Sprintf("%d Rfl/%d", num, reg)
}

func (s *State) buildFreeCompany(f Faction, n int) *Element {
	name, short := fmt.Sprintf("%s Independent Company", Ordinal(n+1)), fmt.Sprintf("%s Coy", Ordinal(n+1))
	if f == SovietUnion {
		short = fmt.Sprintf("%d Ind", n+1)
	}
	return s.buildCompany(f, nil, RoleRifle, name, short)
}

func (s *State) buildCompany(f Faction, parent *Element, role CompanyRole, name, short string) *Element {
	e := s.newElement(f, Company, parent, name, short)
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

// Ordinal writes a number as 1st, 2nd or 3rd.
func Ordinal(n int) string {
	suffix := "th"
	if n%100 < 11 || n%100 > 13 {
		switch n % 10 {
		case 1:
			suffix = "st"
		case 2:
			suffix = "nd"
		case 3:
			suffix = "rd"
		}
	}
	return fmt.Sprintf("%d%s", n, suffix)
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
