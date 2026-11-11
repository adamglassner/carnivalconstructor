// Package synth makes Carnival Constructor's music and sound effects from
// scratch: a calliope waltz and little bells, coins and clanks.
package synth

import (
	"math"
	"math/rand"
)

const sampleRate = 44100

/* ---------- synthesis ---------- */

type buf []float64 // mono samples

func newBuf(sec float64) buf { return make(buf, int(sec*sampleRate)) }

func freq(midi float64) float64 { return 440 * math.Pow(2, (midi-69)/12) }

// add mixes a voice into b starting at t seconds.
func (b buf) add(t, dur float64, f func(tt, p float64) float64, amp float64) {
	start := int(t * sampleRate)
	n := int(dur * sampleRate)
	for i := 0; i < n && start+i < len(b); i++ {
		if start+i < 0 {
			continue
		}
		tt := float64(i) / sampleRate
		b[start+i] += f(tt, tt/dur) * amp
	}
}

func env(tt, dur, a, r float64) float64 {
	e := 1.0
	if tt < a {
		e = tt / a
	}
	if tt > dur-r {
		e *= math.Max(0, (dur-tt)/r)
	}
	return e
}

// calliope: a breathy steam-organ pipe with vibrato.
func calliope(midi, dur float64) func(tt, p float64) float64 {
	f := freq(midi)
	r := rand.New(rand.NewSource(int64(midi * 100)))
	return func(tt, p float64) float64 {
		vib := 1 + 0.006*math.Sin(2*math.Pi*5.6*tt)
		ph := 2 * math.Pi * f * vib * tt
		v := math.Sin(ph) + 0.45*math.Sin(2*ph) + 0.22*math.Sin(3*ph) + 0.1*math.Sin(4*ph)
		v += (r.Float64() - .5) * 0.06
		return v * env(tt, dur, 0.012, 0.07)
	}
}

func tuba(midi, dur float64) func(tt, p float64) float64 {
	f := freq(midi)
	return func(tt, p float64) float64 {
		ph := 2 * math.Pi * f * tt
		v := math.Sin(ph) + 0.35*math.Sin(2*ph) + 0.18*math.Sin(3*ph)
		return v * env(tt, dur, 0.01, 0.08) * math.Exp(-tt*2.2)
	}
}

func bell(midi, dur float64) func(tt, p float64) float64 {
	f := freq(midi)
	return func(tt, p float64) float64 {
		ph := 2 * math.Pi * f * tt
		v := math.Sin(ph) + 0.5*math.Sin(2.76*ph) + 0.25*math.Sin(5.4*ph)
		return v * math.Exp(-tt*5) * env(tt, dur, 0.002, 0.02)
	}
}

func square(midi, dur float64) func(tt, p float64) float64 {
	f := freq(midi)
	return func(tt, p float64) float64 {
		v := 0.0
		for k := 1.0; k <= 7; k += 2 {
			v += math.Sin(2*math.Pi*f*k*tt) / k
		}
		return v * env(tt, dur, 0.004, 0.03)
	}
}

func sweep(f0, f1, dur float64) func(tt, p float64) float64 {
	ph := 0.0
	last := 0.0
	return func(tt, p float64) float64 {
		f := f0 + (f1-f0)*p
		ph += 2 * math.Pi * f * (tt - last)
		last = tt
		return (math.Sin(ph) + 0.3*math.Sin(3*ph)) * env(tt, dur, 0.005, 0.05)
	}
}

func noise(dur, decay float64) func(tt, p float64) float64 {
	r := rand.New(rand.NewSource(7))
	return func(tt, p float64) float64 {
		return (r.Float64()*2 - 1) * math.Exp(-tt*decay) * env(tt, dur, 0.001, 0.02)
	}
}

// pcm turns mono samples into 16-bit stereo bytes, with a soft limiter.
func (b buf) pcm(gain float64) []byte {
	out := make([]byte, len(b)*4)
	for i, v := range b {
		x := math.Tanh(v * gain)
		s := int16(x * 30000)
		out[i*4], out[i*4+1] = byte(s), byte(s>>8)
		out[i*4+2], out[i*4+3] = byte(s), byte(s>>8)
	}
	return out
}

// Waltz writes "Pine Hollow Waltz": 32 bars of oom-pah-pah in C, as 16-bit stereo PCM.
func Waltz() []byte {
	const bpm = 168.0
	beat := 60 / bpm
	type note struct{ m, beats float64 }
	n := func(m, b float64) note { return note{m, b} }
	R := -1.0 // rest
	melody := [][]note{
		{n(67, 1), n(72, 1), n(76, 1)}, {n(79, 2), n(76, 1)}, {n(77, 1), n(74, 1), n(71, 1)}, {n(67, 3)},
		{n(74, 1), n(77, 1), n(79, 1)}, {n(77, 2), n(74, 1)}, {n(76, 1), n(74, 1), n(72, 1)}, {n(72, 2), n(R, 1)},
		{n(76, 1), n(79, 1), n(84, 1)}, {n(83, 1), n(81, 1), n(79, 1)}, {n(81, 2), n(77, 1)}, {n(72, 1), n(74, 1), n(77, 1)},
		{n(76, 2), n(79, 1)}, {n(77, 1), n(74, 1), n(71, 1)}, {n(72, .5), n(74, .5), n(76, 1), n(74, 1)}, {n(72, 3)},
	}
	chords := [][3]float64{ // root (bass), two chord tones
		{48, 64, 67}, {48, 64, 67}, {43, 62, 65}, {43, 62, 65}, {43, 62, 65}, {43, 62, 65}, {48, 64, 67}, {48, 64, 67},
		{48, 64, 67}, {48, 64, 67}, {41, 65, 69}, {41, 65, 69}, {48, 64, 67}, {43, 62, 65}, {48, 64, 67}, {48, 64, 67},
	}
	bars := 32
	b := newBuf(float64(bars*3)*beat + 0.01)
	for bar := 0; bar < bars; bar++ {
		k := bar % 16
		t0 := float64(bar*3) * beat
		ch := chords[k]
		alt := ch[0]
		if bar%2 == 1 {
			alt = ch[0] + 7 // walk the bass to the fifth
			if alt > 50 {
				alt -= 12
			}
		}
		b.add(t0, beat*0.9, tuba(alt, beat*0.9), 0.32)
		for j := 1; j < 3; j++ {
			for _, m := range ch[1:] {
				b.add(t0+float64(j)*beat, beat*0.45, calliope(m-12, beat*0.45), 0.07)
			}
		}
		t := t0
		up := 0.0
		if bar >= 16 {
			up = 12 // second time round, an octave up and brighter
		}
		for _, nt := range melody[k] {
			d := nt.beats * beat
			if nt.m > 0 {
				b.add(t, d*0.92, calliope(nt.m+up-12, d*0.92), 0.2)
				if bar >= 16 {
					b.add(t, d*0.92, calliope(nt.m-12, d*0.92), 0.1)
				}
			}
			t += d
		}
		if bar%4 == 0 { // a glockenspiel tap on the downbeat
			b.add(t0, 0.6, bell(ch[2]+12, 0.6), 0.05)
		}
	}
	return b.pcm(1.6)
}

// Effects makes every sound effect, by name, as 16-bit stereo PCM.
func Effects() map[string][]byte {
	m := map[string][]byte{}
	mk := func(name string, sec, gain float64, f func(b buf)) {
		b := newBuf(sec)
		f(b)
		m[name] = b.pcm(gain)
	}
	mk("click", 0.06, 1, func(b buf) { b.add(0, 0.05, bell(96, 0.05), 0.35) })
	mk("tab", 0.08, 1, func(b buf) { b.add(0, 0.07, square(84, 0.07), 0.15) })
	mk("place", 0.5, 1, func(b buf) {
		b.add(0, 0.18, sweep(160, 55, 0.18), 0.7)
		b.add(0, 0.05, noise(0.05, 60), 0.3)
		for i, n := range []float64{84, 88, 91} {
			b.add(0.1+float64(i)*0.06, 0.25, bell(n, 0.25), 0.18)
		}
	})
	mk("sell", 0.35, 1, func(b buf) {
		b.add(0, 0.2, sweep(400, 120, 0.2), 0.4)
		b.add(0.12, 0.2, bell(88, 0.2), 0.2)
	})
	mk("coin", 0.3, 1, func(b buf) {
		b.add(0, 0.07, square(83, 0.07), 0.16)
		b.add(0.06, 0.22, square(88, 0.22), 0.16)
	})
	mk("cash", 0.6, 1, func(b buf) {
		b.add(0, 0.08, noise(0.08, 40), 0.35)
		b.add(0.06, 0.5, bell(91, 0.5), 0.3)
		b.add(0.09, 0.5, bell(96, 0.5), 0.2)
	})
	mk("win", 0.6, 1, func(b buf) {
		for i, n := range []float64{72, 76, 79, 84} {
			b.add(float64(i)*0.07, 0.3, square(n, 0.3), 0.12)
		}
	})
	mk("rigged", 0.5, 1, func(b buf) { b.add(0, 0.45, sweep(420, 180, 0.45), 0.4) })
	mk("broke", 0.8, 1, func(b buf) {
		b.add(0, 0.12, noise(0.12, 25), 0.6)
		b.add(0.05, 0.6, sweep(640, 120, 0.6), 0.35)
		b.add(0.1, 0.2, bell(55, 0.2), 0.3)
	})
	mk("fixed", 0.5, 1, func(b buf) {
		b.add(0, 0.12, square(76, 0.12), 0.15)
		b.add(0.12, 0.3, square(84, 0.3), 0.15)
	})
	mk("gate", 1.6, 1, func(b buf) {
		for i, n := range []float64{72, 76, 79, 84} {
			b.add(float64(i)*0.12, 1.2, bell(n+12, 1.2), 0.25)
		}
		b.add(0.5, 1.0, calliope(72, 1.0), 0.18)
		b.add(0.5, 1.0, calliope(76, 1.0), 0.14)
	})
	mk("fanfare", 1.2, 1, func(b buf) {
		for i, n := range []float64{67, 72, 76} {
			b.add(float64(i)*0.1, 0.15, square(n, 0.15), 0.12)
		}
		b.add(0.3, 0.8, square(79, 0.8), 0.12)
		b.add(0.3, 0.8, square(84, 0.8), 0.08)
	})
	mk("error", 0.2, 1, func(b buf) { b.add(0, 0.16, square(40, 0.16), 0.3) })
	mk("close", 1.6, 1, func(b buf) {
		for i, n := range []float64{84, 79, 76, 72} {
			b.add(float64(i)*0.22, 0.9, bell(n, 0.9), 0.25)
		}
	})
	return m
}
