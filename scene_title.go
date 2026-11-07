package main

import (
	"image/color"
	"math"
	"math/rand/v2"

	"github.com/hajimehoshi/ebiten/v2"
)

type cruiser struct {
	coal, L     int
	x, y, speed float64
	wake        float64
	alive       float64
}

type Title struct {
	ships []*cruiser
	fx    FX
	next  float64
	lang  *Button
}

func newTitle(g *Game) *Title {
	s := &Title{next: 1.5}
	lanes := []float64{170, 300, 430, 560, 690}
	for i, y := range lanes {
		c := &cruiser{coal: i % 2, L: 1 + (i*3)%4, y: y, speed: 18 + float64(i%3)*9}
		if c.coal == 1 {
			c.speed = -c.speed
		}
		c.x = rf(100, W-100)
		s.ships = append(s.ships, c)
	}
	s.lang = &Button{X: W - 190, Y: 26, W: 160, H: 42, Label: func() string { return tr("Русский", "English") }, Keys: []ebiten.Key{ebiten.KeyL}}
	g.snd.Music("menu")
	return s
}

func (s *Title) Update(g *Game) {
	s.fx.Update(dt)
	for _, c := range s.ships {
		c.x += c.speed * dt
		half := float64(c.L) * 52 / 2
		if c.speed > 0 && c.x-half > W+60 {
			c.x = -half - 60
		}
		if c.speed < 0 && c.x+half < -60 {
			c.x = W + half + 60
		}
		c.wake += dt
		if c.wake > 0.06 {
			c.wake = 0
			dir := math.Copysign(1, c.speed)
			s.fx.Wake(c.x-dir*half, c.y+rf(-5, 5), 7)
			s.fx.Wake(c.x-dir*half*0.3, c.y-14, 4)
			s.fx.Wake(c.x-dir*half*0.3, c.y+14, 4)
		}
	}
	s.next -= dt
	if s.next <= 0 {
		s.next = rf(2.2, 3.6)
		a := s.ships[rand.IntN(len(s.ships))]
		var b *cruiser
		for _, c := range s.ships {
			if c.coal != a.coal && (b == nil || rand.IntN(2) == 0) {
				b = c
			}
		}
		tx, ty := b.x+rf(-80, 80), b.y+rf(-50, 50)
		pan := (a.x/W)*2 - 1
		g.snd.Play("launch", pan, 0.25)
		s.fx.Launch(a.x, a.y, tx, ty, 160, 1.1, func() {
			s.fx.Splash(tx, ty, 1)
			g.snd.Play("splash", (tx/W)*2-1, 0.3)
		})
	}
	if s.lang.Update(g) {
		cfg.Lang = 1 - cfg.Lang
		g.store.Save()
		return
	}
	if g.keyJust(ebiten.KeyEscape) {
		g.quit = true
	}
	if g.click || g.keyJust(ebiten.KeyEnter) || g.keyJust(ebiten.KeySpace) || g.keyJust(ebiten.KeyNumpadEnter) {
		g.snd.Play("ping", 0, 0.6)
		g.Go(newSetup(g))
	}
}

func (s *Title) Draw(g *Game, dst *ebiten.Image) {
	drawOcean(dst, 0, 0, W, H, g.t, seaDeep, seaShallow, 260)
	s.fx.drawParticles(dst, false)
	for _, c := range s.ships {
		ang := 0.0
		if c.speed < 0 {
			ang = math.Pi
		}
		drawShipFree(dst, c.coal, c.L, c.x, c.y+math.Sin(g.t*1.3+c.x*0.01)*1.5, 52, ang, 1)
	}
	for _, m := range s.fx.missiles {
		m.draw(dst)
	}
	s.fx.drawParticles(dst, true)
	drawVignette(dst, 1)
	vgrad(dst, 0, 0, W, 300, color.NRGBA{2, 8, 18, 235}, color.NRGBA{2, 8, 18, 0})
	vgrad(dst, 0, H-220, W, 220, color.NRGBA{2, 8, 18, 0}, color.NRGBA{2, 8, 18, 230})

	title := tr("NORTH SEA CLASH", "БИТВА В СЕВЕРНОМ МОРЕ")
	size := fitSize(title, Display, 92, 1120, 6)
	drawTextGlow(dst, title, Display, size, W/2, 120, colText, colCyan, 1, 6, 1.2)
	sub := tr("NAVAL BATTLE  ·  NORTHERN ALLIANCE vs CONTINENTAL LEAGUE", "МОРСКОЙ БОЙ  ·  СЕВЕРНЫЙ АЛЬЯНС против КОНТИНЕНТАЛЬНОЙ ЛИГИ")
	drawTextOpt(dst, sub, Body, 22, W/2, 126+size*1.05, colCyan, 1, TextOpt{Spacing: 2})

	a := 0.55 + 0.45*math.Sin(g.t*3.2)
	drawTextShadow(dst, tr("PRESS ENTER OR CLICK TO START", "НАЖМИТЕ ENTER ИЛИ КЛИКНИТЕ, ЧТОБЫ НАЧАТЬ"), Display, 26, W/2, H-150, withA(colText, a), 1, 3)
	drawText(dst, tr("F1 sound · F2 music · F11 fullscreen · Esc exit", "F1 звук · F2 музыка · F11 полный экран · Esc выход"), Body, 18, W/2, H-96, colDim, 1)
	s.lang.Draw(g, dst)
}
