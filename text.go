package main

import (
	"bytes"
	"embed"
	"encoding/json"
	"image"
	"image/color"
	_ "image/png"
	"strings"

	"github.com/hajimehoshi/ebiten/v2"
)

// Fonts are pre-rendered glyph atlases (see tools/genfont): Russo One for
// headings, Exo 2 for body text. Both are licensed under the SIL OFL 1.1.

//go:embed assets/fonts
var fontFS embed.FS

type glyph struct {
	X, Y, W, H, OX, OY int
	Adv                float64
}

type atlas struct {
	Size, Ascent, Descent float64
	Glyphs                map[string]glyph
	img                   *ebiten.Image
	g                     map[rune]glyph
	sub                   map[rune]*ebiten.Image
}

type Face int

const (
	Display Face = iota
	Body
)

var faces [2][]*atlas

func loadAtlas(name string) *atlas {
	a := &atlas{g: map[rune]glyph{}, sub: map[rune]*ebiten.Image{}}
	js, err := fontFS.ReadFile("assets/fonts/" + name + ".json")
	if err != nil {
		panic(err)
	}
	if err := json.Unmarshal(js, a); err != nil {
		panic(err)
	}
	pngData, _ := fontFS.ReadFile("assets/fonts/" + name + ".png")
	img, _, err := image.Decode(bytes.NewReader(pngData))
	if err != nil {
		panic(err)
	}
	a.img = ebiten.NewImageFromImage(img)
	for k, v := range a.Glyphs {
		r := []rune(k)[0]
		a.g[r] = v
		if v.W > 0 && v.H > 0 {
			a.sub[r] = a.img.SubImage(image.Rect(v.X, v.Y, v.X+v.W, v.Y+v.H)).(*ebiten.Image)
		}
	}
	return a
}

func initFonts() {
	faces[Display] = []*atlas{loadAtlas("display28"), loadAtlas("display48"), loadAtlas("display96")}
	faces[Body] = []*atlas{loadAtlas("body18"), loadAtlas("body26")}
}

func pickAtlas(f Face, size float64) *atlas {
	l := faces[f]
	for _, a := range l {
		if a.Size >= size*0.9 {
			return a
		}
	}
	return l[len(l)-1]
}

func (a *atlas) glyph(r rune) (glyph, bool) {
	if g, ok := a.g[r]; ok {
		return g, true
	}
	switch r {
	case 'ё':
		return a.glyph('е')
	case 'Ё':
		return a.glyph('Е')
	}
	g, ok := a.g['?']
	return g, ok
}

// measure returns the width of a single line.
func measure(s string, f Face, size, spacing float64) float64 {
	a := pickAtlas(f, size)
	k := size / a.Size
	w := 0.0
	n := 0
	for _, r := range s {
		g, _ := a.glyph(r)
		w += g.Adv*k + spacing
		n++
	}
	if n > 0 {
		w -= spacing
	}
	return w
}

func lineHeight(f Face, size float64) float64 {
	a := pickAtlas(f, size)
	return (a.Ascent + a.Descent) * size / a.Size * 1.12
}

// Text draws (multi-line) text. y is the top of the text block.
// align: 0 left, 1 centre, 2 right.
type TextOpt struct {
	Spacing float64
	Blend   ebiten.Blend
	Alpha   float64
}

func drawText(dst *ebiten.Image, s string, f Face, size, x, y float64, clr color.Color, align int) {
	drawTextOpt(dst, s, f, size, x, y, clr, align, TextOpt{})
}

func drawTextOpt(dst *ebiten.Image, s string, f Face, size, x, y float64, clr color.Color, align int, o TextOpt) {
	a := pickAtlas(f, size)
	k := size / a.Size
	lh := lineHeight(f, size)
	for li, ln := range strings.Split(s, "\n") {
		w := measure(ln, f, size, o.Spacing)
		px := x - w*float64(align)/2
		base := y + float64(li)*lh + a.Ascent*k
		for _, r := range ln {
			g, _ := a.glyph(r)
			if img := a.sub[r]; img != nil {
				op := &ebiten.DrawImageOptions{}
				op.GeoM.Scale(k, k)
				op.GeoM.Translate(px+float64(g.OX)*k, base+float64(g.OY)*k)
				op.ColorScale.ScaleWithColor(clr)
				if o.Alpha > 0 {
					op.ColorScale.ScaleAlpha(float32(o.Alpha))
				}
				op.Filter = ebiten.FilterLinear
				op.Blend = o.Blend
				dst.DrawImage(img, op)
			}
			px += g.Adv*k + o.Spacing
		}
	}
}

// drawTextShadow draws text with a soft drop shadow.
func drawTextShadow(dst *ebiten.Image, s string, f Face, size, x, y float64, clr color.Color, align int, spacing float64) {
	sh := size / 18
	drawTextOpt(dst, s, f, size, x+sh, y+sh*1.4, color.NRGBA{0, 0, 0, 150}, align, TextOpt{Spacing: spacing})
	drawTextOpt(dst, s, f, size, x, y, clr, align, TextOpt{Spacing: spacing})
}

// drawTextGlow draws text with an additive halo around it.
func drawTextGlow(dst *ebiten.Image, s string, f Face, size, x, y float64, clr, glow color.Color, align int, spacing, strength float64) {
	r := size / 14
	for i := 0; i < 12; i++ {
		ax := []float64{1, 0.87, 0.5, 0, -0.5, -0.87, -1, -0.87, -0.5, 0, 0.5, 0.87}[i]
		ay := []float64{0, 0.5, 0.87, 1, 0.87, 0.5, 0, -0.5, -0.87, -1, -0.87, -0.5}[i]
		drawTextOpt(dst, s, f, size, x+ax*r, y+ay*r, glow, align, TextOpt{Spacing: spacing, Blend: ebiten.BlendLighter, Alpha: 0.09 * strength})
		drawTextOpt(dst, s, f, size, x+ax*r*2.2, y+ay*r*2.2, glow, align, TextOpt{Spacing: spacing, Blend: ebiten.BlendLighter, Alpha: 0.04 * strength})
	}
	drawTextShadow(dst, s, f, size, x, y, clr, align, spacing)
}

// fitSize returns the largest size <= size at which s fits into width.
func fitSize(s string, f Face, size, width, spacing float64) float64 {
	if w := measure(s, f, size, spacing); w > width {
		return size * width / w
	}
	return size
}

// wrap breaks text into lines no wider than width.
func wrap(s string, f Face, size, width float64) string {
	var out []string
	for _, para := range strings.Split(s, "\n") {
		line := ""
		for _, w := range strings.Fields(para) {
			try := w
			if line != "" {
				try = line + " " + w
			}
			if line != "" && measure(try, f, size, 0) > width {
				out = append(out, line)
				line = w
			} else {
				line = try
			}
		}
		out = append(out, line)
	}
	return strings.Join(out, "\n")
}
