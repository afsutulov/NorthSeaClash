package main

import (
	"encoding/json"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

type Settings struct {
	Lang          int     // 0 English, 1 Russian
	Sound         bool    // sound effects
	Music         bool    // background music
	MusicVol      float64 // 0..1
	SfxVol        float64 // 0..1
	LastCaptain   string
	LastCoalition int
	Fullscreen    bool
}

// Campaign progress: the front line (see campaign.go).
type Campaign struct {
	Front       int
	Taken, Lost int `json:",omitempty"` // saves of the previous version
}

type Captain struct {
	Name         string
	Games        int
	Wins         int
	Fires        int // shots fired in all battles
	Best         int // fewest shots in a won battle (0 = none yet)
	CampaignsWon int
	Campaign     [2]Campaign
}

type BattleResult struct {
	Name      string
	Fires     int
	Coalition int
	Port      string
	PortRU    string
	Date      string
}

type Store struct {
	Settings Settings
	Captains []*Captain
	Best     []BattleResult
	path     string
}

// defaults: English, sound and music on, Northern Alliance.
var cfg = &Settings{Lang: 0, Sound: true, Music: true, MusicVol: 0.55, SfxVol: 0.9, LastCaptain: "", LastCoalition: 0}

func storePath() string {
	dir, err := os.UserConfigDir()
	if err != nil {
		dir = "."
	}
	return filepath.Join(dir, "NorthSeaClash", "save.json")
}

func loadStore() *Store {
	s := &Store{path: storePath(), Settings: *cfg}
	if b, err := os.ReadFile(s.path); err == nil {
		_ = json.Unmarshal(b, s)
	}
	if s.Settings.MusicVol == 0 && s.Settings.SfxVol == 0 {
		s.Settings.MusicVol, s.Settings.SfxVol = cfg.MusicVol, cfg.SfxVol
	}
	*cfg = s.Settings
	for _, c := range s.Captains {
		for i := range c.Campaign {
			if cp := &c.Campaign[i]; cp.Taken != 0 || cp.Lost != 0 {
				cp.Front = max(-campaignPorts, min(campaignPorts, cp.Taken-cp.Lost))
				cp.Taken, cp.Lost = 0, 0
			}
		}
	}
	return s
}

func (s *Store) Save() {
	s.Settings = *cfg
	b, err := json.MarshalIndent(s, "", "  ")
	if err != nil {
		return
	}
	_ = os.MkdirAll(filepath.Dir(s.path), 0o755)
	_ = os.WriteFile(s.path, b, 0o644)
}

// Captain returns the card of the named captain, creating it if needed.
func (s *Store) Captain(name string) *Captain {
	for _, c := range s.Captains {
		if strings.EqualFold(c.Name, name) {
			return c
		}
	}
	c := &Captain{Name: name}
	s.Captains = append(s.Captains, c)
	return c
}

// TopCaptains returns the six best captains by victories.
func (s *Store) TopCaptains() []*Captain {
	l := append([]*Captain(nil), s.Captains...)
	sort.SliceStable(l, func(i, j int) bool {
		if l[i].Wins != l[j].Wins {
			return l[i].Wins > l[j].Wins
		}
		return l[i].Games < l[j].Games
	})
	var r []*Captain
	for _, c := range l {
		if c.Games > 0 && len(r) < 6 {
			r = append(r, c)
		}
	}
	return r
}

// AddResult records a won battle in the "fewest shots" table (top 6).
func (s *Store) AddResult(r BattleResult) {
	r.Date = time.Now().Format("2006-01-02")
	s.Best = append(s.Best, r)
	sort.SliceStable(s.Best, func(i, j int) bool { return s.Best[i].Fires < s.Best[j].Fires })
	if len(s.Best) > 6 {
		s.Best = s.Best[:6]
	}
}
