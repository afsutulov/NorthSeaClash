package main

import "math/rand/v2"

// Cells is a 12x12 field: the playing area is rows/columns 1..10, the frame
// around it lets the neighbourhood checks run without bounds tests.
// Cell codes are the same as in the first, Python version of the game:
//
//	' ' empty water, '#' ship, 'X' hit, 'o' miss, '-' excluded by the enemy AI
type Cells [12][12]byte

// Ship is a ship on the field. R, C are the 1-based row and column of its
// first cell; vertical ships grow downwards, horizontal ones to the right.
type Ship struct {
	R, C, L int
	V       bool
}

// Fleet composition: one 4-cell, two 3-cell, three 2-cell and four 1-cell ships.
var fleetPlan = []int{4, 3, 3, 2, 2, 2, 1, 1, 1, 1}

func newCells() (a Cells) {
	for i := range a {
		for j := range a[i] {
			a[i][j] = ' '
		}
	}
	return
}

func (s Ship) Cells() [][2]int {
	r := make([][2]int, s.L)
	for w := 0; w < s.L; w++ {
		if s.V {
			r[w] = [2]int{s.R + w, s.C}
		} else {
			r[w] = [2]int{s.R, s.C + w}
		}
	}
	return r
}

func (s Ship) Has(x, y int) bool {
	for _, c := range s.Cells() {
		if c[0] == x && c[1] == y {
			return true
		}
	}
	return false
}

// SetPole fills the field with ships: from 4-cell down to 1-cell ships, at
// random positions, never touching each other (algorithm of the Python version).
func SetPole(a *Cells) (ships []Ship) {
	for z := 4; z > 0; z-- { // From 4-cell ships down to 1-cell ships
		for k := 0; k < 5-z; k++ { // Number of ships depends on their size
			for { // Try random positions until the ship fits
				var x, y int
				Flag2 := rand.IntN(2) == 1 // true - horizontal, false - vertical
				if Flag2 {
					x = rand.IntN(10) + 1
					y = rand.IntN(11-z) + 1
				} else {
					x = rand.IntN(11-z) + 1
					y = rand.IntN(10) + 1
				}
				s := Ship{R: x, C: y, L: z, V: !Flag2}
				if canPlace(a, s) { // No other ship around - place it
					place(a, s)
					ships = append(ships, s)
					break
				}
			}
		}
	}
	return
}

// canPlace reports whether the ship fits on the field without touching other
// ships (the same neighbourhood test SetPole uses).
func canPlace(a *Cells, s Ship) bool {
	if s.R < 1 || s.C < 1 || (s.V && s.R+s.L-1 > 10) || (!s.V && s.C+s.L-1 > 10) || s.R > 10 || s.C > 10 {
		return false
	}
	h, w := 1, s.L
	if s.V {
		h, w = s.L, 1
	}
	for i := s.R - 1; i < s.R+h+1; i++ {
		for j := s.C - 1; j < s.C+w+1; j++ {
			if a[i][j] == '#' {
				return false
			}
		}
	}
	return true
}

func place(a *Cells, s Ship) {
	for _, c := range s.Cells() {
		a[c[0]][c[1]] = '#'
	}
}

// Porajenie reports a defeat: no ship cell is left on the field.
func Porajenie(a *Cells) bool {
	for i := range a {
		for j := range a[i] {
			if a[i][j] == '#' {
				return false
			}
		}
	}
	return true
}

// isSunk reports whether every cell of the ship has been hit.
func isSunk(a *Cells, s Ship) bool {
	for _, c := range s.Cells() {
		if a[c[0]][c[1]] != 'X' {
			return false
		}
	}
	return true
}

// prepareField marks the frame of the player's field with '-' (as Start did
// in the first, Python version of the game).
func prepareField(p1 *Cells) {
	for i := range p1 {
		for j := range p1[i] {
			if i == 0 || j == 0 || i == len(p1)-1 || j == len(p1)-1 {
				p1[i][j] = '-'
			}
		}
	}
}

// What the enemy knows about each cell of the player's field.
const (
	kUnknown = iota // not shot yet, nothing known
	kMiss           // shot, water
	kHit            // shot, part of a ship that is still afloat
	kEmpty          // not shot, but certainly water
	kSunk           // part of a ship reported sunk
)

// AI is the enemy commander. It plays under exactly the same conditions as
// the player: it never sees the player's field. It only names a target
// (Next) and is told the answer — miss or hit (Result) — and, like the player,
// it is told when a ship has been sunk (Sunk). The battle acts as the referee.
//
//   - Ships are straight and never touch, so the diagonal neighbours of a
//     hit and every cell around a sunk ship are certainly water. (The player
//     can deduce the same; on screen only the water around sunk ships is marked.)
//   - Search: for every cell it counts how many placements of the ships that
//     are still afloat pass through it, and shoots at one of the most likely
//     cells. Gaps too small for any remaining ship are never shot.
//   - Finishing: after a hit it only considers placements of the remaining
//     ships that cover all hits of the wounded ship. That keeps the shots on
//     the ship's line and tries the more likely end first.
type AI struct {
	k      [12][12]int
	remain []int // lengths of ships still afloat (known from the sinking reports)
	ready  bool
}

func (ai *AI) init() {
	for i := 0; i < 12; i++ {
		for j := 0; j < 12; j++ {
			if i == 0 || j == 0 || i == 11 || j == 11 {
				ai.k[i][j] = kEmpty
			}
		}
	}
	ai.remain = append([]int(nil), fleetPlan...)
	ai.ready = true
}

// Next chooses the next target from the AI's own knowledge only.
func (ai *AI) Next() (x, y int) {
	if !ai.ready {
		ai.init()
	}
	return ai.choose()
}

// Result tells the AI the answer to its shot at (x, y).
func (ai *AI) Result(x, y int, hit bool) {
	if !ai.ready {
		ai.init()
	}
	if !hit {
		ai.k[x][y] = kMiss
		return
	}
	ai.k[x][y] = kHit
	for _, d := range [][2]int{{-1, -1}, {-1, 1}, {1, -1}, {1, 1}} {
		if ai.k[x+d[0]][y+d[1]] == kUnknown {
			ai.k[x+d[0]][y+d[1]] = kEmpty
		}
	}
}

// referee resolves a shot at the player's field: it marks the cell and
// returns whether a ship was hit. Only the answer is passed to the AI.
func referee(p1 *Cells, x, y int) bool {
	if p1[x][y] == '#' {
		p1[x][y] = 'X'
		return true
	}
	p1[x][y] = 'o'
	return false
}

// Shot is one enemy turn: the AI names a target, the referee answers.
// It returns the cell and the outcome: 1 hit, 2 miss.
func (ai *AI) Shot(p1 *Cells) (x, y, res int) {
	x, y = ai.Next()
	hit := referee(p1, x, y)
	ai.Result(x, y, hit)
	if hit {
		return x, y, 1
	}
	return x, y, 2
}

// Sunk tells the AI that a ship has been sunk (the battle announces it).
func (ai *AI) Sunk(s Ship) {
	if !ai.ready {
		ai.init()
	}
	for _, c := range s.Cells() {
		for i := c[0] - 1; i <= c[0]+1; i++ {
			for j := c[1] - 1; j <= c[1]+1; j++ {
				if ai.k[i][j] == kUnknown {
					ai.k[i][j] = kEmpty
				}
			}
		}
	}
	for _, c := range s.Cells() {
		ai.k[c[0]][c[1]] = kSunk
	}
	for i, l := range ai.remain {
		if l == s.L {
			ai.remain = append(ai.remain[:i], ai.remain[i+1:]...)
			break
		}
	}
}

// placements calls f for every straight segment of length l on the field.
func placements(l int, f func(cells [][2]int)) {
	for r := 1; r <= 10; r++ {
		for c := 1; c <= 10; c++ {
			if c+l-1 <= 10 {
				f(Ship{R: r, C: c, L: l}.Cells())
			}
			if l > 1 && r+l-1 <= 10 {
				f(Ship{R: r, C: c, L: l, V: true}.Cells())
			}
		}
	}
}

func (ai *AI) choose() (int, int) {
	var score [12][12]float64
	// Finishing mode: the wounded ship (connected hits that are not sunk yet).
	var wound [][2]int
	for i := 1; i <= 10 && wound == nil; i++ {
		for j := 1; j <= 10 && wound == nil; j++ {
			if ai.k[i][j] == kHit {
				wound = ai.cluster(i, j)
			}
		}
	}
	if wound != nil {
		inWound := map[[2]int]bool{}
		for _, c := range wound {
			inWound[c] = true
		}
		for _, l := range ai.remain {
			if l < len(wound) {
				continue
			}
			placements(l, func(cells [][2]int) {
				covered, extra := 0, 0
				in := map[[2]int]bool{}
				for _, c := range cells {
					in[c] = true
					switch ai.k[c[0]][c[1]] {
					case kUnknown:
					case kHit:
						if inWound[c] {
							covered++
						} else {
							extra++ // another hit: then both belong to this ship
						}
					default:
						return
					}
				}
				if covered < len(wound) {
					return
				}
				// ships never touch: no other hit may lie next to this placement
				for _, c := range cells {
					for i := c[0] - 1; i <= c[0]+1; i++ {
						for j := c[1] - 1; j <= c[1]+1; j++ {
							if ai.k[i][j] == kHit && !in[[2]int{i, j}] {
								return
							}
						}
					}
				}
				w := 1.0 + 25*float64(extra) // placements explaining more hits are far more likely
				for _, c := range cells {
					if ai.k[c[0]][c[1]] == kUnknown {
						score[c[0]][c[1]] += w
					}
				}
			})
		}
		if x, y, ok := pickBest(&score, 1); ok {
			return x, y
		}
		score = [12][12]float64{}
	}
	// Search mode: probability density of the ships still afloat.
	for _, l := range ai.remain {
		placements(l, func(cells [][2]int) {
			for _, c := range cells {
				if ai.k[c[0]][c[1]] != kUnknown {
					return
				}
			}
			for _, c := range cells {
				score[c[0]][c[1]]++
			}
		})
	}
	if x, y, ok := pickBest(&score, 0.85); ok {
		return x, y
	}
	for { // fallback, should not happen
		x, y := rand.IntN(10)+1, rand.IntN(10)+1
		if ai.k[x][y] == kUnknown {
			return x, y
		}
	}
}

// cluster returns the hits connected to (x, y).
func (ai *AI) cluster(x, y int) [][2]int {
	seen := map[[2]int]bool{{x, y}: true}
	out := [][2]int{{x, y}}
	for i := 0; i < len(out); i++ {
		c := out[i]
		for _, d := range [][2]int{{-1, 0}, {1, 0}, {0, -1}, {0, 1}} {
			n := [2]int{c[0] + d[0], c[1] + d[1]}
			if ai.k[n[0]][n[1]] == kHit && !seen[n] {
				seen[n] = true
				out = append(out, n)
			}
		}
	}
	return out
}

// pickBest returns a random cell among those scoring at least share*max.
func pickBest(score *[12][12]float64, share float64) (int, int, bool) {
	best := 0.0
	for i := 1; i <= 10; i++ {
		for j := 1; j <= 10; j++ {
			best = max(best, score[i][j])
		}
	}
	if best <= 0 {
		return 0, 0, false
	}
	var cand [][2]int
	for i := 1; i <= 10; i++ {
		for j := 1; j <= 10; j++ {
			if score[i][j] >= best*share {
				cand = append(cand, [2]int{i, j})
			}
		}
	}
	c := cand[rand.IntN(len(cand))]
	return c[0], c[1], true
}
