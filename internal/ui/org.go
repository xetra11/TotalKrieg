package ui

import (
	"fmt"
	"strings"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/text/v2"

	"github.com/xetra11/TotalKrieg/internal/game"
)

const (
	orgRowH   = 28.0
	orgColW   = 386.0
	orgColGap = 8.0
	orgIndent = 14.0
)

var orgPanel = rect{16, 76, 1208, 716}

// orgRow is one line of the order of battle window.
type orgRow struct {
	e      *game.Element
	r      rect
	indent float64
}

// orgRows lays out a formation. A division gets one column per regiment.
func orgRows(root *game.Element) []orgRow {
	var rows []orgRow
	top := orgPanel.y + 84
	var walk func(e *game.Element, x float64, y *float64, indent float64)
	walk = func(e *game.Element, x float64, y *float64, indent float64) {
		rows = append(rows, orgRow{e, rect{x, *y, orgColW, orgRowH - 2}, indent})
		*y += orgRowH
		for _, c := range e.Children {
			walk(c, x, y, indent+orgIndent)
		}
	}
	if root.Kind == game.Division {
		for i, reg := range root.Children {
			y := top
			walk(reg, orgPanel.x+16+float64(i)*(orgColW+orgColGap), &y, 0)
		}
		return rows
	}
	y := top
	walk(root, orgPanel.x+16, &y, 0)
	return rows
}

func (r orgRow) detachButton() rect {
	return rect{r.r.x + r.r.w - 92, r.r.y + 2, 88, r.r.h - 4}
}

func orgCloseButton() rect {
	return rect{orgPanel.x + orgPanel.w - 136, orgPanel.y + 14, 120, 34}
}

func orgButton() rect {
	return rect{sideX + 16, 288, 160, 36}
}

func reattachButton() rect {
	return rect{sideX + 184, 288, 160, 36}
}

// hexName is the grid reference of a hex, as printed on staff maps.
func hexName(c game.Coord) string {
	return fmt.Sprintf("%c%d", 'A'+rune(c.Col), c.Row+1)
}

// membership names the superior formations of an element.
func membership(e *game.Element) string {
	var parts []string
	for p := e.Parent; p != nil; p = p.Parent {
		parts = append(parts, p.Short)
	}
	if len(parts) == 0 {
		return "selbständig"
	}
	return strings.Join(parts, ", ")
}

func (a *App) openOrg() {
	if a.selUnit != nil {
		a.showOrg = true
		a.pendingDetach = nil
	}
}

func (a *App) canDetach(e *game.Element) bool {
	return a.humanTurn() && e.Faction == a.st.First && a.st.CanDetach(e) == nil
}

func (a *App) clickOrg(x, y float64) {
	if orgCloseButton().contains(x, y) {
		a.showOrg = false
		return
	}
	for _, row := range orgRows(a.selUnit.Element.Root()) {
		if !row.r.contains(x, y) {
			continue
		}
		switch {
		case row.e.Token != nil:
			a.selectUnit(row.e.Token)
			a.showOrg = false
		case row.detachButton().contains(x, y) && a.canDetach(row.e):
			a.pendingDetach = row.e
			a.showOrg = false
			a.message = fmt.Sprintf("Wähle ein freies Nachbarfeld für %s.", row.e.Short)
		}
		return
	}
}

func (a *App) selectUnit(u *game.Unit) {
	a.selUnit = u
	a.selCity = nil
	a.reach = nil
	if a.humanTurn() && u != nil && u.Faction == a.st.First {
		a.reach = a.st.Reachable(u)
	}
}

// placeDetachment finishes a detachment on the clicked hex. Any other hex
// cancels it.
func (a *App) placeDetachment(c game.Coord) {
	e := a.pendingDetach
	a.pendingDetach = nil
	u, err := a.st.Detach(e, c)
	if err != nil {
		a.message = "Ausgliedern abgebrochen."
		return
	}
	a.selectUnit(u)
	a.message = fmt.Sprintf("%s ausgegliedert, gehört weiter zu %s.", u.Short, membership(e))
}

func (a *App) reattachSelected() {
	t, err := a.st.Reattach(a.selUnit)
	if err != nil {
		a.message = err.Error()
		return
	}
	a.message = fmt.Sprintf("%s wieder bei %s eingegliedert.", a.selUnit.Short, t.Short)
	a.selectUnit(t)
}

func (a *App) drawOrg(dst *ebiten.Image) {
	r := orgPanel
	fillRect(dst, r, colChrome)
	strokeRect(dst, r, 2, colAmber)
	root := a.selUnit.Element.Root()
	a.label(dst, root.Name, r.x+16, r.y+12, 24, colText, true, text.AlignStart)
	summary := fmt.Sprintf("Stärke %d von %d, %d Verbände auf der Karte", root.Strength(), root.FullStrength(), len(root.Tokens()))
	a.label(dst, summary, r.x+16, r.y+46, 15, colMuted, false, text.AlignStart)
	a.drawButton(dst, orgCloseButton(), "Schließen", true)
	for _, row := range orgRows(root) {
		a.drawOrgRow(dst, row)
	}
}

func (a *App) drawOrgRow(dst *ebiten.Image, row orgRow) {
	e := row.e
	r := row.r
	if holder := e.Holder(); holder != nil && holder == a.selUnit {
		fillRect(dst, r, colLine)
	} else {
		fillRect(dst, r, colPanel)
	}
	x := r.x + row.indent + 6
	y := r.y + 6
	fg := colText
	if e.Strength() == 0 {
		fg = colDisabled
	}
	if e.Kind == game.Kompanie {
		a.label(dst, companyLabel(e), x, y, 13, fg, true, text.AlignStart)
		a.label(dst, e.Callsign, x+78, y, 13, colAmber, false, text.AlignStart)
		a.label(dst, e.Leader, x+150, y, 12, fg, false, text.AlignStart)
	} else {
		a.label(dst, e.Short, x, y, 14, fg, true, text.AlignStart)
		a.label(dst, e.Kind.String(), x+96, y, 13, colMuted, false, text.AlignStart)
	}
	a.label(dst, fmt.Sprintf("%d/%d", e.Strength(), e.FullStrength()), r.x+r.w-98, y, 13, fg, true, text.AlignEnd)
	switch {
	case e.Detached():
		a.label(dst, "ausgegl. "+hexName(e.Token.Pos), r.x+r.w-6, y, 12, colAmber, true, text.AlignEnd)
	case e.Kind == game.Kompanie && e.Strength() == 0:
		a.label(dst, "aufgerieben", r.x+r.w-6, y, 12, colDisabled, false, text.AlignEnd)
	case a.canDetach(e):
		b := row.detachButton()
		fillRect(dst, b, colAmber)
		a.label(dst, "Ausgliedern", b.x+b.w/2, b.y+5, 12, colDeep, true, text.AlignCenter)
	}
}

// companyLabel marks machine gun companies, whose number alone doesn't show it.
func companyLabel(e *game.Element) string {
	if e.Faction == game.Wehrmacht && e.Role == game.RoleMG {
		return e.Short + " MG"
	}
	return e.Short
}

func (a *App) drawButton(dst *ebiten.Image, r rect, label string, enabled bool) {
	bg, fg := colLine, colDisabled
	if enabled {
		bg, fg = colAmber, colDeep
	}
	fillRect(dst, r, bg)
	a.label(dst, label, r.x+r.w/2, r.y+r.h/2-8, 14, fg, true, text.AlignCenter)
}

// drawFormationLinks ties the selected unit to the rest of its formation.
func (a *App) drawFormationLinks(dst *ebiten.Image) {
	u := a.selUnit
	if u == nil {
		return
	}
	x0, y0 := hexCenter(u.Pos)
	for _, t := range u.Element.Root().Tokens() {
		if t == u {
			continue
		}
		x1, y1 := hexCenter(t.Pos)
		dashedLine(dst, x0, y0, x1, y1, factionColor(u.Faction))
	}
}

func (a *App) drawDetachSpots(dst *ebiten.Image) {
	e := a.pendingDetach
	if e == nil {
		return
	}
	for _, c := range a.st.DetachSpots(e.Holder()) {
		cx, cy := hexCenter(c)
		strokePoly(dst, hexCorners(cx, cy, hexSize-2), true, 3, colAmber)
	}
}
