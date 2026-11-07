package main

import (
	"image"
	"image/color"
	"math"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/vector"
)

// --- Palette ---

var (
	colText    = color.NRGBA{0xe8, 0xf4, 0xff, 0xff}
	colDim     = color.NRGBA{0x8f, 0xa9, 0xc0, 0xff}
	colCyan    = color.NRGBA{0x3d, 0xe1, 0xf0, 0xff}
	colRed     = color.NRGBA{0xff, 0x4d, 0x4d, 0xff}
	colGreen   = color.NRGBA{0x4d, 0xf0, 0x9a, 0xff}
	colGold    = color.NRGBA{0xff, 0xd1, 0x66, 0xff}
	colPanel   = color.NRGBA{0x06, 0x14, 0x24, 0xd8}
	colBorder  = color.NRGBA{0x3d, 0xe1, 0xf0, 0x55}
	colOutline = color.NRGBA{0x02, 0x08, 0x10, 0xff}
)

func withA(c color.NRGBA, a float64) color.NRGBA {
	c.A = uint8(math.Max(0, math.Min(255, float64(c.A)*a)))
	return c
}

func mixC(a, b color.NRGBA, t float64) color.NRGBA {
	f := func(x, y uint8) uint8 { return uint8(float64(x) + (float64(y)-float64(x))*t) }
	return color.NRGBA{f(a.R, b.R), f(a.G, b.G), f(a.B, b.B), f(a.A, b.A)}
}

// --- Vector helpers ---

func pathOpt(c color.Color) *vector.DrawPathOptions {
	op := &vector.DrawPathOptions{AntiAlias: true}
	op.ColorScale.ScaleWithColor(c)
	return op
}

func fillPath(dst *ebiten.Image, p *vector.Path, c color.Color) {
	vector.FillPath(dst, p, nil, pathOpt(c))
}

func strokePath(dst *ebiten.Image, p *vector.Path, w float32, c color.Color) {
	vector.StrokePath(dst, p, &vector.StrokeOptions{Width: w, LineJoin: vector.LineJoinRound, LineCap: vector.LineCapRound}, pathOpt(c))
}

func fillRect(dst *ebiten.Image, x, y, w, h float32, c color.Color) {
	vector.FillRect(dst, x, y, w, h, c, true)
}

func strokeRect(dst *ebiten.Image, x, y, w, h, sw float32, c color.Color) {
	vector.StrokeRect(dst, x, y, w, h, sw, c, true)
}

func line(dst *ebiten.Image, x0, y0, x1, y1, w float32, c color.Color) {
	vector.StrokeLine(dst, x0, y0, x1, y1, w, c, true)
}

// circle draws a filled anti-aliased disc from a pre-rendered texture
// (much cheaper than a vector path when many small circles are drawn).
func circle(dst *ebiten.Image, x, y, r float32, c color.Color) {
	op := &ebiten.DrawImageOptions{Filter: ebiten.FilterLinear}
	op.GeoM.Translate(-64, -64)
	op.GeoM.Scale(float64(r)/63, float64(r)/63)
	op.GeoM.Translate(float64(x), float64(y))
	op.ColorScale.ScaleWithColor(c)
	dst.DrawImage(discTex, op)
}

func ring(dst *ebiten.Image, x, y, r, w float32, c color.Color) {
	vector.StrokeCircle(dst, x, y, r, w, c, true)
}

func rrPath(x, y, w, h, r float32) *vector.Path {
	p := &vector.Path{}
	r = float32(math.Min(float64(r), math.Min(float64(w), float64(h))/2))
	p.MoveTo(x+r, y)
	p.LineTo(x+w-r, y)
	p.QuadTo(x+w, y, x+w, y+r)
	p.LineTo(x+w, y+h-r)
	p.QuadTo(x+w, y+h, x+w-r, y+h)
	p.LineTo(x+r, y+h)
	p.QuadTo(x, y+h, x, y+h-r)
	p.LineTo(x, y+r)
	p.QuadTo(x, y, x+r, y)
	p.Close()
	return p
}

func fillRR(dst *ebiten.Image, x, y, w, h, r float32, c color.Color) {
	fillPath(dst, rrPath(x, y, w, h, r), c)
}

func strokeRR(dst *ebiten.Image, x, y, w, h, r, sw float32, c color.Color) {
	strokePath(dst, rrPath(x, y, w, h, r), sw, c)
}

// arc draws a circular arc (angles in radians).
func arc(dst *ebiten.Image, x, y, r, a0, a1, w float32, c color.Color) {
	p := &vector.Path{}
	n := 24
	for i := 0; i <= n; i++ {
		a := float64(a0 + (a1-a0)*float32(i)/float32(n))
		px, py := x+r*float32(math.Cos(a)), y+r*float32(math.Sin(a))
		if i == 0 {
			p.MoveTo(px, py)
		} else {
			p.LineTo(px, py)
		}
	}
	strokePath(dst, p, w, c)
}

// --- Textures ---

var softDot, vignette, whitePix, discTex *ebiten.Image

func initTextures() {
	const n = 64
	img := image.NewNRGBA(image.Rect(0, 0, n, n))
	for y := 0; y < n; y++ {
		for x := 0; x < n; x++ {
			d := math.Hypot(float64(x)-n/2+0.5, float64(y)-n/2+0.5) / (n / 2)
			a := math.Max(0, 1-d)
			img.SetNRGBA(x, y, color.NRGBA{255, 255, 255, uint8(255 * a * a)})
		}
	}
	softDot = ebiten.NewImageFromImage(img)
	const vw, vh = 160, 100
	v := image.NewNRGBA(image.Rect(0, 0, vw, vh))
	for y := 0; y < vh; y++ {
		for x := 0; x < vw; x++ {
			dx, dy := (float64(x)/vw-0.5)*2, (float64(y)/vh-0.5)*2
			d := math.Sqrt(dx*dx*0.8 + dy*dy)
			a := math.Pow(math.Max(0, d-0.35)/1.05, 1.6)
			v.SetNRGBA(x, y, color.NRGBA{0, 2, 8, uint8(math.Min(1, a) * 235)})
		}
	}
	vignette = ebiten.NewImageFromImage(v)
	d := image.NewNRGBA(image.Rect(0, 0, 128, 128))
	for y := 0; y < 128; y++ {
		for x := 0; x < 128; x++ {
			dd := math.Hypot(float64(x)-63.5, float64(y)-63.5)
			a := math.Max(0, math.Min(1, 63-dd+0.5))
			d.SetNRGBA(x, y, color.NRGBA{255, 255, 255, uint8(255 * a)})
		}
	}
	discTex = ebiten.NewImageFromImage(d)
	whitePix = ebiten.NewImage(3, 3)
	whitePix.Fill(color.White)
}

// glowDot draws a soft additive light.
func glowDot(dst *ebiten.Image, x, y, r float64, c color.NRGBA, a float64) {
	op := &ebiten.DrawImageOptions{}
	op.GeoM.Translate(-32, -32)
	op.GeoM.Scale(r/32, r/32)
	op.GeoM.Translate(x, y)
	op.ColorScale.ScaleWithColor(c)
	op.ColorScale.ScaleAlpha(float32(a))
	op.Blend = ebiten.BlendLighter
	op.Filter = ebiten.FilterLinear
	dst.DrawImage(softDot, op)
}

func drawVignette(dst *ebiten.Image, a float64) {
	op := &ebiten.DrawImageOptions{}
	b := dst.Bounds()
	op.GeoM.Scale(float64(b.Dx())/160, float64(b.Dy())/100)
	op.GeoM.Translate(float64(b.Min.X), float64(b.Min.Y))
	op.ColorScale.ScaleAlpha(float32(a))
	op.Filter = ebiten.FilterLinear
	dst.DrawImage(vignette, op)
}

// vgrad draws a vertical gradient rectangle.
func vgrad(dst *ebiten.Image, x, y, w, h float32, top, bottom color.NRGBA) {
	tc := func(c color.NRGBA) (float32, float32, float32, float32) {
		a := float32(c.A) / 255
		return float32(c.R) / 255 * a, float32(c.G) / 255 * a, float32(c.B) / 255 * a, a
	}
	r0, g0, b0, a0 := tc(top)
	r1, g1, b1, a1 := tc(bottom)
	vs := []ebiten.Vertex{
		{DstX: x, DstY: y, SrcX: 1, SrcY: 1, ColorR: r0, ColorG: g0, ColorB: b0, ColorA: a0},
		{DstX: x + w, DstY: y, SrcX: 1, SrcY: 1, ColorR: r0, ColorG: g0, ColorB: b0, ColorA: a0},
		{DstX: x, DstY: y + h, SrcX: 1, SrcY: 1, ColorR: r1, ColorG: g1, ColorB: b1, ColorA: a1},
		{DstX: x + w, DstY: y + h, SrcX: 1, SrcY: 1, ColorR: r1, ColorG: g1, ColorB: b1, ColorA: a1},
	}
	op := &ebiten.DrawTrianglesOptions{}
	op.ColorScaleMode = ebiten.ColorScaleModePremultipliedAlpha
	dst.DrawTriangles(vs, []uint16{0, 1, 2, 1, 3, 2}, whitePix, op)
}

// --- Ocean shader ---

const oceanSrc = `//kage:unit pixels
package main

var Time float
var Deep vec3
var Shallow vec3
var Scale float

func hash(p vec2) float {
	return fract(sin(dot(p, vec2(127.1, 311.7))) * 43758.5453)
}

func noise(p vec2) float {
	i := floor(p)
	f := fract(p)
	u := f * f * (3.0 - 2.0*f)
	a := hash(i)
	b := hash(i + vec2(1.0, 0.0))
	c := hash(i + vec2(0.0, 1.0))
	d := hash(i + vec2(1.0, 1.0))
	return mix(mix(a, b, u.x), mix(c, d, u.x), u.y)
}

func fbm(p vec2) float {
	v := 0.0
	a := 0.5
	for i := 0; i < 4; i++ {
		v += a * noise(p)
		p = p*2.03 + vec2(1.7, 9.2)
		a *= 0.5
	}
	return v
}

func Fragment(dstPos vec4, srcPos vec2, color vec4) vec4 {
	p := dstPos.xy / Scale
	t := Time * 0.06
	q := vec2(fbm(p+vec2(t, -t*0.6)), fbm(p+vec2(-t*0.7, t)+vec2(5.2, 1.3)))
	n := fbm(p + 1.8*q + vec2(t*0.5, t*0.3))
	col := mix(Deep, Shallow, smoothstep(0.2, 0.85, n))
	crest := smoothstep(0.6, 0.8, n) * (0.55 + 0.45*sin(Time*1.2+p.x*3.0+p.y*2.0))
	col += vec3(0.30, 0.50, 0.58) * crest * 0.32
	c := abs(sin((q.x-q.y)*16.0 + Time*0.5))
	col += vec3(0.05, 0.13, 0.15) * pow(1.0-c, 6.0)
	return vec4(col, 1.0) * color.a
}
`

var oceanShader *ebiten.Shader

// drawOcean fills a rectangle with animated water.
func drawOcean(dst *ebiten.Image, x, y, w, h float64, t float64, deep, shallow [3]float32, scale float64) {
	op := &ebiten.DrawRectShaderOptions{}
	op.GeoM.Translate(x, y)
	op.Uniforms = map[string]any{
		"Time": float32(t), "Deep": deep[:], "Shallow": shallow[:], "Scale": float32(scale),
	}
	dst.DrawRectShader(int(w), int(h), oceanShader, op)
}

var (
	seaDeep       = [3]float32{0.010, 0.050, 0.105}
	seaShallow    = [3]float32{0.030, 0.200, 0.300}
	boardDeep     = [3]float32{0.020, 0.105, 0.180}
	boardShallow  = [3]float32{0.050, 0.290, 0.400}
	mapSeaDeep    = [3]float32{0.025, 0.110, 0.190}
	mapSeaShallow = [3]float32{0.060, 0.260, 0.360}
)

// --- UI ---

func panel(dst *ebiten.Image, x, y, w, h float32) {
	img := cached("panel"+itoa(int(w))+"x"+itoa(int(h)), int(w)+10, int(h)+12, func(d *ebiten.Image) {
		fillRR(d, 4, 6, w, h, 16, color.NRGBA{0, 0, 0, 90})
		fillRR(d, 0, 0, w, h, 16, colPanel)
		strokeRR(d, 0.75, 0.75, w-1.5, h-1.5, 16, 1.5, colBorder)
		line(d, 18, 1.5, w-18, 1.5, 1.5, color.NRGBA{0x9f, 0xf6, 0xff, 0x50})
	})
	drawLayer(dst, img, float64(x), float64(y))
}

func panelTitle(dst *ebiten.Image, s string, x, y float64, c color.NRGBA) {
	drawTextOpt(dst, s, Display, 20, x, y, c, 0, TextOpt{Spacing: 2})
	w := measure(s, Display, 20, 2)
	line(dst, float32(x), float32(y+30), float32(x+w), float32(y+30), 2, withA(c, 0.8))
}

type Button struct {
	X, Y, W, H float64
	Label      func() string
	Primary    bool
	Disabled   bool
	Keys       []ebiten.Key
	hover      float64
	Accent     color.NRGBA
}

func (b *Button) Update(g *Game) bool {
	in := g.in(b.X, b.Y, b.W, b.H)
	t := 0.0
	if in && !b.Disabled {
		t = 1
	}
	b.hover += (t - b.hover) * 0.25
	if b.Disabled {
		return false
	}
	hit := in && g.click
	for _, k := range b.Keys {
		if g.keyJust(k) {
			hit = true
		}
	}
	if hit {
		g.snd.Play("click", 0, 0.7)
		g.click = false
	}
	return hit
}

func (b *Button) Draw(g *Game, dst *ebiten.Image) {
	x, y, w, h := float32(b.X), float32(b.Y), float32(b.W), float32(b.H)
	acc := colCyan
	if b.Accent.A != 0 {
		acc = b.Accent
	}
	hv := b.hover
	alpha := 1.0
	if b.Disabled {
		alpha = 0.4
	}
	if b.Primary {
		glowDot(dst, b.X+b.W/2, b.Y+b.H/2, b.W*0.75, acc, 0.10+0.2*hv)
		top := mixC(acc, color.NRGBA{255, 255, 255, 255}, 0.15+0.25*hv)
		bot := mixC(acc, color.NRGBA{0, 30, 60, 255}, 0.45)
		fillRR(dst, x, y+3, w, h, 12, withA(color.NRGBA{0, 0, 0, 120}, alpha))
		fillRR(dst, x, y, w, h, 12, withA(bot, alpha))
		fillRR(dst, x+2, y+2, w-4, h*0.55, 10, withA(top, alpha*0.85))
		strokeRR(dst, x, y, w, h, 12, 2, withA(mixC(acc, color.NRGBA{255, 255, 255, 255}, 0.5), alpha))
		drawTextOpt(dst, b.Label(), Display, 21, b.X+b.W/2, b.Y+b.H/2-14, withA(color.NRGBA{0x02, 0x10, 0x20, 0xff}, alpha), 1, TextOpt{Spacing: 1})
		return
	}
	fillRR(dst, x, y, w, h, 12, withA(mixC(color.NRGBA{0x08, 0x1c, 0x30, 0xe0}, color.NRGBA{0x10, 0x3a, 0x55, 0xf0}, hv), alpha))
	strokeRR(dst, x, y, w, h, 12, 1.5, withA(withA(acc, 0.45+0.5*hv), alpha))
	drawTextOpt(dst, b.Label(), Display, 19, b.X+b.W/2, b.Y+b.H/2-12, withA(mixC(colText, acc, hv*0.6), alpha), 1, TextOpt{Spacing: 1})
}

// --- Board ---

const letters = "ABCDEFGHIJ"

// drawBoard draws a 10x10 sea grid with coordinates. The frame and the grid
// are static and cached; only the water is animated.
func drawBoard(g *Game, dst *ebiten.Image, x, y, cell float64, accent color.NRGBA) {
	const o = 46.0
	s := cell * 10
	key := itoa(int(cell)) + itoa(int(accent.R)) + itoa(int(accent.G))
	under := cached("bu"+key, int(s+2*o), int(s+2*o), func(d *ebiten.Image) {
		fs := float32(s)
		glowDot(d, o+s/2, o+s/2, s*0.75, accent, 0.06)
		fillRR(d, o-34, o-34, fs+44, fs+44, 14, color.NRGBA{0x04, 0x10, 0x1e, 0xd0})
		strokeRR(d, o-34, o-34, fs+44, fs+44, 14, 1.2, withA(accent, 0.35))
		ls := math.Min(18, cell*0.4)
		for i := 0; i < 10; i++ {
			c := (float64(i) + 0.5) * cell
			drawText(d, string(letters[i]), Display, ls, o+c, o-ls-10, colDim, 1)
			drawText(d, itoa(i+1), Display, ls, o-14, o+c-ls*0.62, colDim, 2)
		}
	})
	over := cached("bo"+key, int(s+2*o), int(s+2*o), func(d *ebiten.Image) {
		fs := float32(s)
		for i := 0; i <= 10; i++ {
			c := float32(i) * float32(cell)
			a := 0.22
			if i == 0 || i == 10 {
				a = 0.7
			}
			line(d, o+c, o, o+c, o+fs, 1, withA(color.NRGBA{0x9f, 0xf6, 0xff, 0xff}, a))
			line(d, o, o+c, o+fs, o+c, 1, withA(color.NRGBA{0x9f, 0xf6, 0xff, 0xff}, a))
		}
		for _, cr := range [][4]float32{{o, o, 1, 1}, {o + fs, o, -1, 1}, {o, o + fs, 1, -1}, {o + fs, o + fs, -1, -1}} {
			line(d, cr[0]-3*cr[2], cr[1]-3*cr[3], cr[0]+18*cr[2], cr[1]-3*cr[3], 3, accent)
			line(d, cr[0]-3*cr[2], cr[1]-3*cr[3], cr[0]-3*cr[2], cr[1]+18*cr[3], 3, accent)
		}
	})
	drawLayer(dst, under, x-o, y-o)
	drawOcean(dst, x, y, s, s, g.t+x*0.01, boardDeep, boardShallow, 150)
	drawLayer(dst, over, x-o, y-o)
}

func cellCenter(x, y, cell float64, r, c int) (float64, float64) {
	return x + (float64(c)-0.5)*cell, y + (float64(r)-0.5)*cell
}

// cellAt returns the row and column under the point, or 0, 0.
func cellAt(x, y, cell float64, px, py int) (int, int) {
	fx, fy := (float64(px)-x)/cell, (float64(py)-y)/cell
	if fx < 0 || fy < 0 || fx >= 10 || fy >= 10 {
		return 0, 0
	}
	return int(fy) + 1, int(fx) + 1
}

func coord(r, c int) string { return string(letters[c-1]) + itoa(r) }

func itoa(n int) string {
	if n == 0 {
		return "0"
	}
	neg := n < 0
	if neg {
		n = -n
	}
	var b []byte
	for n > 0 {
		b = append([]byte{byte('0' + n%10)}, b...)
		n /= 10
	}
	if neg {
		b = append([]byte{'-'}, b...)
	}
	return string(b)
}

// --- cached static layers ---

var layers = map[string]*ebiten.Image{}

// cached returns an image drawn once by draw and reused afterwards.
func cached(key string, w, h int, draw func(img *ebiten.Image)) *ebiten.Image {
	if img, ok := layers[key]; ok {
		return img
	}
	img := ebiten.NewImage(w, h)
	draw(img)
	layers[key] = img
	return img
}

func drawLayer(dst, img *ebiten.Image, x, y float64) {
	op := &ebiten.DrawImageOptions{}
	op.GeoM.Translate(x, y)
	dst.DrawImage(img, op)
}
