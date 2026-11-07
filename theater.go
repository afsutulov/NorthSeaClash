package main

import (
	"image"
	"image/color"

	"github.com/hajimehoshi/ebiten/v2"
)

// tr returns the English or Russian variant of a text.
func tr(en, ru string) string {
	if cfg.Lang == 1 {
		return ru
	}
	return en
}

type Country struct {
	EN, RU string
	Flag   func(dst *ebiten.Image, x, y, w, h float32)
	Lon    float64 // label position on the campaign map
	Lat    float64
}

type Port struct {
	EN, RU    string
	Country   int
	Lon, Lat  float64
	LabelLeft bool
}

func (p Port) Name() string { return tr(p.EN, p.RU) }

type Coalition struct {
	EN, RU         string
	MottoEN        string
	MottoRU        string
	Countries      []int
	Ports          []Port // ordered from the front line to the rear
	Accent         color.NRGBA
	Hull, Deck     color.NRGBA
	Super, Outline color.NRGBA
}

func (c *Coalition) Name() string { return tr(c.EN, c.RU) }

const (
	cUK = iota
	cNO
	cNL
	cDE
	cDK
	cBE
)

var countries = []Country{
	cUK: {"United Kingdom", "Великобритания", flagUK, -2.0, 54.35},
	cNO: {"Norway", "Норвегия", flagNO, 8.6, 60.4},
	cNL: {"Netherlands", "Нидерланды", flagNL, 5.9, 52.5},
	cDE: {"Germany", "Германия", flagDE, 9.3, 52.9},
	cDK: {"Denmark", "Дания", flagDK, 9.3, 56.3},
	cBE: {"Belgium", "Бельгия", flagBE, 4.6, 50.98},
}

// The North Sea theatre: two fictional coalitions of real maritime nations,
// each defending five real naval and commercial ports.
var coalitions = [2]Coalition{
	{
		EN: "Northern Alliance", RU: "Северный альянс",
		MottoEN: "Masters of the northern waters", MottoRU: "Хозяева северных вод",
		Countries: []int{cUK, cNO},
		Ports: []Port{
			{"Hull", "Халл", cUK, -0.33, 53.74, false},
			{"Rosyth", "Росайт", cUK, -3.43, 56.03, false},
			{"Aberdeen", "Абердин", cUK, -2.08, 57.15, false},
			{"Stavanger", "Ставангер", cNO, 5.73, 58.97, false},
			{"Bergen", "Берген", cNO, 5.32, 60.39, false},
		},
		Accent: color.NRGBA{0x4c, 0xc3, 0xff, 0xff},
		Hull:   color.NRGBA{0x52, 0x66, 0x7c, 0xff}, Deck: color.NRGBA{0x7d, 0x92, 0xa8, 0xff},
		Super: color.NRGBA{0xd6, 0xe2, 0xee, 0xff}, Outline: color.NRGBA{0x12, 0x1c, 0x28, 0xff},
	},
	{
		EN: "Continental League", RU: "Континентальная лига",
		MottoEN: "Steel shield of the coast", MottoRU: "Стальной щит побережья",
		Countries: []int{cNL, cDE, cDK, cBE},
		Ports: []Port{
			{"Den Helder", "Ден-Хелдер", cNL, 4.76, 52.96, false},
			{"Rotterdam", "Роттердам", cNL, 4.12, 51.97, false},
			{"Zeebrugge", "Зебрюгге", cBE, 3.20, 51.33, false},
			{"Wilhelmshaven", "Вильгельмсхафен", cDE, 8.13, 53.52, true},
			{"Esbjerg", "Эсбьерг", cDK, 8.45, 55.47, true},
		},
		Accent: color.NRGBA{0xff, 0xb3, 0x47, 0xff},
		Hull:   color.NRGBA{0x5f, 0x63, 0x5c, 0xff}, Deck: color.NRGBA{0x8c, 0x90, 0x86, 0xff},
		Super: color.NRGBA{0xe4, 0xe2, 0xd8, 0xff}, Outline: color.NRGBA{0x1a, 0x1b, 0x18, 0xff},
	},
}

var shipClassEN = [5]string{"", "Missile boat", "Corvette", "Destroyer", "Cruiser"}
var shipClassRU = [5]string{"", "Ракетный катер", "Корвет", "Эсминец", "Крейсер"}

func shipClass(l int) string { return tr(shipClassEN[l], shipClassRU[l]) }

// Coastlines of the North Sea (longitude, latitude), simplified by hand.
var landPolys = [][][2]float64{
	// Great Britain (east coast)
	{{-7, 50.0}, {1.4, 51.1}, {1.45, 51.38}, {0.9, 51.45}, {0.6, 51.52}, {1.0, 51.78}, {1.3, 51.95}, {1.6, 52.2}, {1.75, 52.5},
		{1.7, 52.75}, {1.3, 52.95}, {0.5, 52.97}, {0.25, 52.8}, {0.35, 53.15}, {0.25, 53.45}, {0.12, 53.6}, {-0.2, 53.72}, {-0.05, 54.1},
		{-0.2, 54.15}, {-0.6, 54.5}, {-1.15, 54.65}, {-1.4, 55.0}, {-1.55, 55.35}, {-1.75, 55.65}, {-2.0, 55.85}, {-2.4, 55.98},
		{-2.9, 56.02}, {-3.45, 56.0}, {-3.1, 56.06}, {-2.7, 56.2}, {-2.58, 56.28}, {-2.85, 56.45}, {-2.55, 56.58}, {-2.4, 56.75},
		{-2.2, 56.95}, {-2.07, 57.15}, {-1.95, 57.35}, {-1.78, 57.5}, {-2.0, 57.69}, {-2.6, 57.68}, {-3.3, 57.7}, {-3.9, 57.6},
		{-4.2, 57.55}, {-3.8, 57.82}, {-3.95, 57.95}, {-3.4, 58.25}, {-3.1, 58.42}, {-3.05, 58.65}, {-3.4, 58.65}, {-5.0, 58.62}, {-7, 58.6}},
	// Orkney and Shetland
	{{-3.4, 58.8}, {-2.75, 58.88}, {-2.4, 59.05}, {-2.8, 59.3}, {-3.35, 59.15}},
	{{-1.4, 59.85}, {-1.1, 60.15}, {-1.0, 60.6}, {-0.8, 60.8}, {-1.3, 60.65}, {-1.5, 60.35}, {-1.35, 60.1}},
	// Continent with Jutland
	{{13, 50.0}, {1.6, 50.0}, {1.6, 50.7}, {1.85, 50.95}, {2.5, 51.08}, {3.2, 51.33}, {3.45, 51.4}, {3.6, 51.6}, {3.9, 51.85},
		{4.05, 51.98}, {4.3, 52.15}, {4.55, 52.45}, {4.65, 52.75}, {4.75, 52.95}, {5.05, 53.12}, {5.5, 53.32}, {6.2, 53.42}, {6.85, 53.45},
		{7.05, 53.3}, {7.2, 53.65}, {7.9, 53.72}, {8.05, 53.55}, {8.15, 53.48}, {8.35, 53.62}, {8.55, 53.55}, {8.7, 53.88}, {9.0, 54.0},
		{8.65, 54.3}, {8.9, 54.48}, {8.65, 54.9}, {8.55, 55.15}, {8.38, 55.45}, {8.1, 55.58}, {8.15, 56.0}, {8.1, 56.55}, {8.25, 56.85},
		{8.6, 57.1}, {9.5, 57.15}, {10.0, 57.45}, {10.6, 57.75}, {10.45, 57.25}, {10.3, 56.65}, {10.9, 56.42}, {10.25, 56.0},
		{10.0, 55.7}, {9.6, 55.5}, {9.8, 55.05}, {10.0, 54.65}, {11.0, 54.4}, {13, 54.4}},
	// Funen and Zealand
	{{9.75, 55.5}, {10.25, 55.6}, {10.8, 55.3}, {10.6, 55.05}, {10.0, 55.1}},
	{{11.0, 55.7}, {11.5, 56.0}, {12.6, 56.05}, {12.4, 55.3}, {11.5, 55.15}, {11.1, 55.3}},
	// Scandinavia
	{{4.9, 62.5}, {5.0, 61.3}, {4.85, 60.8}, {5.0, 60.3}, {5.2, 59.9}, {5.3, 59.3}, {5.6, 58.95}, {5.55, 58.6}, {6.4, 58.25},
		{7.05, 57.98}, {8.0, 58.12}, {8.65, 58.35}, {9.6, 58.9}, {10.5, 59.2}, {10.6, 59.85}, {10.9, 59.2}, {11.2, 58.9},
		{11.2, 58.4}, {11.9, 57.7}, {12.5, 56.9}, {12.9, 56.4}, {14, 56.3}, {14, 62.5}},
}

// Map view (degrees) and the equirectangular scale at 56°N.
const mapLon0, mapLon1, mapLat0, mapLat1, mapKx = -4.9, 11.0, 50.6, 61.4, 0.559

// --- Flags (drawn into dst clipped to the flag rectangle) ---

func clip(dst *ebiten.Image, x, y, w, h float32) *ebiten.Image {
	return dst.SubImage(image.Rect(int(x), int(y), int(x+w+0.5), int(y+h+0.5))).(*ebiten.Image)
}

func hex(v uint32) color.NRGBA {
	return color.NRGBA{uint8(v >> 16), uint8(v >> 8), uint8(v), 0xff}
}

func stripes(dst *ebiten.Image, x, y, w, h float32, vertical bool, cols ...uint32) {
	n := float32(len(cols))
	for i, c := range cols {
		if vertical {
			fillRect(dst, x+w*float32(i)/n, y, w/n+0.6, h, hex(c))
		} else {
			fillRect(dst, x, y+h*float32(i)/n, w, h/n+0.6, hex(c))
		}
	}
}

func flagDE(d *ebiten.Image, x, y, w, h float32) {
	stripes(d, x, y, w, h, false, 0x111111, 0xdd0000, 0xffce00)
}
func flagNL(d *ebiten.Image, x, y, w, h float32) {
	stripes(d, x, y, w, h, false, 0xae1c28, 0xffffff, 0x21468b)
}
func flagBE(d *ebiten.Image, x, y, w, h float32) {
	stripes(d, x, y, w, h, true, 0x111111, 0xfdda24, 0xef3340)
}

func nordic(d *ebiten.Image, x, y, w, h float32, bg, c1, c2 uint32, cx, wd1, wd2 float32) {
	fillRect(d, x, y, w, h, hex(bg))
	fillRect(d, x+w*cx-h*wd1/2, y, h*wd1, h, hex(c1))
	fillRect(d, x, y+h/2-h*wd1/2, w, h*wd1, hex(c1))
	if wd2 > 0 {
		fillRect(d, x+w*cx-h*wd2/2, y, h*wd2, h, hex(c2))
		fillRect(d, x, y+h/2-h*wd2/2, w, h*wd2, hex(c2))
	}
}

func flagNO(d *ebiten.Image, x, y, w, h float32) {
	nordic(d, x, y, w, h, 0xba0c2f, 0xffffff, 0x00205b, 8.0/22, 4.0/16, 2.0/16)
}
func flagDK(d *ebiten.Image, x, y, w, h float32) {
	nordic(d, x, y, w, h, 0xc8102e, 0xffffff, 0, 14.0/37, 4.0/28, 0)
}

func flagUK(dst *ebiten.Image, x, y, w, h float32) {
	d := clip(dst, x, y, w, h)
	fillRect(d, x, y, w, h, hex(0x012169))
	line(d, x, y, x+w, y+h, h*0.2, hex(0xffffff))
	line(d, x+w, y, x, y+h, h*0.2, hex(0xffffff))
	line(d, x, y, x+w, y+h, h*0.07, hex(0xc8102e))
	line(d, x+w, y, x, y+h, h*0.07, hex(0xc8102e))
	fillRect(d, x+w/2-h*0.17, y, h*0.34, h, hex(0xffffff))
	fillRect(d, x, y+h/2-h*0.17, w, h*0.34, hex(0xffffff))
	fillRect(d, x+w/2-h*0.1, y, h*0.2, h, hex(0xc8102e))
	fillRect(d, x, y+h/2-h*0.1, w, h*0.2, hex(0xc8102e))
}

// drawFlag draws a country flag with a thin frame.
func drawFlag(dst *ebiten.Image, c int, x, y, w, h float32) {
	countries[c].Flag(dst, x, y, w, h)
	strokeRect(dst, x, y, w, h, 1, color.NRGBA{0, 0, 0, 120})
}
