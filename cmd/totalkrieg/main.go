// TotalKrieg is a turn-based operational game on a hex map.
package main

import (
	"errors"
	"flag"
	"fmt"
	"os"

	"github.com/hajimehoshi/ebiten/v2"

	"github.com/xetra11/TotalKrieg/internal/game"
	"github.com/xetra11/TotalKrieg/internal/ui"
)

func main() {
	seed := flag.Uint64("seed", 0, "Kartennummer, 0 wählt zufällig")
	faction := flag.String("faction", "", "wehrmacht oder sowjetunion, überspringt das Menü")
	autoplay := flag.Int("autoplay", 0, "so viele KI-Züge vor dem Start spielen")
	shot := flag.String("shot", "", "Bild in diese PNG-Datei schreiben und beenden")
	flag.Parse()

	opts := ui.Options{Seed: *seed, Autoplay: *autoplay, Shot: *shot}
	switch *faction {
	case "":
	case "wehrmacht":
		f := game.Wehrmacht
		opts.Faction = &f
	case "sowjetunion":
		f := game.Sowjetunion
		opts.Faction = &f
	default:
		fmt.Fprintln(os.Stderr, "unbekannte Fraktion:", *faction)
		os.Exit(2)
	}

	app, err := ui.New(opts)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	ebiten.SetWindowTitle("TotalKrieg")
	ebiten.SetWindowSize(1600, 900)
	ebiten.SetWindowResizingMode(ebiten.WindowResizingModeEnabled)
	if err := ebiten.RunGame(app); err != nil && !errors.Is(err, ebiten.Termination) {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
