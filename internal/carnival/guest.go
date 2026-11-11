package carnival

import "math"

type GuestState int

const (
	Idle GuestState = iota
	Walking
	Queuing
	Riding
)

// Guest is one visitor.
type Guest struct {
	ID         int
	Name       string
	Kid        bool
	X, Y       float64 // position in tiles
	State      GuestState
	Target     *Attraction
	Money      int
	Spent      int
	Happy      float64
	Hunger     float64
	Thirst     float64
	Energy     float64
	Visits     map[int]int
	Activities int
	Stay       int // how many things they mean to do before going home
	Thought    string
	ThoughtT   float64
	Prize      int // 0 = none, else which plush they carry
	Look, Skin int
	Facing     float64 // -1 left, 1 right

	cx, cy, nx, ny int
	moving         bool
	goal           int // tile to walk to (wandering or leaving)
	speed          float64
	ox, oy         float64 // where in the tile they like to walk
	wait           float64
	litterT        float64
	wanders        int
	leaving        bool
	outside        bool
	gone           bool
	lastTile       int
	grumbleT       float64
	onPath         bool
}

func (g *Guest) Leaving() bool { return g.leaving }

// Mood is a word for how a guest feels.
func (g *Guest) Mood() string {
	switch h := g.Happy; {
	case h >= 85:
		return "Thrilled"
	case h >= 65:
		return "Happy"
	case h >= 45:
		return "Okay"
	case h >= 25:
		return "Grumpy"
	}
	return "Miserable"
}

// Doing says what the guest is up to.
func (g *Guest) Doing() string {
	switch {
	case g.leaving:
		return "Heading home"
	case g.State == Riding && g.Target != nil:
		switch g.Target.Def.Cat {
		case CatRide:
			return "Riding the " + g.Target.Def.Name
		case CatGame:
			return "Playing " + g.Target.Def.Name
		case CatFood:
			return "Buying " + g.Target.Def.Name
		}
		return "Sitting down"
	case g.State == Queuing && g.Target != nil:
		return "In line for " + g.Target.Def.Name
	case g.Target != nil:
		return "Walking to " + g.Target.Def.Name
	}
	return "Looking around"
}

func (g *Guest) think(s *Sim, t string, p float64) {
	if p >= 1 || s.chance(p) {
		g.Thought, g.ThoughtT = t, 2.6
	}
}

func (g *Guest) update(s *Sim, dt float64) {
	if g.ThoughtT > 0 {
		g.ThoughtT -= dt
	}
	g.grumbleT -= dt
	g.Hunger = clamp(g.Hunger+0.42*dt, 0, 100)
	g.Thirst = clamp(g.Thirst+0.5*dt, 0, 100)
	if g.State == Walking || g.State == Idle {
		g.Energy = clamp(g.Energy-0.16*dt, 0, 100)
		g.Happy -= 0.05 * dt // nothing to do is a little boring
	}
	night := s.T > 0.12
	if g.State != Riding && !g.outside {
		if night {
			if s.IsLit(g.cx, g.cy) {
				g.Happy += 0.05 * dt
			} else {
				g.Happy -= 0.14 * dt
				if g.grumbleT < 0 && s.chance(0.004) {
					g.think(s, "It's awful dark over here.", 1)
					g.grumbleT = 20
				}
			}
		}
		g.onPath = s.in(g.cx, g.cy) && s.Path[s.idx(g.cx, g.cy)]
		if g.State == Walking {
			if g.onPath {
				g.Happy += 0.02 * dt
			} else {
				g.Happy -= 0.04 * dt // trampling through the grass
				if s.Weather == Drizzle {
					g.Happy -= 0.12 * dt
					if g.grumbleT < 0 && s.chance(0.003) {
						g.think(s, "My shoes are all muddy!", 1)
						g.grumbleT = 20
					}
				}
			}
		}
		if s.Weather == Drizzle {
			g.Happy -= 0.05 * dt
		}
	}
	if g.Hunger > 85 || g.Thirst > 85 {
		g.Happy -= 0.15 * dt
	}
	g.Happy = clamp(g.Happy, 0, 100)

	if g.litterT > 0 {
		g.litterT -= dt
		if g.litterT <= 0 && !g.outside && s.in(g.cx, g.cy) {
			i := s.idx(g.cx, g.cy)
			if !s.trashNear[i] || s.chance(0.06) {
				if s.Litter[i] < 9 {
					s.Litter[i]++
				}
			}
		}
	}

	switch g.State {
	case Riding:
		return
	case Queuing:
		g.wait += dt
		if a := g.Target; a != nil {
			for k, q := range a.Queue {
				if q == g {
					tx, ty := a.QueueSpot(k)
					g.X += (tx - g.X) * math.Min(1, dt*6)
					g.Y += (ty - g.Y) * math.Min(1, dt*6)
					break
				}
			}
		}
		patience := 30.0 + g.Happy*0.25
		if g.wait > patience {
			g.Target.dequeue(g)
			g.Target = nil
			g.State = Idle
			g.Happy -= 8
			g.think(s, "This line is taking forever!", 0.8)
			g.decide(s)
		}
		return
	case Idle:
		g.decide(s)
		return
	}
	g.walk(s, dt)
}

// walk moves the guest one step along the way to their target or goal.
func (g *Guest) walk(s *Sim, dt float64) {
	if g.outside {
		// coming in through the gate, or going out of it
		tx := float64(s.GateX) + .5
		if g.leaving {
			tx = -1.2
		}
		ty := float64(s.GateY) + .5 + g.oy
		g.moveToward(tx, ty, dt)
		if math.Abs(g.X-tx) < 0.05 {
			if g.leaving {
				s.depart(g)
				g.gone = true
				return
			}
			g.outside = false
			g.cx, g.cy = s.GateX, s.GateY
			g.State = Idle
		}
		return
	}
	goal := g.goal
	if g.Target != nil {
		ex, ey := g.Target.Entrance()
		goal = s.idx(ex, ey)
	}
	if g.moving {
		tx, ty := float64(g.nx)+.5+g.ox, float64(g.ny)+.5+g.oy
		if g.moveToward(tx, ty, dt) {
			g.cx, g.cy = g.nx, g.ny
			g.moving = false
			i := s.idx(g.cx, g.cy)
			if !s.Path[i] {
				s.Traffic[i]++
			}
			if s.Litter[i] > 0 && i != g.lastTile {
				g.Happy -= 0.8 * float64(min(s.Litter[i], 3))
				if g.grumbleT < 0 && s.chance(0.12) {
					g.think(s, "Ew, litter everywhere.", 1)
					g.grumbleT = 15
				}
			}
			g.lastTile = i
		}
		return
	}
	cur := s.idx(g.cx, g.cy)
	if cur == goal {
		g.arrive(s)
		return
	}
	f := s.flow(goal)
	d := f[cur]
	if d < 0 {
		// can't get there from here any more (something was built)
		g.Target = nil
		if g.leaving {
			s.depart(g)
			g.gone = true
			return
		}
		g.State = Idle
		return
	}
	var opts [4]int
	n := 0
	for _, o := range [4][2]int{{1, 0}, {-1, 0}, {0, 1}, {0, -1}} {
		x, y := g.cx+o[0], g.cy+o[1]
		if !s.in(x, y) {
			continue
		}
		if j := s.idx(x, y); f[j] >= 0 && s.grid[j] == 0 && f[j]+s.StepCost(j) == d {
			opts[n] = s.idx(x, y)
			n++
		}
	}
	if n == 0 {
		g.State = Idle
		g.Target = nil
		return
	}
	next := opts[s.Rand.Intn(n)]
	g.nx, g.ny = next%s.W, next/s.W
	g.moving = true
}

func (g *Guest) moveToward(tx, ty, dt float64) bool {
	dx, dy := tx-g.X, ty-g.Y
	dist := math.Hypot(dx, dy)
	step := g.speed * dt
	if g.onPath {
		step *= 1.3
	}
	if dx > 0.01 {
		g.Facing = 1
	} else if dx < -0.01 {
		g.Facing = -1
	}
	if dist <= step {
		g.X, g.Y = tx, ty
		return true
	}
	g.X += dx / dist * step
	g.Y += dy / dist * step
	return false
}

// arrive: the guest reached the tile they were walking to.
func (g *Guest) arrive(s *Sim) {
	if g.leaving {
		g.outside = true
		return
	}
	a := g.Target
	if a == nil { // finished wandering
		g.State = Idle
		return
	}
	if a.Broken {
		g.Target, g.State = nil, Idle
		g.think(s, "Aw, it's broken.", 0.6)
		g.Happy -= 3
		return
	}
	if a.Price > g.Money {
		g.Target, g.State = nil, Idle
		g.Visits[a.ID] += 2
		g.think(s, "I can't afford that.", 0.5)
		return
	}
	if a.Price > a.Def.FairMax {
		over := float64(a.Price-a.Def.FairMax) / float64(a.Def.MaxPrice-a.Def.FairMax+1)
		if s.chance(0.15 + over*0.85) {
			g.Target, g.State = nil, Idle
			g.Visits[a.ID] += 2
			g.Happy -= 5
			a.Night.Refused++
			g.think(s, "$"+itoa(a.Price)+"? Too pricey!", 0.8)
			return
		}
	}
	if len(a.Queue) >= a.Def.QueueMax {
		g.Target, g.State = nil, Idle
		g.Visits[a.ID]++
		g.Happy -= 3
		g.think(s, "Line's too long.", 0.5)
		return
	}
	g.State = Queuing
	g.wait = 0
	a.Queue = append(a.Queue, g)
}

// decide picks what to do next.
func (g *Guest) decide(s *Sim) {
	if g.leaving {
		g.State = Walking
		return
	}
	switch {
	case s.closing:
		s.sendHome(g, "")
		return
	case g.Happy < 16:
		s.sendHome(g, "I'm going home.")
		return
	case g.Money < 1:
		s.sendHome(g, "I'm out of money!")
		return
	case g.Activities >= g.Stay:
		s.sendHome(g, pick(s, "What a night!", "Time to head home.", "That was swell."))
		return
	}
	g.State = Walking
	g.Target = nil
	cur := s.idx(g.cx, g.cy)

	reach := func(a *Attraction) (int, bool) {
		ex, ey := a.Entrance()
		f := s.flow(s.idx(ex, ey))
		d := f[cur]
		return int(d), d >= 0
	}
	choose := func(ok func(a *Attraction) float64) *Attraction {
		total := 0.0
		w := make([]float64, len(s.Attr))
		for i, a := range s.Attr {
			if !a.Def.Visitable() || a.Broken || len(a.Queue) >= a.Def.QueueMax || a.Price > g.Money {
				continue
			}
			d, can := reach(a)
			if !can {
				continue
			}
			v := ok(a)
			if v <= 0 {
				continue
			}
			v /= 1 + float64(d)/54
			w[i] = v
			total += v
		}
		if total <= 0 {
			return nil
		}
		r := s.Rand.Float64() * total
		for i, v := range w {
			r -= v
			if v > 0 && r <= 0 {
				return s.Attr[i]
			}
		}
		return nil
	}

	// needs first
	if g.Thirst > 65 {
		if a := choose(func(a *Attraction) float64 { return a.Def.Thirst * b2f(a.Def.Thirst > 20) }); a != nil {
			g.Target = a
			return
		}
	}
	if g.Hunger > 62 {
		if a := choose(func(a *Attraction) float64 { return a.Def.Hunger * b2f(a.Def.Hunger > 20) }); a != nil {
			g.Target = a
			return
		}
		if g.grumbleT < 0 {
			g.think(s, "I'm starving!", 0.5)
			g.grumbleT = 12
		}
	}
	if g.Energy < 28 {
		if a := choose(func(a *Attraction) float64 { return a.Def.Rest }); a != nil {
			g.Target = a
			return
		}
		g.Activities++ // too tired to stay as long
	}
	// then fun
	a := choose(func(a *Attraction) float64 {
		d := a.Def
		if d.Cat != CatRide && d.Cat != CatGame {
			return 0
		}
		v := d.Fun / (1 + float64(g.Visits[a.ID])*1.4)
		if a.Price > d.FairMax {
			v *= 0.6
		}
		if g.Kid && (d.ID == "duckpond" || d.ID == "carousel" || d.ID == "teacups") {
			v *= 1.8
		}
		if !g.Kid && (d.ID == "ferris" || d.Cat == CatGame) {
			v *= 1.3
		}
		return v
	})
	if a != nil {
		g.Target = a
		return
	}
	if g.wanders >= 2 {
		s.sendHome(g, "Nothing else to do here.")
		return
	}
	// wander somewhere
	g.wanders++
	g.Activities++
	f := s.flow(cur)
	for tries := 0; tries < 40; tries++ {
		i := s.Rand.Intn(len(f))
		if tries < 30 && !s.Path[i] {
			continue // a stroll along the midway, if there is one
		}
		if f[i] > 0 {
			g.goal = i
			return
		}
	}
	g.goal = cur
}

func b2f(b bool) float64 {
	if b {
		return 1
	}
	return 0
}
