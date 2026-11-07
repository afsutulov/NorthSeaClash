// North Sea Clash — a modern naval battle game.
//
// It continues the author's Python/Tk "Морской бой" (fleet placement rules,
// enemy commander). Game flow: title, captain card, campaign map, deployment,
// battle, records.
package main

import (
	"bytes"
	_ "embed"
	"flag"
	"fmt"
	"image"
	_ "image/png"
	"math"
	"math/rand/v2"
	"os"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/inpututil"
)

//go:embed assets/icon.png
var iconPNG []byte

const W, H = 1280, 800
const dt = 1.0 / 60

type Scene interface {
	Update(g *Game)
	Draw(g *Game, dst *ebiten.Image)
}

type Game struct {
	scene, next Scene
	fade        float64 // 0..1 black overlay during transitions
	fading      bool
	t           float64
	snd         *Sound
	store       *Store
	cap         *Captain
	coal        int
	canvas      *ebiten.Image
	mx, my      int
	click       bool
	rclick      bool
	shake       float64
	ready       bool
	quit        bool
	shot        func(g *Game, img *ebiten.Image) // screenshot hook (tests)
}

func (g *Game) Go(s Scene) {
	if g.next == nil {
		g.next = s
		g.fading = true
	}
}

func (g *Game) in(x, y, w, h float64) bool {
	return float64(g.mx) >= x && float64(g.mx) < x+w && float64(g.my) >= y && float64(g.my) < y+h
}

func (g *Game) keyJust(k ebiten.Key) bool { return inpututil.IsKeyJustPressed(k) }

// keyRep reports a key press with auto-repeat.
func (g *Game) keyRep(k ebiten.Key) bool {
	n := inpututil.KeyPressDuration(k)
	return n == 1 || (n >= 22 && n%4 == 0)
}

func (g *Game) Shake(a float64) { g.shake = math.Max(g.shake, a) }

func (g *Game) init() {
	initFonts()
	initTextures()
	var err error
	if oceanShader, err = ebiten.NewShader([]byte(oceanSrc)); err != nil {
		panic(err)
	}
	buildShips()
	g.ready = true
}

func (g *Game) Update() error {
	if !g.ready {
		g.init()
	}
	if ebiten.IsWindowBeingClosed() || g.quit {
		g.store.Save()
		return ebiten.Termination
	}
	g.t += dt
	g.mx, g.my = ebiten.CursorPosition()
	g.click = inpututil.IsMouseButtonJustPressed(ebiten.MouseButtonLeft)
	g.rclick = inpututil.IsMouseButtonJustPressed(ebiten.MouseButtonRight)
	if g.keyJust(ebiten.KeyF11) || (g.keyJust(ebiten.KeyEnter) && ebiten.IsKeyPressed(ebiten.KeyAlt)) {
		cfg.Fullscreen = !ebiten.IsFullscreen()
		ebiten.SetFullscreen(cfg.Fullscreen)
	}
	if g.keyJust(ebiten.KeyF1) {
		cfg.Sound = !cfg.Sound
	}
	if g.keyJust(ebiten.KeyF2) {
		cfg.Music = !cfg.Music
	}
	g.snd.Update(dt)
	g.shake *= 0.88
	if g.fading {
		if g.next != nil {
			g.fade += dt * 4
			if g.fade >= 1 {
				g.fade = 1
				g.scene, g.next = g.next, nil
			}
		} else {
			g.fade -= dt * 3
			if g.fade <= 0 {
				g.fade, g.fading = 0, false
			}
		}
		if g.next != nil {
			return nil
		}
	}
	g.scene.Update(g)
	return nil
}

func (g *Game) Draw(screen *ebiten.Image) {
	if !g.ready {
		return
	}
	g.canvas.Clear()
	g.scene.Draw(g, g.canvas)
	if g.shot != nil {
		g.shot(g, g.canvas)
	}
	op := &ebiten.DrawImageOptions{}
	if g.shake > 0.3 {
		op.GeoM.Translate(rf(-g.shake, g.shake), rf(-g.shake, g.shake))
	}
	screen.DrawImage(g.canvas, op)
	if g.fade > 0 {
		fillRect(screen, 0, 0, W, H, withA(colOutline, g.fade))
	}
}

func (g *Game) Layout(int, int) (int, int) { return W, H }

func main() {
	mute := flag.Bool("mute", false, "start without audio (for systems without a sound device)")
	lang := flag.String("lang", "", "interface language: en or ru")
	flag.Parse()

	g := &Game{canvas: ebiten.NewImage(W, H)}
	g.store = loadStore()
	switch *lang {
	case "en":
		cfg.Lang = 0
	case "ru":
		cfg.Lang = 1
	}
	g.coal = cfg.LastCoalition
	g.snd = newSound(*mute)
	g.scene = newTitle(g)
	if setupShots != nil {
		setupShots(g)
	}

	ebiten.SetWindowTitle("North Sea Clash — Битва в Северном море")
	ebiten.SetWindowSize(W, H)
	ebiten.SetWindowResizingMode(ebiten.WindowResizingModeEnabled)
	ebiten.SetWindowClosingHandled(true)
	ebiten.SetTPS(60)
	ebiten.SetFullscreen(cfg.Fullscreen)
	if img, _, err := image.Decode(bytes.NewReader(iconPNG)); err == nil {
		ebiten.SetWindowIcon([]image.Image{img})
	}
	if err := ebiten.RunGame(g); err != nil {
		g.store.Save()
		fmt.Fprintln(os.Stderr, err)
		fmt.Fprintln(os.Stderr, "If the error is about audio, start the game with --mute.")
		os.Exit(1)
	}
}

// setupShots is set by the screenshot tool build (go build -tags shots).
var setupShots func(g *Game)

var _ = rand.IntN
