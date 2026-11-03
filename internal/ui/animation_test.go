package ui

import (
	"testing"

	"github.com/xetra11/TotalKrieg/internal/game"
)

// TestQueuedMoveStaysAtStartUntilItPlays guards against units flashing on the
// target hex before their move is animated. The game state already holds the
// destination while the moves wait in the queue.
func TestQueuedMoveStaysAtStartUntilItPlays(t *testing.T) {
	from := game.Coord{Col: 2, Row: 2}
	to := game.Coord{Col: 6, Row: 2}
	first := &game.Unit{Pos: to}
	second := &game.Unit{Pos: to}

	var q animationQueue
	q.startMove(first, from, to)
	q.startMove(second, from, to)

	assertDrawnAt(t, &q, first, from)
	assertDrawnAt(t, &q, second, from)

	// The first move starts, the second keeps waiting at its start hex.
	q.tick()
	if q.current == nil || q.current.unit != first {
		t.Fatal("first move did not start")
	}
	assertDrawnAt(t, &q, second, from)

	// Once the first move is done it sits on its target, the second is still
	// waiting at the start.
	for q.current != nil {
		q.tick()
	}
	assertDrawnAt(t, &q, first, to)
	assertDrawnAt(t, &q, second, from)

	// After the pause the second move starts.
	for i := 0; i <= moveDelayFrames; i++ {
		q.tick()
	}
	if q.current == nil || q.current.unit != second {
		t.Fatal("second move did not start after the delay")
	}
}

func assertDrawnAt(t *testing.T, q *animationQueue, u *game.Unit, c game.Coord) {
	t.Helper()
	x, y := q.positionOf(u)
	wx, wy := hexCenter(c)
	if x != wx || y != wy {
		t.Fatalf("unit drawn at (%v, %v), want %v", x, y, c)
	}
}
