package ui

import (
	"math"

	"github.com/hajimehoshi/ebiten/v2"

	"github.com/xetra11/TotalKrieg/internal/game"
)

type symbolKey struct {
	faction game.Faction
	kind    game.UnitKind
	scale   float64
}

func unitKey(k game.UnitKind) string {
	if k == game.Company {
		return "company"
	}
	return "battalion"
}

// symbol returns the rendered sign of a unit type, cached per scale.
func (a *App) symbol(f game.Faction, k game.UnitKind, scale float64) *ebiten.Image {
	key := symbolKey{f, k, scale}
	if img, ok := a.symbolImg[key]; ok {
		return img
	}
	sym := a.sets[f].Units[unitKey(k)]
	s := float32(scale)
	w := int(math.Ceil(float64(sym.W*s))) + 2
	h := int(math.Ceil(float64(sym.H*s))) + 2
	img := ebiten.NewImage(w, h)
	clr := factionColor(f)
	for _, shape := range sym.Shapes {
		pts := make([][2]float32, len(shape.Points))
		for i, p := range shape.Points {
			pts[i] = [2]float32{p[0]*s + 1, p[1]*s + 1}
		}
		width := shape.Width
		if width == 0 {
			width = 2
		}
		switch shape.Op {
		case "fill":
			fillPoly(img, pts, clr)
		case "outline":
			strokePoly(img, pts, true, width*s, clr)
		default:
			strokePoly(img, pts, false, width*s, clr)
		}
	}
	a.symbolImg[key] = img
	return img
}
