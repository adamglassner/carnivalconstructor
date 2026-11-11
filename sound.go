package main

import (
	"bytes"
	"math"
	"sync"

	"github.com/adamglassner/carnivalconstructor/internal/synth"

	"github.com/hajimehoshi/ebiten/v2/audio"
)

// Music and sound effects are synthesized when the game starts: a calliope
// waltz for the midway and little bells, coins and clanks for everything else.

const sampleRate = 44100

type sound struct {
	mu      sync.Mutex
	ctx     *audio.Context
	sfx     map[string][]byte
	music   *audio.Player
	musicOn bool
	sfxOn   bool
	volume  float64
	fade    float64
	ready   bool
}

var snd = &sound{musicOn: true, sfxOn: true, volume: 0.6}

func (s *sound) init() {
	s.ctx = audio.NewContext(sampleRate)
	go func() {
		fx := synth.Effects()
		tune := synth.Waltz()
		s.mu.Lock()
		defer s.mu.Unlock()
		s.sfx = fx
		loop := audio.NewInfiniteLoop(bytes.NewReader(tune), int64(len(tune)))
		if p, err := s.ctx.NewPlayer(loop); err == nil {
			s.music = p
			p.SetVolume(0)
			p.Play()
		}
		s.ready = true
	}()
}

func (s *sound) play(name string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if !s.sfxOn || s.ctx == nil || s.sfx[name] == nil {
		return
	}
	p := s.ctx.NewPlayerFromBytes(s.sfx[name])
	p.SetVolume(s.volume)
	p.Play()
}

func (s *sound) setMusic(on bool)    { s.mu.Lock(); s.musicOn = on; s.mu.Unlock() }
func (s *sound) setSfx(on bool)      { s.mu.Lock(); s.sfxOn = on; s.mu.Unlock() }
func (s *sound) setVolume(v float64) { s.mu.Lock(); s.volume = v; s.mu.Unlock() }

// update eases the music in and out; quiet is how loud it should be (0-1).
func (s *sound) update(dt, level float64) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.music == nil {
		return
	}
	target := 0.0
	if s.musicOn {
		target = level
	}
	s.fade += (target - s.fade) * math.Min(1, dt/0.4)
	s.music.SetVolume(s.volume * s.fade * 0.55)
}
