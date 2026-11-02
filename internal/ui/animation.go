package ui

import (
	"github.com/xetra11/TotalKrieg/internal/game"
)

const (
	// animationFrames is how long a unit move takes.
	animationFrames = 20
	// moveDelayFrames is the pause between consecutive moves.
	moveDelayFrames = 5
)

// animation is one unit moving from one hex to another.
type animation struct {
	unit   *game.Unit
	from   game.Coord
	to     game.Coord
	frame  int
	frames int
}

// animationQueue holds the pending animations and the current one.
type animationQueue struct {
	current *animation
	pending []animation
	delay   int // frames to wait before starting next animation
}

// startMove begins animating a unit's movement.
func (q *animationQueue) startMove(u *game.Unit, from, to game.Coord) {
	if from == to {
		return
	}
	q.pending = append(q.pending, animation{
		unit:   u,
		from:   from,
		to:     to,
		frame:  0,
		frames: animationFrames,
	})
}

// tick advances the current animation and starts the next one when ready.
func (q *animationQueue) tick() {
	if q.current != nil {
		q.current.frame++
		if q.current.frame >= q.current.frames {
			q.current = nil
			q.delay = moveDelayFrames
		}
		return
	}
	if q.delay > 0 {
		q.delay--
		return
	}
	if len(q.pending) > 0 {
		q.current = &q.pending[0]
		q.pending = q.pending[1:]
	}
}

// isRunning reports whether an animation is active or pending.
func (q *animationQueue) isRunning() bool {
	return q.current != nil || len(q.pending) > 0 || q.delay > 0
}

// clear drops all animations.
func (q *animationQueue) clear() {
	q.current = nil
	q.pending = nil
	q.delay = 0
}

// progress returns the interpolation factor [0, 1] for the current animation.
func (a *animation) progress() float64 {
	if a.frames == 0 {
		return 1
	}
	p := float64(a.frame) / float64(a.frames)
	// Ease out cubic for smoother animation
	return 1 - (1-p)*(1-p)*(1-p)
}

// position returns the interpolated screen position of the animated unit.
func (a *animation) position() (float64, float64) {
	t := a.progress()
	x0, y0 := hexCenter(a.from)
	x1, y1 := hexCenter(a.to)
	return x0 + (x1-x0)*t, y0 + (y1-y0)*t
}
