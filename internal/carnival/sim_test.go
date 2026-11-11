package carnival

import "testing"

func build(t *testing.T, s *Sim, id string, x, y int) *Attraction {
	t.Helper()
	a, err := s.Place(DefByID(id), x, y)
	if err != nil {
		t.Fatalf("place %s at %d,%d: %v", id, x, y, err)
	}
	return a
}

func runNight(t *testing.T, s *Sim) NightReport {
	t.Helper()
	if err := s.OpenGates(); err != nil {
		t.Fatal(err)
	}
	for i := 0; i < 100000 && s.Phase == PhaseNight; i++ {
		s.Update(0.1)
	}
	if s.Phase != PhaseResults {
		t.Fatalf("night never ended (T=%.2f, %d guests left)", s.T, len(s.Guests))
	}
	return s.Reports[len(s.Reports)-1]
}

func TestPlacementRules(t *testing.T) {
	s := New(1)
	if err := s.CanPlace(DefByID("barrel"), 5, 2); err != ErrLocked {
		t.Fatalf("locked ride: %v", err)
	}
	if err := s.CanPlace(DefByID("carousel"), 22, 2); err != ErrEdge {
		t.Fatalf("off the edge: %v", err)
	}
	if err := s.CanPlace(DefByID("bench"), 1, s.GateY); err != ErrGateway {
		t.Fatalf("gate: %v", err)
	}
	build(t, s, "carousel", 5, 2)
	if err := s.CanPlace(DefByID("ringtoss"), 6, 3); err != ErrSpace {
		t.Fatalf("overlap: %v", err)
	}
	// building on a ride's entrance isn't allowed
	if err := s.CanPlace(DefByID("bench"), 6, 5); err != ErrDoor {
		t.Fatalf("entrance: %v", err)
	}
	// a wall of benches that would cut the carousel off from the gate
	for x := 2; x <= 10; x++ {
		if x != 6 {
			build(t, s, "bench", x, 6)
		}
	}
	for y := 1; y <= 5; y++ {
		build(t, s, "trash", 4, y)
		build(t, s, "trash", 8, y)
	}
	for x := 5; x <= 7; x++ {
		build(t, s, "trash", x, 1)
	}
	if err := s.CanPlace(DefByID("trash"), 6, 6); err != ErrBlocks {
		t.Fatalf("blocking check: %v", err)
	}
	cash := s.Cash
	back := s.Sell(s.Attr[0])
	if s.Cash != cash+back || back != int(float64(1800)*SellBack) {
		t.Fatalf("sell back %d", back)
	}
}
func TestNeedsSomethingFun(t *testing.T) {
	s := New(2)
	build(t, s, "corndogs", 5, 5)
	if err := s.OpenGates(); err != ErrNoFun {
		t.Fatalf("expected ErrNoFun, got %v", err)
	}
}

func TestThreeNights(t *testing.T) {
	s := New(42)
	build(t, s, "carousel", 4, 2)
	build(t, s, "ringtoss", 9, 3)
	build(t, s, "corndogs", 12, 3)
	build(t, s, "lemonade", 15, 3)
	build(t, s, "trash", 8, 7)
	build(t, s, "lamp", 11, 7)
	build(t, s, "bench", 14, 8)
	r := runNight(t, s)
	t.Logf("night 1: %d guests, income $%d, profit $%d, happy %.0f, rep %.2f->%.2f, litter %d",
		r.Guests, r.Income(), r.Profit(), r.AvgHappy, r.RepBefore, r.RepAfter, r.Litter)
	if r.Guests < 30 || r.Rides == 0 || r.Games == 0 || r.Food == 0 {
		t.Fatalf("a quiet night: %+v", r)
	}
	s.Next()
	if s.Phase != PhaseBuild || s.Night != 2 {
		t.Fatalf("phase %v night %d", s.Phase, s.Night)
	}
	build(t, s, "ferris", 18, 2)
	build(t, s, "darts", 4, 9)
	build(t, s, "funnel", 8, 10)
	r = runNight(t, s)
	t.Logf("night 2: %d guests, income $%d, profit $%d, happy %.0f, rep %.2f", r.Guests, r.Income(), r.Profit(), r.AvgHappy, r.RepAfter)
	s.Next()
	r = runNight(t, s)
	t.Logf("night 3: %d guests, income $%d, profit $%d, happy %.0f, rep %.2f, cash $%d", r.Guests, r.Income(), r.Profit(), r.AvgHappy, r.RepAfter, s.Cash)
	s.Next()
	if s.Phase != PhaseBuild || s.Night != 4 {
		t.Fatalf("nights should go on: phase %v night %d", s.Phase, s.Night)
	}
	// save and pick the run back up
	sv := s.Save()
	l, err := Load(sv, 9)
	if err != nil || l.Night != 4 || l.Cash != s.Cash || len(l.Attr) != len(s.Attr) || l.Rep != s.Rep {
		t.Fatalf("load: %v night %d cash %d/%d attrs %d/%d", err, l.Night, l.Cash, s.Cash, len(l.Attr), len(s.Attr))
	}
	r = runNight(t, l)
	t.Logf("night 4 after loading: %d guests, profit $%d", r.Guests, r.Profit())
	for n := 5; n <= 12; n++ {
		l.Next()
		r = runNight(t, l)
	}
	t.Logf("night %d: %d guests, profit $%d, cash $%d, rep %.2f", r.Night, r.Guests, r.Profit(), l.Cash, l.Rep)
}

func TestRiggingCostsReputation(t *testing.T) {
	play := func(rig float64) NightReport {
		s := New(7)
		build(t, s, "carousel", 4, 2)
		a := build(t, s, "ringtoss", 9, 3)
		b := build(t, s, "darts", 12, 3)
		build(t, s, "corndogs", 15, 3)
		s.SetRig(a, rig)
		s.SetRig(b, rig)
		return runNight(t, s)
	}
	fair, rigged := play(0), play(1)
	t.Logf("fair: wins %d, prizes $%d, rep %.2f; rigged: wins %d, prizes $%d, rep %.2f, caught %d",
		fair.Wins, fair.Prizes, fair.RepAfter, rigged.Wins, rigged.Prizes, rigged.RepAfter, rigged.RiggedCaught)
	if rigged.Prizes >= fair.Prizes || rigged.RepAfter >= fair.RepAfter || rigged.RiggedCaught == 0 {
		t.Fatal("rigging should pay less in prizes and cost reputation")
	}
}

func TestBreakdownAndRepair(t *testing.T) {
	var s *Sim
	var a *Attraction
	for seed := int64(1); seed < 40; seed++ {
		s = New(seed)
		a = build(t, s, "teacups", 4, 2)
		build(t, s, "corndogs", 9, 3)
		if err := s.OpenGates(); err != nil {
			t.Fatal(err)
		}
		for i := 0; i < 20000 && !a.Broken && s.Phase == PhaseNight; i++ {
			s.Update(0.1)
		}
		if a.Broken {
			break
		}
	}
	if !a.Broken {
		t.Fatal("never broke down")
	}
	cash := s.Cash
	if err := s.Repair(a); err != nil {
		t.Fatal(err)
	}
	if s.Cash != cash-RepairCost {
		t.Fatal("repair should cost money")
	}
	for i := 0; i < 200 && a.Broken; i++ {
		s.Update(0.1)
	}
	if a.Broken {
		t.Fatal("still broken after the repair")
	}
}

func TestGuestsPreferPaths(t *testing.T) {
	s := New(11)
	build(t, s, "carousel", 18, 2)
	// a path from the gate along row 7 and up to the carousel's entrance (19, 5)
	for x := 2; x <= 19; x++ {
		if err := s.PlacePath(x, s.GateY); err != nil {
			t.Fatal(err)
		}
	}
	for y := 5; y < s.GateY; y++ {
		if err := s.PlacePath(19, y); err != nil {
			t.Fatal(err)
		}
	}
	if err := s.PlacePath(5, s.GateY); err != ErrPath {
		t.Fatalf("double path: %v", err)
	}
	r := runNight(t, s)
	on, off := float32(0), float32(0)
	for i, tr := range s.Traffic {
		if s.Path[i] {
			continue
		}
		off += tr
	}
	_ = on
	t.Logf("%d guests, %0.f steps off the path", r.Guests, off)
	if r.Guests < 10 || off > float32(r.Guests)*6 {
		t.Fatalf("guests should mostly keep to the path (%.0f steps off it)", off)
	}
	cash := s.Cash
	if !s.RemovePath(10, s.GateY) || s.Cash != cash+PathRefund {
		t.Fatal("remove path")
	}
	if s.RemovePath(0, s.GateY) {
		t.Fatal("the gate path should stay")
	}
	sv := s.Save()
	l, _ := Load(sv, 1)
	if l.Path[l.idx(10, s.GateY)] || !l.Path[l.idx(11, s.GateY)] {
		t.Fatal("paths not saved")
	}
}
