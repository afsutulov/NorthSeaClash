package main

import (
	"image/color"
	"math"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/vector"
)

// Ships are drawn once as vector art (top view, bow to the right) at SS pixels
// per cell, then scaled and rotated when they are placed on a board.
const SS = 96

var shipImg [2][5]*ebiten.Image

func buildShips() {
	for c := 0; c < 2; c++ {
		for l := 1; l <= 4; l++ {
			shipImg[c][l] = renderShip(&coalitions[c], l)
		}
	}
}

func renderShip(co *Coalition, L int) *ebiten.Image {
	W := float32(L * SS)
	img := ebiten.NewImage(int(W), SS)
	beam := float32(38 + 5*L)
	yT, yB := 48-beam/2, 48+beam/2
	xs, xt := float32(7), W-5
	taper := float32(math.Min(float64(W)*0.34, 92))
	if L == 1 {
		taper = 38
	}
	hull := func(in float32) *vector.Path {
		p := &vector.Path{}
		t, b := yT+in, yB-in
		x0, x1 := xs+in, xt-in*1.8
		xb := x1 - taper + in*0.5
		p.MoveTo(x0+5, t)
		p.LineTo(xb, t)
		p.CubicTo(xb+taper*0.55, t, x1-taper*0.12, 48-beam*0.12, x1, 48)
		p.CubicTo(x1-taper*0.12, 48+beam*0.12, xb+taper*0.55, b, xb, b)
		p.LineTo(x0+5, b)
		p.QuadTo(x0, b, x0, b-5)
		p.LineTo(x0, t+5)
		p.QuadTo(x0, t, x0+5, t)
		p.Close()
		return p
	}
	fillPath(img, hull(0), co.Outline)
	fillPath(img, hull(2.5), co.Hull)
	fillPath(img, hull(6), co.Deck)
	// deck planking / centre line and edge highlight
	line(img, xs+12, 48, xt-taper*0.6, 48, 1.2, withA(co.Hull, 0.6))
	strokePath(img, hull(6), 1, withA(color.NRGBA{255, 255, 255, 255}, 0.18))
	// stern accent band
	fillRect(img, xs+3, yT+6, 4, beam-12, co.Accent)

	X := func(f float64) float32 { return xs + (xt-xs)*float32(f) }
	block := func(x0, x1 float32, hw float32, c color.NRGBA) {
		fillRR(img, x0+2.5, 48-hw+3, x1-x0, hw*2, 4, color.NRGBA{0, 0, 0, 90})
		fillRR(img, x0, 48-hw, x1-x0, hw*2, 4, co.Outline)
		fillRR(img, x0+1.5, 48-hw+1.5, x1-x0-3, hw*2-3, 3, c)
		fillRect(img, x0+2, 48-hw+2, x1-x0-4, (hw*2-4)*0.35, withA(color.NRGBA{255, 255, 255, 255}, 0.22))
	}
	turret := func(x float32, r float32) {
		circle(img, x+1.5, 50, r, color.NRGBA{0, 0, 0, 90})
		line(img, x, 48, x+r*2.6, 48, 2.6, co.Outline)
		circle(img, x, 48, r, co.Outline)
		circle(img, x, 48, r-1.5, co.Super)
		circle(img, x-r*0.25, 47-r*0.25, r*0.35, withA(color.NRGBA{255, 255, 255, 255}, 0.5))
	}
	vls := func(x0 float32, cols, rows int) {
		cw := float32(5.5)
		w, h := float32(cols)*cw, float32(rows)*cw
		fillRect(img, x0-1, 48-h/2-1, w+2, h+2, co.Outline)
		for i := 0; i < cols; i++ {
			for j := 0; j < rows; j++ {
				fillRect(img, x0+float32(i)*cw+0.6, 48-h/2+float32(j)*cw+0.6, cw-1.2, cw-1.2, mixC(co.Deck, co.Hull, 0.5))
			}
		}
	}
	radar := func(x float32, r float32) {
		circle(img, x, 48, r, co.Outline)
		circle(img, x, 48, r-1.4, mixC(co.Super, co.Accent, 0.25))
		line(img, x, 48-r*1.7, x, 48+r*1.7, 2.2, co.Outline)
		circle(img, x, 48, 1.8, co.Accent)
	}
	launchers := func(x float32, n int) {
		for i := 0; i < n; i++ {
			yy := 48 - beam*0.28 + float32(i)*beam*0.56/float32(max(1, n-1))
			fillRR(img, x, yy-3, 16, 6, 2, co.Outline)
			fillRR(img, x+1, yy-2, 14, 4, 1.5, co.Super)
		}
	}
	helipad := func(x0, x1 float32) {
		cx := (x0 + x1) / 2
		r := float32(math.Min(float64(x1-x0)/2, float64(beam)*0.36))
		circle(img, cx, 48, r, withA(co.Outline, 0.55))
		ring(img, cx, 48, r-2, 1.5, withA(color.NRGBA{255, 255, 255, 255}, 0.75))
		line(img, cx-r*0.35, 48-r*0.45, cx-r*0.35, 48+r*0.45, 2, color.NRGBA{255, 255, 255, 220})
		line(img, cx+r*0.35, 48-r*0.45, cx+r*0.35, 48+r*0.45, 2, color.NRGBA{255, 255, 255, 220})
		line(img, cx-r*0.35, 48, cx+r*0.35, 48, 2, color.NRGBA{255, 255, 255, 220})
	}
	hw := beam * 0.3
	switch L {
	case 1: // missile boat
		launchers(X(0.10), 2)
		block(X(0.38), X(0.62), hw, co.Super)
		fillRect(img, X(0.58), 48-hw+3, 3, hw*2-6, withA(co.Outline, 0.8))
		radar(X(0.48), 5)
		turret(X(0.74), 6)
	case 2: // corvette
		launchers(X(0.08), 2)
		block(X(0.24), X(0.36), hw*0.8, co.Deck)
		block(X(0.38), X(0.62), hw, co.Super)
		fillRect(img, X(0.59), 48-hw+3, 3, hw*2-6, withA(co.Outline, 0.8))
		radar(X(0.50), 6.5)
		vls(X(0.64), 2, 4)
		turret(X(0.80), 8)
	case 3: // destroyer
		helipad(X(0.02), X(0.19))
		block(X(0.20), X(0.30), hw, co.Deck)
		launchers(X(0.31), 2)
		block(X(0.40), X(0.62), hw*1.05, co.Super)
		fillRect(img, X(0.60), 48-hw+3, 3, hw*2-6, withA(co.Outline, 0.8))
		radar(X(0.47), 7)
		radar(X(0.56), 5)
		vls(X(0.65), 3, 4)
		turret(X(0.84), 9)
	case 4: // cruiser
		helipad(X(0.02), X(0.13))
		block(X(0.14), X(0.22), hw, co.Deck)
		vls(X(0.235), 4, 4)
		block(X(0.36), X(0.58), hw*1.1, co.Super)
		fillRect(img, X(0.56), 48-hw+3, 3, hw*2-6, withA(co.Outline, 0.8))
		radar(X(0.41), 7)
		radar(X(0.51), 8)
		launchers(X(0.30), 2)
		vls(X(0.61), 4, 4)
		turret(X(0.78), 9)
		turret(X(0.87), 8)
	}
	return img
}

// shipGeo returns the transform for a ship placed with its first cell at
// board cell (r, c).
func shipGeo(s Ship, x, y, cell float64) ebiten.GeoM {
	k := cell / SS
	var m ebiten.GeoM
	if s.V {
		m.Rotate(math.Pi / 2)
		m.Translate(SS, 0)
	}
	m.Scale(k, k)
	m.Translate(x+float64(s.C-1)*cell, y+float64(s.R-1)*cell)
	return m
}

// drawShip draws a ship with its shadow. tint scales the colour (wrecks are dark).
func drawShip(dst *ebiten.Image, coal int, s Ship, x, y, cell float64, tint ebiten.ColorScale, alpha float32) {
	img := shipImg[coal][s.L]
	m := shipGeo(s, x, y, cell)
	op := &ebiten.DrawImageOptions{Filter: ebiten.FilterLinear}
	op.GeoM = m
	op.GeoM.Translate(cell*0.05, cell*0.08)
	op.ColorScale.Scale(0, 0, 0, 0.35*alpha)
	dst.DrawImage(img, op)
	op = &ebiten.DrawImageOptions{Filter: ebiten.FilterLinear}
	op.GeoM = m
	op.ColorScale = tint
	op.ColorScale.ScaleAlpha(alpha)
	dst.DrawImage(img, op)
}

// drawShipFree draws a ship sprite at an arbitrary position and angle
// (centre x, y), used by the title screen and the fleet roster.
func drawShipFree(dst *ebiten.Image, coal, L int, x, y, cell, angle float64, alpha float32) {
	img := shipImg[coal][L]
	k := cell / SS
	op := &ebiten.DrawImageOptions{Filter: ebiten.FilterLinear}
	op.GeoM.Translate(-float64(L*SS)/2, -SS/2)
	op.GeoM.Scale(k, k)
	op.GeoM.Rotate(angle)
	op.GeoM.Translate(x+cell*0.06, y+cell*0.09)
	op.ColorScale.Scale(0, 0, 0, 0.35*alpha)
	dst.DrawImage(img, op)
	op.GeoM.Translate(-cell*0.06, -cell*0.09)
	op.ColorScale = ebiten.ColorScale{}
	op.ColorScale.ScaleAlpha(alpha)
	dst.DrawImage(img, op)
}

// shipBowStern returns the board coordinates of the ship centre.
func shipCenter(s Ship, x, y, cell float64) (float64, float64) {
	if s.V {
		return x + (float64(s.C)-0.5)*cell, y + (float64(s.R-1)+float64(s.L)/2)*cell
	}
	return x + (float64(s.C-1)+float64(s.L)/2)*cell, y + (float64(s.R)-0.5)*cell
}
