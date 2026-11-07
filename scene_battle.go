package main

import (
	"image/color"
	"math"
	"math/rand/v2"
	"sort"

	"github.com/hajimehoshi/ebiten/v2"
)

const (
	bCell       = 46.0
	bY          = 196.0
	bLX, bRX    = 78.0, 742.0
	stPlayer    = 0
	stFlying    = 1
	stEnemyWait = 2
	stEnemyFly  = 3
	stOver      = 4
)

type logLine struct {
	en, ru string
	c      color.NRGBA
}

type Battle struct {
	p1, p2         Cells // p1: player's field (the AI shoots here), p2: enemy field
	my, en         []Ship
	sunkMy, sunkEn []bool
	ai             AI
	shown          [12][12]bool // results of enemy shots already visible on the player's field
	known          [12][12]bool // enemy cells known to be empty (around sunk ships)
	fx             FX
	state          int
	timer          float64
	fires, hits    int
	log            []logLine
	win            bool
	overT          float64
	campaignDone   bool
	fr             Front
	aimR, aimC     int
	kbd            bool
	cont, retreat  *Button
	yes, no        *Button
	confirm        bool
	pulse          float64
}

func newBattle(g *Game, ships []Ship) *Battle {
	s := &Battle{p1: newCells(), p2: newCells(), my: append([]Ship(nil), ships...), aimR: 5, aimC: 5}
	for _, sh := range s.my {
		place(&s.p1, sh)
	}
	prepareField(&s.p1) // the frame is excluded from the AI's future moves
	s.en = SetPole(&s.p2)
	s.sunkMy, s.sunkEn = make([]bool, len(s.my)), make([]bool, len(s.en))
	s.fr = campaignFront(g)
	s.cont = &Button{X: W/2 - 140, Y: 598, W: 280, H: 56, Primary: true, Label: func() string { return tr("CONTINUE", "ПРОДОЛЖИТЬ") }, Keys: []ebiten.Key{ebiten.KeyEnter, ebiten.KeyNumpadEnter}}
	s.retreat = &Button{X: 568, Y: 600, W: 134, H: 50, Label: func() string { return tr("RETREAT", "ОТСТУПИТЬ") }}
	s.yes = &Button{X: W/2 - 190, Y: 440, W: 180, H: 54, Primary: true, Accent: colRed, Label: func() string { return tr("RETREAT", "ОТСТУПИТЬ") }, Keys: []ebiten.Key{ebiten.KeyY}}
	s.no = &Button{X: W/2 + 10, Y: 440, W: 180, H: 54, Label: func() string { return tr("FIGHT ON", "В БОЙ") }, Keys: []ebiten.Key{ebiten.KeyN, ebiten.KeyEscape}}
	s.addLog("Battle for "+s.fr.Target.EN+" begins. Your turn, captain!", "Начинается бой за "+s.fr.Target.RU+". Ваш ход, капитан!", colCyan)
	g.snd.Music("battle")
	return s
}

func (s *Battle) addLog(en, ru string, c color.NRGBA) {
	s.log = append(s.log, logLine{en, ru, c})
	if len(s.log) > 3 {
		s.log = s.log[len(s.log)-3:]
	}
}

func alive(ships []Ship, sunk []bool) []Ship {
	var r []Ship
	for i, sh := range ships {
		if !sunk[i] {
			r = append(r, sh)
		}
	}
	return r
}

func (s *Battle) launchFrom(ships []Ship, sunk []bool, bx float64) (float64, float64) {
	if a := alive(ships, sunk); len(a) > 0 {
		return shipCenter(a[rand.IntN(len(a))], bx, bY, bCell)
	}
	return bx + 5*bCell, bY + 5*bCell
}

// enemyLaunchPoint returns the centre of a random cell of the enemy field
// (water or ship alike), so enemy launches never reveal where its ships are.
func enemyLaunchPoint() (float64, float64) {
	x, y := cellCenter(bRX, bY, bCell, rand.IntN(10)+1, rand.IntN(10)+1)
	return x + rf(-bCell*0.2, bCell*0.2), y + rf(-bCell*0.2, bCell*0.2)
}

func pan(x float64) float64 { return x/W*2 - 1 }

// fire launches the player's missile at the enemy cell (r, c).
func (s *Battle) fire(g *Game, r, c int) {
	s.fires++
	s.state = stFlying
	x0, y0 := s.launchFrom(s.my, s.sunkMy, bLX)
	x1, y1 := cellCenter(bRX, bY, bCell, r, c)
	g.snd.Play("launch", pan(x0), 0.8)
	s.fx.Launch(x0, y0, x1, y1, 170, 0.95, func() { s.playerImpact(g, r, c, x1, y1) })
}

func (s *Battle) playerImpact(g *Game, x, y int, px, py float64) {
	co := coord(x, y)
	switch s.p2[x][y] {
	case '#':
		s.p2[x][y] = 'X'
		s.hits++
		sunk := -1
		for i, sh := range s.en {
			if !s.sunkEn[i] && isSunk(&s.p2, sh) {
				s.sunkEn[i], sunk = true, i
				for _, p := range sh.Cells() {
					for i := p[0] - 1; i <= p[0]+1; i++ {
						for j := p[1] - 1; j <= p[1]+1; j++ {
							s.known[i][j] = true
						}
					}
				}
			}
		}
		if sunk >= 0 {
			cx, cy := shipCenter(s.en[sunk], bRX, bY, bCell)
			s.fx.Explosion(px, py, 1.1)
			s.fx.Explosion(cx, cy, 1.8)
			g.Shake(12)
			g.snd.Play("sunk", pan(px), 1)
			l := s.en[sunk].L
			s.addLog("Captain: "+co+" — enemy "+shipClassEN[l]+" destroyed!", "Капитан: "+co+" — вражеский "+lower(shipClassRU[l])+" уничтожен!", colGold)
		} else {
			s.fx.Explosion(px, py, 1)
			g.Shake(6)
			g.snd.Play("explosion", pan(px), 0.9)
			s.addLog("Captain: shot at "+co+" — target hit! Fire again", "Капитан: выстрел по "+co+" — цель поражена! Стреляйте ещё", colGreen)
		}
		if Porajenie(&s.p2) {
			s.end(g, true)
			return
		}
		s.state = stPlayer
	default:
		s.p2[x][y] = 'o'
		s.fx.Splash(px, py, 1)
		g.snd.Play("splash", pan(px), 0.8)
		s.addLog("Captain: shot at "+co+" — miss. The enemy is firing", "Капитан: выстрел по "+co+" — мимо. Ход противника", colDim)
		s.state, s.timer = stEnemyWait, 1.0
		g.snd.Play("alert", 0.6, 0.5)
	}
}

func (s *Battle) enemyFire(g *Game) {
	x, y, res := s.ai.Shot(&s.p1)
	s.state = stEnemyFly
	x0, y0 := enemyLaunchPoint()
	x1, y1 := cellCenter(bLX, bY, bCell, x, y)
	g.snd.Play("launch", pan(x0), 0.7)
	s.fx.Launch(x0, y0, x1, y1, 170, 0.95, func() { s.enemyImpact(g, x, y, res, x1, y1) })
}

func (s *Battle) enemyImpact(g *Game, x, y, res int, px, py float64) {
	s.shown[x][y] = true
	co := coord(x, y)
	if res == 1 {
		sunk := -1
		for i, sh := range s.my {
			if !s.sunkMy[i] && isSunk(&s.p1, sh) {
				s.sunkMy[i], sunk = true, i
				s.ai.Sunk(sh) // the sinking is announced, the enemy knows it too
			}
		}
		if sunk >= 0 {
			cx, cy := shipCenter(s.my[sunk], bLX, bY, bCell)
			s.fx.Explosion(px, py, 1.1)
			s.fx.Explosion(cx, cy, 1.8)
			g.Shake(14)
			g.snd.Play("sunk", pan(px), 1)
			l := s.my[sunk].L
			s.addLog("Enemy: "+co+" — we have lost a "+lower(shipClassEN[l])+"!", "Противник: "+co+" — потерян наш "+lower(shipClassRU[l])+"!", colRed)
		} else {
			s.fx.Explosion(px, py, 1)
			g.Shake(8)
			g.snd.Play("explosion", pan(px), 0.9)
			s.addLog("Enemy: shot at "+co+" — our ship is hit!", "Противник: выстрел по "+co+" — наш корабль подбит!", colRed)
		}
		if Porajenie(&s.p1) {
			s.end(g, false)
			return
		}
		s.state, s.timer = stEnemyWait, 0.8
		return
	}
	s.fx.Splash(px, py, 1)
	g.snd.Play("splash", pan(px), 0.8)
	s.addLog("Enemy: shot at "+co+" — miss. Your turn", "Противник: выстрел по "+co+" — мимо. Ваш ход", colCyan)
	s.state = stPlayer
	g.snd.Play("ping", -0.4, 0.35)
}

// end finishes the battle and records the result on the captain's card.
func (s *Battle) end(g *Game, win bool) {
	s.state, s.win, s.overT = stOver, win, 0
	c := g.cap
	c.Games++
	c.Fires += s.fires
	cp := &c.Campaign[g.coal]
	if win {
		c.Wins++
		if c.Best == 0 || s.fires < c.Best {
			c.Best = s.fires
		}
		g.store.AddResult(BattleResult{Name: c.Name, Fires: s.fires, Coalition: g.coal, Port: s.fr.Target.EN, PortRU: s.fr.Target.RU})
		cp.Front = min(cp.Front+1, campaignPorts) // the front moves towards the enemy
		if cp.Front >= campaignPorts {
			c.CampaignsWon++
			s.campaignDone = true
		}
		g.snd.Play("victory", 0, 0.9)
	} else {
		cp.Front = max(cp.Front-1, -campaignPorts) // the front moves back
		if cp.Front <= -campaignPorts {
			s.campaignDone = true
		}
		g.snd.Play("defeat", 0, 0.9)
	}
	g.snd.Duck(6)
	g.store.Save()
}

func lower(s string) string {
	r := []rune(s)
	if len(r) > 0 && r[0] < 0x80 && r[0] >= 'A' && r[0] <= 'Z' {
		r[0] += 'a' - 'A'
	} else if len(r) > 0 && r[0] >= 'А' && r[0] <= 'Я' {
		r[0] += 'а' - 'А'
	}
	return string(r)
}

func (s *Battle) Update(g *Game) {
	s.fx.Update(dt)
	s.pulse += dt
	// burning hulls
	for i := 1; i <= 10; i++ {
		for j := 1; j <= 10; j++ {
			if s.p2[i][j] == 'X' {
				x, y := cellCenter(bRX, bY, bCell, i, j)
				s.fx.Burn(x, y, bCell, dt, 1)
			}
			if s.p1[i][j] == 'X' && s.shown[i][j] {
				x, y := cellCenter(bLX, bY, bCell, i, j)
				s.fx.Burn(x, y, bCell, dt, 1)
			}
		}
	}
	if s.state == stOver {
		s.overT += dt
		if s.overT > 1.6 && s.cont.Update(g) {
			if s.campaignDone {
				g.Go(newCampaignEnd(g))
			} else {
				g.Go(newCard(g))
			}
		}
		return
	}
	if s.confirm {
		if s.yes.Update(g) {
			s.confirm = false
			s.addLog("The squadron retreats.", "Эскадра отступает.", colRed)
			s.end(g, false)
		} else if s.no.Update(g) {
			s.confirm = false
		}
		return
	}
	if s.retreat.Update(g) || g.keyJust(ebiten.KeyEscape) {
		s.confirm = true
		return
	}
	switch s.state {
	case stPlayer:
		for _, k := range []struct {
			key    ebiten.Key
			dr, dc int
		}{{ebiten.KeyArrowUp, -1, 0}, {ebiten.KeyArrowDown, 1, 0}, {ebiten.KeyArrowLeft, 0, -1}, {ebiten.KeyArrowRight, 0, 1}} {
			if g.keyRep(k.key) {
				s.kbd = true
				s.aimR = min(10, max(1, s.aimR+k.dr))
				s.aimC = min(10, max(1, s.aimC+k.dc))
			}
		}
		r, c := cellAt(bRX, bY, bCell, g.mx, g.my)
		if r > 0 && !s.kbd {
			s.aimR, s.aimC = r, c
		}
		if r > 0 && g.click {
			s.kbd = false
			s.aimR, s.aimC = r, c
		}
		shoot := (g.click && r > 0) || (g.keyJust(ebiten.KeyEnter) || g.keyJust(ebiten.KeyNumpadEnter) || g.keyJust(ebiten.KeySpace))
		if shoot {
			if s.canFire(s.aimR, s.aimC) {
				s.fire(g, s.aimR, s.aimC)
			} else {
				g.snd.Play("error", 0.5, 0.6)
			}
		}
	case stEnemyWait:
		s.timer -= dt
		if s.timer <= 0 && !s.fx.Busy() {
			s.enemyFire(g)
		}
	}
}

func (s *Battle) canFire(r, c int) bool {
	return r >= 1 && r <= 10 && c >= 1 && c <= 10 && (s.p2[r][c] == ' ' || s.p2[r][c] == '#') && !s.known[r][c]
}

func (s *Battle) Draw(g *Game, dst *ebiten.Image) {
	drawBackdrop(g, dst)
	me, en := &coalitions[g.coal], &coalitions[1-g.coal]
	title := tr("BATTLE FOR ", "БОЙ ЗА ") + upper(s.fr.Target.Name())
	drawTextGlow(dst, title, Display, 34, W/2, 18, colText, colCyan, 1, 4, 0.6)
	switch {
	case s.state == stOver:
	case s.state == stPlayer || s.state == stFlying:
		a := 0.7 + 0.3*math.Sin(s.pulse*4)
		drawTextOpt(dst, tr("YOUR TURN — CHOOSE A TARGET", "ВАШ ХОД — ВЫБЕРИТЕ ЦЕЛЬ"), Display, 20, W/2, 70, withA(colCyan, a), 1, TextOpt{Spacing: 3})
	default:
		a := 0.6 + 0.4*math.Sin(s.pulse*7)
		drawTextOpt(dst, tr("ENEMY IS FIRING", "ПРОТИВНИК ВЕДЁТ ОГОНЬ"), Display, 20, W/2, 70, withA(colRed, a), 1, TextOpt{Spacing: 3})
	}
	// side titles
	sideTitle := func(x float64, co *Coalition, label string) {
		fx := float32(x)
		for _, ct := range co.Countries {
			drawFlag(dst, ct, fx, 112, 30, 19)
			fx += 36
		}
		drawText(dst, label, Display, 19, float64(fx)+6, 110, co.Accent, 0)
	}
	sideTitle(bLX-30, me, tr("YOUR FLEET", "ВАШ ФЛОТ"))
	sideTitle(bRX-30, en, tr("ENEMY: ", "ПРОТИВНИК: ")+upper(en.Name()))

	drawBoard(g, dst, bLX, bY, bCell, me.Accent)
	drawBoard(g, dst, bRX, bY, bCell, en.Accent)

	// ships
	dark := ebiten.ColorScale{}
	dark.Scale(0.32, 0.28, 0.26, 1)
	for i, sh := range s.my {
		cs := ebiten.ColorScale{}
		if s.sunkMy[i] {
			cs = dark
		}
		drawShip(dst, g.coal, sh, bLX, bY, bCell, cs, 1)
	}
	for i, sh := range s.en {
		if s.sunkEn[i] {
			drawShip(dst, 1-g.coal, sh, bRX, bY, bCell, dark, 1)
		} else if s.state == stOver && !s.win && s.overT > 0.8 {
			drawShip(dst, 1-g.coal, sh, bRX, bY, bCell, ebiten.ColorScale{}, float32(math.Min(0.75, s.overT-0.8)))
		}
	}
	// markers
	for i := 1; i <= 10; i++ {
		for j := 1; j <= 10; j++ {
			if s.p1[i][j] == 'o' && s.shown[i][j] {
				missMark(dst, bLX, i, j)
			}
			if s.p1[i][j] == 'X' && s.shown[i][j] {
				hitMark(dst, bLX, i, j, s.pulse)
			}
			switch {
			case s.p2[i][j] == 'o':
				missMark(dst, bRX, i, j)
			case s.p2[i][j] == 'X':
				hitMark(dst, bRX, i, j, s.pulse)
			case s.known[i][j]:
				x, y := cellCenter(bRX, bY, bCell, i, j)
				circle(dst, float32(x), float32(y), 2.5, withA(colDim, 0.6))
			}
		}
	}
	s.fx.Draw(dst)

	// aim reticle
	if s.state == stPlayer && !s.confirm {
		r, c := cellAt(bRX, bY, bCell, g.mx, g.my)
		if r > 0 || s.kbd {
			r, c = s.aimR, s.aimC
			x, y := cellCenter(bRX, bY, bCell, r, c)
			ok := s.canFire(r, c)
			col := colRed
			if !ok {
				col = withA(colDim, 0.8)
			}
			fillRect(dst, float32(bRX), float32(y-bCell/2), float32(bCell*10), bCell, withA(col, 0.06))
			fillRect(dst, float32(x-bCell/2), float32(bY), bCell, float32(bCell*10), withA(col, 0.06))
			k := float32(bCell/2 - 3 + 3*math.Sin(s.pulse*6))
			fx, fy := float32(x), float32(y)
			for _, d := range [][2]float32{{1, 1}, {-1, 1}, {1, -1}, {-1, -1}} {
				line(dst, fx+d[0]*k, fy+d[1]*k, fx+d[0]*(k-10), fy+d[1]*k, 3, col)
				line(dst, fx+d[0]*k, fy+d[1]*k, fx+d[0]*k, fy+d[1]*(k-10), 3, col)
			}
			ring(dst, fx, fy, 6, 1.5, col)
			line(dst, fx-12, fy, fx-4, fy, 1.5, col)
			line(dst, fx+4, fy, fx+12, fy, 1.5, col)
			line(dst, fx, fy-12, fx, fy-4, 1.5, col)
			line(dst, fx, fy+4, fx, fy+12, 1.5, col)
			drawText(dst, coord(r, c), Display, 18, x, y-bCell/2-28, col, 1)
		}
	}

	// fleet status under the boards
	s.fleetRow(dst, g.coal, s.my, s.sunkMy, bLX)
	s.fleetRow(dst, 1-g.coal, s.en, s.sunkEn, bRX)

	// centre column: statistics
	cx := W / 2.0
	stat := func(y float64, label, val string, c color.NRGBA) {
		drawText(dst, label, Body, 16, cx, y, colDim, 1)
		drawText(dst, val, Display, 32, cx, y+22, c, 1)
	}
	stat(210, tr("SHOTS", "ВЫСТРЕЛЫ"), itoa(s.fires), colText)
	stat(290, tr("HITS", "ПОПАДАНИЯ"), itoa(s.hits), colGreen)
	acc := "—"
	if s.fires > 0 {
		acc = itoa(s.hits*100/s.fires) + "%"
	}
	stat(370, tr("ACCURACY", "ТОЧНОСТЬ"), acc, colGold)
	stat(450, tr("ENEMY SHIPS", "КОРАБЛИ ВРАГА"), itoa(len(alive(s.en, s.sunkEn)))+"/10", en.Accent)
	stat(530, tr("OUR SHIPS", "НАШИ КОРАБЛИ"), itoa(len(alive(s.my, s.sunkMy)))+"/10", me.Accent)
	if s.state != stOver {
		s.retreat.Draw(g, dst)
	}

	// battle log
	panel(dst, 48, 704, W-96, 82)
	for i, l := range s.log {
		a := 0.55 + 0.45*float64(i+1)/float64(len(s.log))
		drawText(dst, tr(l.en, l.ru), Body, 19, 70, 714+float64(3-len(s.log)+i)*22, withA(l.c, a), 0)
	}

	if s.confirm {
		fillRect(dst, 0, 0, W, H, color.NRGBA{0, 4, 10, 170})
		panel(dst, W/2-260, 290, 520, 230)
		drawText(dst, tr("RETREAT FROM BATTLE?", "ОТСТУПИТЬ ИЗ БОЯ?"), Display, 28, W/2, 316, colText, 1)
		drawText(dst, tr("The battle will be counted as a defeat.", "Бой будет засчитан как поражение."), Body, 21, W/2, 372, colDim, 1)
		s.yes.Draw(g, dst)
		s.no.Draw(g, dst)
	}
	if s.state == stOver && s.overT > 1.6 {
		s.drawResult(g, dst)
	}
}

func upper(s string) string {
	r := []rune(s)
	for i, c := range r {
		switch {
		case c >= 'a' && c <= 'z':
			r[i] -= 'a' - 'A'
		case c >= 'а' && c <= 'я':
			r[i] -= 'а' - 'А'
		case c == 'ё':
			r[i] = 'Ё'
		}
	}
	return string(r)
}

func missMark(dst *ebiten.Image, bx float64, r, c int) {
	img := cached("miss", 32, 32, func(d *ebiten.Image) {
		ring(d, 16, 16, 9, 1.5, color.NRGBA{0xd8, 0xf6, 0xff, 0x90})
		circle(d, 16, 16, 3.5, color.NRGBA{0xe8, 0xfb, 0xff, 0xd0})
	})
	x, y := cellCenter(bx, bY, bCell, r, c)
	drawLayer(dst, img, x-16, y-16)
}

func hitMark(dst *ebiten.Image, bx float64, r, c int, t float64) {
	x, y := cellCenter(bx, bY, bCell, r, c)
	op := &ebiten.DrawImageOptions{Filter: ebiten.FilterLinear}
	op.GeoM.Translate(-32, -32)
	op.GeoM.Scale(bCell*0.9/64, bCell*0.9/64)
	op.GeoM.Translate(x, y)
	op.ColorScale.Scale(0.05, 0.03, 0.02, 0.85)
	dst.DrawImage(softDot, op)
	glowDot(dst, x, y, bCell*0.55, color.NRGBA{255, 110, 30, 255}, 0.55+0.2*math.Sin(t*9+float64(r*7+c)))
	img := cached("hit", 32, 32, func(d *ebiten.Image) {
		line(d, 9, 9, 23, 23, 3, color.NRGBA{255, 90, 60, 230})
		line(d, 23, 9, 9, 23, 3, color.NRGBA{255, 90, 60, 230})
	})
	drawLayer(dst, img, x-16, y-16)
}

func (s *Battle) fleetRow(dst *ebiten.Image, coal int, ships []Ship, sunk []bool, bx float64) {
	idx := make([]int, len(ships))
	for i := range idx {
		idx[i] = i
	}
	sort.SliceStable(idx, func(a, b int) bool { return ships[idx[a]].L > ships[idx[b]].L })
	x := bx
	for _, i := range idx {
		l := ships[i].L
		w := float64(l) * 17
		a := float32(1)
		if sunk[i] {
			a = 0.3
		}
		drawShipFree(dst, coal, l, x+w/2, 672, 17, 0, a)
		if sunk[i] {
			line(dst, float32(x+2), 672, float32(x+w-2), 672, 2.5, colRed)
		}
		x += w + 9
	}
}

func (s *Battle) drawResult(g *Game, dst *ebiten.Image) {
	a := math.Min(1, (s.overT-1.6)*2)
	fillRect(dst, 0, 0, W, H, withA(color.NRGBA{0, 4, 10, 190}, a))
	panel(dst, W/2-330, 150, 660, 530)
	title, c := tr("VICTORY", "ПОБЕДА"), colGold
	t, l := s.fr.Target, s.fr.Loss
	sub := tr(t.EN+" is captured!", "Порт "+t.RU+" захвачен!")
	if s.fr.Liberate {
		sub = tr(t.EN+" is liberated!", "Порт "+t.RU+" освобождён!")
	}
	if !s.win {
		title, c = tr("DEFEAT", "ПОРАЖЕНИЕ"), colRed
		sub = tr("The enemy has recaptured "+l.EN+".", "Противник отбил порт "+l.RU+".")
		if s.fr.LossOwn {
			sub = tr("The enemy has taken "+l.EN+".", "Противник захватил порт "+l.RU+".")
		}
	}
	glowDot(dst, W/2, 230, 300, c, 0.18)
	drawTextGlow(dst, title, Display, 76, W/2, 176, colText, c, 1, 8, 1.2)
	drawText(dst, sub, Display, 26, W/2, 282, c, 1)
	acc := "—"
	if s.fires > 0 {
		acc = itoa(s.hits*100/s.fires) + "%"
	}
	rows := [][2]string{
		{tr("Shots fired", "Выстрелов"), itoa(s.fires)},
		{tr("Hits", "Попаданий"), itoa(s.hits)},
		{tr("Accuracy", "Точность"), acc},
		{tr("Enemy ships destroyed", "Уничтожено кораблей врага"), itoa(10-len(alive(s.en, s.sunkEn))) + " / 10"},
		{tr("Ships lost", "Потеряно кораблей"), itoa(10-len(alive(s.my, s.sunkMy))) + " / 10"},
	}
	for i, r := range rows {
		y := 340 + float64(i)*36
		drawText(dst, r[0], Body, 22, W/2-250, y, colDim, 0)
		drawText(dst, r[1], Display, 24, W/2+250, y-2, colText, 2)
	}
	if s.campaignDone {
		msg := tr("The last enemy port has fallen — the campaign is won!", "Пал последний порт противника — кампания выиграна!")
		if !s.win {
			msg = tr("Your last port has fallen — the campaign is lost.", "Пал ваш последний порт — кампания проиграна.")
		}
		drawText(dst, msg, Display, fitSize(msg, Display, 22, 620, 0), W/2, 540, c, 1)
	} else if s.win {
		drawText(dst, tr("CONGRATULATIONS!", "ПОЗДРАВЛЯЕМ!"), Display, 22, W/2, 540, colGold, 1)
	} else {
		drawText(dst, tr("Oh! I am sorry...", "Увы, капитан..."), Display, 22, W/2, 540, colDim, 1)
	}
	s.cont.Draw(g, dst)
}
