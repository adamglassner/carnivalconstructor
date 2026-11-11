package main

import (
	"encoding/json"
	"os"
	"path/filepath"
	"time"

	cv "github.com/adamglassner/carnivalconstructor/internal/carnival"
)

type saveData struct {
	Settings struct {
		Music bool `json:"music"`
		Sfx   bool `json:"sfx"`
		Vol   int  `json:"vol"`
		Tips  bool `json:"tipsSeen"`
	} `json:"settings"`
	Stats struct {
		Runs       int `json:"runs"`
		BestNight  int `json:"bestNight"`
		Nights     int `json:"nights"`
		Guests     int `json:"guests"`
		BestProfit int `json:"bestProfit"`
		BestCash   int `json:"bestCash"`
	} `json:"stats"`
	Unlocked map[string]int64 `json:"achievements"`
}

func defaultSave() saveData {
	var s saveData
	s.Settings.Music, s.Settings.Sfx, s.Settings.Vol = true, true, 60
	s.Unlocked = map[string]int64{}
	return s
}

// appDir is %APPDATA%\Ironite on Windows.
func appDir() string {
	if d := os.Getenv("CC_DATA"); d != "" { // test builds
		return d
	}
	d, err := os.UserConfigDir()
	if err != nil {
		return ""
	}
	return filepath.Join(d, "Ironite")
}

func loadSave() saveData {
	s := defaultSave()
	if dir := appDir(); dir != "" {
		if b, err := os.ReadFile(filepath.Join(dir, bundleName, "save.json")); err == nil {
			_ = json.Unmarshal(b, &s)
		}
	}
	if s.Unlocked == nil {
		s.Unlocked = map[string]int64{}
	}
	s.Settings.Vol = min(100, max(0, s.Settings.Vol))
	return s
}

func writeJSON(path string, v any) {
	b, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		return
	}
	_ = os.MkdirAll(filepath.Dir(path), 0o755)
	if os.WriteFile(path+".tmp", b, 0o644) == nil {
		_ = os.Rename(path+".tmp", path)
	}
}

func (g *game) persist() {
	if dir := appDir(); dir != "" {
		writeJSON(filepath.Join(dir, bundleName, "save.json"), g.save)
		g.writeAchievements()
	}
}

/* ---------- achievements (shown in Ironite Launcher too) ---------- */

type achievement struct {
	ID, Name, Desc string
	Hidden         bool
}

var achievements = []achievement{
	{"opening", "Grand Opening", "Open the gates for your first night.", false},
	{"fullmidway", "Full Midway", "Run all three midway games at once.", false},
	{"crowd", "Step Right Up", "Welcome 120 guests in a single night.", false},
	{"pleaser", "Crowd Pleaser", "Finish a night with guests 80% happy or better.", false},
	{"houseedge", "The House Always Wins", "Take $400 at the midway games in one night.", false},
	{"honest", "Honest Carny", "Make it through three nights without ever rigging a game.", false},
	{"wheel", "Big Wheel", "Build a Ferris Wheel.", false},
	{"roadready", "Opening Weekend", "Make it through three nights at Pine Hollow.", false},
	{"regular", "Season Regular", "Keep the carnival running for ten nights.", false},
	{"bankroll", "Bankroll", "Have $25,000 in cash at the end of a night.", false},
	{"paved", "Paved Paradise", "Lay 40 tiles of footpath.", false},
	{"caught", "Caught Red-Handed", "Have a guest call one of your games rigged.", true},
}

// unlock records an achievement; it returns true the first time.
func (g *game) unlock(id string) bool {
	if _, ok := g.save.Unlocked[id]; ok {
		return false
	}
	g.save.Unlocked[id] = time.Now().Unix()
	for _, a := range achievements {
		if a.ID == id {
			g.toast("Achievement: "+a.Name, a.Desc, "ach")
			snd.play("fanfare")
		}
	}
	g.persist()
	return true
}

func (g *game) writeAchievements() {
	dir := appDir()
	if dir == "" {
		return
	}
	type item struct {
		ID          string `json:"id"`
		Name        string `json:"name"`
		Description string `json:"description"`
		Category    string `json:"category"`
		Hidden      bool   `json:"hidden"`
		Unlocked    bool   `json:"unlocked"`
		UnlockedAt  int64  `json:"unlocked_at"`
		Progress    int    `json:"progress"`
		Target      int    `json:"target"`
	}
	out := struct {
		Schema       int    `json:"schema"`
		Game         string `json:"game"`
		Title        string `json:"title"`
		Updated      int64  `json:"updated"`
		Achievements []item `json:"achievements"`
	}{Schema: 1, Game: gameID, Title: productName, Updated: time.Now().Unix()}
	for _, a := range achievements {
		at, ok := g.save.Unlocked[a.ID]
		it := item{ID: a.ID, Name: a.Name, Description: a.Desc, Category: "Demo", Hidden: a.Hidden, Unlocked: ok, UnlockedAt: at, Target: 1}
		if ok {
			it.Progress = 1
		}
		out.Achievements = append(out.Achievements, it)
	}
	writeJSON(filepath.Join(dir, "Achievements", gameID+".json"), out)
}

/* ---------- the run in progress ---------- */

func runPath() string {
	if dir := appDir(); dir != "" {
		return filepath.Join(dir, bundleName, "run.json")
	}
	return ""
}

// saveRun keeps the run as it stands this morning, so Continue can pick it up.
func (g *game) saveRun() {
	path := runPath()
	if path == "" || g.sim == nil {
		return
	}
	if ph := g.sim.Phase; ph != cv.PhaseBuild && !(ph == cv.PhaseResults && g.sim.Cash >= -1500) {
		return
	}
	b, err := json.Marshal(g.sim.Save())
	if err != nil || string(b) == g.savedAt {
		return
	}
	g.savedAt = string(b)
	writeJSON(path, json.RawMessage(b))
}

func (g *game) deleteRun() {
	if path := runPath(); path != "" {
		_ = os.Remove(path)
	}
	g.savedAt = ""
}

// savedRun reads the run in progress, if there is one.
func savedRun() (cv.Saved, bool) {
	var sv cv.Saved
	path := runPath()
	if path == "" {
		return sv, false
	}
	b, err := os.ReadFile(path)
	if err != nil || json.Unmarshal(b, &sv) != nil || sv.Night < 1 {
		return sv, false
	}
	return sv, true
}

func (g *game) continueRun() {
	sv, ok := savedRun()
	if !ok {
		g.newRun()
		return
	}
	s, err := cv.Load(sv, time.Now().UnixNano())
	if err != nil {
		g.newRun()
		return
	}
	g.sim = s
	g.screen, g.panel = "play", ""
	g.placing, g.sel, g.selGuest, g.pathTool = nil, nil, nil, 0
	g.paused, g.speed, g.tab = false, 1, cv.CatRide
	g.floaters, g.toasts, g.log = nil, nil, nil
	g.rigged = false
	for _, a := range s.Attr {
		if a.Rig > 0.001 {
			g.rigged = true
		}
	}
	resultsShown[s] = s.Night - 1
	g.note("Welcome back! Night " + itoa(s.Night) + " at " + s.Fair.Name + ".")
}

// run caches the saved run for the title screen (it's read once a second at most).
var runCache struct {
	at time.Time
	sv cv.Saved
	ok bool
}

func (g *game) run() (cv.Saved, bool) {
	if time.Since(runCache.at) > time.Second {
		runCache.sv, runCache.ok = savedRun()
		runCache.at = time.Now()
	}
	return runCache.sv, runCache.ok
}
