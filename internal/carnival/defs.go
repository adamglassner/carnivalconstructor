// Package carnival is the simulation behind Carnival Constructor Classic: the lot,
// the attractions, the guests and the money. It has no graphics, so it can be
// tested on its own.
package carnival

// Cat is an attraction's category; it decides which build tab it's on.
type Cat int

const (
	CatRide Cat = iota
	CatGame
	CatFood
	CatExtra
)

var CatNames = [...]string{"Rides", "Midway Games", "Food", "Extras"}

// Def describes one kind of attraction.
type Def struct {
	ID, Name string
	Cat      Cat
	W, H     int // footprint in tiles; guests line up below the middle column
	Cost     int // to build
	Price    int // default ticket price
	MaxPrice int
	FairMax  int     // guests start to grumble above this price
	Capacity int     // guests per cycle
	Cycle    float64 // seconds per ride, game or serving
	Fun      float64 // happiness a ride or game gives
	Hunger   float64 // how much it fills you up
	Thirst   float64
	Rest     float64 // energy back (benches)
	Upkeep   int     // wages and running costs per night
	Goods    float64 // cost of the food per serving
	Prize    float64 // cost of a prize
	BaseWin  float64 // chance to win a midway game played fair
	Breaks   float64 // chance to break down per cycle (rides)
	QueueMax int
	Light    float64 // radius it lights at night, in tiles
	Locked   bool    // shown but saved for the full game
	Blurb    string
}

// Visitable reports whether guests go to it (trash cans and lamps they don't).
func (d *Def) Visitable() bool { return d.Capacity > 0 }

var Defs = []*Def{
	// Rides
	{ID: "carousel", Name: "Carousel", Cat: CatRide, W: 3, H: 3, Cost: 1800, Price: 3, MaxPrice: 10, FairMax: 4, Capacity: 10, Cycle: 9, Fun: 32, Upkeep: 140, Breaks: 0.02, QueueMax: 14, Light: 2.6,
		Blurb: "Painted horses and a calliope. Everyone loves it."},
	{ID: "teacups", Name: "Teacup Twirl", Cat: CatRide, W: 3, H: 3, Cost: 2400, Price: 4, MaxPrice: 10, FairMax: 5, Capacity: 8, Cycle: 8, Fun: 40, Upkeep: 170, Breaks: 0.035, QueueMax: 14, Light: 2.4,
		Blurb: "Spin till you're dizzy. Breaks down now and then."},
	{ID: "ferris", Name: "Ferris Wheel", Cat: CatRide, W: 3, H: 3, Cost: 3400, Price: 5, MaxPrice: 12, FairMax: 6, Capacity: 16, Cycle: 14, Fun: 46, Upkeep: 220, Breaks: 0.02, QueueMax: 18, Light: 3.6,
		Blurb: "The big draw. Lights up the whole lot at night."},
	{ID: "barrel", Name: "Gravity Barrel", Cat: CatRide, W: 3, H: 3, Cost: 5200, Locked: true, Blurb: "Coming in the full game."},
	{ID: "coaster", Name: "Wild Mouse", Cat: CatRide, W: 4, H: 3, Cost: 8800, Locked: true, Blurb: "Coming in the full game."},

	// Midway games
	{ID: "ringtoss", Name: "Ring Toss", Cat: CatGame, W: 2, H: 2, Cost: 650, Price: 2, MaxPrice: 6, FairMax: 3, Capacity: 2, Cycle: 3.5, Fun: 14, Upkeep: 60, Prize: 4, BaseWin: 0.32, QueueMax: 5, Light: 1.4,
		Blurb: "Land a ring on a bottle. Set the odds yourself."},
	{ID: "darts", Name: "Balloon Darts", Cat: CatGame, W: 2, H: 2, Cost: 550, Price: 2, MaxPrice: 6, FairMax: 3, Capacity: 2, Cycle: 3, Fun: 12, Upkeep: 55, Prize: 3, BaseWin: 0.38, QueueMax: 5, Light: 1.4,
		Blurb: "Pop three balloons, win a plush. Set the odds yourself."},
	{ID: "duckpond", Name: "Duck Pond", Cat: CatGame, W: 2, H: 2, Cost: 420, Price: 1, MaxPrice: 4, FairMax: 2, Capacity: 3, Cycle: 2.5, Fun: 10, Upkeep: 40, Prize: 1.5, BaseWin: 0.6, QueueMax: 6, Light: 1.4,
		Blurb: "Every duck's a winner. Mostly. Kids love it."},
	{ID: "striker", Name: "High Striker", Cat: CatGame, W: 2, H: 2, Cost: 1100, Locked: true, Blurb: "Coming in the full game."},

	// Food
	{ID: "corndogs", Name: "Corn Dogs", Cat: CatFood, W: 2, H: 2, Cost: 520, Price: 4, MaxPrice: 9, FairMax: 5, Capacity: 1, Cycle: 2.2, Fun: 6, Hunger: 60, Thirst: -8, Upkeep: 50, Goods: 1.2, QueueMax: 6, Light: 1.6,
		Blurb: "Salty and filling. Makes guests thirsty."},
	{ID: "funnel", Name: "Funnel Cake", Cat: CatFood, W: 2, H: 2, Cost: 600, Price: 5, MaxPrice: 10, FairMax: 6, Capacity: 1, Cycle: 2.6, Fun: 10, Hunger: 45, Upkeep: 55, Goods: 1.4, QueueMax: 6, Light: 1.6,
		Blurb: "Powdered sugar everywhere. Guests are happier for it."},
	{ID: "lemonade", Name: "Lemon Shake-Ups", Cat: CatFood, W: 2, H: 2, Cost: 380, Price: 3, MaxPrice: 7, FairMax: 4, Capacity: 1, Cycle: 1.8, Fun: 5, Thirst: 65, Upkeep: 40, Goods: 0.6, QueueMax: 6, Light: 1.6,
		Blurb: "Fresh-squeezed and ice cold."},

	// Extras
	{ID: "bench", Name: "Bench", Cat: CatExtra, W: 1, H: 1, Cost: 80, Capacity: 2, Cycle: 7, Rest: 55, QueueMax: 2,
		Blurb: "Tired guests sit down and stay longer."},
	{ID: "trash", Name: "Trash Can", Cat: CatExtra, W: 1, H: 1, Cost: 60,
		Blurb: "Guests nearby don't drop litter."},
	{ID: "lamp", Name: "Light Pole", Cat: CatExtra, W: 1, H: 1, Cost: 140, Upkeep: 8, Light: 3.2,
		Blurb: "Dark corners make guests uneasy. Light them up."},
}

// DefByID finds a definition, or nil.
func DefByID(id string) *Def {
	for _, d := range Defs {
		if d.ID == id {
			return d
		}
	}
	return nil
}

// The one fair in the demo, which runs night after night. The full game goes on tour.
type Fair struct {
	Name string
	W, H int
	Rep  float64 // the town's starting opinion of you
}

var DemoFair = Fair{Name: "Pine Hollow County Fair", W: 24, H: 14, Rep: 2.5}

const (
	StartCash     = 8000
	NightSeconds  = 210.0 // 6 PM to 11 PM
	RepairCost    = 120
	RepairSeconds = 7.0
	CleanupEach   = 2 // per piece of litter at closing
	DefaultGate   = 5
	MaxGate       = 15
	SellBack      = 0.6
	PathCost      = 10 // per tile
	PathRefund    = 5
)
