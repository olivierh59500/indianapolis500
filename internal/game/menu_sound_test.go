package game

import (
	"encoding/json"
	"os"
	"reflect"
	"testing"

	"github.com/olivierh59500/indianapolis500/internal/traffic"
)

type menuAudioFixture struct {
	Pause, Resume, FirstUpdate playedAudioPauseTransition
}

func originalMenuAudioFixture(t *testing.T) menuAudioFixture {
	t.Helper()
	raw, err := os.ReadFile("../sound/testdata/pause-resume.json")
	if err != nil {
		t.Fatal(err)
	}
	var fixture menuAudioFixture
	if err := json.Unmarshal(raw, &fixture); err != nil {
		t.Fatal(err)
	}
	return fixture
}

func menuAudioGame(frame playedAudioPauseFrame) *Game {
	g := &Game{scene: driveScene, session: true, trafficModel: &traffic.Model{}}
	supplyPlayedAudioPause(g, frame)
	return g
}

// Both normal menu entry2B622 and resume2BA24 reach the same AF06 reset
// captured by ordinary P input. No fixture fields are generated from Go.
func TestOriginalMenuDrivingAudioTransitions(t *testing.T) {
	fixture := originalMenuAudioFixture(t)
	g := menuAudioGame(fixture.Pause.Before)
	beforeCars, beforeRandom := g.cars, g.trafficModel.Random
	beforeVoices := g.soundController.Voices
	g.Back()
	if g.scene != menuScene || g.paused || g.menuDrivingPaused {
		t.Fatal("ordinary Escape did not enter an active menu")
	}
	comparePlayedAudioPause(t, g, fixture.Pause.After)
	for index, voice := range g.soundController.Voices {
		want := beforeVoices[index].Restart
		if index == 3 {
			want++
		}
		if voice.Restart != want {
			t.Fatalf("menu voice%d restarted%d times", index, voice.Restart-beforeVoices[index].Restart)
		}
	}
	menuVoices := g.soundController.Voices
	g.activateMenuItem()
	if g.scene != driveScene || g.paused {
		t.Fatal("Resume did not return to active driving")
	}
	comparePlayedAudioPause(t, g, fixture.Resume.After)
	for index, voice := range g.soundController.Voices {
		want := menuVoices[index].Restart
		if index == 3 {
			want++
		}
		if voice.Restart != want {
			t.Fatalf("resume voice%d restart%d, want%d", index, voice.Restart, want)
		}
	}
	g.advanceSound()
	comparePlayedAudioPause(t, g, fixture.FirstUpdate.After)
	if !reflect.DeepEqual(g.cars, beforeCars) || g.trafficModel.Random != beforeRandom {
		t.Fatal("menu transitions changed the physical cars or shared random state")
	}
}

// B394 returns before AF06 while a separate P pause still owns a reference.
// The menu remains usable, then Resume restores that pause without a reset.
func TestMenuRetainsSeparateDrivingPause(t *testing.T) {
	fixture := originalMenuAudioFixture(t)
	g := menuAudioGame(fixture.Pause.Before)
	g.Pause()
	pausedVoices := g.soundController.Voices
	g.Back()
	if g.scene != menuScene || g.paused || !g.menuDrivingPaused {
		t.Fatal("menu failed to retain the separate manual pause")
	}
	comparePlayedAudioPause(t, g, fixture.Pause.After)
	for index, voice := range g.soundController.Voices {
		want := pausedVoices[index].Restart
		if index == 3 {
			want++
		}
		if voice.Restart != want {
			t.Fatalf("nested menu voice%d restart%d, want%d", index, voice.Restart, want)
		}
	}
	menuControl, menuSources := g.soundController, g.soundSources
	g.activateMenuItem()
	if g.scene != driveScene || !g.paused || g.menuDrivingPaused {
		t.Fatal("Resume lost the separate P pause")
	}
	if !reflect.DeepEqual(g.soundController, menuControl) || !reflect.DeepEqual(g.soundSources, menuSources) {
		t.Fatal("returning to a still-paused session reset audio twice")
	}
	g.Pause()
	g.advanceSound()
	if !reflect.DeepEqual(g.soundController, menuControl) {
		t.Fatal("repeated pause advanced or reset audio")
	}
	g.setPaused(false)
	comparePlayedAudioPause(t, g, fixture.Resume.After)
	for index, voice := range g.soundController.Voices {
		want := menuControl.Voices[index].Restart
		if index == 3 {
			want++
		}
		if voice.Restart != want {
			t.Fatalf("P resume voice%d restart%d, want%d", index, voice.Restart, want)
		}
	}
}

func TestUnavailableResumeLeavesAudioUntouched(t *testing.T) {
	fixture := originalMenuAudioFixture(t)
	g := menuAudioGame(fixture.Pause.Before)
	g.scene, g.session = menuScene, false
	beforeControl, beforeSources := g.soundController, g.soundSources
	g.activateMenuItem()
	if g.scene != menuScene || !reflect.DeepEqual(g.soundController, beforeControl) || !reflect.DeepEqual(g.soundSources, beforeSources) {
		t.Fatal("unavailable Resume changed scene or audio")
	}
}

func TestAttractRetainsManualMenuPauseAndSettings(t *testing.T) {
	for _, paused := range []bool{false, true} {
		name := "driving"
		if paused {
			name = "P-paused"
		}
		t.Run(name, func(t *testing.T) {
			fixture := originalMenuAudioFixture(t)
			g, err := New(Options{Mute: true, Practice: true, SavePath: t.TempDir()})
			if err != nil {
				t.Fatal(err)
			}
			supplyPlayedAudioPause(g, fixture.Pause.Before)
			if paused {
				g.Pause()
			}
			g.Back()
			cars, model, clock := g.cars, g.trafficModel.State, g.clock
			setup, team, rank := g.setup, g.team, g.qualifyingPosition
			menuControl, menuSources := g.soundController, g.soundSources
			g.startDrive(true)
			for updates := 0; updates < 8; updates++ {
				if err := g.Update(); err != nil {
					t.Fatal(err)
				}
			}
			g.stopDemo()
			if g.scene != menuScene || g.paused || g.menuDrivingPaused != paused {
				t.Fatal("demonstration lost the manual menu pause")
			}
			if !reflect.DeepEqual(g.cars, cars) || !reflect.DeepEqual(g.trafficModel.State, model) || g.clock != clock ||
				!reflect.DeepEqual(g.setup, setup) || g.team != team || g.qualifyingPosition != rank {
				t.Fatal("demonstration changed saved manual mechanics or settings")
			}
			if !reflect.DeepEqual(g.soundController, menuControl) || !reflect.DeepEqual(g.soundSources, menuSources) {
				t.Fatal("demonstration did not restore the saved manual audio state")
			}
			g.activateMenuItem()
			if g.scene != driveScene || g.paused != paused {
				t.Fatal("manual Resume restored the wrong pause")
			}
			if paused {
				if !reflect.DeepEqual(g.soundController, menuControl) {
					t.Fatal("still-paused manual Resume reset audio")
				}
			} else {
				comparePlayedAudioPause(t, g, fixture.Resume.After)
				if g.soundController.Voices[3].Restart != menuControl.Voices[3].Restart+1 {
					t.Fatal("active manual Resume must reset noise once")
				}
			}
			g.startDrive(false)
			if g.paused || g.menuDrivingPaused {
				t.Fatal("new manual session retained an old pause")
			}
		})
	}
}
