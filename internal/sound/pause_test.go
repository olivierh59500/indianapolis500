package sound

import (
	"encoding/json"
	"math"
	"os"
	"testing"
)

// The control transitions and three playback snapshots are exported from an
// ordinary original P pause/resume. Playback offsets establish continued muted
// sample progression, not synchronized native/Amiga waveform equivalence.
func TestOriginalPauseKeepsMutedEnginePlaybackRunning(t *testing.T) {
	data, err := os.ReadFile("testdata/pause-resume.json")
	if err != nil {
		t.Fatal(err)
	}
	type frame struct {
		Controller originalMixerState
		Voices     [4]originalVoice
	}
	var fixture struct {
		Pause  struct{ Before, After frame }
		Phases []struct {
			Label  string
			Paused bool
			Voices [4]struct {
				Volume, Period, Length, NextByteOffset int
			}
		}
	}
	if err = json.Unmarshal(data, &fixture); err != nil {
		t.Fatal(err)
	}
	if len(fixture.Phases) != 3 || fixture.Phases[0].Paused || !fixture.Phases[1].Paused || fixture.Phases[2].Paused {
		t.Fatal("original full playback states are missing")
	}
	before, paused := fixture.Phases[0], fixture.Phases[1]
	c := fixture.Pause.Before.Controller.Controller
	s := recoveredStream(t)
	for i, source := range fixture.Pause.Before.Voices {
		sample := EngineSample
		if i == 3 {
			sample = NoiseSample
		}
		c.Voices[i] = Voice{Sample: sample, Rate: originalPlaybackClock / float64(source.Period), Level: uint8(source.Level), Restart: uint64(10 + i)}
		s.playback[i] = playback{phase: float64(before.Voices[i].NextByteOffset), gain: float64(source.Level) / 64, sample: sample, restart: c.Voices[i].Restart}
		if paused.Voices[i].Volume != 0 || paused.Voices[i].Period != fixture.Pause.After.Voices[i].Period {
			t.Fatalf("source paused voice%d is not muted at its retained rate", i)
		}
		if i < 3 && before.Voices[i].NextByteOffset == paused.Voices[i].NextByteOffset {
			t.Fatalf("source muted engine%d did not advance", i)
		}
	}
	c.ResetDrivingControls()
	s.setFrame(c.Voices, false)
	buffer := make([]byte, 4096)
	_, _ = s.Read(buffer)
	for i, state := range s.playback {
		start := float64(before.Voices[i].NextByteOffset)
		if i == 3 {
			start = 0 // Only the original noise channel is restarted by AF06.
		}
		want := math.Mod(start+1024*c.Voices[i].Rate/SampleRate, float64(len(s.samples[c.Voices[i].Sample])))
		if math.Abs(state.phase-want) > 1e-7 {
			t.Fatalf("muted voice%d phase%.9f, expected%.9f", i, state.phase, want)
		}
	}
	for _, value := range buffer[len(buffer)-256:] {
		if value != 0 {
			t.Fatal("paused PCM did not settle to silence")
		}
	}

	// Resume invokes the reset again. Reading actual PCM must retain all three
	// engine positions and restart noise even before the next control update.
	previous := s.playback
	c.ResetDrivingControls()
	s.setFrame(c.Voices, true)
	_, _ = s.Read(buffer[:4])
	for i, state := range s.playback {
		start := previous[i].phase
		if i == 3 {
			start = 0
		}
		want := math.Mod(start+c.Voices[i].Rate/SampleRate, float64(len(s.samples[c.Voices[i].Sample])))
		if math.Abs(state.phase-want) > 1e-7 {
			t.Fatalf("resumed voice%d phase%.9f, expected%.9f", i, state.phase, want)
		}
	}
	c.Step(func() uint16 { t.Fatal("first resumed mixer advanced the source RNG"); return 0 })
	s.setFrame(c.Voices, true)
	_, _ = s.Read(buffer[:4])
	if buffer[0] != 0 || buffer[1] != 0 || buffer[2] != 0 || buffer[3] != 0 {
		t.Fatal("first resumed mixer should still produce silent PCM")
	}
}
