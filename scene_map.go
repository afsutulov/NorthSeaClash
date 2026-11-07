package main

import (
	"image/color"
	"math"
	"strings"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/vector"
)

const mapX, mapY, mapH = 40.0, 110.0, 650.0

var mapW = mapH * (mapLon1 - mapLon0) * mapKx / (mapLat1 - mapLat0)

func proj(lon, lat float64) (float64, float64) {
	return mapX + (lon-mapLon0)/(mapLon1-mapLon0)*mapW, mapY + (mapLat1-lat)/(mapLat1-mapLat0)*mapH
}

type MapScene struct {
	deploy, records, menu *Button
}

func newMap(g *Game) *MapScene {
	g.snd.Music("menu")
	return &MapScene{
		deploy:  &Button{X: W - 310, Y: H - 96, W: 260, H: 56, Primary: true, Label: func() string { return tr("DEPLOY FLEET", "РАЗВЕРНУТЬ ФЛОТ") }, Keys: []ebiten.Key{ebiten.KeyEnter, ebiten.KeyNumpadEnter}},
		records: &Button{X: 600, Y: H - 96, W: 190, H: 56, Label: func() string { return tr("RECORDS", "РЕКОРДЫ") }},
		menu:    &Button{X: 806, Y: H - 96, W: 150, H: 56, Label: func() string { return tr("MENU", "МЕНЮ") }, Keys: []ebiten.Key{ebiten.KeyEscape}},
	}
}

func (s *MapScene) Update(g *Game) {
	if campaignOver(g) {
		g.Go(newCampaignEnd(g))
		return
	}
	if s.deploy.Update(g) {
		g.Go(newPlacement(g))
	}
	if s.records.Update(g) {
		g.Go(newCard(g))
	}
	if s.menu.Update(g) {
		g.Go(newTitle(g))
	}
}

func drawSeaMap(g *Game, dst *ebiten.Image) {
	mx, my, mw, mh := float32(mapX), float32(mapY), float32(mapW), float32(mapH)
	fillRR(dst, mx-10, my-10, mw+20, mh+20, 14, color.NRGBA{2, 10, 20, 230})
	drawOcean(dst, mapX, mapY, mapW, mapH, g.t, mapSeaDeep, mapSeaShallow, 180)
	// land, grid and labels never change: draw them once into a cached layer
	layer := cached("map"+itoa(cfg.Lang), int(mapW), int(mapH), func(img *ebiten.Image) {
		m := img
		ox, oy := float32(mapX), float32(mapY)
		for lon := -4.0; lon <= 11; lon++ {
			x, _ := proj(lon, 0)
			line(m, float32(x)-ox, 0, float32(x)-ox, mh, 1, color.NRGBA{0x9f, 0xf6, 0xff, 0x16})
		}
		for lat := 51.0; lat <= 61; lat++ {
			_, y := proj(0, lat)
			line(m, 0, float32(y)-oy, mw, float32(y)-oy, 1, color.NRGBA{0x9f, 0xf6, 0xff, 0x16})
		}
		for _, poly := range landPolys {
			p := &vector.Path{}
			for i, pt := range poly {
				x, y := proj(pt[0], pt[1])
				if i == 0 {
					p.MoveTo(float32(x)-ox, float32(y)-oy)
				} else {
					p.LineTo(float32(x)-ox, float32(y)-oy)
				}
			}
			p.Close()
			strokePath(m, p, 7, color.NRGBA{0x3d, 0xe1, 0xf0, 0x1c})
			fillPath(m, p, color.NRGBA{0x18, 0x26, 0x2c, 0xff})
			strokePath(m, p, 1.6, color.NRGBA{0x7c, 0xe8, 0xf4, 0xa0})
		}
		for _, c := range countries {
			x, y := proj(c.Lon, c.Lat)
			drawTextOpt(m, strings.ToUpper(tr(c.EN, c.RU)), Body, 14, x-mapX, y-mapY, color.NRGBA{0x9f, 0xc8, 0xd8, 0x90}, 1, TextOpt{Spacing: 2.5})
		}
		x, y := proj(2.6, 56.6)
		drawTextOpt(m, tr("NORTH SEA", "СЕВЕРНОЕ МОРЕ"), Display, 22, x-mapX, y-mapY, color.NRGBA{0xbf, 0xf4, 0xff, 0x70}, 1, TextOpt{Spacing: 8})
		drawVignette(m, 0.5)
	})
	op := &ebiten.DrawImageOptions{}
	op.GeoM.Translate(mapX, mapY)
	dst.DrawImage(layer, op)
	strokeRR(dst, mx, my, mw, mh, 4, 1.5, withA(colCyan, 0.5))
}

func (s *MapScene) Draw(g *Game, dst *ebiten.Image) {
	drawBackdrop(g, dst)
	fr := campaignFront(g)
	own, enemy := fr.Staging, fr.Target
	f := g.cap.Campaign[g.coal].Front
	header(dst, tr("CAMPAIGN MAP", "КАРТА КАМПАНИИ"), "")
	drawSeaMap(g, dst)
	m := clip(dst, float32(mapX), float32(mapY), float32(mapW), float32(mapH))

	// route from staging port to target
	ox, oy := proj(own.Lon, own.Lat)
	tx, ty := proj(enemy.Lon, enemy.Lat)
	sx, sy := proj(2.6, 55.8) // open sea
	cx, cy := (ox+tx)/2-(ty-oy)*0.3, (oy+ty)/2+(tx-ox)*0.3
	if c2x, c2y := (ox+tx)/2+(ty-oy)*0.3, (oy+ty)/2-(tx-ox)*0.3; math.Hypot(c2x-sx, c2y-sy) < math.Hypot(cx-sx, cy-sy) {
		cx, cy = c2x, c2y
	}
	bez := func(t float64) (float64, float64) {
		u := 1 - t
		return u*u*ox + 2*u*t*cx + t*t*tx, u*u*oy + 2*u*t*cy + t*t*ty
	}
	acc := coalitions[g.coal].Accent
	for i := 0; i < 40; i++ {
		t := (float64(i) + math.Mod(g.t*1.5, 1)) / 40
		x, y := bez(t)
		circle(m, float32(x), float32(y), 2.2, withA(acc, 0.85))
	}
	drawPorts(g, m)
	// target crosshair
	r := 18 + 4*math.Sin(g.t*4)
	ring(m, float32(tx), float32(ty), float32(r), 2.5, colRed)
	ring(m, float32(tx), float32(ty), float32(r+10), 1, withA(colRed, 0.6))
	for k := 0; k < 4; k++ {
		a := g.t + float64(k)*math.Pi/2
		ca, sa := math.Cos(a), math.Sin(a)
		line(m, float32(tx+ca*(r+4)), float32(ty+sa*(r+4)), float32(tx+ca*(r+16)), float32(ty+sa*(r+16)), 2.5, colRed)
	}
	// our squadron
	fx, fy := bez(0.18)
	ang := math.Atan2(ty-oy, tx-ox)
	drawShipFree(m, g.coal, 4, fx, fy, 16, ang, 1)

	// right panel
	panel(dst, 600, 110, 630, 570)
	panelTitle(dst, tr("OPERATION", "ОПЕРАЦИЯ"), 630, 134, colGold)
	op := tr("Assault on "+enemy.EN, "Штурм порта "+enemy.RU)
	if fr.Liberate {
		op = tr("Liberation of "+enemy.EN, "Освобождение порта "+enemy.RU)
	}
	drawTextShadow(dst, op, Display, fitSize(op, Display, 38, 570, 0), 630, 184, colText, 0, 0)
	drawFlag(dst, enemy.Country, 632, 240, 46, 29)
	drawText(dst, tr(countries[enemy.Country].EN, countries[enemy.Country].RU)+" · "+coalitions[1-g.coal].Name(), Body, 21, 692, 241, colDim, 0)
	lossEN, lossRU := "the enemy recaptures "+fr.Loss.EN, "противник отобьёт порт "+fr.Loss.RU
	if fr.LossOwn {
		lossEN, lossRU = "the enemy takes "+fr.Loss.EN, "противник захватит порт "+fr.Loss.RU
	}
	brief := tr(
		"Your squadron sails from "+own.EN+". Destroy the enemy squadron defending "+enemy.EN+" and the port is yours. If your fleet is sunk, "+lossEN+".",
		"Ваша эскадра выходит из порта "+own.RU+". Уничтожьте эскадру противника, защищающую "+enemy.RU+", — и порт ваш. Если ваш флот погибнет, "+lossRU+".")
	if fr.Liberate {
		brief = tr(
			"The enemy holds your port "+enemy.EN+". Your squadron sails from "+own.EN+": destroy the occupying squadron and liberate the port. If your fleet is sunk, "+lossEN+".",
			"Противник удерживает ваш порт "+enemy.RU+". Эскадра выходит из порта "+own.RU+": уничтожьте эскадру оккупантов и освободите порт. Если ваш флот погибнет, "+lossRU+".")
	}
	drawText(dst, wrap(brief, Body, 21, 570), Body, 21, 630, 296, colText, 0)

	drawText(dst, tr("FRONT LINE", "ЛИНИЯ ФРОНТА"), Display, 18, 630, 432, colGold, 0)
	progress(dst, 630, 468, tr("Enemy ports taken", "Захвачено портов"), max(f, 0), acc)
	progress(dst, 630, 502, tr("Own ports lost", "Потеряно портов"), max(-f, 0), colRed)
	drawText(dst, wrap(tr("A victory moves the front towards the enemy (lost ports are liberated first), a defeat moves it back. Capture all five enemy ports to win the campaign.",
		"Победа сдвигает фронт к противнику (сначала освобождаются свои порты), поражение — назад. Захватите все пять портов противника, чтобы выиграть кампанию."), Body, 17, 570), Body, 17, 630, 538, colDim, 0)

	// legend
	ly := 614.0
	for side := 0; side < 2; side++ {
		x := 630 + float64(side)*300
		glowDot(dst, x+8, ly+12, 22, coalitions[side].Accent, 0.4)
		circle(dst, float32(x+8), float32(ly+12), 6, coalitions[side].Accent)
		drawText(dst, coalitions[side].Name(), Body, 19, x+24, ly+1, colText, 0)
	}
	ring(dst, 638, float32(ly+50), 9, 2, colRed)
	drawText(dst, tr("Target", "Цель"), Body, 18, 654, ly+39, colDim, 0)
	ring(dst, 788, float32(ly+50), 9, 2, color.NRGBA{255, 255, 255, 200})
	drawText(dst, tr("Captured port", "Захваченный порт"), Body, 18, 804, ly+39, colDim, 0)

	s.deploy.Draw(g, dst)
	s.records.Draw(g, dst)
	s.menu.Draw(g, dst)
}
