package ui

import (
	"github.com/xetra11/TotalKrieg/internal/game"
)

// moveStep is one unit movement to animate.
type moveStep struct {
	unit *game.Unit
	from game.Coord
	to   game.Coord
}

// recordMove saves a movement for later animation.
func (a *App) recordMove(u *game.Unit, from, to game.Coord) {
	if from == to {
		return
	}
	a.moveQueue = append(a.moveQueue, moveStep{
		unit: u,
		from: from,
		to:   to,
	})
}

// playMoves starts animating all recorded moves.
func (a *App) playMoves() {
	if a.opts.NoAnimation {
		a.moveQueue = nil
		return
	}
	for _, m := range a.moveQueue {
		a.animations.startMove(m.unit, m.from, m.to)
	}
	a.moveQueue = nil
}

// waitForAnimations reports whether animations are blocking the game flow.
func (a *App) waitForAnimations() bool {
	if a.opts.NoAnimation {
		return false
	}
	return a.animations.isRunning()
}
