// TotalKrieg is a turn-based operational game on a hex map.
package main

import (
	"errors"
	"flag"
	"fmt"
	"os"

	"github.com/hajimehoshi/ebiten/v2"

	"github.com/xetra11/TotalKrieg/internal/game"
	"github.com/xetra11/TotalKrieg/internal/mission"
	"github.com/xetra11/TotalKrieg/internal/ui"
)

func main() {
	seed := flag.Uint64("seed", 0, "map number, 0 picks at random")
	faction := flag.String("faction", "", "wehrmacht or sovietunion, skips the menu")
	autoplay := flag.Int("autoplay", 0, "play this many AI turns before the start")
	shot := flag.String("shot", "", "write an image to this PNG file and quit")
	missionMaker := flag.String("missionmaker", "../MiusMissionMaker", "MiusMissionMaker folder")
	missionGame := flag.String("miusgame", "", "Mius-Front folder, empty uses the Steam default")
	flag.Parse()

	opts := ui.Options{Seed: *seed, Autoplay: *autoplay, Shot: *shot}
	opts.Mission = mission.Config{ToolDir: *missionMaker, GameDir: *missionGame, OutDir: "missions"}
	switch *faction {
	case "":
	case "wehrmacht":
		f := game.Wehrmacht
		opts.Faction = &f
	case "sovietunion":
		f := game.SovietUnion
		opts.Faction = &f
	default:
		fmt.Fprintln(os.Stderr, "unknown faction:", *faction)
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
