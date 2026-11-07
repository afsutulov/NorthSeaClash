//go:build shots

// Screenshot tool: go build -tags shots, then run with NSC_SHOTS=<dir>.
// It walks through all screens with demo data and saves PNG files.
package main

import (
	"image"
	"image/png"
	"math/rand/v2"
	"os"
	"path/filepath"

	"github.com/hajimehoshi/ebiten/v2"
)

func init() {
	setupShots = func(g *Game) {
		dir := os.Getenv("NSC_SHOTS")
		if dir == "" {
			return
		}
		os.MkdirAll(dir, 0o755)
		frame := 0
		mark, tmark := -1, 0.0
		step := 0
		var battle *Battle
		save := func(img *ebiten.Image, name string) {
			b := img.Bounds()
			pix := make([]byte, 4*b.Dx()*b.Dy())
			img.ReadPixels(pix)
			rgba := &image.RGBA{Pix: pix, Stride: 4 * b.Dx(), Rect: image.Rect(0, 0, b.Dx(), b.Dy())}
			f, _ := os.Create(filepath.Join(dir, name))
			png.Encode(f, rgba)
			f.Close()
		}
		set := func(s Scene) { g.scene, g.next, g.fade, g.fading = s, nil, 0, false; frame = 0; step++ }
		demo := func() {
			for _, d := range []struct {
				n        string
				w, gm, f int
			}{{"Nelson", 7, 9, 412}, {"Tromp", 5, 8, 390}, {"De Ruyter", 4, 6, 301}, {"Jellicoe", 3, 6, 340}, {"Scheer", 2, 5, 288}} {
				c := g.store.Captain(d.n)
				c.Wins, c.Games, c.Fires, c.Best = d.w, d.gm, d.f, 38+d.w
			}
			for i, r := range []BattleResult{{Name: "Nelson", Fires: 41, Coalition: 0, Port: "Den Helder", PortRU: "Ден-Хелдер"},
				{Name: "Tromp", Fires: 44, Coalition: 1, Port: "Aberdeen", PortRU: "Абердин"},
				{Name: "Nelson", Fires: 47, Coalition: 0, Port: "Rotterdam", PortRU: "Роттердам"},
				{Name: "De Ruyter", Fires: 52, Coalition: 1, Port: "Hull", PortRU: "Халл"},
				{Name: "Scheer", Fires: 55, Coalition: 1, Port: "Rosyth", PortRU: "Росайт"}} {
				_ = i
				g.store.AddResult(r)
			}
			g.cap = g.store.Captain("Nelson")
			g.cap.Campaign[0] = Campaign{Front: 2}
			g.cap.CampaignsWon = 1
			g.coal = 0
		}
		var midBattle func() *Battle
		midBattle = func() *Battle {
			c := newCells()
			snd := cfg.Sound
			cfg.Sound = false
			defer func() { cfg.Sound = snd }()
			s := newBattle(g, SetPole(&c))
			for n := 0; n < 30; n++ {
				r, cc := rand.IntN(10)+1, rand.IntN(10)+1
				if !s.canFire(r, cc) {
					continue
				}
				s.fires++
				x, y := cellCenter(bRX, bY, bCell, r, cc)
				s.playerImpact(g, r, cc, x, y)
				if s.state == stEnemyWait {
					x, y, res := s.ai.Shot(&s.p1)
					px, py := cellCenter(bLX, bY, bCell, x, y)
					s.enemyImpact(g, x, y, res, px, py)
				}
				if s.state == stOver {
					return midBattle()
				}
			}
			s.state = stPlayer
			s.fx = FX{}
			return s
		}
		if os.Getenv("NSC_FROM") == "battle" {
			demo()
			step = 4
		}
		g.shot = func(g *Game, img *ebiten.Image) {
			frame++
			switch step {
			case 0:
				if frame == 260 {
					save(img, "01_title.png")
					set(newSetup(g))
				}
			case 1:
				if frame == 40 {
					save(img, "02_setup.png")
					demo()
					set(newCard(g))
				}
			case 2:
				if frame == 40 {
					save(img, "03_records.png")
					set(newMap(g))
				}
			case 3:
				if frame == 40 {
					save(img, "04_campaign_map.png")
					p := newPlacement(g)
					c := newCells()
					p.ships = SetPole(&c)[:7]
					set(p)
				}
			case 4:
				if frame == 40 {
					p, ok := g.scene.(*Placement)
					if !ok {
						p = newPlacement(g)
						g.scene = p
					}
					p.sel, p.vert = 1, false
					cl := p.cells()
					for r := 1; r <= 10 && !p.ghostOn; r++ {
						for c := 1; c <= 10; c++ {
							if canPlace(&cl, Ship{R: r, C: c, L: 1}) {
								p.ghost, p.ghostOn, p.ghostOK = Ship{R: r, C: c, L: 1}, true, true
								break
							}
						}
					}
					p.Draw(g, img)
					save(img, "05_deployment.png")
					battle = midBattle()
					set(battle)
				}
			case 5:
				if frame == 30 {
					for i, sh := range battle.en {
						p := sh.Cells()[0]
						if !battle.sunkEn[i] && battle.p2[p[0]][p[1]] == '#' {
							battle.fire(g, p[0], p[1])
							break
						}
					}
					mark = 0
				}
				if frame > 30 && mark == 0 && len(battle.fx.missiles) > 0 && battle.fx.missiles[0].t > 0.55 {
					save(img, "06_battle_missile.png")
					mark = 1
				}
				if mark == 1 && len(battle.fx.missiles) == 0 {
					mark, tmark = 2, g.t
				}
				if mark == 2 && g.t > tmark+0.12 {
					battle.kbd, battle.aimR, battle.aimC = true, 6, 4
					battle.Draw(g, img)
					save(img, "07_battle_explosion.png")
					mark = 3
					battle.end(g, true)
				}
				if mark == 3 && battle.overT > 2.6 {
					save(img, "08_victory.png")
					cfg.Lang = 1
					set(newSetup(g))
				}
			case 6:
				if frame == 40 {
					save(img, "09_setup_ru.png")
					set(newMap(g))
				}
			case 7:
				if frame == 40 {
					save(img, "10_campaign_map_ru.png")
					battle = midBattle()
					set(battle)
				}
			case 8:
				if frame == 20 {
					battle.state, battle.timer = stEnemyWait, 0
				}
				if frame > 20 && len(battle.fx.missiles) > 0 && battle.fx.missiles[0].t > 0.5 {
					save(img, "11_battle_ru.png")
					cfg.Lang = 0
					g.cap.Campaign[0] = Campaign{Front: campaignPorts}
					set(newCampaignEnd(g))
				}
			case 9:
				if frame == 60 {
					save(img, "12_campaign_won.png")
					g.quit = true
				}
			}
		}
	}
}
