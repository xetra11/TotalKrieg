// Package ui is the Ebitengine front end of TotalKrieg.
package ui

import (
	"bytes"
	"fmt"
	"image/png"
	"iter"
	"math/rand/v2"
	"os"
	"time"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/inpututil"
	"github.com/hajimehoshi/ebiten/v2/text/v2"
	"golang.org/x/image/font/gofont/gobold"
	"golang.org/x/image/font/gofont/goregular"

	"github.com/xetra11/TotalKrieg/internal/campaign"
	"github.com/xetra11/TotalKrieg/internal/game"
	"github.com/xetra11/TotalKrieg/internal/mission"
	"github.com/xetra11/TotalKrieg/internal/symbols"
)

// aiDelayFrames keeps the board readable before the AI answers.
const aiDelayFrames = 30

// menuMode is the page of the main menu.
type menuMode int

const (
	menuMain menuMode = iota
	menuNewName
	menuNewFaction
)

// Options configures a run.
type Options struct {
	Seed        uint64        // 0 picks a random map
	Faction     *game.Faction // set to skip the menu
	Autoplay    int           // AI turns to play before the first frame
	Shot        string        // write the first frames to this PNG and quit
	Mission     mission.Config
	SaveDir     string // campaign folder, empty uses the user's config folder
	NoAnimation bool   // skip animations, for tests
}

// App is the running program.
type App struct {
	opts Options
	st   *game.State

	store        *campaign.Store
	current      *campaign.Record
	saves        []campaign.Summary
	menu         menuMode
	saveErr      string
	campaignName string // name being entered for new campaign

	selUnit *game.Unit
	selCity *game.Object
	reach   map[game.Coord]int
	message string
	aiWait  int

	showOrg    bool
	attack     *game.Fight // waits for the choice of resolver
	battle     *externalBattle
	runMission func(mission.Scenario) error // nil without MiusMissionMaker
	readResult func(mission.Scenario, time.Time) (*game.FieldResult, error)

	// script steps through a commit or an AI turn, which stops at every fight.
	script     func() (*game.Fight, bool)
	stopScript func()

	animations animationQueue

	sets      [2]*symbols.Set
	regular   *text.GoTextFaceSource
	bold      *text.GoTextFaceSource
	mapImg    *ebiten.Image
	mapDirty  bool
	symbolImg map[symbolKey]*ebiten.Image

	frames int
	shot   bool

	closing bool
}

// New creates the app and, when a faction is set, starts the match.
func New(opts Options) (*App, error) {
	a := &App{opts: opts, symbolImg: map[symbolKey]*ebiten.Image{}}
	dir := opts.SaveDir
	if dir == "" {
		if d, err := campaign.DefaultDir(); err == nil {
			dir = d
		}
	}
	if dir != "" {
		a.store = campaign.NewStore(dir)
		a.refreshSaves()
	}
	var err error
	if a.sets[game.Wehrmacht], err = symbols.Load("wehrmacht"); err != nil {
		return nil, err
	}
	if a.sets[game.SovietUnion], err = symbols.Load("sovietunion"); err != nil {
		return nil, err
	}
	if a.regular, err = text.NewGoTextFaceSource(bytes.NewReader(goregular.TTF)); err != nil {
		return nil, err
	}
	if a.bold, err = text.NewGoTextFaceSource(bytes.NewReader(gobold.TTF)); err != nil {
		return nil, err
	}
	if opts.Mission.Available() {
		a.runMission = func(s mission.Scenario) error {
			_, err := mission.Start(opts.Mission, s)
			return err
		}
		a.readResult = func(s mission.Scenario, since time.Time) (*game.FieldResult, error) {
			return mission.ReadResult(opts.Mission, s, since)
		}
	}
	if opts.Faction != nil {
		a.startNew(*opts.Faction, "")
		for i := 0; i < opts.Autoplay && !a.st.Over; i++ {
			a.st.PlayTurn()
		}
		a.showRout()
	}
	return a, nil
}

// startNew begins a fresh campaign and writes it at once, so it shows up in the
// menu even if the program is closed right away.
func (a *App) startNew(f game.Faction, name string) {
	if name == "" {
		name = f.String() + " campaign"
	}
	seed := a.opts.Seed
	if seed == 0 {
		seed = rand.Uint64()
	}
	a.endScript()
	a.st = game.New(seed, f)
	a.clearSelection()
	a.menu = menuMain
	a.message = "Choose a unit or a city."
	a.mapDirty = true
	a.campaignName = ""

	// Create the campaign file only once
	if a.store != nil && a.opts.Shot == "" {
		rec, err := a.store.New(name, a.st)
		if err != nil {
			a.saveErr = err.Error()
			return
		}
		a.current = &rec
		a.saveErr = ""
		a.refreshSaves()
	}
}

// autosave writes the running campaign. It stays away from a turn that is still
// being carried out, so the file always holds a state the menu can continue.
func (a *App) autosave() {
	if a.store == nil || a.st == nil || a.current == nil || a.script != nil || a.opts.Shot != "" {
		return
	}
	snap, err := a.st.Snapshot()
	if err != nil {
		a.saveErr = err.Error()
		return
	}
	a.current.State = snap
	if err := a.store.Save(a.current); err != nil {
		a.saveErr = err.Error()
		return
	}
	a.saveErr = ""
}

// refreshSaves reloads the campaign list for the menu.
func (a *App) refreshSaves() {
	if a.store == nil {
		a.saves = nil
		return
	}
	saves, err := a.store.List()
	if err != nil {
		a.saveErr = err.Error()
		return
	}
	a.saves = saves
}

// continueCampaign loads a save and puts the player back on the map.
func (a *App) continueCampaign(id string) {
	if a.store == nil {
		return
	}
	rec, err := a.store.Load(id)
	if err != nil {
		a.saveErr = err.Error()
		return
	}
	st, err := game.Restore(rec.State)
	if err != nil {
		a.saveErr = err.Error()
		return
	}
	a.endScript()
	a.st = st
	a.current = &rec
	a.clearSelection()
	a.mapDirty = true
	a.message = fmt.Sprintf("Campaign continued, round %d.", st.Round)
	a.saveErr = ""
	a.showRout()
}

// toMenu saves the running campaign and returns to the main menu.
func (a *App) toMenu() {
	a.autosave()
	a.endScript()
	a.st = nil
	a.current = nil
	a.clearSelection()
	a.menu = menuMain
	a.message = ""
	a.refreshSaves()
}

func (a *App) clearSelection() {
	a.selUnit, a.selCity, a.reach = nil, nil, nil
	a.showOrg, a.attack, a.battle = false, nil, nil
}

func (a *App) humanTurn() bool {
	return a.st != nil && !a.st.Over && a.st.Active == a.st.First
}

// Update implements ebiten.Game.
func (a *App) Update() error {
	a.frames++
	if a.shot {
		return ebiten.Termination
	}
	if a.closing && !ebiten.IsWindowBeingClosed() {
		a.closing = false
	}
	if ebiten.IsWindowBeingClosed() && !a.closing {
		a.closing = true
		a.autosave()
		return ebiten.Termination
	}
	// Handle text input in campaign name menu
	if a.st == nil && a.menu == menuNewName {
		a.handleNameInput()
	}
	if inpututil.IsMouseButtonJustPressed(ebiten.MouseButtonLeft) {
		x, y := ebiten.CursorPosition()
		a.click(float64(x), float64(y))
	}
	if inpututil.IsKeyJustPressed(ebiten.KeyEnter) || inpututil.IsKeyJustPressed(ebiten.KeySpace) {
		a.confirm()
	}
	if inpututil.IsKeyJustPressed(ebiten.KeyEscape) {
		a.cancel()
	}
	if inpututil.IsKeyJustPressed(ebiten.KeyG) {
		a.toggleOrg()
	}
	a.tick()
	return nil
}

// handleNameInput processes keyboard input for the campaign name.
func (a *App) handleNameInput() {
	// Handle backspace
	if inpututil.IsKeyJustPressed(ebiten.KeyBackspace) {
		if len(a.campaignName) > 0 {
			a.campaignName = a.campaignName[:len(a.campaignName)-1]
		}
		return
	}

	// Get text input from ebiten
	runes := ebiten.AppendInputChars(nil)
	for _, r := range runes {
		// Allow letters, numbers, spaces, and basic punctuation
		if (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') ||
			(r >= '0' && r <= '9') || r == ' ' || r == '-' || r == '_' {
			if len(a.campaignName) < 50 {
				a.campaignName += string(r)
			}
		}
	}
}

// confirm resolves an open fight, otherwise it ends the turn.
func (a *App) confirm() {
	switch {
	case a.battle != nil, a.routing():
	case a.attack != nil:
		a.resolveAuto()
	case !a.showOrg:
		a.next()
	}
}

// cancel steps back one level: dialog, window, selection. A rout can't be
// stepped back from.
func (a *App) cancel() {
	switch {
	case a.st == nil && a.menu == menuNewFaction:
		a.menu = menuNewName
	case a.st == nil && a.menu == menuNewName:
		a.campaignName = ""
		a.menu = menuMain
	case a.routing():
	case a.battle != nil:
		if a.battle.done == nil && !a.battle.running {
			a.battle = nil
		}
	case a.attack != nil:
		a.cancelAttack()
	case a.showOrg:
		a.showOrg = false
	default:
		a.clearSelection()
	}
}

func (a *App) toggleOrg() {
	if a.attack != nil || a.routing() {
		return
	}
	if a.showOrg {
		a.showOrg = false
		return
	}
	a.openOrg()
}

// tick steps the running turn and lets the AI start once the delay ran out.
// Fights and routs wait for the player.
func (a *App) tick() {
	a.animations.tick()
	a.pollMission()
	a.pollResult()
	if a.st == nil || a.st.Over || a.attack != nil || a.routing() {
		return
	}
	if a.waitForAnimations() {
		return
	}
	if a.script != nil {
		a.stepScript()
		return
	}
	if a.humanTurn() {
		a.aiWait = 0
		return
	}
	a.aiWait++
	if a.aiWait >= aiDelayFrames {
		a.aiWait = 0
		a.startScript(a.st.AITurn())
	}
}

// afterTurn saves after a complete turn, so the file always holds a state that
// can be continued.
func (a *App) afterTurn() {
	a.autosave()
}

// next commits the player's orders.
func (a *App) next() {
	if !a.humanTurn() || a.script != nil {
		return
	}
	a.clearSelection()
	a.message = "The orders are carried out."
	a.startScript(a.st.CommitTurn())
	a.afterTurn()
}

func (a *App) startScript(turn iter.Seq[*game.Fight]) {
	a.script, a.stopScript = iter.Pull(turn)
	a.stepScript()
}

// stepScript runs the turn to its next fight. The AI's attacks ask the player
// only when there is another way to fight them than the automatic one.
func (a *App) stepScript() {
	fight, ok := a.script()
	if !ok {
		a.script, a.stopScript = nil, nil
		a.mapDirty = true
		if a.humanTurn() {
			a.message = "Your turn."
		} else if !a.st.Over {
			a.message = "The AI is moving."
		}
		a.afterTurn()
		return
	}
	// nil fight means moves are ready to animate
	if fight == nil {
		if !a.opts.NoAnimation {
			for _, m := range a.st.TakeMoves() {
				a.animations.startMove(m.Unit, m.From, m.To)
			}
		} else {
			a.st.TakeMoves() // clear moves
		}
		return
	}
	a.attack = fight
	if a.incoming() && a.runMission == nil {
		a.resolveAuto()
	}
}

func (a *App) endScript() {
	if a.stopScript != nil {
		a.stopScript()
	}
	a.script, a.stopScript = nil, nil
	a.animations.clear()
}

func (a *App) click(x, y float64) {
	switch {
	case a.st == nil:
		a.clickMenu(x, y)
	case a.st.Over && newGameButton().contains(x, y):
		a.toMenu()
	case a.battle != nil:
		a.clickBattleDialog(x, y)
	case a.attack != nil:
		a.clickAttackPrompt(x, y)
	case a.routing():
		a.clickRout(x, y)
	case a.script != nil:
	case a.showOrg:
		a.clickOrg(x, y)
	case nextButton().contains(x, y):
		a.next()
	case a.selUnit != nil && orgButton().contains(x, y):
		a.openOrg()
	case a.selCity != nil && a.clickRecruit(x, y):
	default:
		if c, ok := hexAt(x, y); ok {
			a.clickHex(c)
		}
	}
}

func (a *App) clickMenu(x, y float64) {
	switch a.menu {
	case menuMain:
		a.clickMenuMain(x, y)
	case menuNewName:
		a.clickMenuNewName(x, y)
	case menuNewFaction:
		a.clickMenuNewFaction(x, y)
	}
}

func (a *App) clickMenuMain(x, y float64) {
	if newCampaignButton().contains(x, y) {
		a.campaignName = ""
		a.menu = menuNewName
		return
	}
	for i, s := range a.saves {
		if saveButton(i).contains(x, y) {
			a.continueCampaign(s.ID)
			return
		}
	}
}

func (a *App) clickMenuNewName(x, y float64) {
	if menuBackButton().contains(x, y) {
		a.campaignName = ""
		a.menu = menuMain
		return
	}
	if menuContinueButton().contains(x, y) {
		if a.campaignName == "" {
			a.saveErr = "Please enter a campaign name."
			return
		}
		a.menu = menuNewFaction
		a.saveErr = ""
		return
	}
}

func (a *App) clickMenuNewFaction(x, y float64) {
	if menuBackButton().contains(x, y) {
		a.menu = menuNewName
		return
	}
	for _, f := range []game.Faction{game.Wehrmacht, game.SovietUnion} {
		if factionButton(f).contains(x, y) {
			a.startNew(f, a.campaignName)
			return
		}
	}
}

func (a *App) clickRecruit(x, y float64) bool {
	for i, k := range game.UnitKinds {
		if !recruitButton(i).contains(x, y) {
			continue
		}
		if !a.humanTurn() {
			return true
		}
		u, err := a.st.Recruit(a.selCity, k)
		if err != nil {
			a.message = err.Error()
			return true
		}
		a.selUnit = u
		a.mapDirty = true
		switch {
		case k == game.Battalion:
			a.message = fmt.Sprintf("%s raised with %d companies.", u.Name, game.BattalionSize(u.Faction))
		case u.Element.Parent != nil:
			a.message = fmt.Sprintf("%s raised and assigned to %s.", u.Name, u.Element.Parent.Short)
		default:
			a.message = fmt.Sprintf("%s raised as an independent company.", u.Name)
		}
		return true
	}
	return false
}

func (a *App) clickHex(c game.Coord) {
	st := a.st
	own := a.selUnit != nil && a.selUnit.Faction == st.First
	if a.humanTurn() && own {
		_, reachable := a.reach[c]
		_, attackable := st.AttackTargets(a.selUnit)[c]
		if reachable || attackable {
			a.planOrder(c)
			return
		}
		if c == a.selUnit.Pos && st.OrderOf(a.selUnit) != nil {
			st.CancelOrder(a.selUnit)
			a.message = a.selUnit.Short + ": order cancelled."
			a.selectUnit(a.selUnit)
			return
		}
	}
	a.selectUnit(st.UnitAt(c))
	if o := st.ObjectAt(c); o != nil && o.Kind.IsCity() {
		a.selCity = o
	}
}

func (a *App) planOrder(c game.Coord) {
	if err := a.st.Plan(a.selUnit, c); err != nil {
		a.message = err.Error()
		return
	}
	a.message = a.selUnit.Short + ": " + orderText(a.st.OrderOf(a.selUnit)) + " planned."
	a.selectUnit(a.selUnit)
}

// Draw implements ebiten.Game.
func (a *App) Draw(screen *ebiten.Image) {
	a.draw(screen)
	if a.opts.Shot != "" && a.frames > 20 && !a.shot {
		a.shot = true
		if err := writePNG(screen, a.opts.Shot); err != nil {
			fmt.Fprintln(os.Stderr, err)
		}
	}
}

func writePNG(img *ebiten.Image, path string) error {
	f, err := os.Create(path)
	if err != nil {
		return err
	}
	defer f.Close()
	return png.Encode(f, img)
}

// Layout implements ebiten.Game.
func (a *App) Layout(int, int) (int, int) {
	return screenW, screenH
}
