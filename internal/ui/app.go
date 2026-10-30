// Package ui is the Ebitengine front end of TotalKrieg.
package ui

import (
	"bytes"
	"fmt"
	"image/png"
	"math/rand/v2"
	"os"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/inpututil"
	"github.com/hajimehoshi/ebiten/v2/text/v2"
	"golang.org/x/image/font/gofont/gobold"
	"golang.org/x/image/font/gofont/goregular"

	"github.com/xetra11/TotalKrieg/internal/game"
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

	showOrg       bool
	pendingDetach *game.Element
	attack        *pendingAttack

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
	if a.sets[game.Sowjetunion], err = symbols.Load("sowjetunion"); err != nil {
		return nil, err
	}
	if a.regular, err = text.NewGoTextFaceSource(bytes.NewReader(goregular.TTF)); err != nil {
		return nil, err
	}
	if a.bold, err = text.NewGoTextFaceSource(bytes.NewReader(gobold.TTF)); err != nil {
		return nil, err
	}
	if opts.Faction != nil {
		a.start(*opts.Faction)
		for i := 0; i < opts.Autoplay && !a.st.Over; i++ {
			a.st.PlayTurn()
		}
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
	a.message = "Wähle eine Einheit oder eine Stadt."
	a.mapDirty = true
}

func (a *App) clearSelection() {
	a.selUnit, a.selCity, a.reach = nil, nil, nil
	a.showOrg, a.pendingDetach, a.attack = false, nil, nil
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
	case a.attack != nil:
		a.resolveAuto()
	case !a.showOrg:
		a.next()
	}
}

// cancel steps back one level: dialog, window, pending detachment, selection.
func (a *App) cancel() {
	switch {
	case a.attack != nil:
		a.attack = nil
	case a.showOrg:
		a.showOrg = false
	case a.pendingDetach != nil:
		a.pendingDetach = nil
	default:
		a.clearSelection()
	}
}

func (a *App) toggleOrg() {
	if a.showOrg {
		a.showOrg = false
		return
	}
	a.openOrg()
}

// tick lets the AI play once the delay ran out.
func (a *App) tick() {
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
		a.message = "Du bist am Zug."
	}
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
		a.message = "Die KI ist am Zug."
	}
}

func (a *App) click(x, y float64) {
	switch {
	case a.st == nil:
		a.clickMenu(x, y)
	case a.st.Over && newGameButton().contains(x, y):
		a.st = nil
		a.clearSelection()
	case a.attack != nil:
		a.clickAttackPrompt(x, y)
	case a.showOrg:
		a.clickOrg(x, y)
	case nextButton().contains(x, y):
		a.next()
	case a.selUnit != nil && orgButton().contains(x, y):
		a.openOrg()
	case a.selUnit != nil && reattachButton().contains(x, y):
		if a.humanTurn() && a.selUnit.Element.Detached() {
			a.reattachSelected()
		}
	case a.selCity != nil && a.clickRecruit(x, y):
	default:
		if c, ok := hexAt(x, y); ok {
			a.clickHex(c)
		}
	}
}

func (a *App) clickMenu(x, y float64) {
	for _, f := range []game.Faction{game.Wehrmacht, game.Sowjetunion} {
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
		a.message = fmt.Sprintf("%s aufgestellt.", u.Name)
		return true
	}
	return false
}

func (a *App) clickHex(c game.Coord) {
	st := a.st
	if a.pendingDetach != nil {
		a.placeDetachment(c)
		return
	}
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
	a.message = fmt.Sprintf("%s ist gezogen.", a.selUnit.Short)
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
