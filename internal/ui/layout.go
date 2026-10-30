package ui

import (
	"math"

	"github.com/xetra11/TotalKrieg/internal/game"
)

const (
	screenW = 1600
	screenH = 900
	topH    = 60
	mapW    = 1240
	sideX   = mapW
	sideW   = screenW - mapW

	hexSize = 29.0
	rowStep = 1.5 * hexSize

	unitPanelY = topH
	cityPanelY = 330
	logPanelY  = 570
	hintY      = 804
)

var (
	hexW    = math.Sqrt(3) * hexSize
	originX = (mapW-hexW*(game.MapCols+0.5))/2 + hexW/2
	originY = topH + 16 + hexSize
)

type rect struct {
	x, y, w, h float64
}

func (r rect) contains(x, y float64) bool {
	return x >= r.x && x < r.x+r.w && y >= r.y && y < r.y+r.h
}

// hexCenter returns the screen position of a hex center.
func hexCenter(c game.Coord) (x, y float64) {
	x = originX + hexW*(float64(c.Col)+0.5*float64(c.Row&1))
	y = originY + rowStep*float64(c.Row)
	return x, y
}

// hexAt returns the hex under a screen position.
func hexAt(x, y float64) (game.Coord, bool) {
	best, bestDist := game.Coord{}, math.MaxFloat64
	for row := 0; row < game.MapRows; row++ {
		for col := 0; col < game.MapCols; col++ {
			c := game.Coord{Col: col, Row: row}
			cx, cy := hexCenter(c)
			if d := math.Hypot(x-cx, y-cy); d < bestDist {
				best, bestDist = c, d
			}
		}
	}
	return best, bestDist <= hexSize
}

// hexCorners returns the outline of a hex around a center.
func hexCorners(cx, cy, size float64) [][2]float32 {
	pts := make([][2]float32, 6)
	for i := range pts {
		a := math.Pi/3*float64(i) + math.Pi/6
		pts[i] = [2]float32{float32(cx + size*math.Cos(a)), float32(cy + size*math.Sin(a))}
	}
	return pts
}

func nextButton() rect {
	return rect{1396, 10, 188, 40}
}

func recruitButton(i int) rect {
	return rect{sideX + 16, cityPanelY + 70 + float64(i)*52, sideW - 32, 46}
}

func factionButton(f game.Faction) rect {
	return rect{420 + float64(f)*400, 360, 360, 240}
}

func newGameButton() rect {
	return rect{mapW/2 - 110, 500, 220, 48}
}
