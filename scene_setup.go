package main

import (
	"image/color"
	"math"
	"strings"
	"unicode"

	"github.com/hajimehoshi/ebiten/v2"
)

type Setup struct {
	name           []rune
	sel            int
	cont, back     *Button
	langEN, langRU *Button
	sound, music   *Button
	cards          [2][4]float64
}

func newSetup(g *Game) *Setup {
	s := &Setup{sel: g.coal}
	name := cfg.LastCaptain
	if name == "" {
		name = tr("Captain", "Капитан")
	}
	s.name = []rune(name)
	s.cont = &Button{X: W - 290, Y: H - 96, W: 240, H: 56, Primary: true, Label: func() string { return tr("TO THE FLEET", "К ФЛОТУ") }}
	s.back = &Button{X: 50, Y: H - 96, W: 160, H: 56, Label: func() string { return tr("BACK", "НАЗАД") }, Keys: []ebiten.Key{ebiten.KeyEscape}}
	s.langEN = &Button{X: 80, Y: 380, W: 140, H: 44, Label: func() string { return "English" }}
	s.langRU = &Button{X: 236, Y: 380, W: 140, H: 44, Label: func() string { return "Русский" }}
	s.sound = &Button{X: 80, Y: 500, W: 296, H: 44, Label: func() string { return tr("Sound effects: ", "Звуковые эффекты: ") + onOff(cfg.Sound) }}
	s.music = &Button{X: 80, Y: 556, W: 296, H: 44, Label: func() string { return tr("Music: ", "Музыка: ") + onOff(cfg.Music) }}
	for i := 0; i < 2; i++ {
		s.cards[i] = [4]float64{460 + float64(i)*400, 120, 380, 560}
	}
	g.snd.Music("menu")
	return s
}

func onOff(b bool) string {
	if b {
		return tr("ON", "ВКЛ")
	}
	return tr("OFF", "ВЫКЛ")
}

func (s *Setup) Update(g *Game) {
	for _, r := range ebiten.AppendInputChars(nil) {
		if len(s.name) < 16 && (unicode.IsLetter(r) || unicode.IsDigit(r) || strings.ContainsRune(" -.'", r)) {
			s.name = append(s.name, r)
		}
	}
	if g.keyRep(ebiten.KeyBackspace) && len(s.name) > 0 {
		s.name = s.name[:len(s.name)-1]
	}
	if g.keyJust(ebiten.KeyArrowLeft) || g.keyJust(ebiten.KeyArrowRight) || g.keyJust(ebiten.KeyTab) {
		s.sel = 1 - s.sel
		g.snd.Play("click", 0, 0.6)
	}
	for i, c := range s.cards {
		if g.click && g.in(c[0], c[1], c[2], c[3]) {
			if s.sel != i {
				g.snd.Play("place", 0, 0.8)
			}
			s.sel = i
		}
	}
	if s.langEN.Update(g) {
		cfg.Lang = 0
	}
	if s.langRU.Update(g) {
		cfg.Lang = 1
	}
	if s.sound.Update(g) {
		cfg.Sound = !cfg.Sound
	}
	if s.music.Update(g) {
		cfg.Music = !cfg.Music
	}
	if s.back.Update(g) {
		g.Go(newTitle(g))
		return
	}
	name := strings.TrimSpace(string(s.name))
	s.cont.Disabled = name == ""
	if s.cont.Update(g) || (!s.cont.Disabled && (g.keyJust(ebiten.KeyEnter) || g.keyJust(ebiten.KeyNumpadEnter))) {
		g.cap = g.store.Captain(name)
		g.coal = s.sel
		cfg.LastCaptain, cfg.LastCoalition = g.cap.Name, s.sel
		g.store.Save()
		g.snd.Play("ping", 0, 0.6)
		g.Go(newCard(g))
	}
}

func drawBackdrop(g *Game, dst *ebiten.Image) {
	drawOcean(dst, 0, 0, W, H, g.t, seaDeep, seaShallow, 260)
	fillRect(dst, 0, 0, W, H, color.NRGBA{2, 8, 18, 120})
	drawVignette(dst, 1)
}

func header(dst *ebiten.Image, title, sub string) {
	drawTextGlow(dst, title, Display, 40, 50, 28, colText, colCyan, 0, 4, 0.7)
	if sub != "" {
		drawText(dst, sub, Body, 20, 52, 80, colDim, 0)
	}
}

func (s *Setup) Draw(g *Game, dst *ebiten.Image) {
	drawBackdrop(g, dst)
	header(dst, tr("FLEET COMMAND", "КОМАНДОВАНИЕ ФЛОТОМ"), tr("Introduce yourself, captain, and choose the side you will fight for", "Представьтесь, капитан, и выберите сторону, за которую будете сражаться"))

	panel(dst, 50, 120, 370, 560)
	panelTitle(dst, tr("CAPTAIN", "КАПИТАН"), 80, 146, colCyan)
	drawText(dst, tr("Name (letters, digits):", "Имя (буквы, цифры):"), Body, 18, 80, 196, colDim, 0)
	fillRR(dst, 80, 224, 296, 54, 10, color.NRGBA{0, 0, 0, 120})
	strokeRR(dst, 80, 224, 296, 54, 10, 1.5, withA(colCyan, 0.7))
	nm := string(s.name)
	drawText(dst, nm, Display, 26, 96, 236, colText, 0)
	if int(g.t*2)%2 == 0 {
		cx := float32(98 + measure(nm, Display, 26, 0))
		line(dst, cx, 236, cx, 268, 2, colCyan)
	}
	drawText(dst, tr("Backspace — erase", "Backspace — стереть"), Body, 16, 80, 290, withA(colDim, 0.8), 0)
	panelTitle(dst, tr("LANGUAGE", "ЯЗЫК"), 80, 334, colCyan)
	for i, b := range []*Button{s.langEN, s.langRU} {
		b.Accent = colCyan
		b.Primary = cfg.Lang == i
		b.Draw(g, dst)
	}
	panelTitle(dst, tr("AUDIO", "ЗВУК"), 80, 456, colCyan)
	s.sound.Draw(g, dst)
	s.music.Draw(g, dst)
	drawText(dst, tr("Defaults: Northern Alliance,\nEnglish, sound and music on.", "По умолчанию: Северный альянс,\nанглийский, звук и музыка включены."), Body, 16, 80, 616, withA(colDim, 0.8), 0)

	for i := range s.cards {
		s.drawCard(g, dst, i)
	}
	drawText(dst, tr("Arrow keys or click to choose a side", "Стрелки или клик — выбор стороны"), Body, 18, 700, H-80, colDim, 1)
	s.back.Draw(g, dst)
	s.cont.Draw(g, dst)
}

func (s *Setup) drawCard(g *Game, dst *ebiten.Image, i int) {
	c := s.cards[i]
	co := &coalitions[i]
	x, y, w, h := float32(c[0]), float32(c[1]), float32(c[2]), float32(c[3])
	sel := s.sel == i
	if sel {
		glowDot(dst, c[0]+c[2]/2, c[1]+c[3]/2, c[2]*0.9, co.Accent, 0.16+0.05*math.Sin(g.t*3))
	}
	panel(dst, x, y, w, h)
	if sel {
		strokeRR(dst, x, y, w, h, 16, 3, co.Accent)
	}
	// flags
	fx := c[0] + 28
	for _, ct := range co.Countries {
		drawFlag(dst, ct, float32(fx), y+26, 54, 34)
		fx += 66
	}
	drawText(dst, co.Name(), Display, fitSize(co.Name(), Display, 30, c[2]-50, 0), c[0]+28, c[1]+76, colText, 0)
	drawText(dst, tr(co.MottoEN, co.MottoRU), Body, 18, c[0]+28, c[1]+118, co.Accent, 0)
	var names []string
	for _, ct := range co.Countries {
		names = append(names, tr(countries[ct].EN, countries[ct].RU))
	}
	drawText(dst, strings.Join(names, " · "), Body, 17, c[0]+28, c[1]+146, colDim, 0)

	// fleet preview on water
	drawOcean(dst, c[0]+20, c[1]+184, c[2]-40, 110, g.t, boardDeep, boardShallow, 120)
	drawShipFree(dst, i, 4, c[0]+c[2]/2, c[1]+222, 46, 0, 1)
	drawShipFree(dst, i, 2, c[0]+c[2]/2-80, c[1]+268, 34, 0, 1)
	drawShipFree(dst, i, 3, c[0]+c[2]/2+70, c[1]+268, 34, 0, 1)
	strokeRect(dst, float32(c[0]+20), float32(c[1]+184), float32(c[2]-40), 110, 1, withA(co.Accent, 0.5))

	drawText(dst, tr("HOME PORTS", "ПОРТЫ БАЗИРОВАНИЯ"), Display, 17, c[0]+28, c[1]+312, co.Accent, 0)
	for k, p := range co.Ports {
		yy := c[1] + 346 + float64(k)*36
		drawFlag(dst, p.Country, x+28, float32(yy+3), 30, 19)
		drawText(dst, p.Name(), Body, 21, c[0]+70, yy-2, colText, 0)
		drawText(dst, tr(countries[p.Country].EN, countries[p.Country].RU), Body, 16, c[0]+c[2]-28, yy+2, colDim, 2)
	}
	if sel {
		drawText(dst, tr("• SELECTED", "• ВЫБРАНО"), Display, 16, c[0]+c[2]-26, c[1]+30, co.Accent, 2)
	}
}
