package main

import (
	"bytes"
	"encoding/binary"
	"math"
	"math/rand/v2"
	"sync"

	"github.com/hajimehoshi/ebiten/v2/audio"
)

// All sound effects and music are synthesized at start-up: no audio files,
// no licensing questions, and the executable stays self-contained.

const SR = 44100

type Sound struct {
	ctx     *audio.Context
	mu      sync.Mutex
	sfx     map[string][]float64 // mono
	cache   map[string][]byte    // f32 stereo, per pan
	music   map[string][]byte
	want    string
	cur     *audio.Player
	curName string
	vol     float64
	duck    float64
	players []*audio.Player
}

func newSound(mute bool) *Sound {
	s := &Sound{sfx: map[string][]float64{}, cache: map[string][]byte{}, music: map[string][]byte{}}
	if !mute {
		s.ctx = audio.NewContext(SR)
	}
	genSFX(s.sfx)
	go func() {
		m := map[string][]byte{"menu": toF32(menuTrack()), "battle": toF32(battleTrack())}
		s.mu.Lock()
		s.music = m
		s.mu.Unlock()
	}()
	return s
}

// Play starts a sound effect. pan: -1 left .. 1 right.
func (s *Sound) Play(name string, pan, vol float64) {
	if s.ctx == nil || !cfg.Sound {
		return
	}
	p := math.Round(pan * 2)
	key := name + itoa(int(p))
	b, ok := s.cache[key]
	if !ok {
		m := s.sfx[name]
		if m == nil {
			return
		}
		b = monoToF32(m, p/2)
		s.cache[key] = b
	}
	// keep players referenced until they finish; never stack too many sounds
	n := 0
	for _, o := range s.players {
		if o.IsPlaying() {
			s.players[n] = o
			n++
		}
	}
	s.players = s.players[:n]
	if n >= 5 {
		return
	}
	pl := s.ctx.NewPlayerF32FromBytes(b)
	pl.SetVolume(vol * cfg.SfxVol * 0.75)
	pl.Play()
	s.players = append(s.players, pl)
}

func (s *Sound) Music(name string) { s.want = name }
func (s *Sound) Duck(sec float64)  { s.duck = sec }

// Update fades music between tracks; call once per frame.
func (s *Sound) Update(dt float64) {
	if s.ctx == nil {
		return
	}
	target := 0.0
	if cfg.Music {
		target = cfg.MusicVol
	}
	if s.duck > 0 {
		s.duck -= dt
		target *= 0.2
	}
	if s.curName != s.want {
		s.vol -= dt * 1.2
		if s.vol <= 0 || s.cur == nil {
			if s.cur != nil {
				s.cur.Close()
				s.cur = nil
			}
			s.mu.Lock()
			b := s.music[s.want]
			s.mu.Unlock()
			if b != nil {
				loop := audio.NewInfiniteLoopF32(bytes.NewReader(b), int64(len(b)))
				if p, err := s.ctx.NewPlayerF32(loop); err == nil {
					s.cur, s.curName, s.vol = p, s.want, 0
					p.SetVolume(0)
					p.Play()
				}
			}
		}
	} else if s.vol < target {
		s.vol = math.Min(target, s.vol+dt*0.5)
	} else {
		s.vol = math.Max(target, s.vol-dt*1.0)
	}
	if s.cur != nil {
		s.cur.SetVolume(math.Max(0, s.vol))
	}
}

// --- conversion ---

func monoToF32(m []float64, pan float64) []byte {
	l, r := math.Cos((pan+1)*math.Pi/4)*1.41, math.Sin((pan+1)*math.Pi/4)*1.41
	b := make([]byte, len(m)*8)
	for i, v := range m {
		binary.LittleEndian.PutUint32(b[i*8:], math.Float32bits(float32(v*l)))
		binary.LittleEndian.PutUint32(b[i*8+4:], math.Float32bits(float32(v*r)))
	}
	return b
}

type stereo struct{ l, r []float64 }

func newStereo(sec float64) stereo {
	n := int(sec * SR)
	return stereo{make([]float64, n), make([]float64, n)}
}

func toF32(s stereo) []byte {
	b := make([]byte, len(s.l)*8)
	for i := range s.l {
		binary.LittleEndian.PutUint32(b[i*8:], math.Float32bits(float32(s.l[i])))
		binary.LittleEndian.PutUint32(b[i*8+4:], math.Float32bits(float32(s.r[i])))
	}
	return b
}

// --- DSP building blocks ---

var rnd = rand.New(rand.NewPCG(7, 2026))

func noise() float64 { return rnd.Float64()*2 - 1 }

type onepole struct{ y float64 }

func (f *onepole) lp(x, fc float64) float64 {
	w := 2 * math.Pi * fc / SR
	f.y += w / (1 + w) * (x - f.y)
	return f.y
}

type svf struct{ lp, bp float64 }

// run is a Chamberlin state-variable filter; damp = 1/Q.
func (f *svf) run(x, fc, damp float64) (lp, bp, hp float64) {
	F := 2 * math.Sin(math.Pi*math.Min(fc, SR/6.5)/SR)
	hp = x - f.lp - damp*f.bp
	f.bp += F * hp
	f.lp += F * f.bp
	return f.lp, f.bp, hp
}

func midi(n float64) float64 { return 440 * math.Pow(2, (n-69)/12) }

func saw(ph float64) float64 { return 2*(ph-math.Floor(ph)) - 1 }

func normalize(m []float64, peak float64) {
	mx := 1e-9
	for _, v := range m {
		mx = math.Max(mx, math.Abs(v))
	}
	for i := range m {
		m[i] = math.Tanh(m[i]/mx*peak*1.2) / math.Tanh(1.2)
	}
}

// reverb is a small Schroeder reverb (4 combs + 2 allpasses per channel).
func reverb(in []float64, wet, fb float64, spread int) []float64 {
	out := make([]float64, len(in))
	combs := []int{1116, 1188, 1277, 1356}
	for _, d := range combs {
		d += spread
		buf := make([]float64, d)
		var lp float64
		for i, x := range in {
			y := buf[i%d]
			lp += 0.35 * (y - lp)
			buf[i%d] = x + lp*fb
			out[i] += y * 0.25
		}
	}
	for _, d := range []int{556 + spread, 441 + spread} {
		buf := make([]float64, d)
		for i, x := range out {
			y := buf[i%d]
			v := x + y*0.5
			buf[i%d] = v
			out[i] = y - v*0.5
		}
	}
	for i := range out {
		out[i] = in[i]*(1-wet*0.5) + out[i]*wet
	}
	return out
}

func env(t, a, d float64) float64 {
	if t < 0 {
		return 0
	}
	if t < a {
		return t / a
	}
	return math.Exp(-(t - a) * d)
}

// --- sound effects ---

func genSFX(m map[string][]float64) {
	mk := func(sec float64, f func(t float64, i int) float64) []float64 {
		b := make([]float64, int(sec*SR))
		for i := range b {
			b[i] = f(float64(i)/SR, i)
		}
		return b
	}
	m["click"] = mk(0.08, func(t float64, _ int) float64 {
		return 0.5*math.Sin(2*math.Pi*1400*t)*math.Exp(-t*60) + 0.25*math.Sin(2*math.Pi*2100*t)*math.Exp(-t*90)
	})
	var f1 onepole
	m["place"] = mk(0.25, func(t float64, _ int) float64 {
		return 0.7*math.Sin(2*math.Pi*(140+60*math.Exp(-t*30))*t)*math.Exp(-t*22) + 0.4*f1.lp(noise(), 2500)*math.Exp(-t*60)
	})
	m["error"] = mk(0.22, func(t float64, _ int) float64 {
		return 0.35 * math.Tanh(3*math.Sin(2*math.Pi*110*t)) * env(t, 0.01, 9)
	})
	// missile launch: rising filtered roar
	var lf svf
	var lr onepole
	launch := mk(1.6, func(t float64, _ int) float64 {
		_, bp, _ := lf.run(noise(), 400+2600*math.Sqrt(t/1.6), 0.9)
		roar := lr.lp(noise(), 160) * 2.2
		return (0.9*bp + roar*math.Exp(-t*2.5)) * env(t, 0.03, 2.1)
	})
	normalize(launch, 0.8)
	m["launch"] = reverb(launch, 0.35, 0.7, 0)
	// explosion: sub thump, noisy body, crackle
	var ef, ec onepole
	exp := mk(2.6, func(t float64, _ int) float64 {
		thump := math.Sin(2*math.Pi*(38*t+72/8*(1-math.Exp(-t*8)))) * math.Exp(-t*5.5) * 1.2
		body := ef.lp(noise(), 2600*math.Exp(-t*2.8)+120) * env(t, 0.004, 2.4) * 2.4
		cr := 0.0
		if rnd.Float64() < 0.02*math.Exp(-t*2) {
			cr = noise() * 2
		}
		return thump + body + (cr-ec.lp(cr, 900))*0.5
	})
	normalize(exp, 0.95)
	m["explosion"] = reverb(exp, 0.4, 0.75, 0)
	// sinking: two blasts and a metallic groan
	var sf onepole
	sunk := make([]float64, int(4.2*SR))
	for i := range sunk {
		t := float64(i) / SR
		v := 0.0
		if i < len(exp) {
			v += exp[i]
		}
		if j := i - int(0.38*SR); j >= 0 && j < len(exp) {
			v += exp[j] * 0.8
		}
		f0 := 62 * (1 - 0.12*t/4.2)
		gr := 0.0
		for k, p := range []float64{1, 2.76, 5.4, 8.93} {
			gr += math.Sin(2*math.Pi*f0*p*t+0.3*math.Sin(2*math.Pi*(3.1+float64(k))*t)) / (1 + float64(k))
		}
		v += gr * env(t-0.5, 0.5, 0.9) * 0.45
		v += sf.lp(noise(), 300) * env(t-0.8, 0.6, 0.7) * 0.8
		sunk[i] = v
	}
	normalize(sunk, 0.95)
	m["sunk"] = reverb(sunk, 0.45, 0.8, 0)
	// splash: water burst, plop and bubbles
	var wf svf
	type bub struct{ t0, f float64 }
	var bubs []bub
	for i := 0; i < 14; i++ {
		bubs = append(bubs, bub{0.1 + rnd.Float64()*0.9, 500 + rnd.Float64()*900})
	}
	splash := mk(1.5, func(t float64, _ int) float64 {
		_, bp, hp := wf.run(noise(), 1700, 1.1)
		v := (bp*0.8 + hp*0.25) * env(t, 0.008, 4.2)
		v += math.Sin(2*math.Pi*(90+150*math.Exp(-t*18))*t) * math.Exp(-t*10) * 0.8
		for _, b := range bubs {
			if d := t - b.t0; d > 0 && d < 0.06 {
				v += math.Sin(2*math.Pi*b.f*(1+d*12)*d) * math.Exp(-d*60) * 0.18
			}
		}
		return v
	})
	normalize(splash, 0.8)
	m["splash"] = reverb(splash, 0.3, 0.7, 0)
	// sonar ping
	ping := mk(3.2, func(t float64, _ int) float64 {
		v := 0.0
		for k, d := range []float64{0, 0.45, 0.9, 1.35} {
			v += math.Sin(2*math.Pi*1250*(t-d)) * env(t-d, 0.004, 3.6) * math.Pow(0.38, float64(k))
		}
		return v * 0.45
	})
	m["ping"] = reverb(ping, 0.5, 0.82, 0)
	// alert when the enemy starts firing
	m["alert"] = reverb(mk(0.7, func(t float64, _ int) float64 {
		f := 660.0
		if t > 0.18 {
			f = 495
		}
		return 0.25 * math.Tanh(2.5*math.Sin(2*math.Pi*f*t)) * env(math.Mod(t, 0.18), 0.005, 9)
	}), 0.3, 0.6, 0)
	m["victory"] = stinger(true)
	m["defeat"] = stinger(false)
}

// stinger plays a short brass-like chord progression.
func stinger(win bool) []float64 {
	type ch struct {
		t, d  float64
		notes []float64
	}
	var seq []ch
	if win {
		seq = []ch{{0, 0.32, []float64{62, 66, 69, 74}}, {0.34, 0.32, []float64{67, 71, 74}}, {0.68, 0.32, []float64{69, 73, 76}},
			{1.02, 2.8, []float64{50, 62, 66, 69, 74, 78}}}
	} else {
		seq = []ch{{0, 1.1, []float64{50, 57, 62, 65}}, {1.15, 1.1, []float64{46, 58, 62, 65}}, {2.3, 0.6, []float64{43, 55, 58, 62}},
			{2.95, 2.4, []float64{45, 57, 61, 64}}}
	}
	out := make([]float64, int(5.6*SR))
	for _, c := range seq {
		for ni, n := range c.notes {
			f := midi(n)
			var lp onepole
			ph := rnd.Float64()
			ph2 := rnd.Float64()
			start := int(c.t * SR)
			ln := int((c.d + 0.9) * SR)
			for i := 0; i < ln && start+i < len(out); i++ {
				t := float64(i) / SR
				e := math.Min(1, t/0.03)
				if t > c.d {
					e *= math.Exp(-(t - c.d) * 5)
				}
				fc := 700 + 3200*math.Exp(-t*3)
				if !win {
					fc = 400 + 1200*math.Exp(-t*1.5)
				}
				ph += f / SR
				ph2 += f * 1.004 / SR
				v := lp.lp(saw(ph)+saw(ph2), fc)
				out[start+i] += v * e * 0.18 / (1 + float64(ni)*0.15)
			}
		}
		// timpani hit
		for i := 0; i < int(1.2*SR); i++ {
			t := float64(i) / SR
			j := int(c.t*SR) + i
			if j < len(out) {
				out[j] += math.Sin(2*math.Pi*(55+40*math.Exp(-t*20))*t) * math.Exp(-t*4) * 0.5
			}
		}
	}
	normalize(out, 0.9)
	return reverb(out, 0.45, 0.8, 0)
}

// --- music ---

type song struct {
	st   stereo
	beat float64
	loop int
}

func (s *song) add(ch int, start int, v float64) {
	if start >= 0 && start < len(s.st.l) {
		if ch <= 0 {
			s.st.l[start] += v
		}
		if ch >= 0 {
			s.st.r[start] += v
		}
	}
}

func (s *song) pad(t0, dur float64, notes []float64, amp, bright float64) {
	start := int(t0 * SR)
	n := int((dur + 1.2) * SR)
	for _, nt := range notes {
		f := midi(nt)
		for ch := -1; ch <= 1; ch += 2 {
			var a, b onepole
			p1, p2 := rnd.Float64(), rnd.Float64()
			d1, d2 := 1+0.0016*float64(ch), 1-0.0028*float64(ch)
			for i := 0; i < n; i++ {
				t := float64(i) / SR
				e := math.Min(1, t/0.7)
				if t > dur {
					e *= math.Max(0, 1-(t-dur)/1.2)
				}
				p1 += f * d1 / SR
				p2 += f * d2 / SR
				fc := bright * (1 + 0.35*math.Sin(2*math.Pi*0.15*(t0+t)))
				v := b.lp(a.lp(saw(p1)+saw(p2), fc), fc)
				s.add(ch, start+i, v*e*amp)
			}
		}
	}
}

func (s *song) bass(t0, dur, note, amp float64) {
	f := midi(note)
	var lp onepole
	start := int(t0 * SR)
	for i := 0; i < int((dur+0.1)*SR); i++ {
		t := float64(i) / SR
		e := math.Min(1, t/0.005) * math.Exp(-t*2.5)
		if t > dur {
			e *= math.Max(0, 1-(t-dur)/0.1)
		}
		v := math.Sin(2*math.Pi*f*t) + 0.5*lp.lp(saw(f*t), 400+900*math.Exp(-t*10))
		v = math.Tanh(v*1.3) * e * amp
		s.add(0, start+i, v)
	}
}

func (s *song) pluck(t0, note, amp, pan float64) {
	f := midi(note)
	var lp onepole
	start := int(t0 * SR)
	for i := 0; i < int(0.6*SR); i++ {
		t := float64(i) / SR
		v := lp.lp(saw(f*t)*0.6+math.Sin(2*math.Pi*f*t)*0.5, 500+3500*math.Exp(-t*14)) * math.Exp(-t*7) * amp
		s.add(-1, start+i, v*(1-pan)*0.7)
		s.add(1, start+i, v*(1+pan)*0.7)
		// dotted-eighth echo
		d := start + i + int(s.beat*0.75*SR)
		s.add(-1, d, v*(1+pan)*0.25)
		s.add(1, d, v*(1-pan)*0.25)
	}
}

func (s *song) kick(t0, amp float64) {
	start := int(t0 * SR)
	for i := 0; i < int(0.45*SR); i++ {
		t := float64(i) / SR
		v := math.Sin(2*math.Pi*(45*t+110/14*(1-math.Exp(-t*14)))) * math.Exp(-t*7) * amp
		s.add(0, start+i, math.Tanh(v*1.5))
	}
}

func (s *song) snare(t0, amp float64) {
	var f svf
	start := int(t0 * SR)
	for i := 0; i < int(0.35*SR); i++ {
		t := float64(i) / SR
		_, bp, hp := f.run(noise(), 1800, 0.8)
		v := (bp*0.6+hp*0.4)*math.Exp(-t*14) + math.Sin(2*math.Pi*185*t)*math.Exp(-t*22)*0.6
		s.add(0, start+i, v*amp)
	}
}

func (s *song) hat(t0, amp, pan float64) {
	var f svf
	start := int(t0 * SR)
	for i := 0; i < int(0.08*SR); i++ {
		t := float64(i) / SR
		_, _, hp := f.run(noise(), 7000, 0.7)
		v := hp * math.Exp(-t*55) * amp
		s.add(-1, start+i, v*(1-pan))
		s.add(1, start+i, v*(1+pan))
	}
}

func (s *song) ping(t0, amp float64) {
	start := int(t0 * SR)
	for i := 0; i < int(1.8*SR); i++ {
		t := float64(i) / SR
		s.add(0, start+i, math.Sin(2*math.Pi*1250*t)*env(t, 0.004, 3.2)*amp)
	}
}

// finish adds reverb, folds the tail into the start for a seamless loop and normalizes.
func (s *song) finish(wet float64) stereo {
	l := reverb(s.st.l, wet, 0.82, 0)
	r := reverb(s.st.r, wet, 0.82, 23)
	for i := s.loop; i < len(l); i++ {
		l[i-s.loop] += l[i]
		r[i-s.loop] += r[i]
	}
	l, r = l[:s.loop], r[:s.loop]
	mx := 1e-9
	for i := range l {
		mx = math.Max(mx, math.Max(math.Abs(l[i]), math.Abs(r[i])))
	}
	for i := range l {
		l[i] = math.Tanh(l[i]/mx*1.05) * 0.82
		r[i] = math.Tanh(r[i]/mx*1.05) * 0.82
	}
	return stereo{l, r}
}

var chords = map[string][]float64{
	"Dm": {50, 53, 57, 62}, "Bb": {46, 53, 58, 62}, "F": {53, 57, 60, 65}, "C": {48, 52, 55, 60},
	"Gm": {43, 55, 58, 62}, "A": {45, 57, 61, 64}, "Am": {45, 57, 60, 64},
}
var roots = map[string]float64{"Dm": 38, "Bb": 34, "F": 41, "C": 36, "Gm": 43, "A": 33, "Am": 33}

// battleTrack: 110 BPM, D minor, 16 bars.
func battleTrack() stereo {
	prog := []string{"Dm", "Bb", "F", "C", "Dm", "Bb", "Gm", "A"}
	beat := 60.0 / 110
	bar := beat * 4
	bars := 16
	s := &song{st: newStereo(bar*float64(bars) + 5), beat: beat, loop: int(bar * float64(bars) * SR)}
	for b := 0; b < bars; b++ {
		c := prog[b%8]
		t := float64(b) * bar
		s.pad(t, bar, chords[c], 0.022, 1300)
		if b >= 2 {
			for e := 0; e < 8; e++ {
				n := roots[c]
				if e == 2 || e == 6 {
					n += 12
				}
				s.bass(t+float64(e)*beat/2, beat*0.42, n, 0.16)
			}
		}
		if b >= 4 {
			for k := 0; k < 4; k++ {
				s.kick(t+float64(k)*beat, 0.55)
				if k == 1 || k == 3 {
					s.snare(t+float64(k)*beat, 0.32)
				}
			}
			s.kick(t+2.5*beat, 0.35)
			for e := 0; e < 8; e++ {
				a := 0.06
				if e%2 == 1 {
					a = 0.11
				}
				s.hat(t+float64(e)*beat/2, a, 0.3)
			}
			if b%8 == 7 {
				for k := 0; k < 4; k++ {
					s.snare(t+3*beat+float64(k)*beat/4, 0.12+0.05*float64(k))
				}
			}
		}
		if b >= 6 {
			ns := chords[c]
			pat := []int{0, 1, 2, 3, 2, 1, 2, 3}
			for e := 0; e < 8; e++ {
				s.pluck(t+float64(e)*beat/2, ns[pat[e]]+12, 0.07, math.Sin(float64(e)))
			}
		}
	}
	return s.finish(0.3)
}

// menuTrack: 74 BPM, calm ambient with sonar pings, 8 bars.
func menuTrack() stereo {
	prog := []string{"Dm", "Bb", "F", "C", "Dm", "Gm", "Bb", "A"}
	beat := 60.0 / 74
	bar := beat * 4
	s := &song{st: newStereo(bar*8 + 5), beat: beat, loop: int(bar * 8 * SR)}
	for b := 0; b < 8; b++ {
		c := prog[b]
		t := float64(b) * bar
		s.pad(t, bar, chords[c], 0.026, 900)
		s.pad(t, bar, []float64{chords[c][3] + 12}, 0.012, 2400)
		s.bass(t, bar*0.9, roots[c], 0.10)
		ns := chords[c]
		for e := 0; e < 8; e++ {
			if e%2 == 0 || b >= 4 {
				s.pluck(t+float64(e)*beat/2, ns[(e*3)%4]+24, 0.035, math.Sin(float64(e+b)))
			}
		}
		if b%4 == 0 {
			s.ping(t+beat*0.5, 0.08)
		}
	}
	return s.finish(0.45)
}
