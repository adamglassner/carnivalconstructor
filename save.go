package main

import (
	"encoding/json"
	"os"
	"path/filepath"
	"time"
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
		Finished   int `json:"finished"`
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
	{"honest", "Honest Carny", "Finish the demo without ever rigging a game.", false},
	{"wheel", "Big Wheel", "Build a Ferris Wheel.", false},
	{"roadready", "Road Ready", "Finish all three nights at Pine Hollow.", false},
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
