package main

import (
	"image/color"
	"math"
	"sort"

	cv "github.com/adamglassner/carnivalconstructor/internal/carnival"
)

// The lot: 24 x 14 tiles, laid out left of the side panel and above the build bar.
const (
	TS = 46.0 // tile size
	LX = 44.0 // lot left
	LY = 78.0 // lot top
)

var lotW, lotH = 24 * TS, 14 * TS

func tilePx(x, y float64) (float64, float64) { return LX + x*TS, LY + y*TS }

func (g *game) mouseTile() (int, int, bool) {
	tx, ty := math.Floor((g.mx-LX)/TS), math.Floor((g.my-LY)/TS)
	in := g.mx >= LX && g.my >= LY && g.mx < LX+lotW && g.my < LY+lotH
	return int(tx), int(ty), in
}

var shirts = []color.RGBA{cRed, cTeal, cGold, cBlue, cPurple, cOrange, cPink, hex(0x6fbf5a)}
var skins = []color.RGBA{hex(0xf1c7a1), hex(0xe0a87c), hex(0xc68857), hex(0x8d5a3b), hex(0x5e3a26)}
var prizeCols = []color.RGBA{cPink, cBlue, cGold, cPurple, cMint}

// accent colours per attraction
func accentOf(id string) color.RGBA {
	switch id {
	case "ringtoss", "lemonade":
		return cTeal
	case "darts", "corndogs":
		return cRed
	case "duckpond", "funnel":
		return cBlue
	case "teacups":
		return cPink
	}
	return cGold
}

func (g *game) darkness() float64 {
	s := g.sim
	if s == nil || s.Phase != cv.PhaseNight {
		return 0
	}
	return math.Max(0, math.Min(1, (s.T+0.02)/0.22))
}

func (g *game) drawLot(p pen) {
	s := g.sim
	// field around the lot
	p.rect(-200, 0, 1360, 900, hex(0x2c5a37))
	for i := 0; i < 60; i++ { // a few tufts outside the fence
		x := math.Mod(float64(i*197), 1150)
		y := 10 + math.Mod(float64(i*131), 720)
		if x > LX-8 && x < LX+lotW+8 && y > LY-8 && y < LY+lotH+8 {
			continue
		}
		p.circle(x, y, 3, hex(0x2f6340))
	}
	// the road in to the gate
	_, gy := tilePx(0, float64(s.GateY))
	for x := LX - TS*6; x < LX; x += TS {
		drawPathTile(p, x, gy, TS, true, true, false, false)
	}
	// grass
	for y := 0; y < s.H; y++ {
		for x := 0; x < s.W; x++ {
			px, py := tilePx(float64(x), float64(y))
			c := cGrass
			if (x+y)%2 == 1 {
				c = cGrass2
			}
			p.rect(px, py, TS+0.5, TS+0.5, c)
		}
	}
	// worn grass where guests walk most
	for y := 0; y < s.H; y++ {
		for x := 0; x < s.W; x++ {
			t := s.Traffic[y*s.W+x]
			if t < 4 || s.Path[y*s.W+x] {
				continue
			}
			k := math.Min(0.8, float64(t)/110)
			px, py := tilePx(float64(x), float64(y))
			p.rrect(px-4, py-4, TS+8, TS+8, 18, withA(cDirt, k))
		}
	}
	// footpaths
	isPath := func(x, y int) bool { return x >= 0 && y >= 0 && x < s.W && y < s.H && s.Path[y*s.W+x] }
	for y := 0; y < s.H; y++ {
		for x := 0; x < s.W; x++ {
			if !isPath(x, y) {
				continue
			}
			px, py := tilePx(float64(x), float64(y))
			drawPathTile(p, px, py, TS, isPath(x-1, y) || (x == 0 && y == s.GateY), isPath(x+1, y), isPath(x, y-1), isPath(x, y+1))
		}
	}
	if s.Phase == cv.PhaseBuild || g.placing != nil || g.pathTool > 0 {
		for x := 0; x <= s.W; x++ {
			px, _ := tilePx(float64(x), 0)
			p.line(px, LY, px, LY+lotH, 1, withA(cWhite, 0.07))
		}
		for y := 0; y <= s.H; y++ {
			_, py := tilePx(0, float64(y))
			p.line(LX, py, LX+lotW, py, 1, withA(cWhite, 0.07))
		}
	}
	// litter
	for i, l := range s.Litter {
		if l == 0 {
			continue
		}
		x, y := i%s.W, i/s.W
		for k := 0; k < int(l); k++ {
			px, py := tilePx(float64(x)+0.2+math.Mod(float64(k*37+i*11), 60)/100, float64(y)+0.25+math.Mod(float64(k*53+i*7), 55)/100)
			p.rect(px, py, 5, 4, []color.RGBA{cCream, cRed, cGold}[(k+i)%3])
		}
	}
	// fence
	fc := hex(0x8a5a32)
	p.line(LX, LY, LX+lotW, LY, 3, fc)
	p.line(LX, LY+lotH, LX+lotW, LY+lotH, 3, fc)
	p.line(LX+lotW, LY, LX+lotW, LY+lotH, 3, fc)
	p.line(LX, LY, LX, gy, 3, fc)
	p.line(LX, gy+TS, LX, LY+lotH, 3, fc)
	for x := 0; x <= s.W; x++ {
		px, _ := tilePx(float64(x), 0)
		p.rect(px-2, LY-4, 4, 8, fc)
		p.rect(px-2, LY+lotH-4, 4, 8, fc)
	}
}

// drawGate: the ticket booth and arch, drawn among the other things so it sorts by depth.
func (g *game) drawGate(p pen) {
	gx, gy := tilePx(0, float64(g.sim.GateY))
	// ticket booth just outside the fence
	bx, by := gx-38, gy-6
	p.ellipse(bx+16, by+2, 20, 5, cShadow)
	p.rect(bx, by-34, 32, 36, cCream)
	p.rect(bx+5, by-26, 22, 12, cNavy)
	p.rect(bx-3, by-44, 38, 12, cRed)
	for i := 0; i < 4; i++ {
		p.rect(bx-3+float64(i)*10, by-44, 5, 12, cCream)
	}
	p.text("TICKETS", body(7, 700), bx+16, by-6, alCenter, cRed)
	// arch posts either side of the path
	for _, y := range []float64{gy - 2, gy + TS + 2} {
		p.rect(gx-4, y-46, 8, 46, cRedDk)
		p.circle(gx, y-48, 6, cGold)
	}
	p.line(gx, gy-48, gx, gy+TS-46, 7, cRed)
}

func (g *game) drawAttraction(p pen, d *cv.Def, x, y float64, a *cv.Attraction) {
	w, h := float64(d.W)*TS, float64(d.H)*TS
	cx, by := x+w/2, y+h-4
	t := g.clock
	running := false
	anim := 0.0
	if a != nil {
		running = a.Running
		anim = a.Anim
	}
	switch d.ID {
	case "carousel":
		drawCarousel(p, cx, by, w, anim, running, t)
	case "teacups":
		drawTeacups(p, cx, by, w, anim)
	case "ferris":
		drawFerris(p, cx, by, w, anim)
	case "barrel", "coaster", "striker":
		p.rrect(x+6, y+6, w-12, h-12, 8, cPanel2)
		p.text("?", display(40), cx, y+h/2, alCenter, cMuted)
	case "ringtoss", "darts", "duckpond":
		drawBooth(p, d, x, y, w, h, t, a)
	case "corndogs", "funnel", "lemonade":
		drawTruck(p, d, x, y, w, h, t)
	case "bench":
		p.ellipse(cx, by-2, 18, 4, cShadow)
		p.rect(cx-17, by-16, 34, 5, cBrown)
		p.rect(cx-17, by-24, 34, 5, hex(0x9a6238))
		p.rect(cx-15, by-11, 3, 10, hex(0x3a2a1a))
		p.rect(cx+12, by-11, 3, 10, hex(0x3a2a1a))
	case "trash":
		p.ellipse(cx, by-2, 11, 4, cShadow)
		p.rrect(cx-9, by-26, 18, 24, 3, hex(0x2f7d5a))
		p.rect(cx-9, by-18, 18, 2, hex(0x235e43))
		p.rrect(cx-11, by-30, 22, 6, 3, hex(0x3d9a70))
	case "lamp":
		p.ellipse(cx, by-2, 9, 3, cShadow)
		p.rect(cx-2.5, by-64, 5, 62, hex(0x2a2a3a))
		p.rect(cx-6, by-6, 12, 6, hex(0x2a2a3a))
		p.rrect(cx-8, by-74, 16, 12, 5, hex(0x2a2a3a))
		p.circle(cx, by-66, 5, cGold)
	}
	if a != nil && a.Broken {
		bx, bby := cx, y+h*0.35
		if d.ID == "ferris" {
			bby = by - 110
		}
		pulse := 0.75 + 0.25*math.Sin(t*6)
		p.circle(bx, bby, 17, withA(cRed, pulse))
		p.ring(bx, bby, 17, 2, cCream)
		if a.Repairing {
			frac := 1 - a.RepairT/cv.RepairSeconds
			for i := 0; i < int(frac*24); i++ {
				ang := -math.Pi/2 + float64(i)/24*2*math.Pi
				p.circle(bx+math.Cos(ang)*22, bby+math.Sin(ang)*22, 2.4, cMint)
			}
			p.text("FIXING", head(9), bx, bby, alCenter, cCream)
		} else {
			// a little wrench
			p.line(bx-7, bby+7, bx+5, bby-5, 4, cCream)
			p.circle(bx+6, bby-6, 5, cCream)
			p.circle(bx+8, bby-8, 2.5, withA(cRed, pulse))
		}
	}
}

func drawCarousel(p pen, cx, by, w, anim float64, running bool, t float64) {
	r := w * 0.44
	p.ellipse(cx, by, r+6, 15, cShadow)
	p.ellipse(cx, by-6, r, 15, cRedDk)
	p.ellipse(cx, by-11, r, 14, cCream)
	p.ellipse(cx, by-11, r-8, 10, hex(0xf3e2bc))
	top := by - 82
	ang := anim * 0.9
	type horse struct{ x, y, z, bob float64 }
	var hs []horse
	for i := 0; i < 8; i++ {
		a := ang + float64(i)*math.Pi/4
		bob := 0.0
		if running {
			bob = math.Sin(anim*4+float64(i)) * 4
		}
		hs = append(hs, horse{cx + math.Cos(a)*r*0.78, by - 11 + math.Sin(a)*9, math.Sin(a), bob})
	}
	sort.Slice(hs, func(i, j int) bool { return hs[i].z < hs[j].z })
	drawHorse := func(h horse) {
		p.line(h.x, top+8, h.x, h.y, 2, cGold)
		hy := h.y - 24 - h.bob
		col := mix(cCream, hex(0xc9b48a), (1-h.z)/4)
		p.ellipse(h.x, hy, 9, 5, col)
		p.circle(h.x+8, hy-5, 3.5, col)
		p.line(h.x-5, hy+3, h.x-7, hy+10, 2, col)
		p.line(h.x+5, hy+3, h.x+7, hy+10, 2, col)
		p.rect(h.x-3, hy-6, 6, 3, cRed)
	}
	for _, h := range hs {
		if h.z < 0 {
			drawHorse(h)
		}
	}
	p.rect(cx-9, top+6, 18, by-11-top-6, cPlum)
	p.rect(cx-9, top+16, 18, 4, cGold)
	p.rect(cx-9, by-28, 18, 4, cGold)
	for _, h := range hs {
		if h.z >= 0 {
			drawHorse(h)
		}
	}
	// canopy
	half := r + 6
	p.rect(cx-half, top-2, half*2, 12, cRed)
	for i := 0; i < 9; i++ {
		x := cx - half + (float64(i)+0.5)*half*2/9
		p.circle(x, top+10, half/9, []color.RGBA{cCream, cRed}[i%2])
	}
	apex := top - 58
	for i := 0; i < 8; i++ {
		x0 := cx - half + float64(i)*half*2/8
		x1 := x0 + half*2/8
		c := cRed
		if i%2 == 1 {
			c = cCream
		}
		p.poly(c, x0, top, x1, top, cx, apex)
	}
	p.line(cx, apex, cx, apex-14, 2, cTan)
	p.poly(cTeal, cx, apex-14, cx+14, apex-10, cx, apex-6)
	_ = t
}

func drawTeacups(p pen, cx, by, w, anim float64) {
	r := w * 0.46
	p.ellipse(cx, by, r+4, 20, cShadow)
	p.ellipse(cx, by-6, r, 20, hex(0x1f6f66))
	p.ellipse(cx, by-10, r, 19, cTeal)
	p.ellipse(cx, by-10, r*0.62, 12, hex(0x37c4b3))
	ang := anim * 1.4
	type cup struct {
		x, y, z float64
		c       color.RGBA
		spin    float64
	}
	var cups []cup
	cols := []color.RGBA{cPink, cGold, cBlue, cRed}
	for i := 0; i < 4; i++ {
		a := ang + float64(i)*math.Pi/2
		cups = append(cups, cup{cx + math.Cos(a)*r*0.62, by - 12 + math.Sin(a)*12, math.Sin(a), cols[i], anim*5 + float64(i)})
	}
	sort.Slice(cups, func(i, j int) bool { return cups[i].z < cups[j].z })
	drawCup := func(c cup) {
		p.ellipse(c.x, c.y-6, 19, 7, mix(c.c, cBlack, 0.25))
		p.poly(c.c, c.x-19, c.y-14, c.x+19, c.y-14, c.x+14, c.y, c.x-14, c.y)
		p.ellipse(c.x, c.y-14, 19, 6, mix(c.c, cWhite, 0.35))
		p.ellipse(c.x, c.y-14, 15, 4, hex(0x5a3a3a))
		hx := c.x + math.Cos(c.spin)*20
		p.ring(hx, c.y-8, 5, 2.5, c.c)
		p.circle(c.x-5, c.y-18, 3.5, skins[1])
		p.circle(c.x+5, c.y-18, 3.5, skins[3])
	}
	for _, c := range cups {
		if c.z < 0 {
			drawCup(c)
		}
	}
	// teapot in the middle
	p.ellipse(cx, by-30, 15, 13, cCream)
	p.poly(cCream, cx+12, by-32, cx+26, by-44, cx+24, by-40, cx+12, by-26)
	p.ring(cx-16, by-30, 6, 3, cCream)
	p.ellipse(cx, by-42, 9, 4, cPink)
	p.circle(cx, by-47, 3, cGold)
	for _, c := range cups {
		if c.z >= 0 {
			drawCup(c)
		}
	}
}

func ferrisHub(cx, by, w float64) (float64, float64, float64) {
	return cx, by - 104, w * 0.6
}

func drawFerris(p pen, cx, by, w, anim float64) {
	hx, hy, R := ferrisHub(cx, by, w)
	p.ellipse(cx, by, w*0.46, 12, cShadow)
	// platform
	p.rect(cx-w*0.42, by-12, w*0.84, 12, hex(0x4a3a5a))
	p.rect(cx-w*0.42, by-14, w*0.84, 4, cGold)
	for i := 0; i < 3; i++ {
		p.rect(cx-14+float64(i)*2, by-4+float64(i)*-3, 28-float64(i)*4, 3, cTan)
	}
	// legs (back)
	p.line(cx-w*0.36, by-12, hx, hy, 5, hex(0xcfc0a0))
	p.line(cx+w*0.36, by-12, hx, hy, 5, hex(0xcfc0a0))
	ang := anim * 0.32
	p.ring(hx, hy, R, 4, cGold)
	p.ring(hx, hy, R*0.78, 2, cRed)
	n := 12
	for i := 0; i < n; i++ {
		a := ang + float64(i)*2*math.Pi/float64(n)
		p.line(hx, hy, hx+math.Cos(a)*R, hy+math.Sin(a)*R, 1.5, cTan)
	}
	cols := []color.RGBA{cRed, cTeal, cGold}
	for i := 0; i < n; i++ {
		a := ang + float64(i)*2*math.Pi/float64(n)
		gx, gy := hx+math.Cos(a)*R, hy+math.Sin(a)*R
		p.line(gx, gy, gx, gy+5, 1.5, cTan)
		p.rrect(gx-8, gy+4, 16, 12, 4, cols[i%3])
		p.rect(gx-6, gy+6, 12, 3, withA(cNavy, 0.6))
	}
	p.circle(hx, hy, 9, cRed)
	p.ring(hx, hy, 9, 2.5, cGold)
	// legs (front)
	p.line(cx-w*0.3, by-12, hx, hy, 4, cTan)
	p.line(cx+w*0.3, by-12, hx, hy, 4, cTan)
}

func drawBooth(p pen, d *cv.Def, x, y, w, h, t float64, a *cv.Attraction) {
	acc := accentOf(d.ID)
	cx, by := x+w/2, y+h-4
	p.ellipse(cx, by, w*0.48, 8, cShadow)
	// back wall and prizes
	p.rect(x+8, by-74, w-16, 44, cPlumDk)
	for i := 0; i < 4; i++ {
		px := x + 16 + float64(i)*(w-32)/3
		p.circle(px, by-64, 6, prizeCols[i%len(prizeCols)])
		p.circle(px-4, by-70, 2.5, prizeCols[i%len(prizeCols)])
		p.circle(px+4, by-70, 2.5, prizeCols[i%len(prizeCols)])
	}
	switch d.ID {
	case "ringtoss":
		for i := 0; i < 6; i++ {
			bx := x + 18 + float64(i)*(w-36)/5
			p.rect(bx-2.5, by-50, 5, 14, hex(0x2f8a4a))
			p.rect(bx-1.2, by-55, 2.4, 6, hex(0x2f8a4a))
		}
		if a != nil && a.Running {
			k := math.Mod(a.Anim*1.6, 1)
			p.ring(x+10+k*(w-20), by-40-math.Sin(k*math.Pi)*26, 5, 2, cGold)
		}
	case "darts":
		for i := 0; i < 7; i++ {
			bx := x + 16 + float64(i%4)*(w-32)/3 + float64(i/4)*10
			byy := by - 52 + float64(i/4)*-12
			if a != nil && a.Running && i == int(a.Anim*3)%7 {
				continue // popped
			}
			p.circle(bx, byy, 5.5, []color.RGBA{cRed, cGold, cBlue, cPink}[i%4])
		}
	case "duckpond":
		p.rrect(x+10, by-44, w-20, 14, 6, cBlue)
		for i := 0; i < 5; i++ {
			k := math.Mod(t*0.18+float64(i)/5, 1)
			dx := x + 14 + k*(w-28)
			p.circle(dx, by-40, 4.5, cGold)
			p.circle(dx+3.5, by-44, 2.8, cGold)
			p.poly(cOrange, dx+6, by-44, dx+9, by-43, dx+6, by-42)
		}
	}
	// counter
	p.rect(x+4, by-30, w-8, 28, cPlum)
	p.rect(x+4, by-30, w-8, 5, mix(cPlum, cWhite, 0.25))
	for i := 0; i < 5; i++ {
		p.rect(x+10+float64(i)*(w-20)/4-2, by-22, 4, 18, cPlumDk)
	}
	// awning
	sw := (w - 4) / 7
	for i := 0; i < 7; i++ {
		c := acc
		if i%2 == 1 {
			c = cCream
		}
		p.rect(x+2+float64(i)*sw, by-90, sw+0.5, 16, c)
		p.circle(x+2+(float64(i)+0.5)*sw, by-74, sw/2, c)
	}
	// sign
	p.rrect(cx-34, by-110, 68, 18, 4, cRed)
	p.rrectStroke(cx-34, by-110, 68, 18, 4, 1.5, cGold)
	name := map[string]string{"ringtoss": "RING TOSS", "darts": "DARTS", "duckpond": "DUCKS"}[d.ID]
	p.text(name, head(10), cx, by-101, alCenter, cCream)
}

func drawTruck(p pen, d *cv.Def, x, y, w, h, t float64) {
	acc := accentOf(d.ID)
	cx, by := x+w/2, y+h-4
	p.ellipse(cx, by, w*0.48, 7, cShadow)
	p.rrect(x+4, by-60, w-8, 50, 8, cCream)
	p.rect(x+4, by-60, w-8, 10, acc)
	p.rect(x+4, by-22, w-8, 8, mix(acc, cBlack, 0.2))
	p.rect(x+14, by-46, w-34, 20, hex(0x2a2440))
	p.rect(x+14, by-30, w-34, 4, cTan)
	// window light
	p.rect(x+16, by-44, w-38, 14, withA(cGold, 0.35))
	// awning flap
	p.poly(acc, x+10, by-50, x+w-16, by-50, x+w-12, by-44, x+6, by-44)
	p.rect(x+w-18, by-50, 10, 26, hex(0xd8cbb0))
	p.circle(x+20, by-8, 8, hex(0x1d1d24))
	p.circle(x+20, by-8, 3.5, cMuted)
	p.circle(x+w-20, by-8, 8, hex(0x1d1d24))
	p.circle(x+w-20, by-8, 3.5, cMuted)
	// roof sign
	sy := by - 74
	p.rrect(cx-22, sy-10, 44, 20, 6, cNavy)
	p.rrectStroke(cx-22, sy-10, 44, 20, 6, 2, acc)
	switch d.ID {
	case "corndogs":
		p.line(cx-12, sy+6, cx-4, sy-2, 2, cTan)
		p.rrect(cx-8, sy-9, 22, 9, 4.5, hex(0xc98a3a))
	case "funnel":
		for i := 0; i < 3; i++ {
			p.ring(cx-6+float64(i)*6, sy, 5, 2.4, hex(0xe8c27a))
		}
		p.circle(cx-3, sy-3, 1.6, cWhite)
		p.circle(cx+5, sy-2, 1.6, cWhite)
	case "lemonade":
		p.poly(hex(0xfff0a0), cx-7, sy-7, cx+7, sy-7, cx+5, sy+8, cx-5, sy+8)
		p.line(cx+2, sy-7, cx+6, sy-13, 2, cRed)
		p.circle(cx-8, sy-6, 4, cGold)
	}
	_ = t
}

func (g *game) drawGuest(p pen, gu *cv.Guest) {
	x, y := tilePx(gu.X, gu.Y)
	k := 1.0
	if gu.Kid {
		k = 0.78
	}
	walking := gu.State == cv.Walking
	ph := 0.0
	if walking {
		ph = math.Sin(g.clock*12+float64(gu.ID)) * 3 * k
	}
	p.ellipse(x, y, 5.5*k, 2.2, cShadow)
	leg := hex(0x2a2f55)
	p.line(x-2*k, y-8*k, x-2*k+ph*0.6, y, 2.4*k, leg)
	p.line(x+2*k, y-8*k, x+2*k-ph*0.6, y, 2.4*k, leg)
	shirt := shirts[gu.Look%len(shirts)]
	p.rrect(x-5*k, y-18*k, 10*k, 11*k, 3*k, shirt)
	skin := skins[gu.Skin%len(skins)]
	p.circle(x, y-22*k, 4.4*k, skin)
	hair := []color.RGBA{hex(0x3a2a1a), hex(0x6a4a2a), hex(0xd9b56a), hex(0x1a1a1a), hex(0xa0522d)}[(gu.Look+gu.Skin)%5]
	p.ellipse(x, y-24.5*k, 4.6*k, 2.4*k, hair)
	if gu.Prize > 0 {
		bc := prizeCols[(gu.Prize-1)%len(prizeCols)]
		bx := x + 7*k
		p.line(x+4*k, y-14*k, bx, y-30*k, 1, cTan)
		p.ellipse(bx, y-35*k, 5, 6, bc)
		p.circle(bx-1.5, y-37*k, 1.4, withA(cWhite, 0.6))
	}
	if g.selGuest == gu {
		p.ring(x, y-12*k, 15, 2, cGold)
	}
}

func (g *game) drawThought(p pen, gu *cv.Guest) {
	if gu.ThoughtT <= 0 || gu.Thought == "" || gu.State == cv.Riding {
		return
	}
	x, y := tilePx(gu.X, gu.Y)
	a := math.Min(1, gu.ThoughtT/0.4)
	q := p.fade(a)
	f := body(11, 600)
	w := textW(gu.Thought, f) + 14
	bx, by := x-w/2, y-50
	c := cCream
	tc := cNavy
	if gu.Thought == "Rigged!?" {
		c, tc = cRed, cCream
	}
	q.rrect(bx, by, w, 20, 9, c)
	q.poly(c, x-4, by+19, x+4, by+19, x, by+26)
	q.text(gu.Thought, f, x, by+10, alCenter, tc)
}

// drawWorld draws the lot and everything on it, sorted back to front.
func (g *game) drawWorld(p pen) {
	s := g.sim
	g.drawLot(p)
	type item struct {
		y    float64
		draw func()
	}
	var items []item
	_, gy := tilePx(0, float64(s.GateY))
	items = append(items, item{gy + TS + 2, func() { g.drawGate(p) }})
	for _, a := range s.Attr {
		a := a
		x, y := tilePx(float64(a.X), float64(a.Y))
		items = append(items, item{y + float64(a.Def.H)*TS, func() {
			q := p
			if g.sel == a {
				q.rrect(x+2, y+2, float64(a.Def.W)*TS-4, float64(a.Def.H)*TS-4, 8, withA(cGold, 0.18))
			}
			g.drawAttraction(q, a.Def, x, y, a)
		}})
	}
	for _, gu := range s.Guests {
		gu := gu
		if gu.State == cv.Riding {
			continue
		}
		_, y := tilePx(gu.X, gu.Y)
		items = append(items, item{y, func() { g.drawGuest(p, gu) }})
	}
	sort.SliceStable(items, func(i, j int) bool { return items[i].y < items[j].y })
	for _, it := range items {
		it.draw()
	}
	g.drawNight(p)
	if s.Weather == cv.Drizzle && s.Phase == cv.PhaseNight {
		for i := 0; i < 140; i++ {
			fx := math.Mod(float64(i*173)+g.clock*40, 1180)
			fy := math.Mod(float64(i*97)+g.clock*520, 760)
			p.line(fx, fy, fx-3, fy+12, 1, withA(hex(0xb4c8ff), 0.28))
		}
	}
	var placed [][4]float64
	for i := len(s.Guests) - 1; i >= 0; i-- {
		gu := s.Guests[i]
		if gu.ThoughtT <= 0 || gu.Thought == "" || gu.State == cv.Riding {
			continue
		}
		x, y := tilePx(gu.X, gu.Y)
		w := textW(gu.Thought, body(11, 600)) + 14
		r := [4]float64{x - w/2, y - 50, w, 26}
		clash := false
		for _, o := range placed {
			if r[0] < o[0]+o[2] && o[0] < r[0]+r[2] && r[1] < o[1]+o[3] && o[1] < r[1]+r[3] {
				clash = true
				break
			}
		}
		if clash || len(placed) >= 10 {
			continue
		}
		placed = append(placed, r)
		g.drawThought(p, gu)
	}
}

// drawNight darkens the lot as the evening goes on and lights it with the rides.
func (g *game) drawNight(p pen) {
	d := g.darkness()
	s := g.sim
	if d <= 0 {
		return
	}
	dusk := mix(hex(0x6a3050), hex(0x0a0d38), d)
	p.rect(-200, 0, 1376, 900, withA(dusk, 0.18+0.42*d))
	warm := color.RGBA{255, 190, 120, 255}
	gx, gy := tilePx(0, float64(s.GateY))
	p.glow(gx, gy+TS/2, 2.6*TS, warm, 0.2*d)
	for _, a := range s.Attr {
		if a.Def.Light <= 0 {
			continue
		}
		cx, cy := a.Center()
		x, y := tilePx(cx, cy)
		r := (a.Def.Light + float64(a.Def.W)/2) * TS
		k := 0.15
		if a.Broken {
			k = 0.05
		}
		if a.Def.ID == "lamp" {
			p.glow(x, y-56, 26, cGold, 0.45*d)
			k = 0.2
		}
		p.glow(x, y, r, warm, k*d)
		g.drawBulbs(p, a, d)
	}
}

// drawBulbs: the strings of lights on rides and booths, bright after dark.
func (g *game) drawBulbs(p pen, a *cv.Attraction, d float64) {
	x, y := tilePx(float64(a.X), float64(a.Y))
	w, h := float64(a.Def.W)*TS, float64(a.Def.H)*TS
	cx, by := x+w/2, y+h-4
	t := g.clock
	cols := []color.RGBA{cGold, cRed, cMint, cPink}
	bulb := func(bx, byy float64, i int) {
		on := a.Broken || math.Sin(t*3+float64(i)*0.9) > -0.3
		if a.Broken {
			on = i%3 == 0
		}
		if !on {
			return
		}
		c := cols[i%len(cols)]
		p.circle(bx, byy, 2.2, mix(c, cWhite, 0.4))
		p.glow(bx, byy, 7, c, 0.28*d)
	}
	switch a.Def.ID {
	case "ferris":
		hx, hy, R := ferrisHub(cx, by, w)
		ang := a.Anim * 0.32
		for i := 0; i < 24; i++ {
			an := ang + float64(i)*2*math.Pi/24
			bulb(hx+math.Cos(an)*R, hy+math.Sin(an)*R, i)
		}
	case "carousel":
		half := w*0.44 + 6
		for i := 0; i < 11; i++ {
			bulb(cx-half+float64(i)*half*2/10, by-84, i)
		}
	case "teacups":
		for i := 0; i < 14; i++ {
			an := float64(i) * 2 * math.Pi / 14
			bulb(cx+math.Cos(an)*w*0.46, by-8+math.Sin(an)*20, i)
		}
	case "ringtoss", "darts", "duckpond":
		for i := 0; i < 7; i++ {
			bulb(x+6+float64(i)*(w-12)/6, by-92, i)
		}
	case "corndogs", "funnel", "lemonade":
		for i := 0; i < 5; i++ {
			bulb(x+10+float64(i)*(w-20)/4, by-61, i)
		}
	}
}

// drawGhost shows what's about to be built under the mouse.
func (g *game) drawGhost(p pen) {
	d := g.placing
	if d == nil {
		return
	}
	tx, ty, in := g.mouseTile()
	if !in {
		return
	}
	x0, y0 := tx-d.W/2, ty-d.H/2
	err := g.sim.CanPlace(d, x0, y0)
	ok := err == nil
	col := withA(cMint, 0.35)
	if !ok {
		col = withA(cRed, 0.4)
	}
	for yy := y0; yy < y0+d.H; yy++ {
		for xx := x0; xx < x0+d.W; xx++ {
			px, py := tilePx(float64(xx), float64(yy))
			p.rect(px+1, py+1, TS-2, TS-2, col)
		}
	}
	if d.Visitable() {
		ex, ey := x0+d.W/2, y0+d.H
		px, py := tilePx(float64(ex), float64(ey))
		p.rect(px+6, py+6, TS-12, TS-12, withA(cGold, 0.35))
		p.poly(cGold, px+TS/2, py+10, px+TS/2-8, py+22, px+TS/2+8, py+22)
	}
	x, y := tilePx(float64(x0), float64(y0))
	g.drawAttraction(p.fade(0.75), d, x, y, nil)
	if !ok && err != nil {
		f := body(12, 600)
		w := textW(err.Error(), f) + 20
		mx, my := g.mx, g.my-34
		p.rrect(mx-w/2, my-12, w, 24, 8, cRedDk)
		p.text(err.Error(), f, mx, my, alCenter, cCream)
	}
}

// drawPathTile draws one square of footpath, joined up to its neighbours.
func drawPathTile(p pen, x, y, ts float64, l, r, u, d bool) {
	const in = 5.0
	edge, top := hex(0xa88a5c), hex(0xd8c194)
	x0, y0, x1, y1 := x+in, y+in, x+ts-in, y+ts-in
	if l {
		x0 = x
	}
	if r {
		x1 = x + ts
	}
	if u {
		y0 = y
	}
	if d {
		y1 = y + ts
	}
	p.rect(x0, y0, x1-x0, y1-y0, edge)
	p.rect(x0+1.5, y0+1.5, x1-x0-3, y1-y0-3, top)
	// a few paving stones
	k := ts / 46
	for i, o := range [][2]float64{{10, 12}, {28, 9}, {18, 28}, {34, 31}, {8, 36}} {
		sx, sy := x+o[0]*k, y+o[1]*k
		p.rrect(sx, sy, 7*k, 5*k, 2*k, []color.RGBA{hex(0xcbb283), hex(0xe2cfa6)}[i%2])
	}
}

// drawPathGhost shows the tile a path tool will change.
func (g *game) drawPathGhost(p pen) {
	tx, ty, in := g.mouseTile()
	if !in || g.pathTool == 0 {
		return
	}
	ok := false
	if g.pathTool == 1 {
		ok = g.sim.CanPath(tx, ty) == nil
	} else {
		ok = g.sim.Path[ty*g.sim.W+tx] && !(ty == g.sim.GateY && tx <= 1)
	}
	c := withA(cMint, 0.4)
	if !ok {
		c = withA(cRed, 0.35)
	}
	px, py := tilePx(float64(tx), float64(ty))
	p.rect(px+1, py+1, TS-2, TS-2, c)
}
