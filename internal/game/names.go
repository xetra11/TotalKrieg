package game

import "fmt"

// The division numbers are placeholders until the list of real formations is
// verified, see the open decisions in docs/GDD.md.
var divisionNumbers = [2][]int{
	Wehrmacht:   {294, 306, 336, 111, 17, 302, 335, 79, 370, 73},
	Sowjetunion: {127, 130, 315, 271, 387, 221, 216, 346, 118, 243},
}

var cityNames = [2][]string{
	Wehrmacht:   {"Stalino", "Mariupol", "Gorlowka"},
	Sowjetunion: {"Rostow", "Schachty", "Nowotscherkassk"},
}

// nameUnit assigns the designation for the n-th unit of a kind, counted from 0.
func nameUnit(u *Unit, n int) {
	switch u.Faction {
	case Wehrmacht:
		switch u.Kind {
		case Division:
			num := divisionNumber(Wehrmacht, n)
			u.Name = fmt.Sprintf("%d. Infanterie-Division", num)
			u.Short = fmt.Sprintf("%d. I.D.", num)
		case Bataillon:
			u.Name = fmt.Sprintf("%s./Infanterie-Regiment %d", roman(n%3+1), 513+n/3)
			u.Short = fmt.Sprintf("%s./%d", roman(n%3+1), 513+n/3)
		default:
			u.Name = fmt.Sprintf("%d./Infanterie-Regiment %d", n%12+1, 513+n/12)
			u.Short = fmt.Sprintf("%d./%d", n%12+1, 513+n/12)
		}
	default:
		switch u.Kind {
		case Division:
			num := divisionNumber(Sowjetunion, n)
			u.Name = fmt.Sprintf("%d. Schützendivision", num)
			u.Short = fmt.Sprintf("%d сд", num)
			u.Flag = u.Short
		case Bataillon:
			u.Name = fmt.Sprintf("%d. Bataillon, %d. Schützenregiment", n%3+1, 10+n/3)
			u.Short = fmt.Sprintf("%d/%d сп", n%3+1, 10+n/3)
			u.Flag = fmt.Sprint(n%3 + 1)
		default:
			u.Name = fmt.Sprintf("%d. Kompanie, %d. Schützenregiment", n%9+1, 10+n/9)
			u.Short = fmt.Sprintf("%d/%d сп", n%9+1, 10+n/9)
		}
	}
}

func divisionNumber(f Faction, n int) int {
	list := divisionNumbers[f]
	return list[n%len(list)]
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
