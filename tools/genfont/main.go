//go:build ignore

// genfont pre-renders the TTF fonts into PNG glyph atlases plus JSON metrics.
// The game embeds the results, so it needs no font-rendering dependencies at runtime.
//
//	go run tools/genfont/main.go <RussoOne.ttf> <Exo2.ttf> <outdir>
package main

import (
	"encoding/json"
	"fmt"
	"image"
	"image/color"
	"image/draw"
	"image/png"
	"os"
	"path/filepath"

	"golang.org/x/image/font"
	"golang.org/x/image/font/opentype"
	"golang.org/x/image/math/fixed"
)

type glyph struct {
	X, Y, W, H int
	OX, OY     int // offset of the bitmap from the pen position (top-left, baseline-relative)
	Adv        float64
}

type metrics struct {
	Size    float64
	Ascent  float64
	Descent float64
	Glyphs  map[string]glyph
}

func charset() []rune {
	var r []rune
	for c := rune(32); c < 127; c++ {
		r = append(r, c)
	}
	for c := 'А'; c <= 'я'; c++ {
		r = append(r, c)
	}
	return append(r, []rune("ЁёЇїІіЄє—–«»…№•·°×←→↑↓✓★")...)
}

func gen(ttf, name string, size float64, out string) {
	data, err := os.ReadFile(ttf)
	if err != nil {
		panic(err)
	}
	f, err := opentype.Parse(data)
	if err != nil {
		panic(err)
	}
	face, err := opentype.NewFace(f, &opentype.FaceOptions{Size: size, DPI: 72, Hinting: font.HintingNone})
	if err != nil {
		panic(err)
	}
	m := face.Metrics()
	met := metrics{Size: size, Ascent: float64(m.Ascent) / 64, Descent: float64(m.Descent) / 64, Glyphs: map[string]glyph{}}
	const W = 2048
	atlas := image.NewRGBA(image.Rect(0, 0, W, 4096))
	x, y, rowH, pad := 1, 1, 0, 2
	for _, c := range charset() {
		b, adv, ok := face.GlyphBounds(c)
		if !ok {
			continue
		}
		gx0, gy0 := b.Min.X.Floor()-1, b.Min.Y.Floor()-1
		gx1, gy1 := b.Max.X.Ceil()+1, b.Max.Y.Ceil()+1
		w, h := gx1-gx0, gy1-gy0
		if c == ' ' {
			w, h = 0, 0
		}
		if x+w+pad >= W {
			x, y, rowH = 1, y+rowH+pad, 0
		}
		if w > 0 && h > 0 {
			d := font.Drawer{Dst: atlas, Src: image.NewUniform(color.White), Face: face,
				Dot: fixed.P(x-gx0, y-gy0)}
			d.DrawString(string(c))
		}
		met.Glyphs[string(c)] = glyph{X: x, Y: y, W: w, H: h, OX: gx0, OY: gy0, Adv: float64(adv) / 64}
		x += w + pad
		if h > rowH {
			rowH = h
		}
	}
	H := y + rowH + pad
	img := atlas.SubImage(image.Rect(0, 0, W, H)).(*image.RGBA)
	trim := image.NewNRGBA(image.Rect(0, 0, W, H))
	draw.Draw(trim, trim.Bounds(), img, image.Point{}, draw.Src)
	fp, _ := os.Create(filepath.Join(out, name+".png"))
	png.Encode(fp, trim)
	fp.Close()
	js, _ := json.Marshal(met)
	os.WriteFile(filepath.Join(out, name+".json"), js, 0o644)
	fmt.Println(name, W, H, len(met.Glyphs))
}

func main() {
	out := os.Args[3]
	os.MkdirAll(out, 0o755)
	gen(os.Args[1], "display96", 96, out)
	gen(os.Args[1], "display48", 48, out)
	gen(os.Args[1], "display28", 28, out)
	gen(os.Args[2], "body26", 26, out)
	gen(os.Args[2], "body18", 18, out)
}
