package main

import (
	"image/color"

	"github.com/hajimehoshi/ebiten/v2"
)

// The campaign is a single front line. Campaign.Front > 0 is the number of
// enemy ports held by the player, Front < 0 the number of the player's ports
// held by the enemy. A victory moves the front one port towards the enemy
// (first liberating the player's own ports), a defeat moves it back.
// Front = +5 wins the campaign, -5 loses it.
const campaignPorts = 5

type Front struct {
	Staging  Port // port the squadron sails from
	Target   Port // port the battle is fought for (always held by the enemy)
	Liberate bool // the target is one of the player's own ports
	Loss     Port // port the enemy takes if the battle is lost
	LossOwn  bool // Loss is one of the player's own ports
}

func campaignFront(g *Game) Front {
	f := g.cap.Campaign[g.coal].Front
	own, en := coalitions[g.coal].Ports, coalitions[1-g.coal].Ports
	switch {
	case f > 0:
		return Front{Staging: en[f-1], Target: en[min(f, campaignPorts-1)], Loss: en[f-1]}
	case f == 0:
		return Front{Staging: own[0], Target: en[0], Loss: own[0], LossOwn: true}
	default:
		n := -f
		st := own[min(n, campaignPorts-1)]
		return Front{Staging: st, Target: own[n-1], Liberate: true, Loss: st, LossOwn: true}
	}
}

// portOwner returns the coalition holding port i of the given side.
func portOwner(g *Game, side, i int) int {
	f := g.cap.Campaign[g.coal].Front
	if side == g.coal {
		if i < -f {
			return 1 - g.coal
		}
		return g.coal
	}
	if i < f {
		return g.coal
	}
	return side
}

func campaignOver(g *Game) bool {
	f := g.cap.Campaign[g.coal].Front
	return f >= campaignPorts || f <= -campaignPorts
}

// drawPorts draws all ports on the campaign map in the colour of their holder.
func drawPorts(g *Game, m *ebiten.Image) {
	for side := 0; side < 2; side++ {
		for i, p := range coalitions[side].Ports {
			x, y := proj(p.Lon, p.Lat)
			owner := portOwner(g, side, i)
			c := coalitions[owner].Accent
			glowDot(m, x, y, 26, c, 0.45)
			circle(m, float32(x), float32(y), 8, color.NRGBA{2, 10, 20, 255})
			circle(m, float32(x), float32(y), 6, c)
			if owner != side {
				ring(m, float32(x), float32(y), 11, 2, color.NRGBA{255, 255, 255, 200})
			}
			al, lx := 0, x+14
			if p.LabelLeft {
				al, lx = 2, x-14
			}
			drawTextShadow(m, p.Name(), Body, 18, lx, y-12, colText, al, 0)
		}
	}
}

// CampaignEnd announces the end of the campaign; only from here a new
// campaign is started.
type CampaignEnd struct {
	won   bool
	again *Button
}

func newCampaignEnd(g *Game) *CampaignEnd {
	s := &CampaignEnd{won: g.cap.Campaign[g.coal].Front >= campaignPorts}
	s.again = &Button{X: W - 330, Y: H - 96, W: 280, H: 56, Primary: true, Label: func() string { return tr("NEW CAMPAIGN", "НОВАЯ КАМПАНИЯ") }, Keys: []ebiten.Key{ebiten.KeyEnter, ebiten.KeyNumpadEnter}}
	g.snd.Music("menu")
	if s.won {
		g.snd.Play("victory", 0, 0.9)
	} else {
		g.snd.Play("defeat", 0, 0.9)
	}
	g.snd.Duck(6)
	return s
}

func (s *CampaignEnd) Update(g *Game) {
	if s.again.Update(g) {
		g.cap.Campaign[g.coal] = Campaign{}
		g.store.Save()
		g.Go(newCard(g))
	}
}

func (s *CampaignEnd) Draw(g *Game, dst *ebiten.Image) {
	drawBackdrop(g, dst)
	me, en := &coalitions[g.coal], &coalitions[1-g.coal]
	header(dst, tr("CAMPAIGN COMPLETE", "КАМПАНИЯ ЗАВЕРШЕНА"), "")
	drawSeaMap(g, dst)
	drawPorts(g, clip(dst, float32(mapX), float32(mapY), float32(mapW), float32(mapH)))

	panel(dst, 600, 110, 630, 570)
	title, c := tr("CAMPAIGN WON", "КАМПАНИЯ ВЫИГРАНА"), colGold
	text := tr("All five enemy ports are captured and all your own ports are held. The North Sea belongs to the "+me.EN+"!",
		"Все пять портов противника захвачены, все ваши порты удержаны. Северное море принадлежит коалиции «"+me.RU+"»!")
	if !s.won {
		title, c = tr("CAMPAIGN LOST", "КАМПАНИЯ ПРОИГРАНА"), colRed
		text = tr("All your ports have fallen. The "+en.EN+" now controls the North Sea.",
			"Все ваши порты пали. Северное море теперь контролирует коалиция «"+en.RU+"».")
	}
	glowDot(dst, 915, 230, 320, c, 0.16)
	drawTextGlow(dst, title, Display, fitSize(title, Display, 54, 570, 4), 915, 150, colText, c, 1, 4, 1.2)
	fx := float32(915 - len(me.Countries)*33)
	for _, ct := range me.Countries {
		drawFlag(dst, ct, fx, 240, 54, 34)
		fx += 66
	}
	drawText(dst, wrap(text, Body, 23, 560), Body, 23, 630, 310, colText, 0)
	cp := g.cap
	rows := [][2]string{
		{tr("Captain", "Капитан"), cp.Name},
		{tr("Battles fought", "Проведено боёв"), itoa(cp.Games)},
		{tr("Victories", "Побед"), itoa(cp.Wins)},
		{tr("Campaigns won", "Выиграно кампаний"), itoa(cp.CampaignsWon)},
	}
	for i, r := range rows {
		y := 470 + float64(i)*42
		drawText(dst, r[0], Body, 22, 640, y, colDim, 0)
		drawText(dst, r[1], Display, 24, 1190, y-2, colText, 2)
	}
	s.again.Draw(g, dst)
}
