package main

import (
	"math"
	"strings"
	"time"

	cv "github.com/adamglassner/carnivalconstructor/internal/carnival"
	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/inpututil"
)

type hit struct {
	x, y, w, h float64
	fn         func()
	drag       func(mx float64) // sliders
}

type floater struct {
	x, y, t float64
	text    string
	good    bool
}

type toastMsg struct {
	title, body, kind string
	t                 float64
	a                 *cv.Attraction
}

type game struct {
	sim      *cv.Sim
	save     saveData
	screen   string // "title" or "play"
	panel    string // "", "menu", "help", "settings"
	speed    int    // 1, 2 or 3
	paused   bool
	tab      cv.Cat
	placing  *cv.Def
	sel      *cv.Attraction
	selGuest *cv.Guest

	hits     []hit
	dragging func(mx float64)
	mx, my   float64
	scale    float64
	offX     float64
	offY     float64

	clock      float64
	floaters   []floater
	toasts     []toastMsg
	log        []string
	rigged     bool // a game was rigged at some point this run
	quit       bool
	canvas     *ebiten.Image
	fixedMouse bool // screenshots
	heroShot   bool
	runStart   time.Time
}

func newGame() *game {
	g := &game{save: loadSave(), screen: "title", speed: 1, scale: 1}
	return g
}

func (g *game) newRun() {
	g.sim = cv.New(time.Now().UnixNano())
	g.screen = "play"
	g.panel = ""
	g.placing, g.sel, g.selGuest = nil, nil, nil
	g.paused, g.speed = false, 1
	g.tab = cv.CatRide
	g.floaters, g.toasts, g.log = nil, nil, nil
	g.rigged = false
	g.runStart = time.Now()
	g.save.Stats.Runs++
	g.persist()
	g.note("Welcome to " + g.sim.Fair.Name + "!")
}

// applyTestQuery sets up a scene for automated screenshots.
func (g *game) applyTestQuery(q string) {
	if q == "" {
		return
	}
	has := func(k string) bool { return strings.Contains(q, k) }
	g.heroShot = has("hero")
	if has("panel=help") {
		g.panel = "help"
	}
	if has("panel=settings") {
		g.panel = "settings"
	}
	if !has("play") && !has("night") && !has("results") && !has("done") {
		return
	}
	g.newRun()
	s := g.sim
	s.Rand.Seed(5)
	if has("build") {
		s.Cash = 1 << 20
		for _, b := range []struct {
			id   string
			x, y int
		}{{"carousel", 3, 1}, {"ferris", 8, 1}, {"teacups", 14, 2}, {"ringtoss", 4, 7}, {"darts", 7, 8}, {"duckpond", 10, 8},
			{"corndogs", 15, 8}, {"funnel", 18, 8}, {"lemonade", 19, 3}, {"lamp", 13, 6}, {"lamp", 3, 11}, {"trash", 9, 11}, {"trash", 14, 11},
			{"bench", 12, 11}, {"bench", 20, 11}, {"lamp", 22, 7}} {
			s.Place(cv.DefByID(b.id), b.x, b.y)
		}
		s.Cash = 3120
	}
	if has("night") || has("results") || has("done") {
		s.OpenGates()
		target := 0.55
		if has("late") {
			target = 0.85
		}
		for s.T < target {
			s.Update(0.05)
		}
		s.Events = nil
		if len(s.Attr) > 0 {
			for _, a := range s.Attr {
				if a.Def.ID == "ringtoss" {
					s.SetRig(a, 0.7)
					g.sel = a
				}
			}
		}
	}
	if has("results") || has("done") {
		for s.Phase == cv.PhaseNight {
			s.Update(0.1)
		}
		s.Events = nil
	}
	if has("done") {
		for s.Phase != cv.PhaseDone {
			s.Next()
			if s.Phase == cv.PhaseBuild {
				s.OpenGates()
				for s.Phase == cv.PhaseNight {
					s.Update(0.1)
				}
			}
		}
	}
	if has("place") {
		g.placing = cv.DefByID("ferris")
		g.mx, g.my = 640, 420
		g.fixedMouse = true
	}
	if has("guest") && len(s.Guests) > 0 {
		g.sel = nil
		g.selGuest = s.Guests[len(s.Guests)/2]
	}
	if has("paused") {
		g.paused = true
	}
	if has("tab=food") {
		g.tab = cv.CatFood
	}
}

func (g *game) Layout(ow, oh int) (int, int) {
	k := ebiten.Monitor().DeviceScaleFactor()
	return int(float64(ow) * k), int(float64(oh) * k)
}

func (g *game) Update() error {
	if g.quit {
		return ebiten.Termination
	}
	const dt = 1.0 / 60
	g.clock += dt
	if !g.fixedMouse {
		cx, cy := ebiten.CursorPosition()
		g.mx, g.my = float64(cx)/g.scale-g.offX, float64(cy)/g.scale-g.offY
	}

	g.keys()
	g.mouse()

	if g.screen == "play" && g.sim != nil {
		s := g.sim
		if s.Phase == cv.PhaseNight && g.panel == "" && !g.paused {
			mul := []float64{1, 1, 2, 4}[g.speed]
			s.Update(dt * mul)
			for _, a := range s.Attr {
				if a.Rig > 0.001 {
					g.rigged = true
				}
			}
		}
		g.events()
		if g.selGuest != nil && !g.guestAlive(g.selGuest) {
			g.selGuest = nil
		}
	}
	for i := range g.floaters {
		g.floaters[i].t += dt
		g.floaters[i].y -= dt * 22
	}
	for len(g.floaters) > 0 && g.floaters[0].t > 1.4 {
		g.floaters = g.floaters[1:]
	}
	keep := g.toasts[:0]
	for _, t := range g.toasts {
		t.t += dt
		if t.t < 6 {
			keep = append(keep, t)
		}
	}
	g.toasts = keep

	level := 1.0
	if g.screen == "play" && g.sim != nil {
		switch g.sim.Phase {
		case cv.PhaseBuild:
			level = 0.55
		case cv.PhaseNight:
			level = 1
			if g.paused {
				level = 0.35
			}
		default:
			level = 0.45
		}
	}
	snd.update(dt, level)
	return nil
}

func (g *game) guestAlive(gu *cv.Guest) bool {
	for _, x := range g.sim.Guests {
		if x == gu {
			return true
		}
	}
	return false
}

func (g *game) keys() {
	if inpututil.IsKeyJustPressed(ebiten.KeyF11) || (inpututil.IsKeyJustPressed(ebiten.KeyEnter) && ebiten.IsKeyPressed(ebiten.KeyAlt)) {
		ebiten.SetFullscreen(!ebiten.IsFullscreen())
	}
	if inpututil.IsKeyJustPressed(ebiten.KeyM) {
		g.save.Settings.Music = !g.save.Settings.Music
		snd.setMusic(g.save.Settings.Music)
		g.persist()
	}
	if inpututil.IsKeyJustPressed(ebiten.KeyEscape) {
		switch {
		case g.placing != nil:
			g.placing = nil
		case g.panel != "":
			g.panel = ""
		case g.sel != nil || g.selGuest != nil:
			g.sel, g.selGuest = nil, nil
		case g.screen == "play":
			g.panel = "menu"
		}
	}
	if g.screen != "play" || g.panel != "" || g.sim == nil {
		return
	}
	if inpututil.IsKeyJustPressed(ebiten.KeySpace) && g.sim.Phase == cv.PhaseNight {
		g.paused = !g.paused
	}
	for i, k := range []ebiten.Key{ebiten.Key1, ebiten.Key2, ebiten.Key3} {
		if inpututil.IsKeyJustPressed(k) {
			g.speed, g.paused = i+1, false
		}
	}
	if inpututil.IsMouseButtonJustPressed(ebiten.MouseButtonRight) {
		g.placing = nil
	}
	if g.sel != nil && (inpututil.IsKeyJustPressed(ebiten.KeyDelete)) {
		g.sellSelected()
	}
}

func (g *game) mouse() {
	if g.dragging != nil {
		if ebiten.IsMouseButtonPressed(ebiten.MouseButtonLeft) {
			g.dragging(g.mx)
			return
		}
		g.dragging = nil
		g.persist()
	}
	if !inpututil.IsMouseButtonJustPressed(ebiten.MouseButtonLeft) {
		return
	}
	for i := len(g.hits) - 1; i >= 0; i-- {
		h := g.hits[i]
		if g.mx >= h.x && g.my >= h.y && g.mx < h.x+h.w && g.my < h.y+h.h {
			if h.drag != nil {
				g.dragging = h.drag
				h.drag(g.mx)
				return
			}
			if h.fn != nil {
				h.fn()
			}
			return
		}
	}
	if g.screen == "play" && g.panel == "" && g.sim != nil {
		g.clickWorld()
	}
}

func (g *game) clickWorld() {
	s := g.sim
	if s.Phase != cv.PhaseBuild && s.Phase != cv.PhaseNight {
		return
	}
	tx, ty, in := g.mouseTile()
	if g.placing != nil {
		if !in {
			return
		}
		d := g.placing
		a, err := s.Place(d, tx-d.W/2, ty-d.H/2)
		if err != nil {
			snd.play("error")
			return
		}
		snd.play("place")
		x, y := tilePx(float64(a.X)+float64(d.W)/2, float64(a.Y))
		g.floaters = append(g.floaters, floater{x: x, y: y, text: "-" + money(d.Cost)})
		if d.ID == "ferris" {
			g.unlock("wheel")
		}
		g.checkMidway()
		if d.Cat != cv.CatExtra || s.Cash < d.Cost {
			g.placing = nil
			g.sel, g.selGuest = a, nil
		}
		return
	}
	// a guest under the mouse?
	var best *cv.Guest
	bd := 14.0
	for _, gu := range s.Guests {
		if gu.State == cv.Riding {
			continue
		}
		x, y := tilePx(gu.X, gu.Y)
		if d := math.Hypot(g.mx-x, g.my-(y-12)); d < bd {
			best, bd = gu, d
		}
	}
	if best != nil {
		g.selGuest, g.sel = best, nil
		snd.play("click")
		return
	}
	if !in {
		g.sel, g.selGuest = nil, nil
		return
	}
	if a := s.AttrAt(tx, ty); a != nil {
		g.sel, g.selGuest = a, nil
		snd.play("click")
		return
	}
	g.sel, g.selGuest = nil, nil
}

func (g *game) checkMidway() {
	have := map[string]bool{}
	for _, a := range g.sim.Attr {
		have[a.Def.ID] = true
	}
	if have["ringtoss"] && have["darts"] && have["duckpond"] {
		g.unlock("fullmidway")
	}
}

func (g *game) sellSelected() {
	if g.sel == nil {
		return
	}
	a := g.sel
	back := g.sim.Sell(a)
	x, y := tilePx(float64(a.X)+float64(a.Def.W)/2, float64(a.Y))
	g.floaters = append(g.floaters, floater{x: x, y: y, text: "+" + money(back), good: true})
	snd.play("sell")
	g.sel = nil
}

func (g *game) openGates() {
	if err := g.sim.OpenGates(); err != nil {
		g.toast(err.Error(), "", "warn")
		snd.play("error")
		return
	}
	g.placing = nil
	g.paused = false
	g.unlock("opening")
	g.checkMidway()
}

func (g *game) toast(title, body, kind string) {
	g.toasts = append(g.toasts, toastMsg{title: title, body: body, kind: kind})
	if len(g.toasts) > 3 {
		g.toasts = g.toasts[len(g.toasts)-3:]
	}
}

func (g *game) note(s string) {
	g.log = append(g.log, s)
	if len(g.log) > 5 {
		g.log = g.log[len(g.log)-5:]
	}
}

// events turns what happened in the simulation into sounds and pop-ups.
func (g *game) events() {
	s := g.sim
	coins := 0
	for _, e := range s.Events {
		x, y := tilePx(e.X, e.Y)
		switch e.Kind {
		case "coin":
			if len(g.floaters) < 30 {
				g.floaters = append(g.floaters, floater{x: x, y: y - 20, text: e.Text, good: true})
			}
			coins++
		case "win":
			snd.play("win")
		case "rigged":
			snd.play("rigged")
			g.unlock("caught")
			if g.clock-g.lastRigNote() > 10 {
				g.note("A guest thinks your " + e.A.Def.Name + " is rigged!")
			}
		case "broke":
			snd.play("broke")
			g.toasts = append(g.toasts, toastMsg{title: e.Text, body: "Click it and send a mechanic (" + money(cv.RepairCost) + ").", kind: "warn", a: e.A})
			g.note(e.Text)
		case "fixed":
			snd.play("fixed")
			g.note(e.Text)
		case "gate":
			snd.play("gate")
			g.note("The gates are open! Night " + itoa(s.Night) + " has begun.")
		}
	}
	if coins > 0 && int(g.clock*10)%3 == 0 {
		snd.play("coin")
	}
	s.Events = s.Events[:0]
	if s.Phase == cv.PhaseResults && !g.resultsSeen() {
		g.onResults()
	}
}

var rigNoteAt float64

func (g *game) lastRigNote() float64 {
	v := rigNoteAt
	rigNoteAt = g.clock
	return v
}

var resultsShown = map[*cv.Sim]int{}

func (g *game) resultsSeen() bool { return resultsShown[g.sim] >= g.sim.Night }

func (g *game) onResults() {
	s := g.sim
	resultsShown[s] = s.Night
	r := s.Reports[len(s.Reports)-1]
	snd.play("close")
	g.sel, g.selGuest, g.placing = nil, nil, nil
	g.save.Stats.Nights++
	g.save.Stats.Guests += r.Guests
	if r.Profit() > g.save.Stats.BestProfit {
		g.save.Stats.BestProfit = r.Profit()
	}
	g.persist()
	if r.Guests >= 120 {
		g.unlock("crowd")
	}
	if r.AvgHappy >= 80 {
		g.unlock("pleaser")
	}
	if r.Games >= 400 {
		g.unlock("houseedge")
	}
}

// finishDemo is called when the player tears down after the last night.
func (g *game) finishDemo() {
	s := g.sim
	s.Next()
	if s.Phase == cv.PhaseDone {
		g.save.Stats.Finished++
		if s.Cash > g.save.Stats.BestCash {
			g.save.Stats.BestCash = s.Cash
		}
		g.unlock("roadready")
		if !g.rigged {
			g.unlock("honest")
		}
		snd.play("fanfare")
	}
	g.persist()
}

func (g *game) Draw(screen *ebiten.Image) {
	W, H := float64(screen.Bounds().Dx()), float64(screen.Bounds().Dy())
	s := math.Min(W/logicalW, H/logicalH)
	g.scale = s
	g.offX, g.offY = (W/s-logicalW)/2, (H/s-logicalH)/2
	ss := math.Max(1, math.Min(2, 3200/W))
	cw, ch := int(math.Ceil(W*ss)), int(math.Ceil(H*ss))
	if g.canvas == nil || g.canvas.Bounds().Dx() != cw || g.canvas.Bounds().Dy() != ch {
		if g.canvas != nil {
			g.canvas.Deallocate()
		}
		g.canvas = ebiten.NewImage(cw, ch)
	}
	g.canvas.Fill(cNavy)
	p := pen{dst: g.canvas, s: s * ss, ox: g.offX, oy: g.offY, alpha: 1}
	g.hits = g.hits[:0]
	if g.screen == "title" {
		g.drawTitle(p)
	} else {
		g.drawPlay(p)
	}
	switch g.panel {
	case "menu":
		g.drawMenu(p)
	case "help":
		g.drawHelp(p)
	case "settings":
		g.drawSettings(p)
	}
	op := &ebiten.DrawImageOptions{}
	op.GeoM.Scale(1/ss, 1/ss)
	op.Filter = ebiten.FilterLinear
	screen.DrawImage(g.canvas, op)
}
