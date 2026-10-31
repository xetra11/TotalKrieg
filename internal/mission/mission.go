// Package mission is the adapter to MiusMissionMaker. It turns a TotalKrieg
// fight into a scenario for Graviteam Tactics: Mius-Front and starts it.
//
// The mapping is fixed: every company becomes its company HQ platoon plus the
// platoons of its role, and every battalion staff in the fight its battalion HQ
// platoon. See companyPlans and the section Mius-Front Integration in docs/GDD.md.
package mission

import (
	"fmt"
	"strings"

	"github.com/xetra11/TotalKrieg/internal/game"
)

const (
	// maxPlatoonsPerSide is the generator's limit of 15 platoons per battalion
	// type. Both sides use one type each.
	maxPlatoonsPerSide = 15
	riflePlatoons      = 3
	maxMinutes         = 60
)

// Scenario is the JSON format of MiusMissionMaker.
type Scenario struct {
	ID          string            `json:"id"`
	Title       string            `json:"title"`
	Description string            `json:"description"`
	Base        string            `json:"base"`
	MaxMinutes  int               `json:"max_minutes"`
	Points      map[string][2]int `json:"points"`
	Sides       Sides             `json:"sides"`
	Victory     Victory           `json:"victory"`
	Defeat      *Defeat           `json:"defeat,omitempty"`
	Texts       map[string]string `json:"texts"`

	// Roster lists every platoon, player side first, in mission order. The
	// battle result refers to platoons by that order.
	Roster []RosterEntry `json:"-"`
	// Defending is set when the player holds against an AI attack. The game
	// reports its result from the player's side.
	Defending bool `json:"-"`
}

// RosterEntry says what a Mius-Front platoon is in TotalKrieg.
type RosterEntry struct {
	Owner *game.Element
	Label string
}

// Sides holds both forces.
type Sides struct {
	Player Side `json:"player"`
	Enemy  Side `json:"enemy"`
}

// Side is one force.
type Side struct {
	Deploy   Deploy      `json:"deploy"`
	Platoons []Platoon   `json:"platoons"`
	Orders   []SideOrder `json:"orders,omitempty"`
}

// SideOrder sends all platoons of a side somewhere.
type SideOrder struct {
	To   string `json:"to"`
	Mode string `json:"mode"`
}

// Deploy is the initial set-up of a side.
type Deploy struct {
	Type      string `json:"type"`
	At        string `json:"at"`
	Formation string `json:"formation,omitempty"`
}

// Platoon is one Mius-Front platoon.
type Platoon struct {
	Callsign string `json:"callsign"`
	Unit     string `json:"unit"`
	Plan     string `json:"plan,omitempty"`
}

// Victory is the player's win condition.
type Victory struct {
	OwnReaches string  `json:"own_reaches,omitempty"`
	Radius     int     `json:"radius,omitempty"`
	Share      float64 `json:"share,omitempty"`
	Balance    float64 `json:"balance"`
}

// Defeat is the player's loss condition.
type Defeat struct {
	EnemyReaches string `json:"enemy_reaches"`
}

// unitSet names the Mius-Front platoons of one faction, from docs/UNITS.md of
// MiusMissionMaker.
type unitSet struct {
	callsign                                     string
	staff, hq, rifle, mgHQ, mg, mortarHQ, mortar string
}

var unitSets = [2]unitSet{
	game.Wehrmacht: {
		callsign: "panther",
		staff:    "wer_mibn_1141_combat",
		hq:       "wer_mibn_1141_comcoy", rifle: "wer_mibn_1141_plat",
		mgHQ: "wer_mibn_1141_mg_com", mg: "wer_mibn_1141_mg_plat1",
	},
	game.SovietUnion: {
		callsign: "pine",
		staff:    "ra_rbn_1242_combat",
		hq:       "ra_rbn_1242_comcoy", rifle: "ra_rbn_1242_plat2",
		mgHQ: "ra_rbn_1242_mg_com", mg: "ra_rbn_1242_mg_plat",
		mortarHQ: "ra_rbn_1242_mt82_com", mortar: "ra_rbn_1242_mt82_plat",
	},
}

// basePlan is a shipped mission that fits an attacker of a faction, with tested
// positions from its original script.
type basePlan struct {
	id            string
	player, enemy [2]int
}

var basePlans = [2]basePlan{
	game.Wehrmacht:   {id: "tr06", player: [2]int{-113, -558}, enemy: [2]int{1264, -785}},
	game.SovietUnion: {id: "tr02", player: [2]int{-268, 2190}, enemy: [2]int{1384, 1720}},
}

// defensePlans are the shipped defensive missions, for fights the AI starts.
var defensePlans = [2]basePlan{
	game.Wehrmacht:   {id: "tr04", player: [2]int{-401, -716}, enemy: [2]int{224, -910}},
	game.SovietUnion: {id: "sm01", player: [2]int{1328, 622}, enemy: [2]int{2163, 533}},
}

// Battle describes a fight of the human player, who attacks or defends.
type Battle struct {
	Round        int
	Hex          game.Coord
	Faction      game.Faction    // the human player's side
	Player       []*game.Element // the human player's companies
	Enemy        []*game.Element // the AI's companies
	PlayerStaffs []*game.Element // battalion staffs in command of them
	EnemyStaffs  []*game.Element
	Fortified    bool // the defender holds a city
	Defending    bool // the AI attacks
}

// NewBattle collects a fight's forces from the rules. Staffs take part when
// they stand within command range of a company in the fight.
func NewBattle(s *game.State, u *game.Unit, target game.Coord) Battle {
	attackers, defenders := s.Engagement(u, target)
	o := s.ObjectAt(target)
	b := Battle{Round: s.Round, Hex: target, Faction: s.First, Fortified: o != nil && o.Kind.IsCity()}
	if u.Faction != s.First {
		b.Defending = true
		attackers, defenders = defenders, attackers
	}
	b.Player, b.PlayerStaffs = forces(attackers)
	b.Enemy, b.EnemyStaffs = forces(defenders)
	return b
}

func forces(units []*game.Unit) (companies, staffs []*game.Element) {
	seen := map[*game.Element]bool{}
	addStaff := func(e *game.Element) {
		if !seen[e] {
			seen[e] = true
			staffs = append(staffs, e)
		}
	}
	for _, u := range units {
		switch {
		case u.Kind == game.Battalion:
			addStaff(u.Element)
		default:
			companies = append(companies, u.Element)
			if u.Linked() {
				addStaff(u.Element.Parent)
			}
		}
	}
	return companies, staffs
}

// Build creates the scenario. Companies become platoons at the scale of the
// fight, so 2 against 3 companies stays 2 against 3 in Mius-Front.
func Build(b Battle) Scenario {
	own, foe := unitSets[b.Faction], unitSets[b.Faction.Other()]
	ownPlans, foePlans := companyPlans(own, b.Player, b.PlayerStaffs), companyPlans(foe, b.Enemy, b.EnemyStaffs)
	ownTotal, foeTotal := planSize(ownPlans), planSize(foePlans)
	hex := b.Hex.Name()
	ownPicked := pick(ownPlans, scaled(ownTotal, ownTotal, foeTotal))
	foePicked := pick(foePlans, scaled(foeTotal, ownTotal, foeTotal))
	s := Scenario{
		ID:         strings.ToLower(fmt.Sprintf("tk_r%02d_%s", b.Round, hex)),
		Title:      fmt.Sprintf("TotalKrieg %s, round %d", hex, b.Round),
		MaxMinutes: maxMinutes,
		Sides: Sides{
			Player: Side{Platoons: platoons(own.callsign, ownPicked, "")},
			Enemy:  Side{Platoons: platoons(foe.callsign, foePicked, "")},
		},
		Roster:    append(own.roster(ownPicked), foe.roster(foePicked)...),
		Defending: b.Defending,
	}
	if b.Defending {
		s.setDefense(b)
	} else {
		s.setAttack(b)
	}
	return s
}

// setAttack lets the player take a position the enemy holds.
func (s *Scenario) setAttack(b Battle) {
	base := basePlans[b.Faction]
	ownForce, foeForce := force(b.Faction, len(b.Player)), force(b.Faction.Other(), len(b.Enemy))
	s.Base = base.id
	s.Description = fmt.Sprintf("%s attacks %s. Take the enemy position.", ownForce, foeForce)
	s.Points = map[string][2]int{"own_start": base.player, "enemy_pos": base.enemy}
	s.Sides.Player.Deploy = Deploy{Type: "place", At: "own_start"}
	s.Sides.Enemy.Deploy = holdingDeploy("enemy_pos", b.Fortified)
	setPlan(s.Sides.Enemy.Platoons, "defend")
	s.Victory = Victory{OwnReaches: "enemy_pos", Radius: 250, Share: 0.5, Balance: 2}
	s.Texts = map[string]string{
		"brief_own":   "Your attacking force: " + ownForce,
		"brief_foe":   "The enemy holds this position: " + foeForce,
		"info_attack": "Take the enemy position",
	}
}

// setDefense lets the player hold a position the enemy attacks.
func (s *Scenario) setDefense(b Battle) {
	base := defensePlans[b.Faction]
	ownForce, foeForce := force(b.Faction, len(b.Player)), force(b.Faction.Other(), len(b.Enemy))
	s.Base = base.id
	s.Description = fmt.Sprintf("%s attacks %s. Hold the position.", foeForce, ownForce)
	s.Points = map[string][2]int{"own_pos": base.player, "enemy_start": base.enemy}
	s.Sides.Player.Deploy = holdingDeploy("own_pos", b.Fortified)
	s.Sides.Enemy.Deploy = Deploy{Type: "place", At: "enemy_start"}
	s.Sides.Enemy.Orders = []SideOrder{{To: "own_pos", Mode: "attack"}}
	setPlan(s.Sides.Enemy.Platoons, "attack")
	s.Victory = Victory{Balance: 2}
	s.Defeat = &Defeat{EnemyReaches: "own_pos"}
	s.Texts = map[string]string{
		"brief_own":   "Your defending force: " + ownForce,
		"brief_foe":   "The enemy attacks with: " + foeForce,
		"info_attack": "Hold this position",
	}
}

// holdingDeploy digs a defender in when it holds a city.
func holdingDeploy(at string, fortified bool) Deploy {
	if fortified {
		return Deploy{Type: "trench", At: at}
	}
	return Deploy{Type: "place", At: at, Formation: "line"}
}

func setPlan(platoons []Platoon, plan string) {
	for i := range platoons {
		platoons[i].Plan = plan
	}
}

func force(f game.Faction, companies int) string {
	nation := "German"
	if f == game.SovietUnion {
		nation = "Soviet"
	}
	word := "companies"
	if companies == 1 {
		word = "company"
	}
	return fmt.Sprintf("%d %s %s", companies, nation, word)
}

// Describe says what a company becomes in Mius-Front.
func Describe(c *game.Element) string {
	switch c.Role {
	case game.RoleMG:
		return "Company HQ, MG platoon"
	case game.RoleMortar:
		return "Company HQ, mortar platoon"
	}
	if n := rifleCount(c); n > 1 {
		return fmt.Sprintf("Company HQ, %d rifle platoons", n)
	}
	return "Company HQ, 1 rifle platoon"
}

// rifleCount gives a weakened company fewer rifle platoons, rounded up.
func rifleCount(c *game.Element) int {
	return (riflePlatoons*c.Current + c.Full - 1) / c.Full
}

// companyPlan is the platoon list of one company: combat platoons, then its HQ.
type companyPlan struct {
	owner  *game.Element
	combat []string
	hq     string
}

// companyPlans maps companies to platoons. Each staff comes last, as a plan
// with only an HQ, so it is the first to go when a fight gets scaled down.
func companyPlans(set unitSet, companies, staffs []*game.Element) []companyPlan {
	var out []companyPlan
	for _, c := range companies {
		switch c.Role {
		case game.RoleMG:
			out = append(out, companyPlan{c, []string{set.mg}, set.mgHQ})
		case game.RoleMortar:
			out = append(out, companyPlan{c, []string{set.mortar}, set.mortarHQ})
		default:
			out = append(out, companyPlan{c, repeat(set.rifle, rifleCount(c)), set.hq})
		}
	}
	for _, s := range staffs {
		out = append(out, companyPlan{s, nil, set.staff})
	}
	return out
}

func repeat(s string, n int) []string {
	out := make([]string, n)
	for i := range out {
		out[i] = s
	}
	return out
}

func planSize(plans []companyPlan) int {
	n := 0
	for _, p := range plans {
		n += len(p.combat) + 1
	}
	return n
}

// scaled shrinks a side's platoon count when the larger side passes the limit,
// so the ratio between the sides holds.
func scaled(total, a, b int) int {
	largest := max(a, b)
	if largest <= maxPlatoonsPerSide {
		return total
	}
	return max(1, (total*maxPlatoonsPerSide+largest/2)/largest)
}

// picked is a chosen platoon and the element it stands for.
type picked struct {
	unit  string
	owner *game.Element
}

// pick takes platoons evenly across the companies, combat units before HQs.
func pick(plans []companyPlan, n int) []picked {
	var out []picked
	for round := 0; len(out) < n; round++ {
		added := false
		for _, p := range plans {
			if round < len(p.combat) && len(out) < n {
				out = append(out, picked{p.combat[round], p.owner})
				added = true
			}
		}
		if !added {
			break
		}
	}
	for _, p := range plans {
		if len(out) < n {
			out = append(out, picked{p.hq, p.owner})
		}
	}
	return out
}

func platoons(name string, units []picked, plan string) []Platoon {
	out := make([]Platoon, len(units))
	for i, u := range units {
		out[i] = Platoon{Callsign: fmt.Sprintf("%s_%02d", name, i+1), Unit: u.unit, Plan: plan}
	}
	return out
}

// roster names the platoons, numbering the rifle platoons of each company.
func (set unitSet) roster(units []picked) []RosterEntry {
	rifles := map[*game.Element]int{}
	out := make([]RosterEntry, len(units))
	for i, u := range units {
		label := "Company HQ"
		switch u.unit {
		case set.staff:
			label = "Battalion HQ"
		case set.rifle:
			rifles[u.owner]++
			label = game.Ordinal(rifles[u.owner]) + " Rifle Platoon"
		case set.mg:
			label = "MG Platoon"
		case set.mortar:
			label = "Mortar Platoon"
		}
		out[i] = RosterEntry{u.owner, label}
	}
	return out
}
