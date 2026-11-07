package main

import (
	"image/color"
	"math"

	"github.com/hajimehoshi/ebiten/v2"
)

const plX, plY, plCell = 112.0, 178.0, 52.0

type Placement struct {
	ships                      []Ship
	sel                        int
	vert                       bool
	kr, kc                     int
	kbd                        bool
	ghost                      Ship
	ghostOn, ghostOK           bool
	rotate, auto, clear, start *Button
	back                       *Button
	t                          float64
}

func newPlacement(g *Game) *Placement {
	s := &Placement{sel: 4, kr: 1, kc: 1}
	s.rotate = &Button{X: 700, Y: 528, W: 170, H: 50, Label: func() string { return tr("ROTATE", "ПОВОРОТ") }}
	s.auto = &Button{X: 880, Y: 528, W: 170, H: 50, Label: func() string { return tr("AUTO", "АВТО") }, Keys: []ebiten.Key{ebiten.KeyA}}
	s.clear = &Button{X: 1060, Y: 528, W: 160, H: 50, Label: func() string { return tr("CLEAR", "ОЧИСТИТЬ") }, Keys: []ebiten.Key{ebiten.KeyDelete}}
	s.start = &Button{X: W - 330, Y: H - 96, W: 280, H: 56, Primary: true, Label: func() string { return tr("START BATTLE", "В БОЙ!") }}
	s.back = &Button{X: 700, Y: H - 96, W: 160, H: 56, Label: func() string { return tr("MAP", "КАРТА") }, Keys: []ebiten.Key{ebiten.KeyEscape}}
	g.snd.Music("battle")
	return s
}

func (s *Placement) remaining(l int) int {
	n := 0
	for _, x := range fleetPlan {
		if x == l {
			n++
		}
	}
	for _, sh := range s.ships {
		if sh.L == l {
			n--
		}
	}
	return n
}

func (s *Placement) cells() Cells {
	c := newCells()
	for _, sh := range s.ships {
		place(&c, sh)
	}
	return c
}

func (s *Placement) nextSel() {
	if s.sel > 0 && s.remaining(s.sel) > 0 {
		return
	}
	s.sel = 0
	for l := 4; l >= 1; l-- {
		if s.remaining(l) > 0 {
			s.sel = l
			return
		}
	}
}

func (s *Placement) Update(g *Game) {
	s.t += dt
	if s.rotate.Update(g) || g.rclick || g.keyJust(ebiten.KeyR) || g.keyJust(ebiten.KeySpace) || g.keyJust(ebiten.KeyInsert) || g.keyJust(ebiten.KeyPageDown) {
		s.vert = !s.vert
		g.snd.Play("click", 0, 0.5)
	}
	if s.auto.Update(g) {
		c := newCells()
		s.ships = SetPole(&c)
		s.sel = 0
		g.snd.Play("place", 0, 0.9)
	}
	if s.clear.Update(g) {
		s.ships, s.sel = nil, 4
	}
	if g.keyJust(ebiten.KeyBackspace) && len(s.ships) > 0 {
		last := s.ships[len(s.ships)-1]
		s.ships = s.ships[:len(s.ships)-1]
		s.sel, s.vert = last.L, last.V
	}
	if s.back.Update(g) {
		g.Go(newMap(g))
		return
	}
	s.start.Disabled = len(s.ships) < len(fleetPlan)
	if s.start.Update(g) || (!s.start.Disabled && g.keyJust(ebiten.KeyEnter)) {
		g.snd.Play("alert", 0, 0.8)
		g.Go(newBattle(g, s.ships))
		return
	}
	// roster selection
	for i, l := range []int{4, 3, 2, 1} {
		if g.click && g.in(700, 214+float64(i)*72, 520, 66) && s.remaining(l) > 0 {
			s.sel = l
			g.snd.Play("click", 0, 0.5)
		}
	}
	// keyboard cursor
	for _, k := range []struct {
		key    ebiten.Key
		dr, dc int
	}{{ebiten.KeyArrowUp, -1, 0}, {ebiten.KeyArrowDown, 1, 0}, {ebiten.KeyArrowLeft, 0, -1}, {ebiten.KeyArrowRight, 0, 1}} {
		if g.keyRep(k.key) {
			s.kbd = true
			s.kr = min(10, max(1, s.kr+k.dr))
			s.kc = min(10, max(1, s.kc+k.dc))
		}
	}
	r, c := cellAt(plX, plY, plCell, g.mx, g.my)
	if r > 0 && (g.click || (!s.kbd)) {
		s.kbd = false
	}
	if s.kbd {
		r, c = s.kr, s.kc
	} else if r > 0 {
		s.kr, s.kc = r, c
	}
	s.nextSel()
	s.ghostOn = r > 0 && s.sel > 0
	if r > 0 && s.sel > 0 {
		if s.vert {
			r = min(r, 11-s.sel)
		} else {
			c = min(c, 11-s.sel)
		}
		s.ghost = Ship{R: r, C: c, L: s.sel, V: s.vert}
		cl := s.cells()
		s.ghostOK = canPlace(&cl, s.ghost)
	}
	place := (g.click && !s.kbd && r > 0) || (s.kbd && (g.keyJust(ebiten.KeyEnter) || g.keyJust(ebiten.KeyNumpadEnter)))
	if place {
		pr, pc := s.kr, s.kc
		// pick up an already placed ship
		for i, sh := range s.ships {
			if sh.Has(pr, pc) {
				s.ships = append(s.ships[:i], s.ships[i+1:]...)
				s.sel, s.vert = sh.L, sh.V
				g.snd.Play("click", 0, 0.6)
				return
			}
		}
		if s.ghostOn {
			if s.ghostOK {
				s.ships = append(s.ships, s.ghost)
				g.snd.Play("place", (float64(s.ghost.C)-5)/10, 0.9)
				s.nextSel()
			} else {
				g.snd.Play("error", 0, 0.7)
			}
		}
	}
}

func (s *Placement) Draw(g *Game, dst *ebiten.Image) {
	drawBackdrop(g, dst)
	fr := campaignFront(g)
	enemy := fr.Target
	co := &coalitions[g.coal]
	header(dst, tr("DEPLOY YOUR FLEET", "РАССТАНОВКА ФЛОТА"), map[bool]string{false: tr("Operation: assault on ", "Операция: штурм порта "), true: tr("Operation: liberation of ", "Операция: освобождение порта ")}[fr.Liberate]+enemy.Name()+tr(". Ships may not touch each other, not even at the corners.", ". Корабли не должны касаться друг друга, даже углами."))
	drawBoard(g, dst, plX, plY, plCell, co.Accent)
	for _, sh := range s.ships {
		drawShip(dst, g.coal, sh, plX, plY, plCell, ebiten.ColorScale{}, 1)
	}
	if s.ghostOn {
		c := colGreen
		if !s.ghostOK {
			c = colRed
		}
		for _, p := range s.ghost.Cells() {
			fillRect(dst, float32(plX+float64(p[1]-1)*plCell+2), float32(plY+float64(p[0]-1)*plCell+2), plCell-4, plCell-4, withA(c, 0.22))
		}
		drawShip(dst, g.coal, s.ghost, plX, plY, plCell, ebiten.ColorScale{}, 0.75)
		x0 := plX + float64(s.ghost.C-1)*plCell
		y0 := plY + float64(s.ghost.R-1)*plCell
		w, h := plCell*float64(s.ghost.L), plCell
		if s.ghost.V {
			w, h = h, w
		}
		strokeRR(dst, float32(x0+1), float32(y0+1), float32(w-2), float32(h-2), 6, 2, withA(c, 0.6+0.4*math.Sin(s.t*8)))
	}

	panel(dst, 680, 130, 560, 470)
	panelTitle(dst, tr("FLEET", "ФЛОТ"), 708, 152, co.Accent)
	drawText(dst, itoa(len(s.ships))+" / 10", Display, 22, 1210, 150, colText, 2)
	for i, l := range []int{4, 3, 2, 1} {
		y := 214 + float64(i)*72
		rem := s.remaining(l)
		active := s.sel == l
		bg := color.NRGBA{255, 255, 255, 8}
		if active {
			bg = withA(co.Accent, 0.18)
		}
		fillRR(dst, 700, float32(y), 520, 64, 10, bg)
		if active {
			strokeRR(dst, 700, float32(y), 520, 64, 10, 2, co.Accent)
		}
		a := float32(1)
		if rem == 0 {
			a = 0.3
		}
		drawShipFree(dst, g.coal, l, 720+float64(l)*40/2+10, y+32, 40, 0, a)
		cl := colText
		if rem == 0 {
			cl = withA(colDim, 0.6)
		}
		drawText(dst, shipClass(l), Display, 20, 900, y+10, cl, 0)
		drawText(dst, itoa(l)+tr(map[bool]string{true: " cell", false: " cells"}[l == 1], map[bool]string{true: " клетка", false: " клетки"}[l == 1]), Body, 17, 900, y+36, colDim, 0)
		drawText(dst, "× "+itoa(rem), Display, 26, 1200, y+16, cl, 2)
	}
	s.rotate.Draw(g, dst)
	s.auto.Draw(g, dst)
	s.clear.Draw(g, dst)

	help := tr("Mouse: click — place / pick up, right click — rotate\nKeys: arrows — move, Enter — place, R / Space / Ins — rotate,\nA — auto, Backspace — undo, Del — clear",
		"Мышь: клик — поставить / поднять, правый клик — повернуть\nКлавиши: стрелки — двигать, Enter — поставить, R / Пробел / Ins —\nповорот, A — авто, Backspace — отменить, Del — очистить")
	drawText(dst, help, Body, 17, 700, 616, colDim, 0)
	s.back.Draw(g, dst)
	s.start.Draw(g, dst)
}
