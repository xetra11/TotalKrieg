package ui

import (
	"fmt"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/text/v2"

	"github.com/xetra11/TotalKrieg/internal/game"
)

// tacticalOptions are the planned tactical layer resolvers. None is wired up yet.
var tacticalOptions = []string{
	"Hexfront (Advanced Squad Leader 2 Digital)",
	"MiusPlayer (Mius Front Multiplayer)",
	"Combat Mission Multiplayer",
}

var attackPrompt = rect{mapW/2 - 310, 170, 620, 450}

func autoResolveButton() rect {
	return rect{attackPrompt.x + 24, attackPrompt.y + 176, attackPrompt.w - 48, 52}
}

func tacticalRow(i int) rect {
	return rect{attackPrompt.x + 24, attackPrompt.y + 272 + float64(i)*36, attackPrompt.w - 48, 32}
}

func cancelAttackButton() rect {
	return rect{attackPrompt.x + attackPrompt.w - 144, attackPrompt.y + attackPrompt.h - 54, 120, 38}
}

func (a *App) clickAttackPrompt(x, y float64) {
	switch {
	case autoResolveButton().contains(x, y):
		a.resolveAuto()
	case cancelAttackButton().contains(x, y):
		a.attack = nil
		a.message = "Angriff abgebrochen."
	}
}

// resolveAuto lets the rules decide the pending fight.
func (a *App) resolveAuto() {
	u, target := a.attack.unit, a.attack.target
	a.attack = nil
	r, err := a.st.Attack(u, target)
	if err != nil {
		a.message = err.Error()
		return
	}
	a.message = fmt.Sprintf("Gefecht: eigene Verluste %d, Feind %d.", r.AttackerLoss, r.DefenderLoss)
	a.mapDirty = true
	if r.AttackerDestroyed {
		a.clearSelection()
		return
	}
	a.selectUnit(u)
}

func (a *App) drawAttackPrompt(dst *ebiten.Image) {
	r := attackPrompt
	u := a.attack.unit
	d := a.st.UnitAt(a.attack.target)
	fillRect(dst, r, colChrome)
	strokeRect(dst, r, 2, colAmber)
	a.label(dst, "Gefecht bei "+hexName(a.attack.target), r.x+24, r.y+16, 16, colAmber, true, text.AlignStart)
	a.label(dst, "Wie soll der Angriff aufgelöst werden?", r.x+24, r.y+40, 24, colText, true, text.AlignStart)

	a.drawCombatant(dst, u, r.x+24, r.y+84, "Angreifer")
	a.label(dst, "gegen", r.x+r.w/2-16, r.y+112, 15, colMuted, true, text.AlignCenter)
	a.drawCombatant(dst, d, r.x+r.w/2+24, r.y+84, "Verteidiger")

	b := autoResolveButton()
	fillRect(dst, b, colAmber)
	a.label(dst, "Automatisch auflösen", b.x+b.w/2, b.y+16, 18, colDeep, true, text.AlignCenter)

	a.label(dst, "Tactical Layer, noch nicht verfügbar", r.x+24, r.y+248, 14, colMuted, false, text.AlignStart)
	for i, name := range tacticalOptions {
		row := tacticalRow(i)
		fillRect(dst, row, colPanel)
		strokeRect(dst, row, 1, colLine)
		a.label(dst, name, row.x+12, row.y+9, 14, colDisabled, false, text.AlignStart)
	}

	c := cancelAttackButton()
	strokeRect(dst, c, 2, colMuted)
	a.label(dst, "Abbrechen", c.x+c.w/2, c.y+11, 15, colText, true, text.AlignCenter)
}

func (a *App) drawCombatant(dst *ebiten.Image, u *game.Unit, x, y float64, role string) {
	fillRect(dst, rect{x, y, 60, 76}, colPaper)
	img := a.symbol(u.Faction, u.Kind, 1.2)
	op := &ebiten.DrawImageOptions{}
	op.GeoM.Translate(x+4, y+6)
	dst.DrawImage(img, op)
	a.label(dst, role, x+72, y+2, 13, factionText(u.Faction), true, text.AlignStart)
	a.label(dst, u.Short, x+72, y+22, 17, colText, true, text.AlignStart)
	a.label(dst, fmt.Sprintf("Stärke %d", u.Strength()), x+72, y+48, 14, colMuted, false, text.AlignStart)
}
