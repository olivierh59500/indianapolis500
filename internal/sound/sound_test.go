package sound

import (
	"encoding/binary"
	"sync"
	"testing"

	gameassets "github.com/olivierh59500/indianapolis500/assets"
)

func recoveredStream(t *testing.T) *stream {
	t.Helper()
	read := func(name string) []byte {
		data, err := gameassets.Original.ReadFile("original/" + name)
		if err != nil {
			t.Fatal(err)
		}
		return data
	}
	s, err := newStream(read("engine-sample.bin"), read("noise-sample.bin"), read("collision-sample.bin"))
	if err != nil {
		t.Fatal(err)
	}
	return s
}

func TestContinuousEngineStreamHasNoChunkBoundaryClick(t *testing.T) {
	s := recoveredStream(t)
	c := NewController()
	c.Voices[0].Rate, c.Voices[0].Level = 12500, 60
	s.setFrame(c.Voices, true)
	first, second := make([]byte, 4096), make([]byte, 4096)
	if n, err := s.Read(first); n != len(first) || err != nil {
		t.Fatalf("first read returned %d, %v", n, err)
	}
	if n, err := s.Read(second); n != len(second) || err != nil {
		t.Fatalf("second read returned %d, %v", n, err)
	}
	last, next := int16(binary.LittleEndian.Uint16(first[len(first)-4:])), int16(binary.LittleEndian.Uint16(second))
	delta := int(next) - int(last)
	if delta < 0 {
		delta = -delta
	}
	if delta > 12000 {
		t.Fatalf("PCM chunk boundary jumps by %d samples", delta)
	}
	if testing.AllocsPerRun(100, func() { _, _ = s.Read(first) }) != 0 {
		t.Fatal("continuous PCM rendering allocates")
	}
}

func TestCollisionCueRestartsAndUsesTheCorrectSample(t *testing.T) {
	s := recoveredStream(t)
	c := NewController()
	c.Requests[0] = 80
	c.Step(func() uint16 { return 1 })
	s.setFrame(c.Voices, true)
	buffer := make([]byte, 1024)
	_, _ = s.Read(buffer)
	old := s.playback[3].phase
	if old <= 0 || old >= float64(len(s.samples[CollisionSample])) || s.playback[3].sample != CollisionSample {
		t.Fatal("collision did not use its recovered waveform")
	}
	c.Requests[0] = 80
	c.Step(func() uint16 { return 1 })
	s.setFrame(c.Voices, true)
	_, _ = s.Read(buffer[:4])
	if s.playback[3].phase >= old {
		t.Fatal("collision cue did not restart")
	}
}

func TestOriginalStereoVoiceRouting(t *testing.T) {
	for voice := 0; voice < 4; voice++ {
		s := recoveredStream(t)
		var voices [4]Voice
		for i := range voices {
			voices[i] = Voice{Sample: EngineSample, Rate: 10000}
		}
		voices[voice].Level = 64
		s.setFrame(voices, true)
		buffer := make([]byte, 4096)
		_, _ = s.Read(buffer)
		left, right := false, false
		for i := 0; i < len(buffer); i += 4 {
			left = left || binary.LittleEndian.Uint16(buffer[i:]) != 0
			right = right || binary.LittleEndian.Uint16(buffer[i+2:]) != 0
		}
		if left != (voice == 0 || voice == 3) || right != (voice == 1 || voice == 2) {
			t.Fatalf("voice%d stereo routing left%v right%v", voice, left, right)
		}
	}
}

func TestPlaybackFramePublication(t *testing.T) {
	s := recoveredStream(t)
	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer wg.Done()
		c := NewController()
		for i := 0; i < 1000; i++ {
			c.Voices[0].Level = uint8(i % 64)
			s.setFrame(c.Voices, i%2 == 0)
		}
	}()
	buffer := make([]byte, 4096)
	for i := 0; i < 100; i++ {
		_, _ = s.Read(buffer)
	}
	wg.Wait()
}
