package ui

import (
	"fmt"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/text/v2"

	"github.com/xetra11/TotalKrieg/internal/game"
	"github.com/xetra11/TotalKrieg/internal/mission"
)

const (
	orgRowH   = 32.0
	orgIndent = 18.0
)

var orgPanel = rect{16, 76, 1208, 716}

// orgRow is one line of the order of battle window.
type orgRow struct {
	e      *game.Element
	r      rect
	indent float64
}

// orgRows lays out a formation, the staff first.
func orgRows(root *game.Element) []orgRow {
	x, y := orgPanel.x+16, orgPanel.y+110
	w := orgPanel.w - 32
	rows := []orgRow{{root, rect{x, y, w, orgRowH - 2}, 0}}
	for i, c := range root.Children {
		rows = append(rows, orgRow{c, rect{x, y + float64(i+1)*orgRowH, w, orgRowH - 2}, orgIndent})
	}
	return rows
}

func orgCloseButton() rect {
	return rect{orgPanel.x + orgPanel.w - 136, orgPanel.y + 14, 120, 34}
}

func orgButton() rect {
	return rect{sideX + 16, 288, 160, 36}
}

// hexName is the grid reference of a hex, as printed on staff maps.
func hexName(c game.Coord) string {
	return c.Name()
}

// membership names the unit's battalion and its link to it.
func membership(u *game.Unit) string {
	switch {
	case u.Kind == game.Battalion:
		return "Command staff, doesn't attack"
	case u.Element.Parent == nil:
		return "independent, no staff link"
	case u.Linked():
		return u.Element.Parent.Short + ", staff linked"
	}
	return u.Element.Parent.Short + ", no staff link"
}

func (a *App) openOrg() {
	if a.selUnit != nil {
		a.showOrg = true
	}
}

func (a *App) clickOrg(x, y float64) {
	if orgCloseButton().contains(x, y) {
		a.showOrg = false
		return
	}
	for _, row := range orgRows(a.selUnit.Element.Root()) {
		if row.r.contains(x, y) && row.e.Token != nil {
			a.selectUnit(row.e.Token)
			a.showOrg = false
			return
		}
	}
}

func (a *App) selectUnit(u *game.Unit) {
	a.selUnit = u
	a.selCity = nil
	a.reach = nil
	if a.humanTurn() && u != nil && u.Faction == a.st.First {
		a.reach = a.st.OrderTargets(u)
	}
}

func (a *App) drawOrg(dst *ebiten.Image) {
	r := orgPanel
	fillRect(dst, r, colChrome)
	strokeRect(dst, r, 2, colAmber)
	root := a.selUnit.Element.Root()
	a.label(dst, root.Name, r.x+16, r.y+12, 24, colText, true, text.AlignStart)
	summary := fmt.Sprintf("Strength %d of %d", root.Strength(), root.FullStrength())
	a.label(dst, summary, r.x+16, r.y+46, 15, colMuted, false, text.AlignStart)
	a.drawButton(dst, orgCloseButton(), "Close", true)

	y := r.y + 86
	for _, h := range []struct {
		name string
		x    float64
	}{{"Unit", 22}, {"Callsign", 190}, {"Leader", 290}, {"Strength", 440}, {"Position", 510}, {"In Mius-Front", 800}} {
		a.label(dst, h.name, r.x+16+h.x, y, 13, colMuted, true, text.AlignStart)
	}
	for _, row := range orgRows(root) {
		a.drawOrgRow(dst, row)
	}
}

func (a *App) drawOrgRow(dst *ebiten.Image, row orgRow) {
	e, r := row.e, row.r
	bg := colPanel
	if e.Token != nil && e.Token == a.selUnit {
		bg = colLine
	}
	fillRect(dst, r, bg)
	y := r.y + 8
	fg := colText
	if e.Token == nil {
		fg = colDisabled
	}
	a.label(dst, companyLabel(e), r.x+6+row.indent, y, 14, fg, true, text.AlignStart)
	a.label(dst, e.Callsign, r.x+190, y, 14, colAmber, false, text.AlignStart)
	a.label(dst, e.Leader, r.x+290, y, 14, fg, false, text.AlignStart)
	a.label(dst, fmt.Sprintf("%d/%d", e.Strength(), e.FullStrength()), r.x+440, y, 14, fg, true, text.AlignStart)
	a.label(dst, orgStatus(e), r.x+510, y, 14, fg, false, text.AlignStart)
	if e.Kind == game.Company && e.Token != nil {
		a.label(dst, mission.Describe(e), r.x+800, y, 14, colMuted, false, text.AlignStart)
	}
	if e.Kind == game.Battalion {
		a.label(dst, "Battalion HQ", r.x+800, y, 14, colMuted, false, text.AlignStart)
	}
}

func orgStatus(e *game.Element) string {
	if e.Token != nil && e.Token.OutOfAction() {
		return "on " + hexName(e.Token.Pos) + ", beaten"
	}
	switch {
	case e.Token == nil && e.Kind == game.Company:
		return "wiped out"
	case e.Token == nil:
		return "staff destroyed"
	case e.Kind == game.Battalion:
		return "staff on " + hexName(e.Token.Pos)
	case e.Token.Linked():
		return "on " + hexName(e.Token.Pos) + ", linked"
	}
	return "on " + hexName(e.Token.Pos) + ", no link"
}

// companyLabel marks machine gun companies, whose number alone doesn't show it.
func companyLabel(e *game.Element) string {
	if e.Faction == game.Wehrmacht && e.Kind == game.Company && e.Role == game.RoleMG {
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

// drawFormationLinks ties a formation's companies to its staff.
func (a *App) drawFormationLinks(dst *ebiten.Image) {
	u := a.selUnit
	if u == nil {
		return
	}
	anchor := u
	if staff := u.Staff(); staff != nil {
		anchor = staff
	}
	x0, y0 := hexCenter(anchor.Pos)
	for _, t := range u.Element.Root().Tokens() {
		if t != anchor {
			x1, y1 := hexCenter(t.Pos)
			dashedLine(dst, x0, y0, x1, y1, factionColor(u.Faction))
		}
	}
}
