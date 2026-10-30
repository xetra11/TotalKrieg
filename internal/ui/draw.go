package ui

import (
	"fmt"
	"image/color"
	"math"
	"strings"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/text/v2"
	"github.com/hajimehoshi/ebiten/v2/vector"

	"github.com/xetra11/TotalKrieg/internal/game"
)

var (
	colChrome   = color.RGBA{0x23, 0x26, 0x1f, 0xff}
	colPanel    = color.RGBA{0x2a, 0x2e, 0x27, 0xff}
	colDeep     = color.RGBA{0x18, 0x1a, 0x15, 0xff}
	colLine     = color.RGBA{0x4b, 0x50, 0x44, 0xff}
	colText     = color.RGBA{0xe8, 0xe5, 0xd8, 0xff}
	colMuted    = color.RGBA{0xb9, 0xba, 0xab, 0xff}
	colAmber    = color.RGBA{0xd9, 0xbc, 0x62, 0xff}
	colPaper    = color.RGBA{0xe5, 0xe1, 0xd0, 0xff}
	colPaperW   = color.RGBA{0xd8, 0xda, 0xd2, 0xff}
	colPaperS   = color.RGBA{0xe8, 0xd2, 0xc2, 0xff}
	colHexLine  = color.RGBA{0xa9, 0xa5, 0x91, 0xff}
	colInk      = color.RGBA{0x23, 0x23, 0x1d, 0xff}
	colBlue     = color.RGBA{0x1c, 0x4f, 0xa0, 0xff}
	colRed      = color.RGBA{0xc8, 0x28, 0x1c, 0xff}
	colDisabled = color.RGBA{0x9a, 0x9c, 0x8e, 0xff}
)

func factionColor(f game.Faction) color.RGBA {
	if f == game.Wehrmacht {
		return colBlue
	}
	return colRed
}

func (a *App) draw(screen *ebiten.Image) {
	screen.Fill(colChrome)
	if a.st == nil {
		a.drawMenu(screen)
		return
	}
	a.drawTopBar(screen)
	a.drawMap(screen)
	a.drawHighlights(screen)
	a.drawUnits(screen)
	a.drawCityNames(screen)
	a.drawHint(screen)
	a.drawSidebar(screen)
	if a.showOrg {
		a.drawOrg(screen)
	}
	if a.st.Over {
		a.drawEnd(screen)
	}
}

func (a *App) label(dst *ebiten.Image, s string, x, y, size float64, clr color.Color, bold bool, align text.Align) {
	src := a.regular
	if bold {
		src = a.bold
	}
	op := &text.DrawOptions{}
	op.GeoM.Translate(x, y)
	op.ColorScale.ScaleWithColor(clr)
	op.PrimaryAlign = align
	text.Draw(dst, s, &text.GoTextFace{Source: src, Size: size}, op)
}

func (a *App) wrapped(dst *ebiten.Image, s string, x, y, width, size float64, clr color.Color) {
	face := &text.GoTextFace{Source: a.regular, Size: size}
	line := ""
	for _, w := range strings.Fields(s) {
		next := strings.TrimSpace(line + " " + w)
		if tw, _ := text.Measure(next, face, 0); tw > width && line != "" {
			a.label(dst, line, x, y, size, clr, false, text.AlignStart)
			y += size * 1.3
			next = w
		}
		line = next
	}
	a.label(dst, line, x, y, size, clr, false, text.AlignStart)
}

func polyPath(pts [][2]float32, closed bool) *vector.Path {
	p := &vector.Path{}
	for i, pt := range pts {
		if i == 0 {
			p.MoveTo(pt[0], pt[1])
		} else {
			p.LineTo(pt[0], pt[1])
		}
	}
	if closed {
		p.Close()
	}
	return p
}

func pathOptions(clr color.Color) *vector.DrawPathOptions {
	op := &vector.DrawPathOptions{AntiAlias: true}
	op.ColorScale.ScaleWithColor(clr)
	return op
}

func fillPoly(dst *ebiten.Image, pts [][2]float32, clr color.Color) {
	vector.FillPath(dst, polyPath(pts, true), &vector.FillOptions{}, pathOptions(clr))
}

func strokePoly(dst *ebiten.Image, pts [][2]float32, closed bool, width float32, clr color.Color) {
	opts := &vector.StrokeOptions{Width: width, LineJoin: vector.LineJoinMiter, MiterLimit: 4}
	vector.StrokePath(dst, polyPath(pts, closed), opts, pathOptions(clr))
}

func fillRect(dst *ebiten.Image, r rect, clr color.Color) {
	vector.FillRect(dst, float32(r.x), float32(r.y), float32(r.w), float32(r.h), clr, false)
}

func strokeRect(dst *ebiten.Image, r rect, width float32, clr color.Color) {
	vector.StrokeRect(dst, float32(r.x), float32(r.y), float32(r.w), float32(r.h), width, clr, false)
}

func (a *App) drawTopBar(dst *ebiten.Image) {
	st := a.st
	fillRect(dst, rect{0, topH - 2, screenW, 2}, colDeep)
	a.label(dst, "TotalKrieg", 16, 9, 22, colText, true, text.AlignStart)
	a.label(dst, st.First.String(), 16, 36, 14, colMuted, false, text.AlignStart)
	a.label(dst, "Runde", 200, 10, 14, colMuted, false, text.AlignStart)
	a.label(dst, fmt.Sprintf("%d / %d", st.Round, game.MaxRounds), 200, 28, 20, colText, true, text.AlignStart)

	for i, p := range []game.Phase{game.PhaseAufstellung, game.PhaseBewegung, game.PhaseAngriff} {
		r := rect{310 + float64(i)*126, 14, 120, 32}
		clr, bg := colMuted, colPanel
		if a.humanTurn() && st.Phase == p {
			clr, bg = colDeep, colAmber
		}
		fillRect(dst, r, bg)
		a.label(dst, p.String(), r.x+r.w/2, r.y+8, 15, clr, true, text.AlignCenter)
	}
	turn := "Du bist am Zug"
	if !a.humanTurn() {
		turn = "KI ist am Zug"
	}
	a.label(dst, turn, 700, 22, 16, colAmber, true, text.AlignStart)

	stock := st.Stocks[st.First]
	a.label(dst, "Manpower", 900, 10, 14, colMuted, false, text.AlignStart)
	a.label(dst, fmt.Sprint(stock.Manpower), 900, 28, 20, colText, true, text.AlignStart)
	a.label(dst, "Ressourcen", 1010, 10, 14, colMuted, false, text.AlignStart)
	a.label(dst, fmt.Sprint(stock.Resources), 1010, 28, 20, colText, true, text.AlignStart)
	cities, objects := st.Holdings(st.First)
	a.label(dst, "Besitz", 1130, 10, 14, colMuted, false, text.AlignStart)
	a.label(dst, fmt.Sprintf("%d Städte, %d Objekte", cities, objects), 1130, 30, 15, colText, true, text.AlignStart)

	b := nextButton()
	if a.humanTurn() {
		fillRect(dst, b, colAmber)
		name := "Zug beenden"
		if st.Phase != game.PhaseAngriff {
			name = "Weiter: " + (st.Phase + 1).String()
		}
		a.label(dst, name, b.x+b.w/2, b.y+11, 15, colDeep, true, text.AlignCenter)
	} else {
		fillRect(dst, b, colLine)
		a.label(dst, "Bitte warten", b.x+b.w/2, b.y+11, 15, colMuted, true, text.AlignCenter)
	}
}

func (a *App) drawMap(dst *ebiten.Image) {
	if a.mapImg == nil {
		a.mapImg = ebiten.NewImage(mapW, screenH-topH)
		a.mapDirty = true
	}
	if a.mapDirty {
		a.renderMap(a.mapImg)
		a.mapDirty = false
	}
	op := &ebiten.DrawImageOptions{}
	op.GeoM.Translate(0, topH)
	dst.DrawImage(a.mapImg, op)
}

// renderMap draws the board into the cached map image.
func (a *App) renderMap(img *ebiten.Image) {
	img.Fill(colPaper)
	st := a.st
	for row := 0; row < game.MapRows; row++ {
		for col := 0; col < game.MapCols; col++ {
			c := game.Coord{Col: col, Row: row}
			cx, cy := hexCenter(c)
			pts := hexCorners(cx, cy-topH, hexSize)
			fill := colPaperW
			if a.controller(c) == game.Sowjetunion {
				fill = colPaperS
			}
			fillPoly(img, pts, fill)
			strokePoly(img, pts, true, 1, colHexLine)
		}
	}
	for _, o := range st.Objects {
		a.renderObject(img, o)
	}
}

// controller returns the owner of the nearest object, which paints the front.
func (a *App) controller(c game.Coord) game.Faction {
	best, owner := math.MaxInt, game.NoFaction
	for _, o := range a.st.Objects {
		if d := game.Distance(c, o.Pos); d < best {
			best, owner = d, o.Owner
		}
	}
	return owner
}

func (a *App) renderObject(img *ebiten.Image, o *game.Object) {
	cx, cy := hexCenter(o.Pos)
	cy -= topH
	strokePoly(img, hexCorners(cx, cy, hexSize-2.5), true, 3, factionColor(o.Owner))
	x, y := float32(cx), float32(cy)
	switch o.Kind {
	case game.Hauptstadt:
		fillPoly(img, starPoints(cx, cy, 15, 6.5), colInk)
	case game.Stadt:
		vector.FillRect(img, x-11, y-5, 9, 13, colInk, false)
		vector.FillRect(img, x, y-10, 10, 18, colInk, false)
	case game.Dorf:
		vector.FillRect(img, x-5, y-4, 10, 8, colInk, false)
	case game.Fabrik:
		fillPoly(img, [][2]float32{{x - 13, y + 9}, {x - 13, y - 3}, {x - 5, y + 2}, {x - 5, y - 3}, {x + 3, y + 2}, {x + 3, y - 12}, {x + 9, y - 12}, {x + 9, y + 9}}, colInk)
	}
}

// drawCityNames runs after the units, so a garrison can't hide a city's name.
func (a *App) drawCityNames(dst *ebiten.Image) {
	for _, o := range a.st.Objects {
		if o.Kind.IsCity() {
			cx, cy := hexCenter(o.Pos)
			a.label(dst, o.Name, cx, cy+20, 11, colInk, true, text.AlignCenter)
		}
	}
}

func starPoints(cx, cy, outer, inner float64) [][2]float32 {
	pts := make([][2]float32, 10)
	for i := range pts {
		r := outer
		if i%2 == 1 {
			r = inner
		}
		ang := -math.Pi/2 + math.Pi/5*float64(i)
		pts[i] = [2]float32{float32(cx + r*math.Cos(ang)), float32(cy + r*math.Sin(ang))}
	}
	return pts
}

func dashedLine(dst *ebiten.Image, x0, y0, x1, y1 float64, clr color.Color) {
	const dash, gap = 10.0, 7.0
	length := math.Hypot(x1-x0, y1-y0)
	dx, dy := (x1-x0)/length, (y1-y0)/length
	for d := 0.0; d < length; d += dash + gap {
		end := math.Min(d+dash, length)
		vector.StrokeLine(dst, float32(x0+dx*d), float32(y0+dy*d), float32(x0+dx*end), float32(y0+dy*end), 2, clr, true)
	}
}

func (a *App) drawHighlights(dst *ebiten.Image) {
	st := a.st
	a.drawFormationLinks(dst)
	a.drawDetachSpots(dst)
	for c := range a.reach {
		cx, cy := hexCenter(c)
		fillPoly(dst, hexCorners(cx, cy, hexSize-1), color.RGBA{0x6b, 0x5c, 0x22, 0x50})
	}
	if u := a.selUnit; u != nil {
		cx, cy := hexCenter(u.Pos)
		strokePoly(dst, hexCorners(cx, cy, hexSize-1), true, 3, colAmber)
		if a.humanTurn() && st.Phase == game.PhaseAngriff && u.Faction == st.First && !u.Attacked {
			for _, n := range u.Pos.Neighbors() {
				if st.CanAttack(u, n) == nil {
					nx, ny := hexCenter(n)
					strokePoly(dst, hexCorners(nx, ny, hexSize-1), true, 3, colRed)
				}
			}
		}
	}
	if o := a.selCity; o != nil {
		cx, cy := hexCenter(o.Pos)
		strokePoly(dst, hexCorners(cx, cy, hexSize-1), true, 3, colAmber)
	}
}

func (a *App) drawUnits(dst *ebiten.Image) {
	st := a.st
	for _, u := range st.Units {
		cx, cy := hexCenter(u.Pos)
		spent := u.Faction == st.First && a.humanTurn() && a.spent(u)
		plate := color.RGBA{0xef, 0xeb, 0xdb, 0xe6}
		fillRect(dst, rect{cx - 22, cy - 25, 44, 46}, plate)
		img := a.symbol(u.Faction, u.Kind, 0.56)
		op := &ebiten.DrawImageOptions{}
		w := float64(img.Bounds().Dx())
		op.GeoM.Translate(cx-w/2-5, cy-25)
		if spent {
			op.ColorScale.ScaleAlpha(0.45)
		}
		dst.DrawImage(img, op)
		a.label(dst, u.Short, cx, cy+8, 10, factionColor(u.Faction), true, text.AlignCenter)
		a.label(dst, fmt.Sprint(u.Strength()), cx+20, cy-24, 11, colInk, true, text.AlignEnd)
	}
}

func (a *App) spent(u *game.Unit) bool {
	switch a.st.Phase {
	case game.PhaseBewegung:
		return u.Moved || u.Fresh
	case game.PhaseAngriff:
		return u.Attacked
	}
	return false
}

func (a *App) drawHint(dst *ebiten.Image) {
	a.label(dst, a.message, 24, hintY, 18, colInk, true, text.AlignStart)
	seed := fmt.Sprintf("Karte %d", a.st.Seed)
	a.label(dst, seed, 24, hintY+30, 13, colInk, false, text.AlignStart)
	a.label(dst, "Enter: weiter   Esc: Auswahl aufheben", 24, hintY+50, 13, colInk, false, text.AlignStart)
	a.drawKey(dst)
}

func (a *App) drawKey(dst *ebiten.Image) {
	x := 620.0
	a.label(dst, "Hauptstadt", x+24, hintY+8, 13, colInk, false, text.AlignStart)
	fillPoly(dst, starPoints(x+8, hintY+15, 10, 4.5), colInk)
	a.label(dst, "Stadt", x+124, hintY+8, 13, colInk, false, text.AlignStart)
	fillRect(dst, rect{x + 108, hintY + 8, 10, 14}, colInk)
	a.label(dst, "Dorf", x+204, hintY+8, 13, colInk, false, text.AlignStart)
	fillRect(dst, rect{x + 188, hintY + 11, 10, 8}, colInk)
	a.label(dst, "Fabrik", x+284, hintY+8, 13, colInk, false, text.AlignStart)
	fillRect(dst, rect{x + 268, hintY + 8, 10, 14}, colInk)
	fillRect(dst, rect{x + 268, hintY + 8, 4, 6}, colPaper)
	fillRect(dst, rect{x, hintY + 40, 18, 12}, colBlue)
	a.label(dst, "Wehrmacht", x+24, hintY+38, 13, colInk, false, text.AlignStart)
	fillRect(dst, rect{x + 108, hintY + 40, 18, 12}, colRed)
	a.label(dst, "Sowjetunion", x+132, hintY+38, 13, colInk, false, text.AlignStart)
}

func (a *App) drawSidebar(dst *ebiten.Image) {
	fillRect(dst, rect{sideX, topH, sideW, screenH - topH}, colPanel)
	fillRect(dst, rect{sideX, topH, 2, screenH - topH}, colDeep)
	a.drawUnitPanel(dst)
	a.drawCityPanel(dst)
	a.drawLog(dst)
}

func (a *App) panelTitle(dst *ebiten.Image, s string, y float64) {
	a.label(dst, s, sideX+16, y+10, 17, colAmber, true, text.AlignStart)
}

func (a *App) drawUnitPanel(dst *ebiten.Image) {
	a.panelTitle(dst, "Einheit", unitPanelY)
	u := a.selUnit
	x := float64(sideX + 16)
	y := float64(unitPanelY + 40)
	if u == nil {
		a.wrapped(dst, "Klicke auf eine Einheit, um sie zu sehen.", x, y, sideW-32, 15, colMuted)
		return
	}
	st := u.Stats()
	fillRect(dst, rect{x, y, 96, 112}, colPaper)
	img := a.symbol(u.Faction, u.Kind, 1.6)
	op := &ebiten.DrawImageOptions{}
	op.GeoM.Translate(x+8, y+6)
	dst.DrawImage(img, op)
	a.drawFlagLabel(dst, u, x+8, y+6, 1.6)
	tx := x + 110
	a.label(dst, u.Kind.String(), tx, y, 13, colMuted, false, text.AlignStart)
	a.wrapped(dst, u.Short, tx, y+16, 220, 18, colText)
	a.label(dst, fmt.Sprintf("Stärke %d / %d", u.Strength(), u.FullStrength()), tx, y+42, 16, colText, true, text.AlignStart)
	a.label(dst, "Bewegung "+fieldCount(st.Move), tx, y+66, 15, colText, false, text.AlignStart)
	if e := u.Element; e.Kind == game.Kompanie {
		a.label(dst, e.Callsign, tx, y+90, 15, colAmber, true, text.AlignStart)
		a.label(dst, e.Leader, tx+90, y+90, 14, colText, false, text.AlignStart)
	}
	a.label(dst, u.Name, x, y+122, 14, colText, false, text.AlignStart)
	a.label(dst, "Historische Sollstärke: "+st.Men+" Mann", x, y+142, 14, colMuted, false, text.AlignStart)
	a.label(dst, "Gehört zu: "+membership(u.Element), x, y+162, 14, colMuted, false, text.AlignStart)
	a.drawButton(dst, orgButton(), "Gliederung (G)", true)
	if u.Element.Detached() {
		_, err := a.st.CanReattach(u)
		a.drawButton(dst, reattachButton(), "Eingliedern", a.humanTurn() && err == nil)
	}
}

func fieldCount(n int) string {
	if n == 1 {
		return "1 Feld"
	}
	return fmt.Sprintf("%d Felder", n)
}

// drawFlagLabel writes the designation into a Soviet flag, as the manual shows.
func (a *App) drawFlagLabel(dst *ebiten.Image, u *game.Unit, x, y, scale float64) {
	sym := a.sets[u.Faction].Units[unitKey(u.Kind)]
	if sym.Label == nil || u.Flag == "" {
		return
	}
	a.label(dst, u.Flag, x+float64(sym.Label.X)*scale, y+float64(sym.Label.Y)*scale,
		float64(sym.Label.Size)*scale, factionColor(u.Faction), true, text.AlignStart)
}

func (a *App) drawCityPanel(dst *ebiten.Image) {
	fillRect(dst, rect{sideX + 2, cityPanelY, sideW - 2, 1}, colLine)
	a.panelTitle(dst, "Stadt", cityPanelY)
	o := a.selCity
	x := float64(sideX + 16)
	if o == nil {
		a.wrapped(dst, "In der Aufstellungsphase eine eigene Stadt anklicken.", x, cityPanelY+40, sideW-32, 15, colMuted)
		return
	}
	a.label(dst, fmt.Sprintf("%s, %s (%s)", o.Name, o.Kind, o.Owner), x, cityPanelY+38, 15, colText, true, text.AlignStart)
	for i, k := range game.UnitKinds {
		r := recruitButton(i)
		stats := game.Stats(a.st.First, k)
		ok := a.humanTurn() && a.st.CanRecruit(o, k) == nil
		bg, fg := colDeep, colDisabled
		if ok {
			bg, fg = colLine, colText
		}
		fillRect(dst, r, bg)
		strokeRect(dst, r, 1, colLine)
		a.label(dst, k.String(), r.x+12, r.y+6, 16, fg, true, text.AlignStart)
		a.label(dst, fmt.Sprintf("Stärke %d, %s", stats.Strength, fieldCount(stats.Move)), r.x+12, r.y+26, 13, fg, false, text.AlignStart)
		a.label(dst, fmt.Sprintf("%d Manpower", stats.Cost), r.x+r.w-12, r.y+14, 15, fg, true, text.AlignEnd)
	}
}

func (a *App) drawLog(dst *ebiten.Image) {
	fillRect(dst, rect{sideX + 2, logPanelY, sideW - 2, 1}, colLine)
	a.panelTitle(dst, "Kampfbericht", logPanelY)
	x := float64(sideX + 16)
	y := float64(logPanelY + 40)
	reports := a.st.Reports
	if len(reports) == 0 {
		a.label(dst, "Noch keine Gefechte.", x, y, 15, colMuted, false, text.AlignStart)
		return
	}
	for i := len(reports) - 1; i >= 0 && y < screenH-44; i-- {
		r := reports[i]
		a.label(dst, fmt.Sprintf("Runde %d: %s gegen %s", r.Round, r.Attacker, r.Defender), x, y, 14, factionText(r.AttackerFaction), true, text.AlignStart)
		a.label(dst, lossText(r), x, y+18, 13, colMuted, false, text.AlignStart)
		y += 44
	}
}

func factionText(f game.Faction) color.RGBA {
	if f == game.Wehrmacht {
		return color.RGBA{0xa9, 0xc3, 0xee, 0xff}
	}
	return color.RGBA{0xf0, 0xa7, 0x9f, 0xff}
}

func lossText(r game.Report) string {
	s := fmt.Sprintf("Verluste %d zu %d", r.AttackerLoss, r.DefenderLoss)
	if r.DefenderDestroyed {
		s += ", Verteidiger vernichtet"
	}
	if r.AttackerDestroyed {
		s += ", Angreifer vernichtet"
	}
	return s
}

func (a *App) drawEnd(dst *ebiten.Image) {
	st := a.st
	r := rect{mapW/2 - 300, 300, 600, 270}
	fillRect(dst, r, colChrome)
	strokeRect(dst, r, 3, colAmber)
	title := "Unentschieden"
	switch st.Winner {
	case st.First:
		title = "Sieg"
	case st.First.Other():
		title = "Niederlage"
	}
	a.label(dst, title, mapW/2, r.y+36, 44, colText, true, text.AlignCenter)
	a.label(dst, st.Reason, mapW/2, r.y+120, 17, colMuted, false, text.AlignCenter)
	b := newGameButton()
	fillRect(dst, b, colAmber)
	a.label(dst, "Neue Partie", b.x+b.w/2, b.y+13, 18, colDeep, true, text.AlignCenter)
}

func (a *App) drawMenu(dst *ebiten.Image) {
	a.label(dst, "TotalKrieg", screenW/2, 150, 72, colText, true, text.AlignCenter)
	a.label(dst, "Operationsspiel auf der Hexkarte, Prototyp", screenW/2, 250, 22, colMuted, false, text.AlignCenter)
	a.label(dst, "Wähle deine Seite", screenW/2, 310, 20, colAmber, true, text.AlignCenter)
	for _, f := range []game.Faction{game.Wehrmacht, game.Sowjetunion} {
		r := factionButton(f)
		fillRect(dst, r, colPanel)
		strokeRect(dst, r, 2, colLine)
		fillRect(dst, rect{r.x + 130, r.y + 20, 100, 120}, colPaper)
		img := a.symbol(f, game.Division, 1.8)
		op := &ebiten.DrawImageOptions{}
		w := float64(img.Bounds().Dx())
		op.GeoM.Translate(r.x+180-w/2, r.y+30)
		dst.DrawImage(img, op)
		a.label(dst, f.String(), r.x+r.w/2, r.y+160, 28, colText, true, text.AlignCenter)
		a.label(dst, "Kartenhälfte "+map[game.Faction]string{game.Wehrmacht: "West", game.Sowjetunion: "Ost"}[f], r.x+r.w/2, r.y+200, 15, colMuted, false, text.AlignCenter)
	}
}
