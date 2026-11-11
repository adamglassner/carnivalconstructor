package carnival

import "math"

// AttrStats counts what an attraction did.
type AttrStats struct {
	Customers, Revenue, Costs int
	Wins, Losses, Refused     int
}

// Attraction is one ride, game, stand or extra on the lot.
type Attraction struct {
	ID       int
	Def      *Def
	X, Y     int // top-left tile
	Price    int
	Rig      float64 // midway games: 0 = fair & square, 1 = rigged
	Operator string

	Queue   []*Guest
	Riding  []*Guest
	Running bool
	Timer   float64 // time left on this ride or game
	loadT   float64
	Anim    float64 // seconds it has been running, for the screen

	Broken, Repairing bool
	RepairT           float64

	Night, Total AttrStats
}

// Entrance is the tile guests line up on: just below the middle of the footprint.
func (a *Attraction) Entrance() (int, int) { return a.X + a.Def.W/2, a.Y + a.Def.H }

func (a *Attraction) entranceCenter() (float64, float64) {
	x, y := a.Entrance()
	return float64(x) + .5, float64(y) + .5
}

// Center of the footprint, in tile coordinates.
func (a *Attraction) Center() (float64, float64) {
	return float64(a.X) + float64(a.Def.W)/2, float64(a.Y) + float64(a.Def.H)/2
}

// WinChance is the chance a guest wins one go at a midway game.
func (a *Attraction) WinChance() float64 { return a.Def.BaseWin * (1 - 0.9*a.Rig) }

// QueueSpot is where the k-th guest in line stands, in tile coordinates.
func (a *Attraction) QueueSpot(k int) (float64, float64) {
	ex, ey := a.entranceCenter()
	row, col := k/4, k%4
	if row%2 == 1 {
		col = 3 - col
	}
	return ex + (float64(col)-1.5)*0.23, ey - 0.12 + float64(row)*0.22
}

func (a *Attraction) dequeue(g *Guest) {
	for i, q := range a.Queue {
		if q == g {
			a.Queue = append(a.Queue[:i], a.Queue[i+1:]...)
			return
		}
	}
}

func (a *Attraction) update(s *Sim, dt float64) {
	if a.Broken {
		if a.Repairing {
			a.RepairT -= dt
			if a.RepairT <= 0 {
				a.Broken, a.Repairing = false, false
				cx, cy := a.Center()
				s.emit(Event{Kind: "fixed", Text: a.Def.Name + " is running again", X: cx, Y: cy, A: a})
			}
		}
		return
	}
	if a.Running {
		a.Anim += dt
		a.Timer -= dt
		if a.Timer <= 0 {
			a.finish(s)
		}
		return
	}
	if len(a.Queue) == 0 {
		a.loadT = 0
		return
	}
	a.loadT += dt
	wait := 0.3
	if a.Def.Cat == CatRide {
		wait = 3.0
	}
	if len(a.Queue) >= a.Def.Capacity || a.loadT >= wait {
		a.start(s)
	}
}

func (a *Attraction) start(s *Sim) {
	n := min(a.Def.Capacity, len(a.Queue))
	a.Riding = append(a.Riding[:0], a.Queue[:n]...)
	a.Queue = append([]*Guest{}, a.Queue[n:]...)
	cx, cy := a.Center()
	for _, g := range a.Riding {
		g.State = Riding
		g.wait = 0
		g.X, g.Y = cx, cy
		if a.Price > 0 && g.Money >= a.Price {
			g.Money -= a.Price
			g.Spent += a.Price
			s.Cash += a.Price
			a.Night.Revenue += a.Price
			a.Total.Revenue += a.Price
			switch a.Def.Cat {
			case CatRide:
				s.Report.Rides += a.Price
			case CatGame:
				s.Report.Games += a.Price
			case CatFood:
				s.Report.Food += a.Price
			}
		}
		if a.Def.Cat != CatExtra {
			a.Night.Customers++
			a.Total.Customers++
		}
	}
	if a.Def.Cat == CatRide && a.Price > 0 {
		s.emit(Event{Kind: "coin", X: cx, Y: cy - float64(a.Def.H)/2, A: a, Text: "+$" + itoa(a.Price*n)})
	}
	a.Running = true
	a.Timer = a.Def.Cycle * (0.9 + s.Rand.Float64()*0.2)
	a.loadT = 0
}

func (a *Attraction) finish(s *Sim) {
	a.Running = false
	riders := a.Riding
	a.Riding = nil
	d := a.Def
	broke := d.Cat == CatRide && s.chance(d.Breaks)
	cx, cy := a.Center()
	for _, g := range riders {
		g.Activities++
		g.Visits[a.ID]++
		g.State = Idle
		g.X, g.Y = a.entranceCenter()
		g.X += g.ox * 0.6
		g.cx, g.cy = a.Entrance()
		g.moving = false
		switch d.Cat {
		case CatRide:
			if broke {
				g.Happy -= 10
				g.think(s, "We got stuck up there!", 0.7)
			} else {
				g.Happy += d.Fun * (0.5 + s.Rand.Float64()*0.25)
				g.Energy -= 4
				g.think(s, pick(s, "Wheee!", "Again! Again!", "Best ride ever!", "That was fun!", "I'm dizzy!"), 0.25)
			}
		case CatGame:
			s.playGame(a, g)
		case CatFood:
			g.Hunger = clamp(g.Hunger-d.Hunger, 0, 100)
			g.Thirst = clamp(g.Thirst-d.Thirst, 0, 100)
			g.Happy += d.Fun * 0.7
			g.litterT = 3 + s.Rand.Float64()*7
			goods := int(math.Round(d.Goods))
			s.Cash -= goods
			s.Report.Goods += goods
			a.Night.Costs += goods
			g.think(s, pick(s, "Yum!", "Mmm, "+d.Name+"!", "Just what I needed."), 0.25)
			if a.Price > 0 {
				s.emit(Event{Kind: "coin", X: cx, Y: cy - 1, A: a, Text: "+$" + itoa(a.Price)})
			}
		case CatExtra:
			g.Energy = clamp(g.Energy+d.Rest, 0, 100)
			g.Happy += 3
		}
	}
	for _, g := range riders {
		if g.State == Idle {
			g.decide(s)
		}
	}
	if broke {
		a.Broken = true
		s.emit(Event{Kind: "broke", Text: d.Name + " broke down!", X: cx, Y: cy, A: a})
		for _, g := range a.Queue {
			g.Target = nil
			g.State = Idle
			g.Happy -= 6
			g.think(s, "It's broken!", 0.5)
		}
		q := a.Queue
		a.Queue = nil
		for _, g := range q {
			g.decide(s)
		}
	}
}

func (s *Sim) playGame(a *Attraction, g *Guest) {
	cx, cy := a.Center()
	if s.chance(a.WinChance()) {
		prize := int(math.Round(a.Def.Prize))
		s.Cash -= prize
		s.Report.Prizes += prize
		s.Report.Wins++
		a.Night.Costs += prize
		a.Night.Wins++
		g.Happy += a.Def.Fun*0.6 + 12
		g.Prize = 1 + s.Rand.Intn(4)
		g.think(s, pick(s, "I won!", "A prize!", "Look what I won!"), 0.9)
		s.emit(Event{Kind: "win", X: cx, Y: cy - 1, A: a, Text: "+$" + itoa(a.Price)})
		return
	}
	s.Report.Losses++
	a.Night.Losses++
	g.Happy += a.Def.Fun*0.25 - 4 - 16*a.Rig
	s.emit(Event{Kind: "coin", X: cx, Y: cy - 1, A: a, Text: "+$" + itoa(a.Price)})
	if a.Rig > 0.45 && s.chance((a.Rig-0.35)*0.55) {
		g.Happy -= 8
		g.think(s, "Rigged!?", 1)
		s.rigHits += 0.012
		s.Report.RiggedCaught++
		s.emit(Event{Kind: "rigged", X: g.X, Y: g.Y, A: a})
		return
	}
	g.think(s, pick(s, "So close!", "Aw, nuts.", "One more try?"), 0.3)
	if g.Money >= a.Price && len(a.Queue) < a.Def.QueueMax && s.chance(0.32*(1-a.Rig*0.6)) {
		g.State = Queuing
		g.Target = a
		g.wait = 0
		a.Queue = append(a.Queue, g)
	}
}

func pick(s *Sim, opts ...string) string { return opts[s.Rand.Intn(len(opts))] }

func itoa(n int) string {
	if n < 0 {
		return "-" + itoa(-n)
	}
	if n < 10 {
		return string(rune('0' + n))
	}
	return itoa(n/10) + string(rune('0'+n%10))
}
