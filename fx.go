package main

import (
	"image/color"
	"math"
	"math/rand/v2"

	"github.com/hajimehoshi/ebiten/v2"
)

type particle struct {
	x, y, vx, vy  float64
	life, max     float64
	size, grow    float64
	drag, gravity float64
	c0, c1        color.NRGBA
	add           bool
	ring          bool
	alpha         float64
}

type FX struct {
	ps       []particle
	missiles []*Missile
}

func rf(a, b float64) float64 { return a + rand.Float64()*(b-a) }

func (f *FX) add(p particle) {
	if p.alpha == 0 {
		p.alpha = 1
	}
	p.max = p.life
	f.ps = append(f.ps, p)
}

func (f *FX) Update(dt float64) {
	n := 0
	for i := range f.ps {
		p := &f.ps[i]
		p.life -= dt
		if p.life <= 0 {
			continue
		}
		p.vx *= 1 - p.drag*dt
		p.vy *= 1 - p.drag*dt
		p.vy += p.gravity * dt
		p.x += p.vx * dt
		p.y += p.vy * dt
		p.size += p.grow * dt
		f.ps[n] = *p
		n++
	}
	f.ps = f.ps[:n]
	m := 0
	for _, ms := range f.missiles {
		if ms.update(f, dt) {
			f.missiles[m] = ms
			m++
		}
	}
	f.missiles = f.missiles[:m]
}

func (f *FX) drawParticles(dst *ebiten.Image, additive bool) {
	for i := range f.ps {
		p := &f.ps[i]
		if p.add != additive {
			continue
		}
		t := 1 - p.life/p.max
		c := mixC(p.c0, p.c1, t)
		a := p.alpha * math.Min(1, p.life/p.max*2.2)
		if p.ring {
			ring(dst, float32(p.x), float32(p.y), float32(p.size), float32(2+3*(1-t)), withA(c, a))
			continue
		}
		op := &ebiten.DrawImageOptions{Filter: ebiten.FilterLinear}
		op.GeoM.Translate(-32, -32)
		op.GeoM.Scale(p.size/32, p.size/32)
		op.GeoM.Translate(p.x, p.y)
		op.ColorScale.ScaleWithColor(c)
		op.ColorScale.ScaleAlpha(float32(a))
		if p.add {
			op.Blend = ebiten.BlendLighter
		}
		dst.DrawImage(softDot, op)
	}
}

func (f *FX) Draw(dst *ebiten.Image) {
	f.drawParticles(dst, false)
	for _, m := range f.missiles {
		m.draw(dst)
	}
	f.drawParticles(dst, true)
}

func (f *FX) Busy() bool { return len(f.missiles) > 0 }

var (
	cFire0  = color.NRGBA{0xff, 0xf2, 0xb0, 0xff}
	cFire1  = color.NRGBA{0xff, 0x5a, 0x10, 0xff}
	cSmoke0 = color.NRGBA{0x3a, 0x38, 0x36, 0xc0}
	cSmoke1 = color.NRGBA{0x14, 0x16, 0x1a, 0x00}
	cWater0 = color.NRGBA{0xe8, 0xfb, 0xff, 0xff}
	cWater1 = color.NRGBA{0x7c, 0xd6, 0xf0, 0x80}
)

// Explosion at x, y; scale 1 for a hit, about 1.8 for a sinking ship.
func (f *FX) Explosion(x, y, s float64) {
	f.add(particle{x: x, y: y, life: 0.22, size: 40 * s, grow: 150 * s, c0: color.NRGBA{255, 250, 220, 255}, c1: cFire1, add: true})
	f.add(particle{x: x, y: y, life: 0.6, size: 10 * s, grow: 160 * s, c0: color.NRGBA{255, 220, 160, 255}, c1: color.NRGBA{255, 120, 40, 0}, ring: true})
	for i := 0; i < int(26*s); i++ {
		a, v := rf(0, 2*math.Pi), rf(30, 150)*s
		f.add(particle{x: x, y: y, vx: math.Cos(a) * v, vy: math.Sin(a) * v, life: rf(0.4, 0.9), size: rf(10, 22) * s, grow: 20,
			drag: 2.5, c0: cFire0, c1: cFire1, add: true})
	}
	for i := 0; i < int(20*s); i++ {
		a, v := rf(0, 2*math.Pi), rf(20, 70)*s
		f.add(particle{x: x, y: y, vx: math.Cos(a) * v, vy: math.Sin(a)*v - 25, life: rf(1.2, 2.4), size: rf(12, 20) * s, grow: 26 * s,
			drag: 1.4, gravity: -12, c0: cSmoke0, c1: cSmoke1})
	}
	for i := 0; i < int(30*s); i++ {
		a, v := rf(0, 2*math.Pi), rf(150, 420)*s
		f.add(particle{x: x, y: y, vx: math.Cos(a) * v, vy: math.Sin(a)*v - 80, life: rf(0.3, 0.8), size: rf(2, 4), gravity: 420,
			drag: 1.2, c0: color.NRGBA{255, 240, 180, 255}, c1: color.NRGBA{255, 100, 20, 255}, add: true})
	}
	for i := 0; i < int(10*s); i++ {
		a, v := rf(0, 2*math.Pi), rf(80, 220)*s
		f.add(particle{x: x, y: y, vx: math.Cos(a) * v, vy: math.Sin(a)*v - 60, life: rf(0.5, 1.0), size: rf(3, 6), gravity: 380,
			drag: 1, c0: color.NRGBA{30, 30, 30, 255}, c1: color.NRGBA{10, 10, 10, 200}})
	}
}

// Splash of a missile hitting the water.
func (f *FX) Splash(x, y, s float64) {
	for i := 0; i < 3; i++ {
		f.add(particle{x: x, y: y, life: 0.9 + float64(i)*0.35, size: 6, grow: 55 + float64(i)*18, c0: cWater0, c1: cWater1, ring: true, alpha: 0.9})
	}
	for i := 0; i < int(34*s); i++ {
		a := rf(-math.Pi, 0)
		v := rf(60, 230) * s
		f.add(particle{x: x + rf(-4, 4), y: y, vx: math.Cos(a) * v * 0.6, vy: math.Sin(a)*v - 60, life: rf(0.5, 1.0), size: rf(3, 7), gravity: 520,
			c0: cWater0, c1: cWater1, add: true, alpha: 0.85})
	}
	for i := 0; i < 10; i++ {
		f.add(particle{x: x + rf(-8, 8), y: y + rf(-8, 8), vx: rf(-20, 20), vy: rf(-40, -10), life: rf(0.7, 1.4), size: rf(10, 18), grow: 30,
			drag: 1.5, c0: color.NRGBA{230, 250, 255, 170}, c1: color.NRGBA{200, 240, 255, 0}})
	}
}

// Burn emits flames and smoke from a burning cell (call every frame).
func (f *FX) Burn(x, y, cell, dt, intensity float64) {
	if rand.Float64() < 14*dt*intensity {
		f.add(particle{x: x + rf(-cell*0.22, cell*0.22), y: y + rf(-cell*0.18, cell*0.18), vx: rf(-6, 6), vy: rf(-30, -14), life: rf(0.35, 0.7),
			size: rf(cell*0.16, cell*0.28), grow: -6, c0: color.NRGBA{255, 230, 140, 255}, c1: color.NRGBA{255, 70, 10, 255}, add: true, alpha: 0.9})
	}
	if rand.Float64() < 7*dt*intensity {
		f.add(particle{x: x + rf(-cell*0.15, cell*0.15), y: y + rf(-cell*0.15, cell*0.1), vx: rf(4, 14), vy: rf(-36, -20), life: rf(1.6, 2.6),
			size: rf(cell*0.15, cell*0.24), grow: cell * 0.35, drag: 0.3, c0: color.NRGBA{0x30, 0x2c, 0x2a, 0x9a}, c1: color.NRGBA{0x20, 0x22, 0x28, 0}})
	}
}

// Wake emits foam behind a moving ship.
func (f *FX) Wake(x, y, size float64) {
	f.add(particle{x: x + rf(-2, 2), y: y + rf(-2, 2), life: rf(1.6, 2.6), size: size, grow: size * 0.9,
		c0: color.NRGBA{235, 250, 255, 110}, c1: color.NRGBA{200, 240, 255, 0}})
}

// Missile flies along a quadratic Bezier curve and calls hit on arrival.
type Missile struct {
	x0, y0, cx, cy, x1, y1 float64
	t, dur                 float64
	hit                    func()
	px, py, ang            float64
	trail                  float64
}

func (f *FX) Launch(x0, y0, x1, y1, arc, dur float64, hit func()) {
	mx, my := (x0+x1)/2, math.Min(y0, y1)-arc
	m := &Missile{x0: x0, y0: y0, cx: mx, cy: my, x1: x1, y1: y1, dur: dur, hit: hit, px: x0, py: y0}
	f.missiles = append(f.missiles, m)
	// launch flash and smoke
	f.add(particle{x: x0, y: y0, life: 0.3, size: 26, grow: 60, c0: color.NRGBA{255, 240, 200, 255}, c1: cFire1, add: true})
	for i := 0; i < 14; i++ {
		f.add(particle{x: x0 + rf(-6, 6), y: y0 + rf(-6, 6), vx: rf(-40, 40), vy: rf(-20, 30), life: rf(0.8, 1.6), size: rf(8, 14), grow: 22,
			drag: 2, c0: color.NRGBA{0xd8, 0xdc, 0xe0, 0xa0}, c1: color.NRGBA{0x80, 0x88, 0x90, 0}})
	}
}

func (m *Missile) pos(t float64) (float64, float64) {
	u := 1 - t
	return u*u*m.x0 + 2*u*t*m.cx + t*t*m.x1, u*u*m.y0 + 2*u*t*m.cy + t*t*m.y1
}

func (m *Missile) update(f *FX, dt float64) bool {
	m.t += dt / m.dur
	// ease-in: missiles accelerate after launch
	e := m.t * m.t * (0.35 + 0.65*m.t)
	e = 0.25*m.t + 0.75*e
	if m.t >= 1 {
		if m.hit != nil {
			m.hit()
		}
		return false
	}
	x, y := m.pos(e)
	m.ang = math.Atan2(y-m.py, x-m.px)
	steps := int(math.Max(1, math.Hypot(x-m.px, y-m.py)/4))
	for i := 0; i < steps; i++ {
		k := float64(i) / float64(steps)
		sx, sy := m.px+(x-m.px)*k, m.py+(y-m.py)*k
		f.add(particle{x: sx + rf(-1, 1), y: sy + rf(-1, 1), vx: rf(-6, 6), vy: rf(-6, 6), life: rf(0.6, 1.1), size: rf(4, 6), grow: 14,
			drag: 1, c0: color.NRGBA{0xe6, 0xea, 0xee, 0x90}, c1: color.NRGBA{0x70, 0x78, 0x80, 0}})
	}
	m.px, m.py = x, y
	return true
}

func (m *Missile) draw(dst *ebiten.Image) {
	ca, sa := math.Cos(m.ang), math.Sin(m.ang)
	tx, ty := m.px-ca*12, m.py-sa*12
	glowDot(dst, tx, ty, 26, color.NRGBA{255, 160, 60, 255}, 0.9)
	glowDot(dst, tx, ty, 10, color.NRGBA{255, 255, 220, 255}, 1)
	line(dst, float32(m.px-ca*12), float32(m.py-sa*12), float32(m.px+ca*6), float32(m.py+sa*6), 4.5, color.NRGBA{0x1c, 0x22, 0x2a, 0xff})
	line(dst, float32(m.px-ca*11), float32(m.py-sa*11), float32(m.px+ca*5), float32(m.py+sa*5), 2.5, color.NRGBA{0xe0, 0xe6, 0xec, 0xff})
	circle(dst, float32(m.px+ca*6), float32(m.py+sa*6), 2.2, color.NRGBA{0xff, 0x50, 0x40, 0xff})
}
