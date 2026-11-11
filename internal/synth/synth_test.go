package synth

import (
	"testing"
	"time"
)

func TestMake(t *testing.T) {
	start := time.Now()
	w := Waltz()
	fx := Effects()
	t.Logf("waltz %.1fs of audio, %d effects, made in %v", float64(len(w))/4/sampleRate, len(fx), time.Since(start))
	if len(w) == 0 || len(w)%4 != 0 || len(fx) < 10 {
		t.Fatal("bad audio")
	}
	for _, n := range []string{"click", "place", "coin", "win", "rigged", "broke", "fixed", "gate", "fanfare", "error", "sell", "cash", "close", "tab"} {
		if len(fx[n]) == 0 {
			t.Fatalf("missing %s", n)
		}
	}
}
