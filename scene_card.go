package main

import (
	"fmt"
	"image/color"
	"strings"

	"github.com/hajimehoshi/ebiten/v2"
)

// Card is the captain's card and the records tables
// ("SEA BATTLE DATABASE").
type Card struct {
	cont, change, exit *Button
}

func newCard(g *Game) *Card {
	g.snd.Music("menu")
	return &Card{
		cont:   &Button{X: W - 330, Y: H - 96, W: 280, H: 56, Primary: true, Label: func() string { return tr("CAMPAIGN MAP", "КАРТА КАМПАНИИ") }, Keys: []ebiten.Key{ebiten.KeyEnter, ebiten.KeyNumpadEnter}},
		change: &Button{X: 50, Y: H - 96, W: 250, H: 56, Label: func() string { return tr("CHANGE CAPTAIN", "СМЕНИТЬ КАПИТАНА") }},
		exit:   &Button{X: 316, Y: H - 96, W: 170, H: 56, Label: func() string { return tr("EXIT", "ВЫХОД") }},
	}
}

func (s *Card) Update(g *Game) {
	if s.cont.Update(g) {
		g.Go(newMap(g))
	}
	if s.change.Update(g) {
		g.Go(newSetup(g))
	}
	if s.exit.Update(g) {
		g.quit = true
	}
	if g.keyJust(ebiten.KeyEscape) {
		g.Go(newTitle(g))
	}
}

func pct(a, b int) string {
	if b == 0 {
		return "—"
	}
	return fmt.Sprintf("%d%%", a*100/b)
}

func (s *Card) Draw(g *Game, dst *ebiten.Image) {
	drawBackdrop(g, dst)
	header(dst, tr("NAVAL DATABASE", "МОРСКАЯ БАЗА ДАННЫХ"), tr("Current card of the captain and the best results of the fleet", "Текущая карточка капитана и лучшие результаты флота"))
	c := g.cap
	co := &coalitions[g.coal]

	// captain card
	panel(dst, 50, 120, 520, 560)
	glowDot(dst, 310, 210, 260, co.Accent, 0.08)
	drawText(dst, tr("CAPTAIN", "КАПИТАН"), Display, 18, 80, 146, co.Accent, 0)
	drawTextShadow(dst, c.Name, Display, fitSize(c.Name, Display, 42, 450, 0), 80, 172, colText, 0, 0)
	fx := float32(80)
	for _, ct := range co.Countries {
		drawFlag(dst, ct, fx, 236, 40, 25)
		fx += 50
	}
	drawText(dst, co.Name(), Body, 21, float64(fx)+8, 236, colText, 0)

	cp := c.Campaign[g.coal]
	rows := [][2]string{
		{tr("Battles", "Сражений"), itoa(c.Games)},
		{tr("Victories", "Побед"), itoa(c.Wins)},
		{tr("Win rate", "Процент побед"), pct(c.Wins, c.Games)},
		{tr("Shots fired", "Выстрелов"), itoa(c.Fires)},
		{tr("Best battle (fewest shots)", "Лучший бой (меньше выстрелов)"), bestStr(c.Best)},
		{tr("Campaigns won", "Выиграно кампаний"), itoa(c.CampaignsWon)},
	}
	for i, r := range rows {
		y := 290 + float64(i)*44
		if i%2 == 0 {
			fillRR(dst, 66, float32(y-6), 488, 40, 8, color.NRGBA{255, 255, 255, 8})
		}
		drawText(dst, r[0], Body, 21, 82, y, colDim, 0)
		drawText(dst, r[1], Display, 24, 536, y-2, colText, 2)
	}
	drawText(dst, tr("CURRENT CAMPAIGN", "ТЕКУЩАЯ КАМПАНИЯ"), Display, 18, 80, 568, co.Accent, 0)
	progress(dst, 80, 604, tr("Enemy ports taken", "Захвачено портов"), max(cp.Front, 0), co.Accent)
	progress(dst, 80, 638, tr("Own ports lost", "Потеряно портов"), max(-cp.Front, 0), colRed)

	// top captains
	panel(dst, 600, 120, 630, 270)
	panelTitle(dst, tr("6 BEST CAPTAINS — VICTORIES", "6 ЛУЧШИХ КАПИТАНОВ — ПОБЕДЫ"), 630, 140, colGold)
	drawText(dst, tr("CAPTAIN", "КАПИТАН"), Body, 16, 680, 186, colDim, 0)
	drawText(dst, tr("WINS", "ПОБЕДЫ"), Body, 16, 1060, 186, colDim, 2)
	drawText(dst, tr("BATTLES", "БОИ"), Body, 16, 1196, 186, colDim, 2)
	top := g.store.TopCaptains()
	for i := 0; i < 6; i++ {
		y := 212 + float64(i)*28
		drawText(dst, itoa(i+1)+".", Display, 18, 664, y, colDim, 2)
		if i < len(top) {
			cl := colText
			if top[i] == c {
				cl = co.Accent
			}
			drawText(dst, top[i].Name, Body, 21, 680, y-2, cl, 0)
			drawText(dst, itoa(top[i].Wins), Display, 20, 1060, y-2, cl, 2)
			drawText(dst, itoa(top[i].Games), Display, 20, 1196, y-2, cl, 2)
		} else {
			drawText(dst, "· · ·", Body, 20, 680, y-2, withA(colDim, 0.4), 0)
		}
	}

	// best battles
	panel(dst, 600, 410, 630, 270)
	panelTitle(dst, tr("6 BEST BATTLES — FEWEST SHOTS", "6 ЛУЧШИХ БОЁВ — МЕНЬШЕ ВЫСТРЕЛОВ"), 630, 430, colGold)
	drawText(dst, tr("CAPTAIN", "КАПИТАН"), Body, 16, 680, 476, colDim, 0)
	drawText(dst, tr("PORT", "ПОРТ"), Body, 16, 900, 476, colDim, 0)
	drawText(dst, tr("SHOTS", "ВЫСТРЕЛЫ"), Body, 16, 1196, 476, colDim, 2)
	for i := 0; i < 6; i++ {
		y := 502 + float64(i)*28
		drawText(dst, itoa(i+1)+".", Display, 18, 664, y, colDim, 2)
		if i < len(g.store.Best) {
			r := g.store.Best[i]
			cl := colText
			if strings.EqualFold(r.Name, c.Name) {
				cl = coalitions[r.Coalition].Accent
			}
			drawText(dst, r.Name, Body, 21, 680, y-2, cl, 0)
			drawText(dst, tr(r.Port, r.PortRU), Body, 19, 900, y, colDim, 0)
			drawText(dst, itoa(r.Fires), Display, 20, 1196, y-2, cl, 2)
		} else {
			drawText(dst, "· · ·", Body, 20, 680, y-2, withA(colDim, 0.4), 0)
		}
	}
	s.cont.Draw(g, dst)
	s.change.Draw(g, dst)
	s.exit.Draw(g, dst)
}

func bestStr(n int) string {
	if n == 0 {
		return "—"
	}
	return itoa(n)
}

func progress(dst *ebiten.Image, x, y float64, label string, n int, c color.NRGBA) {
	drawText(dst, label, Body, 19, x, y, colDim, 0)
	for i := 0; i < 5; i++ {
		px := float32(x + 270 + float64(i)*38)
		if i < n {
			fillRR(dst, px, float32(y+2), 30, 20, 5, c)
		} else {
			strokeRR(dst, px, float32(y+2), 30, 20, 5, 1.5, withA(colDim, 0.5))
		}
	}
}
