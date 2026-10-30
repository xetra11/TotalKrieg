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

// App is the running program.
type App struct {
	opts Options
	st   *game.State

	selUnit *game.Unit
	selCity *game.Object
	reach   map[game.Coord]int
	message string
	aiWait  int

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
	a.message = "Wähle eine Stadt, um Einheiten aufzustellen."
	a.mapDirty = true
}

func (a *App) clearSelection() {
	a.selUnit, a.selCity, a.reach = nil, nil, nil
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
		a.next()
	}
	if inpututil.IsKeyJustPressed(ebiten.KeyEscape) {
		a.clearSelection()
	}
	a.tick()
	return nil
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
		a.message = "Du bist am Zug: Wähle eine Stadt, um Einheiten aufzustellen."
	}
}

// next moves on to the next phase or ends the turn.
func (a *App) next() {
	if !a.humanTurn() {
		return
	}
	a.st.Advance()
	a.clearSelection()
	a.mapDirty = true
	switch {
	case a.st.Over:
	case !a.humanTurn():
		a.message = "Die KI ist am Zug."
	case a.st.Phase == game.PhaseBewegung:
		a.message = "Wähle eine eigene Einheit und ein markiertes Feld."
	default:
		a.message = "Wähle eine eigene Einheit und einen benachbarten Gegner."
	}
}

func (a *App) click(x, y float64) {
	switch {
	case a.st == nil:
		a.clickMenu(x, y)
	case a.st.Over && newGameButton().contains(x, y):
		a.st = nil
		a.clearSelection()
	case nextButton().contains(x, y):
		a.next()
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
	own := a.selUnit != nil && a.selUnit.Faction == st.First
	if a.humanTurn() {
		switch st.Phase {
		case game.PhaseBewegung:
			if _, ok := a.reach[c]; ok && own {
				a.moveSelected(c)
				return
			}
		case game.PhaseAngriff:
			if t := st.UnitAt(c); t != nil && own && t.Faction != st.First {
				a.attackSelected(c)
				return
			}
		}
	}
	a.selUnit = st.UnitAt(c)
	a.selCity = nil
	a.reach = nil
	if o := st.ObjectAt(c); o != nil && o.Kind.IsCity() {
		a.selCity = o
	}
	if a.humanTurn() && st.Phase == game.PhaseBewegung && a.selUnit != nil && a.selUnit.Faction == st.First {
		a.reach = st.Reachable(a.selUnit)
	}
}

func (a *App) moveSelected(c game.Coord) {
	if err := a.st.Move(a.selUnit, c); err != nil {
		a.message = err.Error()
		return
	}
	a.reach = nil
	a.message = fmt.Sprintf("%s ist gezogen.", a.selUnit.Short)
}

func (a *App) attackSelected(c game.Coord) {
	r, err := a.st.Attack(a.selUnit, c)
	if err != nil {
		a.message = err.Error()
		return
	}
	a.message = fmt.Sprintf("Gefecht: eigene Verluste %d, Feind %d.", r.AttackerLoss, r.DefenderLoss)
	if r.AttackerDestroyed {
		a.selUnit = nil
	}
	a.mapDirty = true
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
