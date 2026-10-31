// Package ui is the Ebitengine front end of TotalKrieg.
package ui

import (
	"bytes"
	"fmt"
	"image/png"
	"math/rand/v2"
	"os"
	"time"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/inpututil"
	"github.com/hajimehoshi/ebiten/v2/text/v2"
	"golang.org/x/image/font/gofont/gobold"
	"golang.org/x/image/font/gofont/goregular"

	"github.com/xetra11/TotalKrieg/internal/game"
	"github.com/xetra11/TotalKrieg/internal/mission"
	"github.com/xetra11/TotalKrieg/internal/symbols"
)

// aiDelayFrames keeps the board readable before the AI answers.
const aiDelayFrames = 30

// Options configures a run.
type Options struct {
	Seed     uint64        // 0 picks a random map
	Faction  *game.Faction // set to skip the menu
	Autoplay int           // AI turns to play before the first frame
	Shot     string        // write the first frames to this PNG and quit
	Mission  mission.Config
}

// pendingAttack is a fight waiting for the player's choice of resolver.
type pendingAttack struct {
	unit   *game.Unit
	target game.Coord
}

// App is the running program.
type App struct {
	opts Options
	st   *game.State

	selUnit *game.Unit
	selCity *game.Object
	reach   map[game.Coord]int
	message string
	aiWait  int

	showOrg    bool
	attack     *pendingAttack
	battle     *externalBattle
	runMission func(mission.Scenario) error // nil without MiusMissionMaker
	readResult func(mission.Scenario, time.Time) (*game.FieldResult, error)

	sets      [2]*symbols.Set
	regular   *text.GoTextFaceSource
	bold      *text.GoTextFaceSource
	mapImg    *ebiten.Image
	mapDirty  bool
	symbolImg map[symbolKey]*ebiten.Image

	frames int
	shot   bool
}

// New creates the app and, when a faction is set, starts the match.
func New(opts Options) (*App, error) {
	a := &App{opts: opts, symbolImg: map[symbolKey]*ebiten.Image{}}
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
		a.start(*opts.Faction)
		for i := 0; i < opts.Autoplay && !a.st.Over; i++ {
			a.st.PlayTurn()
		}
		a.showRout()
	}
	return a, nil
}

func (a *App) start(f game.Faction) {
	seed := a.opts.Seed
	if seed == 0 {
		seed = rand.Uint64()
	}
	a.st = game.New(seed, f)
	a.clearSelection()
	a.message = "Choose a unit or a city."
	a.mapDirty = true
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
	case a.routing():
	case a.battle != nil:
		if a.battle.done == nil && !a.battle.running {
			a.battle = nil
		}
	case a.attack != nil:
		a.attack = nil
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

// tick lets the AI play once the delay ran out.
func (a *App) tick() {
	a.pollMission()
	a.pollResult()
	if a.st == nil || a.st.Over || a.humanTurn() {
		a.aiWait = 0
		return
	}
	a.aiWait++
	if a.aiWait < aiDelayFrames {
		return
	}
	a.st.PlayTurn()
	a.mapDirty = true
	a.aiWait = 0
	if !a.st.Over {
		a.message = "Your turn."
	}
	a.showRout()
}

// next ends the player's turn.
func (a *App) next() {
	if !a.humanTurn() {
		return
	}
	a.st.EndTurn()
	a.clearSelection()
	a.mapDirty = true
	if !a.st.Over {
		a.message = "The AI is moving."
	}
}

func (a *App) click(x, y float64) {
	switch {
	case a.st == nil:
		a.clickMenu(x, y)
	case a.st.Over && newGameButton().contains(x, y):
		a.st = nil
		a.clearSelection()
	case a.battle != nil:
		a.clickBattleDialog(x, y)
	case a.attack != nil:
		a.clickAttackPrompt(x, y)
	case a.routing():
		a.clickRout(x, y)
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
	for _, f := range []game.Faction{game.Wehrmacht, game.SovietUnion} {
		if factionButton(f).contains(x, y) {
			a.start(f)
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
		a.message = fmt.Sprintf("%s raised.", u.Name)
		return true
	}
	return false
}

func (a *App) clickHex(c game.Coord) {
	st := a.st
	own := a.selUnit != nil && a.selUnit.Faction == st.First
	if a.humanTurn() && own {
		if _, ok := a.reach[c]; ok {
			a.moveSelected(c)
			return
		}
		if _, ok := st.AttackTargets(a.selUnit)[c]; ok {
			a.attack = &pendingAttack{a.selUnit, c}
			return
		}
	}
	a.selectUnit(st.UnitAt(c))
	if o := st.ObjectAt(c); o != nil && o.Kind.IsCity() {
		a.selCity = o
	}
}

func (a *App) moveSelected(c game.Coord) {
	if err := a.st.Move(a.selUnit, c); err != nil {
		a.message = err.Error()
		return
	}
	a.message = fmt.Sprintf("%s moved.", a.selUnit.Short)
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
