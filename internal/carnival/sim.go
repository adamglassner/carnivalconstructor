package carnival

import (
	"errors"
	"math"
	"math/rand"
)

type Phase int

const (
	PhaseBuild   Phase = iota // daytime: build and set prices
	PhaseNight                // gates open
	PhaseResults              // the night's takings
	PhaseBroke                // ran out of money
)

type Weather int

const (
	Clear Weather = iota
	Breezy
	Drizzle
)

var WeatherNames = [...]string{"Clear", "Breezy", "Drizzle"}

// Event tells the screen something happened (for sounds and pop-ups).
type Event struct {
	Kind string // coin, win, lose, rigged, broke, fixed, gate, full, litter
	Text string
	X, Y float64 // tile coordinates
	A    *Attraction
}

// NightReport is one night's takings.
type NightReport struct {
	Night                    int
	Weather                  Weather
	Guests                   int
	Admissions, Rides, Games int
	Food                     int
	Prizes, Goods, Repairs   int
	Upkeep, Cleanup          int
	Litter                   int
	AvgHappy                 float64
	RepBefore, RepAfter      float64
	MaxRig                   float64 // the most any game was rigged tonight
	RiggedCaught             int     // guests who said "Rigged!?"
	PeakGuests               int
	Wins, Losses             int
}

func (r NightReport) Income() int { return r.Admissions + r.Rides + r.Games + r.Food }
func (r NightReport) Costs() int  { return r.Prizes + r.Goods + r.Repairs + r.Upkeep + r.Cleanup }
func (r NightReport) Profit() int { return r.Income() - r.Costs() }

// Sim is one run of the demo.
type Sim struct {
	Fair   Fair
	W, H   int
	Rand   *rand.Rand
	Phase  Phase
	Night  int     // 1-based
	T      float64 // fraction of the night gone (1 = closing time)
	Cash   int
	Rep    float64 // 0-5 stars
	Gate   int     // admission price
	Attr   []*Attraction
	Guests []*Guest

	Weather Weather

	Report  NightReport
	Reports []NightReport
	Events  []Event

	Litter  []uint8
	Path    []bool    // footpath tiles the player laid
	Traffic []float32 // footsteps per tile; worn grass
	Lit     []float32 // how lit each tile is at night (0-1)

	GateX, GateY int

	grid      []int // attraction ID per tile, 0 = free
	trashNear []bool
	flows     map[int][]int16
	nextID    int
	spawnAcc  float64
	closing   bool
	happySum  float64
	departed  int
	rigHits   float64
	Version   int // bumps whenever the layout changes
}

// New starts a fresh demo.
func New(seed int64) *Sim {
	f := DemoFair
	s := &Sim{Fair: f, W: f.W, H: f.H, Rand: rand.New(rand.NewSource(seed)), Night: 1, Cash: StartCash, Rep: f.Rep, Gate: DefaultGate}
	n := s.W * s.H
	s.grid = make([]int, n)
	s.Litter = make([]uint8, n)
	s.Path = make([]bool, n)
	s.Traffic = make([]float32, n)
	s.Lit = make([]float32, n)
	s.trashNear = make([]bool, n)
	s.GateX, s.GateY = 0, s.H/2
	s.Path[s.idx(0, s.GateY)], s.Path[s.idx(1, s.GateY)] = true, true
	s.Weather = Clear // a gentle first night
	s.layoutChanged()
	return s
}

func (s *Sim) idx(x, y int) int       { return y*s.W + x }
func (s *Sim) in(x, y int) bool       { return x >= 0 && y >= 0 && x < s.W && y < s.H }
func (s *Sim) Free(x, y int) bool     { return s.in(x, y) && s.grid[s.idx(x, y)] == 0 }
func (s *Sim) emit(e Event)           { s.Events = append(s.Events, e) }
func (s *Sim) GateTile() (int, int)   { return s.GateX, s.GateY }
func (s *Sim) Closing() bool          { return s.closing }
func (s *Sim) reserved(x, y int) bool { return y == s.GateY && x <= 1 }
func clamp(v, lo, hi float64) float64 { return math.Max(lo, math.Min(hi, v)) }
func (s *Sim) chance(p float64) bool  { return s.Rand.Float64() < p }
func (s *Sim) AttrAt(x, y int) *Attraction {
	if !s.in(x, y) {
		return nil
	}
	id := s.grid[s.idx(x, y)]
	for _, a := range s.Attr {
		if a.ID == id {
			return a
		}
	}
	return nil
}

/* ---------- building ---------- */

var (
	ErrLocked  = errors.New("Coming in the full game")
	ErrCash    = errors.New("Not enough cash")
	ErrSpace   = errors.New("Something's in the way")
	ErrEdge    = errors.New("Off the edge of the lot")
	ErrBlocks  = errors.New("That would block the way through")
	ErrGateway = errors.New("Keep the gate clear")
	ErrDoor    = errors.New("Keep entrances clear")
	ErrNoFun   = errors.New("Build at least one ride or midway game first")
)

// CanPlace checks whether def fits with its top-left tile at (x, y).
func (s *Sim) CanPlace(d *Def, x, y int) error {
	if d.Locked {
		return ErrLocked
	}
	if x < 0 || y < 0 || x+d.W > s.W || y+d.H > s.H {
		return ErrEdge
	}
	for ty := y; ty < y+d.H; ty++ {
		for tx := x; tx < x+d.W; tx++ {
			if s.reserved(tx, ty) {
				return ErrGateway
			}
			if !s.Free(tx, ty) {
				return ErrSpace
			}
			for _, a := range s.Attr {
				if ex, ey := a.Entrance(); a.Def.Visitable() && ex == tx && ey == ty {
					return ErrDoor
				}
			}
		}
	}
	if d.Visitable() {
		ex, ey := x+d.W/2, y+d.H
		if !s.Free(ex, ey) {
			return ErrSpace
		}
	}
	// Try it and make sure every entrance can still be reached from the gate.
	id := -1
	s.stamp(d, x, y, id)
	ok := s.allReachable()
	s.stamp(d, x, y, 0)
	if !ok {
		return ErrBlocks
	}
	if s.Cash < d.Cost {
		return ErrCash
	}
	return nil
}

func (s *Sim) stamp(d *Def, x, y, id int) {
	for ty := y; ty < y+d.H; ty++ {
		for tx := x; tx < x+d.W; tx++ {
			s.grid[s.idx(tx, ty)] = id
		}
	}
}

func (s *Sim) allReachable() bool {
	f := s.bfs(s.idx(s.GateX, s.GateY))
	for _, a := range s.Attr {
		if !a.Def.Visitable() {
			continue
		}
		ex, ey := a.Entrance()
		if !s.in(ex, ey) || s.grid[s.idx(ex, ey)] != 0 || f[s.idx(ex, ey)] < 0 {
			return false
		}
	}
	return true
}

// Place builds def at (x, y).
func (s *Sim) Place(d *Def, x, y int) (*Attraction, error) {
	if err := s.CanPlace(d, x, y); err != nil {
		return nil, err
	}
	s.nextID++
	a := &Attraction{ID: s.nextID, Def: d, X: x, Y: y, Price: d.Price, Operator: s.operatorName()}
	s.Attr = append(s.Attr, a)
	s.stamp(d, x, y, a.ID)
	s.Cash -= d.Cost
	for ty := y; ty < y+d.H; ty++ {
		for tx := x; tx < x+d.W; tx++ {
			if i := s.idx(tx, ty); s.Path[i] {
				s.Path[i] = false
				s.Cash += PathRefund
			}
		}
	}
	s.layoutChanged()
	s.unstickGuests()
	return a, nil
}

// Sell removes an attraction for part of what it cost.
func (s *Sim) Sell(a *Attraction) int {
	back := int(float64(a.Def.Cost) * SellBack)
	for _, g := range append(append([]*Guest{}, a.Queue...), a.Riding...) {
		g.Target = nil
		g.State = Idle
		g.X, g.Y = a.entranceCenter()
		g.cx, g.cy = a.Entrance()
		g.think(s, "Aw, it's closed.", 0.5)
		g.Happy -= 4
	}
	a.Queue, a.Riding = nil, nil
	s.stamp(a.Def, a.X, a.Y, 0)
	for i, b := range s.Attr {
		if b == a {
			s.Attr = append(s.Attr[:i], s.Attr[i+1:]...)
			break
		}
	}
	for _, g := range s.Guests {
		if g.Target == a {
			g.Target = nil
			g.State = Idle
		}
	}
	s.Cash += back
	s.layoutChanged()
	return back
}

var ErrPath = errors.New("There's already a path here")

// CanPath checks whether a footpath can go on (x, y).
func (s *Sim) CanPath(x, y int) error {
	if !s.in(x, y) {
		return ErrEdge
	}
	i := s.idx(x, y)
	if s.grid[i] != 0 {
		return ErrSpace
	}
	if s.Path[i] {
		return ErrPath
	}
	if s.Cash < PathCost {
		return ErrCash
	}
	return nil
}

// PlacePath lays a footpath tile.
func (s *Sim) PlacePath(x, y int) error {
	if err := s.CanPath(x, y); err != nil {
		return err
	}
	s.Path[s.idx(x, y)] = true
	s.Cash -= PathCost
	s.pathsChanged()
	return nil
}

// RemovePath takes a footpath tile up again (the gate path stays).
func (s *Sim) RemovePath(x, y int) bool {
	if !s.in(x, y) || s.reserved(x, y) || !s.Path[s.idx(x, y)] {
		return false
	}
	s.Path[s.idx(x, y)] = false
	s.Cash += PathRefund
	s.pathsChanged()
	return true
}

func (s *Sim) pathsChanged() {
	s.flows = map[int][]int16{}
	s.Version++
}

// Repair sends a mechanic to a broken ride.
func (s *Sim) Repair(a *Attraction) error {
	if !a.Broken || a.Repairing {
		return nil
	}
	if s.Cash < RepairCost {
		return ErrCash
	}
	s.Cash -= RepairCost
	s.Report.Repairs += RepairCost
	a.Repairing, a.RepairT = true, RepairSeconds
	return nil
}

func (s *Sim) SetPrice(a *Attraction, p int) {
	a.Price = int(clamp(float64(p), 0, float64(a.Def.MaxPrice)))
}

func (s *Sim) SetRig(a *Attraction, r float64) {
	a.Rig = clamp(r, 0, 1)
}

func (s *Sim) SetGate(p int) { s.Gate = int(clamp(float64(p), 0, MaxGate)) }

func (s *Sim) layoutChanged() {
	s.flows = map[int][]int16{}
	s.Version++
	for i := range s.Lit {
		s.Lit[i] = 0
		s.trashNear[i] = false
	}
	light := func(cx, cy, r float64) {
		for y := 0; y < s.H; y++ {
			for x := 0; x < s.W; x++ {
				d := math.Hypot(float64(x)+.5-cx, float64(y)+.5-cy)
				if v := float32(1 - d/r); v > s.Lit[s.idx(x, y)] {
					s.Lit[s.idx(x, y)] = v
				}
			}
		}
	}
	light(float64(s.GateX)+.5, float64(s.GateY)+.5, 2.6)
	for _, a := range s.Attr {
		cx, cy := a.Center()
		if a.Def.Light > 0 {
			light(cx, cy, a.Def.Light+float64(a.Def.W)/2)
		}
		if a.Def.ID == "trash" {
			for y := 0; y < s.H; y++ {
				for x := 0; x < s.W; x++ {
					if math.Hypot(float64(x)+.5-cx, float64(y)+.5-cy) <= 3.2 {
						s.trashNear[s.idx(x, y)] = true
					}
				}
			}
		}
	}
}

// IsLit reports whether a tile is lit well enough at night.
func (s *Sim) IsLit(x, y int) bool { return s.in(x, y) && s.Lit[s.idx(x, y)] > 0.12 }

// unstickGuests moves anyone standing where something was just built.
func (s *Sim) unstickGuests() {
	for _, g := range s.Guests {
		if g.State == Queuing || g.State == Riding || g.outside {
			continue
		}
		if s.Free(g.cx, g.cy) {
			if g.moving && !s.Free(g.nx, g.ny) {
				g.moving = false
				g.X, g.Y = float64(g.cx)+.5, float64(g.cy)+.5
			}
			continue
		}
		f := s.bfs(s.idx(g.cx, g.cy))
		best, bd := -1, int16(math.MaxInt16)
		for i, d := range f {
			if d >= 0 && d < bd && s.grid[i] == 0 {
				best, bd = i, d
			}
		}
		if best >= 0 {
			g.cx, g.cy = best%s.W, best/s.W
			g.X, g.Y = float64(g.cx)+.5, float64(g.cy)+.5
			g.moving = false
		}
	}
}

// bfs measures walking distance from tile `from` to every free tile (-1 = can't get there).
// The start tile counts as walkable even when something is built on it.
func (s *Sim) bfs(from int) []int16 {
	d := make([]int16, len(s.grid))
	for i := range d {
		d[i] = -1
	}
	d[from] = 0
	q := []int{from}
	for len(q) > 0 {
		i := q[0]
		q = q[1:]
		x, y := i%s.W, i/s.W
		for _, n := range [4][2]int{{1, 0}, {-1, 0}, {0, 1}, {0, -1}} {
			nx, ny := x+n[0], y+n[1]
			if !s.in(nx, ny) {
				continue
			}
			j := s.idx(nx, ny)
			if d[j] < 0 && s.grid[j] == 0 {
				d[j] = d[i] + 1
				q = append(q, j)
			}
		}
	}
	return d
}

// StepCost is how much guests mind walking onto a tile: they'd rather keep to the paths.
func (s *Sim) StepCost(i int) int16 {
	if s.Path[i] {
		return 2
	}
	return 3
}

// flow measures the cost of walking from every tile to `target`, preferring paths.
// A guest on tile i steps to the neighbour j where f[j]+StepCost(j) == f[i].
func (s *Sim) flow(target int) []int16 {
	if f, ok := s.flows[target]; ok {
		return f
	}
	f := make([]int16, len(s.grid))
	for i := range f {
		f[i] = -1
	}
	f[target] = 0
	// costs are small integers, so a bucket queue does the job of a priority queue
	buckets := [][]int{{target}}
	for c := 0; c < len(buckets); c++ {
		for k := 0; k < len(buckets[c]); k++ {
			i := buckets[c][k]
			if int(f[i]) != c {
				continue
			}
			x, y := i%s.W, i/s.W
			step := s.StepCost(i)
			for _, n := range [4][2]int{{1, 0}, {-1, 0}, {0, 1}, {0, -1}} {
				nx, ny := x+n[0], y+n[1]
				if !s.in(nx, ny) {
					continue
				}
				j := s.idx(nx, ny)
				if s.grid[j] != 0 {
					continue
				}
				nd := f[i] + step
				if f[j] < 0 || nd < f[j] {
					f[j] = nd
					for len(buckets) <= int(nd) {
						buckets = append(buckets, nil)
					}
					buckets[nd] = append(buckets[nd], j)
				}
			}
		}
	}
	s.flows[target] = f
	return f
}

/* ---------- nights ---------- */

// FunCount is how many rides and games are built.
func (s *Sim) FunCount() int {
	n := 0
	for _, a := range s.Attr {
		if a.Def.Cat == CatRide || a.Def.Cat == CatGame {
			n++
		}
	}
	return n
}

// OpenGates starts the night.
func (s *Sim) OpenGates() error {
	if s.Phase != PhaseBuild {
		return nil
	}
	if s.FunCount() == 0 {
		return ErrNoFun
	}
	s.Phase = PhaseNight
	s.T = 0
	s.closing = false
	s.spawnAcc = 0
	s.happySum, s.departed, s.rigHits = 0, 0, 0
	s.Report = NightReport{Night: s.Night, Weather: s.Weather, RepBefore: s.Rep}
	for _, a := range s.Attr {
		a.Night = AttrStats{}
		s.Report.MaxRig = math.Max(s.Report.MaxRig, a.Rig)
	}
	s.emit(Event{Kind: "gate", Text: "The gates are open!"})
	return nil
}

func (s *Sim) appeal() float64 {
	v := 0.2
	for _, a := range s.Attr {
		switch a.Def.Cat {
		case CatRide:
			v += 0.28
		case CatGame:
			v += 0.12
		case CatFood:
			v += 0.08
		}
	}
	return clamp(v, 0.2, 1.6)
}

// SpawnRate is how many guests arrive per second right now.
func (s *Sim) SpawnRate() float64 {
	curve := 1.0
	switch {
	case s.T < 0.25:
		curve = 0.5 + 2*s.T
	case s.T > 0.7:
		curve = math.Max(0.15, 1-(s.T-0.7)*4)
	}
	w := []float64{1, 0.9, 0.62}[s.Weather]
	gate := clamp(1.35-float64(s.Gate)/11, 0.25, 1.35)
	return 0.62 * (0.45 + s.Rep/4.5) * s.appeal() * w * curve * gate
}

// Update advances the simulation by dt seconds of game time.
func (s *Sim) Update(dt float64) {
	if s.Phase != PhaseNight {
		return
	}
	for dt > 0 {
		step := math.Min(dt, 0.05)
		s.step(step)
		dt -= step
		if s.Phase != PhaseNight {
			return
		}
	}
}

func (s *Sim) step(dt float64) {
	s.T += dt / NightSeconds
	if !s.closing && s.T >= 1 {
		s.closing = true
		for _, g := range s.Guests {
			if g.State != Riding && !g.leaving {
				s.sendHome(g, "Closing time already?")
			}
		}
		for _, a := range s.Attr {
			a.Queue = nil
		}
	}
	if !s.closing && s.T < 0.86 && len(s.Guests) < 240 {
		s.spawnAcc += s.SpawnRate() * dt
		for s.spawnAcc >= 1 {
			s.spawnAcc--
			s.spawn()
		}
	}
	for _, a := range s.Attr {
		a.update(s, dt)
	}
	alive := s.Guests[:0]
	for _, g := range s.Guests {
		g.update(s, dt)
		if !g.gone {
			alive = append(alive, g)
		}
	}
	for i := len(alive); i < len(s.Guests); i++ {
		s.Guests[i] = nil
	}
	s.Guests = alive
	if n := len(s.Guests); n > s.Report.PeakGuests {
		s.Report.PeakGuests = n
	}
	if s.closing && (len(s.Guests) == 0 || s.T > 1.3) {
		s.endNight()
	}
}

func (s *Sim) spawn() {
	g := &Guest{ID: s.Report.Guests + 1 + (s.Night-1)*1000}
	g.Kid = s.chance(0.25)
	g.Name = guestName(s.Rand, g.Kid)
	g.Money = 18 + s.Rand.Intn(32)
	if g.Kid {
		g.Money = 10 + s.Rand.Intn(16)
	}
	g.Happy = 58 + s.Rand.Float64()*14
	g.Hunger = s.Rand.Float64() * 45
	g.Thirst = s.Rand.Float64() * 45
	g.Energy = 70 + s.Rand.Float64()*30
	g.Stay = 5 + s.Rand.Intn(6)
	g.Look = s.Rand.Intn(8)
	g.Skin = s.Rand.Intn(5)
	g.Visits = map[int]int{}
	g.speed = 1.55 + s.Rand.Float64()*0.35
	if g.Kid {
		g.speed += 0.2
	}
	g.cx, g.cy = s.GateX, s.GateY
	g.X, g.Y = -0.9, float64(s.GateY)+.5
	g.outside = true // walking in through the gate
	g.ox, g.oy = (s.Rand.Float64()-.5)*.36, (s.Rand.Float64()-.5)*.36
	s.Cash += s.Gate
	s.Report.Admissions += s.Gate
	s.Report.Guests++
	s.Guests = append(s.Guests, g)
}

func (s *Sim) sendHome(g *Guest, why string) {
	if g.State == Queuing && g.Target != nil {
		g.Target.dequeue(g)
	}
	g.leaving = true
	g.Target = nil
	g.State = Walking
	g.goal = s.idx(s.GateX, s.GateY)
	if why != "" {
		g.think(s, why, 0.35)
	}
}

func (s *Sim) endNight() {
	for _, g := range s.Guests {
		s.depart(g)
	}
	s.Guests = nil
	for _, a := range s.Attr {
		a.Queue, a.Riding, a.Running = nil, nil, false
		a.Broken, a.Repairing = false, false // the crew fixes everything overnight
		s.Report.Upkeep += a.Def.Upkeep
	}
	litter := 0
	for i, l := range s.Litter {
		litter += int(l)
		s.Litter[i] = 0
	}
	r := &s.Report
	r.Litter = litter
	r.Cleanup = litter * CleanupEach
	s.Cash -= r.Upkeep + r.Cleanup
	avg := 60.0
	if s.departed > 0 {
		avg = s.happySum / float64(s.departed)
	}
	r.AvgHappy = avg
	delta := (avg-58)/45*0.5 - s.rigHits
	delta -= math.Max(0, float64(litter)-20) * 0.006
	s.Rep = clamp(s.Rep+delta, 0.5, 5)
	r.RepAfter = s.Rep
	s.Reports = append(s.Reports, *r)
	s.Phase = PhaseResults
}

func (s *Sim) depart(g *Guest) {
	s.happySum += clamp(g.Happy, 0, 100)
	s.departed++
}

// Next moves on from the results screen.
func (s *Sim) Next() {
	if s.Phase != PhaseResults {
		return
	}
	if s.Cash < -1500 {
		s.Phase = PhaseBroke
		return
	}
	s.Night++
	s.Weather = []Weather{Clear, Clear, Breezy, Drizzle}[s.Rand.Intn(4)]
	s.Phase = PhaseBuild
}

// Totals adds up every night so far.
func (s *Sim) Totals() NightReport {
	var t NightReport
	for _, r := range s.Reports {
		t.Guests += r.Guests
		t.Admissions += r.Admissions
		t.Rides += r.Rides
		t.Games += r.Games
		t.Food += r.Food
		t.Prizes += r.Prizes
		t.Goods += r.Goods
		t.Repairs += r.Repairs
		t.Upkeep += r.Upkeep
		t.Cleanup += r.Cleanup
		t.Litter += r.Litter
		t.Wins += r.Wins
		t.Losses += r.Losses
		t.RiggedCaught += r.RiggedCaught
		t.MaxRig = math.Max(t.MaxRig, r.MaxRig)
		t.AvgHappy += r.AvgHappy / float64(len(s.Reports))
	}
	return t
}

// AvgHappy is the mood of the guests on the lot right now.
func (s *Sim) AvgHappy() float64 {
	if len(s.Guests) == 0 {
		return 0
	}
	t := 0.0
	for _, g := range s.Guests {
		t += clamp(g.Happy, 0, 100)
	}
	return t / float64(len(s.Guests))
}

// Clock is the time of night as hours and minutes (6:00 PM to 11:00 PM).
func (s *Sim) Clock() (h, m int) {
	mins := 18*60 + int(math.Min(s.T, 1)*300)
	return mins / 60, mins % 60
}

func (s *Sim) operatorName() string {
	return []string{"Dolores", "Big Lou", "Rita", "Marv", "Tammy", "Earl", "Shirley", "Duke", "Vonda", "Gus", "Pearl", "Hank", "Lorraine", "Skeeter", "Bev", "Otis"}[s.Rand.Intn(16)]
}

var adultNames = []string{"Amy", "Ben", "Carla", "Dave", "Erin", "Frank", "Gina", "Hal", "Iris", "Joe", "Kim", "Lou", "Mae", "Nate", "Olive", "Pete", "Rosa", "Sam", "Tina", "Vic", "Wanda", "Walt", "Jo", "Ray", "Deb", "Cal", "June", "Russ"}
var kidNames = []string{"Tommy", "Lily", "Max", "Ruby", "Sammy", "Ella", "Jake", "Zoe", "Billy", "Rosie", "Theo", "Maisie"}

func guestName(r *rand.Rand, kid bool) string {
	if kid {
		return kidNames[r.Intn(len(kidNames))]
	}
	return adultNames[r.Intn(len(adultNames))]
}
