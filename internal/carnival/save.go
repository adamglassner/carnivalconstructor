package carnival

import (
	"errors"
	"math/rand"
)

// Saved is a run as it stands at the start of a day, for continuing later.
type Saved struct {
	Schema  int           `json:"schema"`
	Night   int           `json:"night"`
	Cash    int           `json:"cash"`
	Rep     float64       `json:"rep"`
	Gate    int           `json:"gate"`
	Weather Weather       `json:"weather"`
	Attr    []SavedAttr   `json:"attractions"`
	Traffic []float32     `json:"traffic"`
	Paths   []int         `json:"paths"`
	Reports []NightReport `json:"reports"`
}

type SavedAttr struct {
	ID       string    `json:"id"`
	X        int       `json:"x"`
	Y        int       `json:"y"`
	Price    int       `json:"price"`
	Rig      float64   `json:"rig"`
	Operator string    `json:"operator"`
	Total    AttrStats `json:"total"`
}

// Save captures the run as it stands in the morning. Saved on the results
// screen, it's the next morning.
func (s *Sim) Save() Saved {
	sv := Saved{Schema: 1, Night: s.Night, Cash: s.Cash, Rep: s.Rep, Gate: s.Gate, Weather: s.Weather,
		Traffic: append([]float32{}, s.Traffic...)}
	if s.Phase == PhaseResults {
		sv.Night, sv.Weather = s.Night+1, Clear
	}
	for i, p := range s.Path {
		if p {
			sv.Paths = append(sv.Paths, i)
		}
	}
	for _, a := range s.Attr {
		sv.Attr = append(sv.Attr, SavedAttr{a.Def.ID, a.X, a.Y, a.Price, a.Rig, a.Operator, a.Total})
	}
	r := s.Reports
	if len(r) > 30 {
		r = r[len(r)-30:]
	}
	sv.Reports = append([]NightReport{}, r...)
	return sv
}

// Load picks a saved run back up at the start of its day.
func Load(sv Saved, seed int64) (*Sim, error) {
	if sv.Schema != 1 || sv.Night < 1 {
		return nil, errors.New("not a saved run")
	}
	s := New(seed)
	s.Rand = rand.New(rand.NewSource(seed))
	s.Cash = 1 << 30
	for _, a := range sv.Attr {
		d := DefByID(a.ID)
		if d == nil {
			continue
		}
		at, err := s.Place(d, a.X, a.Y)
		if err != nil {
			continue
		}
		at.Price, at.Rig, at.Total = a.Price, a.Rig, a.Total
		if a.Operator != "" {
			at.Operator = a.Operator
		}
	}
	for _, i := range sv.Paths {
		if i >= 0 && i < len(s.Path) && s.grid[i] == 0 {
			s.Path[i] = true
		}
	}
	s.pathsChanged()
	s.Night, s.Cash, s.Rep, s.Gate, s.Weather = sv.Night, sv.Cash, sv.Rep, sv.Gate, sv.Weather
	if len(sv.Traffic) == len(s.Traffic) {
		copy(s.Traffic, sv.Traffic)
	}
	s.Reports = sv.Reports
	return s, nil
}
