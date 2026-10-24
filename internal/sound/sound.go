// Package sound plays the game's recovered engine, road-noise and impact PCM.
package sound

import (
	"encoding/binary"
	"fmt"
	"io"
	"sync"
	"time"

	"github.com/hajimehoshi/ebiten/v2/audio"
)

const SampleRate = 44100

var (
	contextOnce sync.Once
	context     *audio.Context
)

// Engine connects the shared native playback stream to Ebitengine audio.
type Engine struct {
	stream *stream
	player *audio.Player
}

type playback struct {
	phase, gain float64
	sample      Sample
	restart     uint64
}

type stream struct {
	samples  [4][]byte
	mu       sync.Mutex
	voices   [4]Voice
	active   bool
	playback [4]playback
}

func newStream(engine, noise, collision []byte) (*stream, error) {
	if len(engine) != 22256 || len(noise) != 4960 || len(collision) != 10000 {
		return nil, fmt.Errorf("invalid recovered sound samples")
	}
	s := &stream{}
	s.samples[EngineSample] = append([]byte(nil), engine...)
	s.samples[NoiseSample] = append([]byte(nil), noise...)
	s.samples[CollisionSample] = append([]byte(nil), collision...)
	s.samples[FailureSample] = s.samples[CollisionSample][2704:4704]
	return s, nil
}

// New starts a silent, allocation-free PCM stream from the original waveforms.
func New(engine, noise, collision []byte) (*Engine, error) {
	source, err := newStream(engine, noise, collision)
	if err != nil {
		return nil, err
	}
	contextOnce.Do(func() { context = audio.NewContext(SampleRate) })
	player, err := newPCMPlayer(context, source)
	if err != nil {
		return nil, err
	}
	return &Engine{stream: source, player: player}, nil
}

func newPCMPlayer(context *audio.Context, source io.Reader) (*audio.Player, error) {
	player, err := context.NewPlayer(source)
	if err != nil {
		return nil, err
	}
	// Keep about two control updates buffered instead of the backend's half
	// second default. The output device has its own additional audio queue.
	player.SetBufferSize(time.Second / 30)
	player.SetVolume(.72)
	player.Play()
	return player, nil
}

// SetFrame publishes complete voice parameters together, avoiding torn pitch,
// sample and gain changes while the audio goroutine reads its next buffer.
func (e *Engine) SetFrame(voices [4]Voice, active bool) {
	if e == nil {
		return
	}
	e.stream.setFrame(voices, active)
}

func (s *stream) setFrame(voices [4]Voice, active bool) {
	s.mu.Lock()
	s.voices, s.active = voices, active
	s.mu.Unlock()
}

func (e *Engine) Close() error {
	if e == nil || e.player == nil {
		return nil
	}
	return e.player.Close()
}

func (s *stream) Read(buffer []byte) (int, error) {
	s.mu.Lock()
	voices, active := s.voices, s.active
	s.mu.Unlock()
	var steps, gains [4]float64
	for i, v := range voices {
		p := &s.playback[i]
		if p.sample != v.Sample || p.restart != v.Restart {
			p.phase, p.sample, p.restart = 0, v.Sample, v.Restart
		}
		steps[i] = v.Rate / SampleRate
		if active {
			gains[i] = float64(min(v.Level, uint8(64))) / 64
		}
	}
	frames := len(buffer) / 4
	for frame := 0; frame < frames; frame++ {
		left, right := 0.0, 0.0
		for i, v := range voices {
			p := &s.playback[i]
			// A short gain transition prevents clicks on pause and cue changes. The
			// game's own four-step pitch interpolation is retained without another ramp.
			p.gain += (gains[i] - p.gain) * .02
			sample := s.samples[v.Sample]
			value := sampleAt(sample, p.phase) * p.gain * .5
			if i == 0 || i == 3 {
				left += value
			} else {
				right += value
			}
			p.phase += steps[i]
			for p.phase >= float64(len(sample)) {
				p.phase -= float64(len(sample))
			}
		}
		writeSample(buffer[frame*4:], left, right)
	}
	clear(buffer[frames*4:])
	return len(buffer), nil
}

func sampleAt(sample []byte, phase float64) float64 {
	if len(sample) == 0 {
		return 0
	}
	for phase >= float64(len(sample)) {
		phase -= float64(len(sample))
	}
	for phase < 0 {
		phase += float64(len(sample))
	}
	index := int(phase)
	next := index + 1
	if next == len(sample) {
		next = 0
	}
	fraction := phase - float64(index)
	first, second := float64(int8(sample[index])), float64(int8(sample[next]))
	return (first + (second-first)*fraction) * 256
}

func writeSample(target []byte, left, right float64) {
	left = max(-32768, min(32767, left))
	right = max(-32768, min(32767, right))
	binary.LittleEndian.PutUint16(target, uint16(int16(left)))
	binary.LittleEndian.PutUint16(target[2:], uint16(int16(right)))
}

var _ io.Reader = (*stream)(nil)
