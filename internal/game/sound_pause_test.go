package game

import (
	"encoding/json"
	"math"
	"os"
	"reflect"
	"testing"

	"github.com/olivierh59500/indianapolis500/internal/sound"
)

type playedAudioPauseFrame struct {
	Label      string
	Clock      int
	Controller struct {
		sound.Controller
		Random uint16
	}
	Sources sound.Sources
	Voices  [4]struct {
		Sample                string
		Length, Period, Level int
	}
}

type playedAudioPauseTransition struct{ Before, After playedAudioPauseFrame }

func supplyPlayedAudioPause(g *Game, frame playedAudioPauseFrame) {
	g.soundController = frame.Controller.Controller
	g.soundSources = frame.Sources
	g.trafficModel.Random = frame.Controller.Random
	samples := map[string]sound.Sample{"engine": sound.EngineSample, "noise": sound.NoiseSample, "collision": sound.CollisionSample, "failure": sound.FailureSample}
	for i, voice := range frame.Voices {
		g.soundController.Voices[i] = sound.Voice{Sample: samples[voice.Sample], Rate: 3546895 / float64(voice.Period), Level: uint8(voice.Level), Restart: uint64(10 + i)}
	}
}

func comparePlayedAudioPause(t *testing.T, g *Game, frame playedAudioPauseFrame) {
	t.Helper()
	control := g.soundController
	control.Voices = [4]sound.Voice{}
	if !reflect.DeepEqual(control, frame.Controller.Controller) || !reflect.DeepEqual(g.soundSources, frame.Sources) || g.trafficModel.Random != frame.Controller.Random {
		t.Fatalf("original pause state differs: native control%+v source%+v; native cursors%v source%v; RNG%x source%x", control, frame.Controller.Controller, g.soundSources.Cursor, frame.Sources.Cursor, g.trafficModel.Random, frame.Controller.Random)
	}
	samples := map[string]sound.Sample{"engine": sound.EngineSample, "noise": sound.NoiseSample, "collision": sound.CollisionSample, "failure": sound.FailureSample}
	for i, source := range frame.Voices {
		got := g.soundController.Voices[i]
		if got.Sample != samples[source.Sample] || got.Level != uint8(source.Level) || math.Abs(got.Rate-3546895/float64(source.Period)) > 1e-9 {
			t.Fatalf("original voice%d native%+v source%+v", i, got, source)
		}
	}
}

// Ordinary original P input reaches B3D4/AF06. Android's shared public Pause
// entry must apply that same reset, without changing vehicle or random state.
func TestOriginalDrivingAudioPause(t *testing.T) {
	data, err := os.ReadFile("../sound/testdata/pause-resume.json")
	if err != nil {
		t.Fatal(err)
	}
	var fixture struct{ Pause, PausedWait, Resume, FirstUpdate playedAudioPauseTransition }
	if err = json.Unmarshal(data, &fixture); err != nil {
		t.Fatal(err)
	}
	g, err := New(Options{Mute: true, Practice: true})
	if err != nil {
		t.Fatal(err)
	}
	supplyPlayedAudioPause(g, fixture.Pause.Before)
	before := g.cars[0].physics
	g.Pause()
	if !g.paused || g.cars[0].physics != before {
		t.Fatal("pause changed the physical vehicle state")
	}
	comparePlayedAudioPause(t, g, fixture.Pause.After)
	for i, voice := range g.soundController.Voices {
		want := uint64(10 + i)
		if i == 3 {
			want++
		}
		if voice.Restart != want {
			t.Fatalf("pause voice%d restart%d, expected%d", i, voice.Restart, want)
		}
	}
	pausedVoices := g.soundController.Voices
	g.Pause()
	if g.soundController.Voices != pausedVoices {
		t.Fatal("repeated Android pause restarted a voice")
	}

	// The original dispatcher receives interrupts during pause but its silent
	// callback leaves the controls and shared random sequence unchanged.
	comparePlayedAudioPause(t, g, fixture.PausedWait.Before)
	for ticks := fixture.PausedWait.After.Clock - fixture.PausedWait.Before.Clock; ticks > 0; ticks-- {
		g.advanceSound()
	}
	comparePlayedAudioPause(t, g, fixture.PausedWait.After)
	comparePlayedAudioPause(t, g, fixture.Resume.Before)
	g.setPaused(false)
	if g.paused || g.cars[0].physics != before {
		t.Fatal("resume changed the physical vehicle state")
	}
	comparePlayedAudioPause(t, g, fixture.Resume.After)
	for i, voice := range g.soundController.Voices {
		want := pausedVoices[i].Restart
		if i == 3 {
			want++
		}
		if voice.Restart != want {
			t.Fatalf("resume voice%d restart%d, expected%d", i, voice.Restart, want)
		}
	}
	comparePlayedAudioPause(t, g, fixture.FirstUpdate.Before)
	g.advanceSound()
	comparePlayedAudioPause(t, g, fixture.FirstUpdate.After)
}
