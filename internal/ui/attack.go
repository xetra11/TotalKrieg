package ui

import (
	"fmt"
	"slices"
	"time"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/text/v2"

	"github.com/xetra11/TotalKrieg/internal/game"
	"github.com/xetra11/TotalKrieg/internal/mission"
)

// tacticalOptions are the planned tactical layer resolvers. None is wired up yet.
var tacticalOptions = []string{
	"Hexfront (Advanced Squad Leader 2 Digital)",
	"MiusPlayer (Mius Front Multiplayer)",
	"Combat Mission Multiplayer",
}

var outcomes = []game.Outcome{game.AttackerWins, game.Draw, game.DefenderWins}

var attackPrompt = rect{mapW/2 - 330, 110, 660, 560}

func autoResolveButton() rect {
	return rect{attackPrompt.x + 24, attackPrompt.y + 176, attackPrompt.w - 48, 48}
}

func missionButton() rect {
	return rect{attackPrompt.x + 24, attackPrompt.y + 232, attackPrompt.w - 48, 48}
}

func tacticalRow(i int) rect {
	return rect{attackPrompt.x + 24, attackPrompt.y + 322 + float64(i)*36, attackPrompt.w - 48, 32}
}

func cancelAttackButton() rect {
	return rect{attackPrompt.x + attackPrompt.w - 144, attackPrompt.y + attackPrompt.h - 54, 120, 38}
}

func outcomeButton(i int) rect {
	w := (attackPrompt.w - 64) / 3
	return rect{attackPrompt.x + 24 + float64(i)*(w+8), attackPrompt.y + 480, w, 52}
}

func startMissionButton() rect {
	return rect{attackPrompt.x + 24, attackPrompt.y + 480, attackPrompt.w - 196, 52}
}

func backButton() rect {
	return rect{attackPrompt.x + attackPrompt.w - 164, attackPrompt.y + 480, 140, 52}
}

// externalBattle is a fight handed to Mius-Front. It starts as a preview of the
// line-up, and only a confirmed preview launches the game.
type externalBattle struct {
	scenario mission.Scenario
	own, foe []string       // what each unit of the fight becomes in Mius-Front
	sides    mission.Battle // the forces, for showing the result
	done     chan error     // set while the generator runs
	running  bool           // the game started, the result is due
	failure  string         // why the last step failed
	started  time.Time      // when the game was launched
	frames   int            // since the last look for a result
	result   *game.FieldResult
}

// resultCheckFrames spaces the looks at the game's result file, about once a
// second.
const resultCheckFrames = 60

// resultPanel replaces the prompt once a result is in, for the list of losses.
var resultPanel = rect{40, 72, mapW - 80, 756}

func applyResultButton() rect {
	return rect{resultPanel.x + 24, resultPanel.y + resultPanel.h - 72, resultPanel.w - 48, 52}
}

func (a *App) clickAttackPrompt(x, y float64) {
	switch {
	case autoResolveButton().contains(x, y):
		a.resolveAuto()
	case missionButton().contains(x, y) && a.runMission != nil:
		a.previewMission()
	case cancelAttackButton().contains(x, y):
		a.attack = nil
		a.message = "Attack cancelled."
	}
}

// resolveAuto lets the rules decide the pending fight.
func (a *App) resolveAuto() {
	u, target := a.attack.unit, a.attack.target
	a.attack = nil
	r, err := a.st.Attack(u, target)
	a.finishFight(u, r, err)
}

func (a *App) finishFight(u *game.Unit, r game.Report, err error) {
	if err != nil {
		a.message = err.Error()
		return
	}
	a.message = fmt.Sprintf("Battle: own losses %d, enemy %d.", r.AttackerLoss, r.DefenderLoss)
	a.mapDirty = true
	if r.AttackerDestroyed {
		a.clearSelection()
	} else {
		a.selectUnit(u)
	}
	a.showRout()
}

func (a *App) routing() bool {
	return a.st != nil && a.st.NextRout() != nil
}

// showRout points the player at the next beaten unit and the hexes it can flee to.
func (a *App) showRout() {
	r := a.st.NextRout()
	if r == nil {
		return
	}
	a.clearSelection()
	a.selUnit = r.Unit
	a.reach = map[game.Coord]int{}
	for _, c := range a.st.RoutOptions(r) {
		a.reach[c] = 1
	}
	a.message = fmt.Sprintf("%s is beaten. Choose the hex it flees to.", r.Unit.Short)
}

func (a *App) clickRout(x, y float64) {
	c, ok := hexAt(x, y)
	if !ok {
		return
	}
	r := a.st.NextRout()
	if err := a.st.Flee(r, c); err != nil {
		a.message = err.Error()
		return
	}
	a.message = fmt.Sprintf("%s flees to %s.", r.Unit.Short, hexName(c))
	a.mapDirty = true
	a.selectUnit(r.Unit)
	a.showRout()
}

// previewMission shows the line-up of the fight. Nothing starts yet.
func (a *App) previewMission() {
	b := mission.NewBattle(a.st, a.attack.unit, a.attack.target)
	a.battle = &externalBattle{
		scenario: mission.Build(b),
		own:      mappingRows(b.Player, len(b.PlayerStaffs)),
		foe:      mappingRows(b.Enemy, len(b.EnemyStaffs)),
		sides:    b,
	}
}

// launchMission builds the mission and starts Mius-Front. The generator takes
// seconds, so it runs beside the game loop.
func (a *App) launchMission() {
	done := make(chan error, 1)
	a.battle.done, a.battle.failure, a.battle.started = done, "", time.Now()
	s, run := a.battle.scenario, a.runMission
	go func() { done <- run(s) }()
}

func mappingRows(companies []*game.Element, staffs int) []string {
	var out []string
	for _, c := range companies {
		out = append(out, fmt.Sprintf("%s %s: %s", c.Short, c.Callsign, mission.Describe(c)))
	}
	switch {
	case staffs == 1:
		out = append(out, "1 battalion HQ")
	case staffs > 1:
		out = append(out, fmt.Sprintf("%d battalion HQs", staffs))
	}
	return out
}

// lossRow is a line of the list of losses: a company or staff, or a platoon.
type lossRow struct {
	text string
	head bool
}

// lossRows lists each company's strength before and after the battle with the
// losses of its platoons, then the staffs that took part.
func lossRows(companies, staffs []*game.Element, res *game.FieldResult) []lossRow {
	var out []lossRow
	for _, c := range companies {
		after := game.ScaledStrength(c.Current, res.ShareOf(c, companies))
		out = append(out, lossRow{fmt.Sprintf("%s %s: strength %d → %d", c.Short, c.Callsign, c.Current, after), true})
		rows := platoonRows(c, res)
		if len(rows) == 0 {
			rows = []lossRow{{text: "not in the mission, losses as the rest of its side"}}
		}
		out = append(out, rows...)
	}
	for _, s := range staffs {
		share, ok := res.Remaining[s]
		if !ok {
			continue
		}
		state := "staff intact"
		if share <= 0 {
			state = "staff lost"
		}
		out = append(out, lossRow{s.Short + ": " + state, true})
		out = append(out, platoonRows(s, res)...)
	}
	return out
}

func platoonRows(e *game.Element, res *game.FieldResult) []lossRow {
	var out []lossRow
	for _, p := range res.Platoons {
		if p.Owner == e {
			out = append(out, lossRow{text: p.Name + ": " + platoonLoss(p)})
		}
	}
	return out
}

func platoonLoss(p game.PlatoonLoss) string {
	condition := int(p.Condition*100 + 0.5)
	switch p.Lost {
	case 0:
		return fmt.Sprintf("no squad lost, condition %d%%", condition)
	case p.Squads:
		return fmt.Sprintf("all %d squads lost", p.Squads)
	}
	return fmt.Sprintf("%d of %d squads lost, the rest at %d%% condition", p.Lost, p.Squads, condition)
}

// sideLoss sums the squads of a side.
func sideLoss(side []*game.Element, res *game.FieldResult) (lost, squads int) {
	for _, p := range res.Platoons {
		if slices.Contains(side, p.Owner) {
			lost, squads = lost+p.Lost, squads+p.Squads
		}
	}
	return lost, squads
}

// pollMission checks whether the generator has finished.
func (a *App) pollMission() {
	if a.battle == nil || a.battle.done == nil {
		return
	}
	select {
	case err := <-a.battle.done:
		a.battle.done = nil
		if err != nil {
			a.battle.failure = err.Error()
			return
		}
		a.battle.running = true
	default:
	}
}

// pollResult looks for the battle's result once the game runs. The game writes
// it when the battle ends, which needs no action in TotalKrieg.
func (a *App) pollResult() {
	b := a.battle
	if b == nil || !b.running || b.result != nil || a.readResult == nil {
		return
	}
	b.frames++
	if b.frames < resultCheckFrames {
		return
	}
	b.frames = 0
	res, err := a.readResult(b.scenario, b.started)
	if err != nil {
		b.failure = err.Error()
		return
	}
	b.failure = ""
	b.result = res
}

// applyResult books the reported losses.
func (a *App) applyResult() {
	u, target, res := a.attack.unit, a.attack.target, a.battle.result
	a.battle, a.attack = nil, nil
	r, err := a.st.ApplyFieldResult(u, target, res)
	a.finishFight(u, r, err)
}

func (a *App) clickBattleDialog(x, y float64) {
	switch {
	case a.battle.done != nil:
	case a.battle.result != nil:
		if applyResultButton().contains(x, y) {
			a.applyResult()
		}
	case a.battle.running:
		a.clickOutcome(x, y)
	case startMissionButton().contains(x, y):
		a.launchMission()
	case backButton().contains(x, y):
		a.battle = nil
	}
}

func (a *App) clickOutcome(x, y float64) {
	for i, o := range outcomes {
		if outcomeButton(i).contains(x, y) {
			u, target := a.attack.unit, a.attack.target
			a.battle, a.attack = nil, nil
			r, err := a.st.ResolveBattle(u, target, o)
			a.finishFight(u, r, err)
			return
		}
	}
}

func (a *App) drawResult(dst *ebiten.Image) {
	r, res, s := resultPanel, a.battle.result, a.battle.sides
	fillRect(dst, r, colChrome)
	strokeRect(dst, r, 2, colAmber)
	a.label(dst, "Battle at "+hexName(a.attack.target), r.x+24, r.y+16, 16, colAmber, true, text.AlignStart)
	a.label(dst, "Mius-Front reports: "+res.Text, r.x+24, r.y+40, 24, colText, true, text.AlignStart)
	a.drawLossColumn(dst, "Own forces", s.Player, s.PlayerStaffs, res, r.x+24)
	a.drawLossColumn(dst, "Enemy", s.Enemy, s.EnemyStaffs, res, r.x+r.w/2+8)
	a.drawButton(dst, applyResultButton(), "Apply result", true)
}

func (a *App) drawLossColumn(dst *ebiten.Image, title string, companies, staffs []*game.Element, res *game.FieldResult, x float64) {
	y := resultPanel.y + 92
	lost, squads := sideLoss(append(slices.Clone(companies), staffs...), res)
	a.label(dst, fmt.Sprintf("%s: %d of %d squads lost", title, lost, squads), x, y, 16, colAmber, true, text.AlignStart)
	y += 30
	for _, row := range lossRows(companies, staffs, res) {
		if row.head {
			y += 6
			a.label(dst, row.text, x, y, 14, colText, true, text.AlignStart)
			y += 21
			continue
		}
		a.label(dst, row.text, x+14, y, 13, colMuted, false, text.AlignStart)
		y += 19
	}
}

func (a *App) drawAttackPrompt(dst *ebiten.Image) {
	if a.battle != nil && a.battle.result != nil {
		a.drawResult(dst)
		return
	}
	r := attackPrompt
	attackers, defenders := a.st.Engagement(a.attack.unit, a.attack.target)
	fillRect(dst, r, colChrome)
	strokeRect(dst, r, 2, colAmber)
	a.label(dst, "Battle at "+hexName(a.attack.target), r.x+24, r.y+16, 16, colAmber, true, text.AlignStart)
	a.label(dst, a.promptTitle(), r.x+24, r.y+40, 24, colText, true, text.AlignStart)

	a.drawCombatant(dst, attackers, r.x+24, r.y+84, "Attacker")
	a.label(dst, "vs.", r.x+r.w/2-16, r.y+112, 15, colMuted, true, text.AlignCenter)
	a.drawCombatant(dst, defenders, r.x+r.w/2+24, r.y+84, "Defender")

	if a.battle != nil {
		a.drawBattleStatus(dst)
		return
	}
	a.drawResolveOptions(dst)
}

func (a *App) promptTitle() string {
	switch {
	case a.battle == nil:
		return "How should the attack be resolved?"
	case a.battle.done != nil:
		return "Building the mission."
	case a.battle.running:
		return "The battle is running in Mius-Front."
	}
	return "Battle line-up"
}

func (a *App) drawResolveOptions(dst *ebiten.Image) {
	r := attackPrompt
	b := autoResolveButton()
	fillRect(dst, b, colAmber)
	a.label(dst, "Resolve automatically", b.x+b.w/2, b.y+14, 18, colDeep, true, text.AlignCenter)

	m := missionButton()
	name := "Play in Mius-Front (MiusMissionMaker)"
	if a.runMission == nil {
		name = "Mius-Front: MiusMissionMaker not found"
	}
	a.drawButton(dst, m, name, a.runMission != nil)

	a.label(dst, "Tactical layers, not available yet", r.x+24, r.y+298, 14, colMuted, false, text.AlignStart)
	for i, name := range tacticalOptions {
		row := tacticalRow(i)
		fillRect(dst, row, colPanel)
		strokeRect(dst, row, 1, colLine)
		a.label(dst, name, row.x+12, row.y+9, 14, colDisabled, false, text.AlignStart)
	}

	c := cancelAttackButton()
	strokeRect(dst, c, 2, colMuted)
	a.label(dst, "Cancel", c.x+c.w/2, c.y+11, 15, colText, true, text.AlignCenter)
}

func (a *App) drawBattleStatus(dst *ebiten.Image) {
	r := attackPrompt
	s := a.battle.scenario
	summary := fmt.Sprintf("Map %s: %d vs. %d platoons", s.Base, len(s.Sides.Player.Platoons), len(s.Sides.Enemy.Platoons))
	a.label(dst, summary, r.x+24, r.y+176, 16, colText, true, text.AlignStart)
	a.drawMappingColumn(dst, a.battle.own, r.x+24, r.y+206)
	a.drawMappingColumn(dst, a.battle.foe, r.x+r.w/2+8, r.y+206)
	switch {
	case a.battle.done != nil:
		a.label(dst, "Mius-Front is about to open.", r.x+24, r.y+450, 15, colMuted, false, text.AlignStart)
	case a.battle.running:
		if a.battle.failure != "" {
			a.wrapped(dst, "Result unreadable: "+a.battle.failure, r.x+24, r.y+440, r.w-48, 14, colText)
		} else {
			a.label(dst, "The result comes in by itself when the battle ends. Otherwise enter it here.", r.x+24, r.y+450, 15, colMuted, false, text.AlignStart)
		}
		for i, o := range outcomes {
			a.drawButton(dst, outcomeButton(i), o.String(), true)
		}
	default:
		if a.battle.failure != "" {
			a.wrapped(dst, "Not started: "+a.battle.failure, r.x+24, r.y+440, r.w-48, 14, colText)
		}
		a.drawButton(dst, startMissionButton(), "Start Mius-Front", true)
		a.drawButton(dst, backButton(), "Back", true)
	}
}

func (a *App) drawMappingColumn(dst *ebiten.Image, rows []string, x, y float64) {
	for i, row := range rows {
		a.wrapped(dst, row, x, y+float64(i)*34, attackPrompt.w/2-32, 13, colText)
	}
}

// drawCombatant shows the lead unit and the size of its side.
func (a *App) drawCombatant(dst *ebiten.Image, units []*game.Unit, x, y float64, role string) {
	u := units[0]
	fillRect(dst, rect{x, y, 60, 76}, colPaper)
	img := a.symbol(u.Faction, u.Kind, 1.2)
	op := &ebiten.DrawImageOptions{}
	op.GeoM.Translate(x+4, y+6)
	dst.DrawImage(img, op)
	strength, companies := 0, 0
	for _, x := range units {
		strength += x.Strength()
		if x.Kind == game.Company {
			companies++
		}
	}
	a.label(dst, role, x+72, y+2, 13, factionText(u.Faction), true, text.AlignStart)
	a.label(dst, u.Short, x+72, y+20, 17, colText, true, text.AlignStart)
	a.label(dst, fmt.Sprintf("Strength %d, %d coy.", strength, companies), x+72, y+46, 14, colMuted, false, text.AlignStart)
}
