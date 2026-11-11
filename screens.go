package main

import (
	"image/color"
	"math"
	"runtime"

	cv "github.com/adamglassner/carnivalconstructor/internal/carnival"
	"github.com/hajimehoshi/ebiten/v2"
)

// scaleAt returns a pen that draws k times bigger with its origin at (x, y).
func scaleAt(p pen, k, x, y float64) pen {
	q := p
	q.s = p.s * k
	q.ox = (p.ox + x) / k
	q.oy = (p.oy + y) / k
	return q
}

func (g *game) dialog(p pen, w, h float64) (float64, float64) {
	p.rect(-400, -400, 2400, 1800, withA(cNight, 0.72))
	g.block()
	x, y := (logicalW-w)/2, (logicalH-h)/2
	p.rrect(x, y+6, w, h, 16, withA(cBlack, 0.4))
	p.rrect(x, y, w, h, 16, cPanel)
	p.rrectStroke(x, y, w, h, 16, 2, cGold)
	return x, y
}

/* ---------- title ---------- */

func (g *game) drawTitle(p pen) {
	t := g.clock
	// sky
	for i := 0; i < 30; i++ {
		c := mix(hex(0x0b0e27), hex(0x3a2560), float64(i)/29)
		p.rect(-400, float64(i)*24, 2240, 25, c)
	}
	for i := 0; i < 90; i++ {
		x := math.Mod(float64(i*211), 1600) - 80
		y := math.Mod(float64(i*137), 520)
		tw := 0.5 + 0.5*math.Sin(t*2+float64(i))
		p.circle(x, y, 1+float64(i%3)*0.5, withA(cCream, 0.3+0.5*tw))
	}
	p.circle(1240, 110, 42, withA(cCream, 0.95))
	p.circle(1258, 98, 38, hex(0x16163d))
	// glow behind the midway
	p.glow(980, 520, 520, cGold, 0.35)
	p.glow(420, 640, 420, cRed, 0.18)
	// the big wheel, the carousel and a booth, drawn large
	wheel := scaleAt(p, 2.6, 1070, 800)
	drawFerris(wheel, 0, 0, 3*TS, t*1.4)
	hx, hy, R := ferrisHub(0, 0, 3*TS)
	for i := 0; i < 24; i++ {
		an := t*1.4*0.32 + float64(i)*2*math.Pi/24
		bx, by := 1070+(hx+math.Cos(an)*R)*2.6, 800+(hy+math.Sin(an)*R)*2.6
		if math.Sin(t*3+float64(i)) > -0.2 {
			c := []color.RGBA{cGold, cRed, cMint}[i%3]
			p.circle(bx, by, 4, mix(c, cWhite, 0.4))
			p.glow(bx, by, 16, c, 0.7)
		}
	}
	// ground
	p.rect(-400, 800, 2240, 200, hex(0x120f2c))
	p.poly(hex(0x2a2140), 560, 800, 880, 800, 1060, 900, 380, 900)
	drawCarousel(scaleAt(p, 2.0, 720, 836), 0, 0, 3*TS, t*2, true, t)
	drawBooth(scaleAt(p, 1.55, 1250, 664), cv.DefByID("ringtoss"), 0, 0, 2*TS, 2*TS, t, nil)
	// string lights
	for i := 0; i < 30; i++ {
		k := float64(i) / 29
		x := -40 + k*1500
		y := 60 + 40*math.Sin(k*math.Pi*3)
		c := []color.RGBA{cGold, cRed, cMint, cPink}[i%4]
		p.circle(x, y, 4, mix(c, cWhite, 0.4))
		p.glow(x, y, 18, c, 0.6)
	}
	// crowd silhouettes
	for i := 0; i < 16; i++ {
		x := math.Mod(float64(i*97), 1440)
		h := 30 + float64(i%3)*8
		p.circle(x, 900-h-6, 9, hex(0x1d1a40))
		p.rrect(x-10, 900-h, 20, h, 8, hex(0x1d1a40))
	}

	if g.heroShot {
		return
	}
	// logo
	lx, ly := 96.0, 170.0
	p.text("Carnival", display(104), lx+6, ly+6, alLeft, cRed)
	p.text("Carnival", display(104), lx, ly, alLeft, cGold)
	p.text("Constructor", display(104), lx+6, ly+112, alLeft, cRed)
	p.text("Constructor", display(104), lx, ly+106, alLeft, cGold)
	p.rrect(lx+4, ly+178, 118, 32, 8, cNavy)
	p.rrectStroke(lx+4, ly+178, 118, 32, 8, 2, cGold)
	p.text("CLASSIC", head(20), lx+63, ly+194, alCenter, cGold)
	p.rrect(lx+132, ly+178, 96, 32, 8, cRed)
	p.text("DEMO", head(20), lx+180, ly+194, alCenter, cCream)
	p.text("Build it. Run it. Pack it up. Roll to the next town.", body(20, 600), lx+244, ly+194, alLeft, cCream)

	bx, by, bw := lx, ly+250, 300.0
	if sv, ok := g.run(); ok {
		g.button(p, bx, by, bw, 58, "CONTINUE · NIGHT "+itoa(sv.Night), bRed, true, func() { g.continueRun() })
		by += 72
		g.button(p, bx, by, bw, 46, "New Game", bDark, true, func() { g.newRun() })
		by += 58
	} else {
		g.button(p, bx, by, bw, 58, "PLAY THE DEMO", bRed, true, func() { g.newRun() })
		by += 72
	}
	g.button(p, bx, by, bw, 46, "How to Play", bDark, true, func() { g.panel = "help" })
	g.button(p, bx, by+58, bw, 46, "Settings", bDark, true, func() { g.panel = "settings" })
	if runtime.GOOS != "js" {
		g.button(p, bx, by+116, bw, 46, "Quit", bDark, true, func() { g.quit = true })
	}
	st := g.save.Stats
	if st.BestNight > 0 {
		p.text("Longest run: "+itoa(st.BestNight)+" nights · Best cash "+money(st.BestCash), body(13, 500), bx, by+186, alLeft, cTan)
	}
	p.text("v"+version+" · Demo · © 2026 Ironite", body(12, 400), 24, 880, alLeft, withA(cTan, 0.8))
	if art.logo != nil {
		op := &ebiten.DrawImageOptions{}
		lw := float64(art.logo.Bounds().Dx())
		k := 120 / lw * p.s
		op.GeoM.Scale(k, k)
		op.GeoM.Translate(float64(p.X(1300)), float64(p.Y(842)))
		op.Filter = ebiten.FilterLinear
		op.ColorScale.ScaleAlpha(0.85)
		p.dst.DrawImage(art.logo, op)
	}
}

/* ---------- menus ---------- */

func (g *game) drawMenu(p pen) {
	x, y := g.dialog(p, 380, 400)
	p.text("Paused", display(34), x+190, y+48, alCenter, cGold)
	bx, bw := x+40, 300.0
	by := y + 96
	g.button(p, bx, by, bw, 48, "Resume", bGold, true, func() { g.panel = "" })
	g.button(p, bx, by+60, bw, 44, "How to Play", bDark, true, func() { g.panel = "help" })
	g.button(p, bx, by+114, bw, 44, "Settings", bDark, true, func() { g.panel = "settings" })
	g.button(p, bx, by+168, bw, 44, "Start Over", bDark, true, func() { g.newRun() })
	g.button(p, bx, by+222, bw, 44, "Quit to Title", bRed, true, func() { g.panel = ""; g.screen = "title"; g.sim = nil })
}

func (g *game) drawSettings(p pen) {
	x, y := g.dialog(p, 460, 380)
	p.text("Settings", display(34), x+230, y+48, alCenter, cGold)
	st := &g.save.Settings
	row := func(yy float64, label string, on bool, fn func()) {
		p.text(label, body(16, 500), x+40, yy+20, alLeft, cCream)
		lbl, sty := "Off", bDark
		if on {
			lbl, sty = "On", bTeal
		}
		g.button(p, x+300, yy, 120, 40, lbl, sty, true, func() { fn(); g.persist() })
	}
	row(y+96, "Music", st.Music, func() { st.Music = !st.Music; snd.setMusic(st.Music) })
	row(y+150, "Sound effects", st.Sfx, func() { st.Sfx = !st.Sfx; snd.setSfx(st.Sfx) })
	p.text("Volume", body(16, 500), x+40, y+224, alLeft, cCream)
	p.rect(x+311, y+223, 14, 2.5, cCream)
	g.button(p, x+300, y+204, 36, 40, "", bDark, true, func() {
		st.Vol = max(0, st.Vol-10)
		snd.setVolume(float64(st.Vol) / 100)
		g.persist()
	})
	p.text(itoa(st.Vol)+"%", head(17), x+360, y+224, alCenter, cGold)
	defer func() {
		p.rect(x+395, y+223, 14, 2.5, cCream)
		p.rect(x+400.75, y+217.25, 2.5, 14, cCream)
	}()
	g.button(p, x+384, y+204, 36, 40, "", bDark, true, func() {
		st.Vol = min(100, st.Vol+10)
		snd.setVolume(float64(st.Vol) / 100)
		g.persist()
	})
	row(y+258, "Full screen (F11)", ebiten.IsFullscreen(), func() { ebiten.SetFullscreen(!ebiten.IsFullscreen()) })
	g.button(p, x+150, y+318, 160, 44, "Done", bGold, true, func() { g.back() })
}

func (g *game) back() {
	if g.screen == "play" {
		g.panel = "menu"
	} else {
		g.panel = ""
	}
}

func (g *game) drawHelp(p pen) {
	x, y := g.dialog(p, 860, 540)
	p.text("How to Play", display(34), x+430, y+46, alCenter, cGold)
	col := func(cx float64, sections [][2]string) {
		yy := y + 92
		for _, s := range sections {
			p.text(s[0], head(17), cx, yy, alLeft, cMint)
			yy = p.para(s[1], body(13, 400), cx, yy+24, 370, 18, cCream) + 14
		}
	}
	col(x+40, [][2]string{
		{"Your carnival", "You run a traveling carnival at the Pine Hollow County Fair, night after night, for as long as you can keep it going. The game saves every morning, so you can Continue later."},
		{"Daytime: build", "Pick something from the bar at the bottom and click the lot to build it. Guests line up at the gold square in front. Click anything you've built to change its price, or sell it. On the Paths tab, click and drag to lay footpaths: guests keep to them and walk faster."},
		{"Night: open the gates", "From 6 to 11 PM guests pay admission, ride, play, eat and spend. Pause with Space, and speed up with 1, 2 and 3."},
	})
	col(x+450, [][2]string{
		{"Keep guests happy", "Rides make them happy, food and drink keep them going, benches rest them. Light poles keep dark corners friendly, and trash cans keep the litter down."},
		{"Fair or rigged?", "Each midway game has a prize odds dial. Rig it and you pay out fewer prizes, but guests catch on and your reputation drops. A better reputation brings bigger crowds."},
		{"Watch for breakdowns", "Rides break down now and then. Click a broken ride and send a mechanic."},
	})
	p.text("Esc: cancel or menu · M: music on/off · F11: full screen · Delete: sell what's selected", body(12, 500), x+430, y+500, alCenter, cMuted)
	g.button(p, x+710, y+20, 120, 40, "Close", bGold, true, func() { g.back() })
}

/* ---------- end of night ---------- */

func (g *game) drawResults(p pen) {
	s := g.sim
	r := s.Reports[len(s.Reports)-1]
	x, y := g.dialog(p, 720, 610)
	p.text("Night "+itoa(r.Night)+" at Pine Hollow", display(32), x+360, y+46, alCenter, cGold)
	p.text(itoa(r.Guests)+" guests · "+cv.WeatherNames[r.Weather]+" · Gates closed at 11 PM", body(14, 500), x+360, y+80, alCenter, cTan)
	line := func(xx, yy float64, label string, v int, neg bool) {
		p.text(label, body(14, 400), xx, yy, alLeft, cCream)
		vs := money(v)
		c := cCream
		if neg && v > 0 {
			vs, c = "-"+money(v), cBad
		}
		p.text(vs, head(16), xx+280, yy, alRight, c)
	}
	lx, rx := x+50, x+390
	yy := y + 124
	p.text("Money in", head(16), lx, yy, alLeft, cMint)
	p.text("Money out", head(16), rx, yy, alLeft, cBad)
	yy += 32
	in := [][2]any{{"Admissions", r.Admissions}, {"Rides", r.Rides}, {"Midway games", r.Games}, {"Food & drink", r.Food}}
	out := [][2]any{{"Prizes", r.Prizes}, {"Food supplies", r.Goods}, {"Wages & upkeep", r.Upkeep}, {"Repairs", r.Repairs}, {"Litter cleanup", r.Cleanup}}
	for i, l := range in {
		line(lx, yy+float64(i)*28, l[0].(string), l[1].(int), false)
	}
	for i, l := range out {
		line(rx, yy+float64(i)*28, l[0].(string), l[1].(int), true)
	}
	yy += 5*28 + 14
	p.line(x+40, yy, x+680, yy, 1, cLine)
	yy += 34
	pc := cGold
	if r.Profit() < 0 {
		pc = cBad
	}
	p.text("Profit tonight", head(20), lx, yy, alLeft, cCream)
	p.text(money(r.Profit()), display(30), x+670, yy, alRight, pc)
	yy += 44
	p.text("Guests left "+itoa(int(r.AvgHappy))+"% happy", body(15, 600), lx, yy, alLeft, cCream)
	p.text("Reputation", body(15, 600), rx, yy, alLeft, cCream)
	stars(p, rx+110, yy, 18, r.RepAfter)
	yy += 30
	note := ""
	switch {
	case r.RiggedCaught > 0:
		note = itoa(r.RiggedCaught) + " guests said your games were rigged. Word gets around."
	case r.RepAfter > r.RepBefore+0.05:
		note = "The town is talking. Expect a bigger crowd next time."
	case r.Litter > 20:
		note = "Litter everywhere. More trash cans would help."
	case r.RepAfter < r.RepBefore-0.05:
		note = "Guests went home grumbling. Lights, benches and fair prices help."
	default:
		note = "A solid night on the midway."
	}
	p.para(note, body(13, 400), lx, yy, 620, 18, cTan)
	g.button(p, x+210, y+540, 300, 50, "On to Day "+itoa(r.Night+1), bRed, true, g.nextDay)
}

func (g *game) drawBroke(p pen) {
	x, y := g.dialog(p, 560, 300)
	p.text("Out of Money", display(36), x+280, y+60, alCenter, cBad)
	p.para("The bank won't spot you any more. The carnival has to fold, for now. Try fewer, cheaper attractions and fair prices.", body(15, 400), x+50, y+110, 460, 21, cCream)
	g.button(p, x+50, y+220, 220, 50, "Try Again", bRed, true, func() { g.newRun() })
	g.button(p, x+290, y+220, 220, 50, "Main Menu", bDark, true, func() { g.screen = "title"; g.sim = nil })
}
