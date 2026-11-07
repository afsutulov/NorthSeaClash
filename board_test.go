package main

import (
	"math/rand/v2"
	"testing"
)

// SetPole must place the whole fleet without ships touching each other.
func TestSetPole(t *testing.T) {
	for n := 0; n < 500; n++ {
		a := newCells()
		ships := SetPole(&a)
		if len(ships) != len(fleetPlan) {
			t.Fatalf("got %d ships", len(ships))
		}
		cells := 0
		for i := range a {
			for j := range a[i] {
				if a[i][j] == '#' {
					cells++
					if i == 0 || j == 0 || i == 11 || j == 11 {
						t.Fatal("ship on the frame")
					}
				}
			}
		}
		if cells != 20 {
			t.Fatalf("got %d ship cells", cells)
		}
		for k, s := range ships {
			b := newCells()
			for m, o := range ships {
				if m != k {
					place(&b, o)
				}
			}
			if !canPlace(&b, s) {
				t.Fatalf("ships touch: %+v", s)
			}
		}
	}
}

// playAI lets the AI sink a random fleet and returns the number of shots.
func playAI(t *testing.T) int {
	p1 := newCells()
	ships := SetPole(&p1)
	prepareField(&p1)
	var ai AI
	seen := map[[2]int]bool{}
	sunk := make([]bool, len(ships))
	shots := 0
	for !Porajenie(&p1) {
		x, y, res := ai.Shot(&p1)
		if seen[[2]int{x, y}] {
			t.Fatalf("AI shot %d,%d twice", x, y)
		}
		seen[[2]int{x, y}] = true
		if x < 1 || x > 10 || y < 1 || y > 10 {
			t.Fatalf("AI shot outside the field: %d,%d", x, y)
		}
		// cells around sunk ships and diagonals of hits are certainly water
		for i, sh := range ships {
			if sunk[i] {
				for _, c := range sh.Cells() {
					if abs(c[0]-x) <= 1 && abs(c[1]-y) <= 1 {
						t.Fatalf("AI shot %d,%d next to a sunk ship", x, y)
					}
				}
			}
		}
		if res == 1 {
			for i, sh := range ships {
				if !sunk[i] && isSunk(&p1, sh) {
					sunk[i] = true
					ai.Sunk(sh)
				}
			}
		}
		if shots++; shots > 100 {
			t.Fatal("AI needs more than 100 shots")
		}
	}
	return shots
}

// The AI must always sink the whole fleet and never shoot a cell twice.
func TestAIFinishes(t *testing.T) {
	for n := 0; n < 300; n++ {
		playAI(t)
	}
}

// legacyShot is the enemy loop of the first, Python version of the game,
// kept here only to measure the improvement.
func legacyShot(p1 *Cells, hod *[][2]int) (int, int, int) {
	for {
		var x, y int
		if len(*hod) == 0 {
			for {
				x, y = rand.IntN(10)+1, rand.IntN(10)+1
				if p1[x][y] != '-' && p1[x][y] != 'o' {
					break
				}
			}
		} else {
			x, y = (*hod)[0][0], (*hod)[0][1]
			*hod = (*hod)[1:]
		}
		switch p1[x][y] {
		case '#':
			for i := x - 1; i < x+2; i++ {
				for j := y - 1; j < y+2; j++ {
					if p1[i][j] != '-' && p1[i][j] != 'X' && ((i == x && j != y) || (j == y && i != x)) {
						*hod = append(*hod, [2]int{i, j})
					}
					if p1[i][j] == ' ' {
						p1[i][j] = '-'
					}
				}
			}
			p1[x][y] = 'X'
			return x, y, 1
		case ' ', '-':
			p1[x][y] = 'o'
			return x, y, 2
		}
	}
}

// The new AI must need clearly fewer shots than the first version.
func TestAIBetterThanLegacy(t *testing.T) {
	const games = 2000
	newTotal, oldTotal := 0, 0
	for n := 0; n < games; n++ {
		newTotal += playAI(t)
		p1 := newCells()
		SetPole(&p1)
		prepareField(&p1)
		var hod [][2]int
		for !Porajenie(&p1) {
			legacyShot(&p1, &hod)
			oldTotal++
		}
	}
	nw, old := float64(newTotal)/games, float64(oldTotal)/games
	t.Logf("average shots to sink the fleet: first version %.1f, new AI %.1f", old, nw)
	if nw > old-5 {
		t.Fatalf("new AI is not better: %.1f vs %.1f", nw, old)
	}
}

// A whole battle, played through the battle scene without graphics.
func TestBattle(t *testing.T) {
	cfg.Sound = false
	g := &Game{snd: newSound(true), store: &Store{path: t.TempDir() + "/save.json"}}
	g.cap = g.store.Captain("Tester")
	for round := 0; round < 20; round++ {
		c := newCells()
		s := newBattle(g, SetPole(&c))
		for steps := 0; s.state != stOver && steps < 100000; steps++ {
			if s.state == stPlayer {
				for r := 1; r <= 10; r++ {
					for c := 1; c <= 10; c++ {
						if s.state == stPlayer && s.canFire(r, c) {
							s.fire(g, r, c)
						}
					}
				}
			}
			s.Update(g)
		}
		if s.state != stOver {
			t.Fatal("battle did not finish")
		}
		for i := 1; i <= 10; i++ {
			for j := 1; j <= 10; j++ {
				if s.known[i][j] && s.p2[i][j] == '#' {
					t.Fatalf("cell %d,%d is marked as water but holds a ship", i, j)
				}
			}
		}
	}
	if g.cap.Games != 20 || g.cap.Wins+0 > 20 {
		t.Fatalf("bad record: %+v", g.cap)
	}
	if cp := g.cap.Campaign[0]; cp.Front > campaignPorts || cp.Front < -campaignPorts {
		t.Fatalf("front out of range: %+v", cp)
	}
}

// Once two neighbouring cells of a ship are hit, the AI must never shoot
// across the ship's line next to it.
func TestAIFollowsLine(t *testing.T) {
	for n := 0; n < 300; n++ {
		p1 := newCells()
		ships := SetPole(&p1)
		prepareField(&p1)
		var ai AI
		sunk := make([]bool, len(ships))
		for !Porajenie(&p1) {
			x, y, res := ai.Shot(&p1)
			for i, sh := range ships {
				if sunk[i] || sh.L < 2 {
					continue
				}
				hits := 0
				for _, c := range sh.Cells() {
					if p1[c[0]][c[1]] == 'X' && !(c[0] == x && c[1] == y) {
						hits++
					}
				}
				if hits >= 2 && !sh.Has(x, y) {
					for _, c := range sh.Cells() {
						if abs(c[0]-x)+abs(c[1]-y) == 1 && ((sh.V && c[1] != y) || (!sh.V && c[0] != x)) {
							t.Fatalf("shot %d,%d across ship %+v", x, y, sh)
						}
					}
				}
			}
			if res == 1 {
				for i, sh := range ships {
					if !sunk[i] && isSunk(&p1, sh) {
						sunk[i] = true
						ai.Sunk(sh)
					}
				}
			}
		}
	}
}

func abs(a int) int {
	if a < 0 {
		return -a
	}
	return a
}

// Enemy launches start from a random cell of the enemy field, so they do not
// reveal ship positions: every cell must be used about equally often.
func TestEnemyLaunchPoint(t *testing.T) {
	count := map[[2]int]int{}
	for n := 0; n < 100000; n++ {
		x, y := enemyLaunchPoint()
		r, c := cellAt(bRX, bY, bCell, int(x), int(y))
		if r == 0 {
			t.Fatalf("launch point %.0f,%.0f is outside the enemy field", x, y)
		}
		count[[2]int{r, c}]++
	}
	if len(count) != 100 {
		t.Fatalf("only %d cells used", len(count))
	}
	for k, v := range count {
		if v < 800 || v > 1200 {
			t.Fatalf("cell %v used %d times of 100000", k, v)
		}
	}
}

// The campaign is a single front line: victories push it towards the enemy,
// defeats push it back, and lost ports are liberated before enemy ports fall.
func TestCampaignFront(t *testing.T) {
	cfg.Sound = false
	g := &Game{snd: newSound(true), store: &Store{path: t.TempDir() + "/save.json"}}
	g.cap = g.store.Captain("Tester")
	own, en := coalitions[0].Ports, coalitions[1].Ports
	battle := func(win bool) *Battle {
		c := newCells()
		s := newBattle(g, SetPole(&c))
		s.end(g, win)
		return s
	}
	check := func(front int, target Port, liberate bool) {
		t.Helper()
		if f := g.cap.Campaign[0].Front; f != front {
			t.Fatalf("front %d, want %d", f, front)
		}
		fr := campaignFront(g)
		if fr.Target != target || fr.Liberate != liberate {
			t.Fatalf("front %d: target %s liberate %v", front, fr.Target.EN, fr.Liberate)
		}
	}
	check(0, en[0], false)
	battle(true) // capture the first enemy port
	check(1, en[1], false)
	if portOwner(g, 1, 0) != 0 {
		t.Fatal("captured port has the wrong owner")
	}
	battle(false) // the enemy recaptures it
	check(0, en[0], false)
	battle(false) // the enemy takes our first port
	check(-1, own[0], true)
	battle(false)
	check(-2, own[1], true)
	if fr := campaignFront(g); fr.Staging != own[2] || fr.Loss != own[2] {
		t.Fatalf("staging %s loss %s", fr.Staging.EN, fr.Loss.EN)
	}
	battle(true) // liberate our second port first
	check(-1, own[0], true)
	battle(true) // liberate the first one
	check(0, en[0], false)
	for i := 1; i <= 4; i++ {
		if battle(true).campaignDone {
			t.Fatal("campaign ended too early")
		}
		check(i, en[i], false)
	}
	if s := battle(true); !s.campaignDone || !campaignOver(g) || g.cap.CampaignsWon != 1 {
		t.Fatal("campaign must be won after the fifth enemy port")
	}
	if f := g.cap.Campaign[0].Front; f != campaignPorts {
		t.Fatalf("front %d must stay at %d until the end screen", f, campaignPorts)
	}
	// defeat: five losses in a row from the start
	g.cap.Campaign[0] = Campaign{}
	for i := 0; i < 4; i++ {
		if battle(false).campaignDone {
			t.Fatal("campaign lost too early")
		}
	}
	if s := battle(false); !s.campaignDone || g.cap.Campaign[0].Front != -campaignPorts {
		t.Fatal("campaign must be lost after the fifth own port")
	}
}

// Saves of the previous version (separate counters) become a front line.
func TestCampaignMigration(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", dir)
	t.Setenv("HOME", dir)
	st := &Store{path: storePath()}
	st.Captains = []*Captain{{Name: "Old", Campaign: [2]Campaign{{Taken: 3, Lost: 1}}}}
	st.Save()
	got := loadStore()
	if cp := got.Captains[0].Campaign[0]; cp.Front != 2 || cp.Taken != 0 || cp.Lost != 0 {
		t.Fatalf("migration: %+v", cp)
	}
}
