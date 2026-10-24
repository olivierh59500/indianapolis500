package game

import (
	"encoding/json"
	"os"
	"reflect"
	"strconv"
	"testing"

	"github.com/olivierh59500/indianapolis500/internal/sound"
)

// Played replay poses pass through the real game camera and emitter adapter,
// independently of the standalone source-selection implementation tests.
func TestOriginalReplaySoundThroughGame(t *testing.T) {
	data, err := os.ReadFile("../sound/testdata/exterior-sources.json")
	if err != nil {
		t.Fatal(err)
	}
	var rows []struct {
		Kind                        string
		Tick                        int
		Before, After               sound.Controller
		BeforeSources, AfterSources sound.Sources
		Actors                      []sound.MovingSource
		Listener                    sound.Listener
	}
	if err = json.Unmarshal(data, &rows); err != nil {
		t.Fatal(err)
	}
	g, err := New(Options{Mute: true, Practice: true})
	if err != nil {
		t.Fatal(err)
	}
	for _, row := range rows {
		if row.Tick >= 24 {
			continue
		}
		t.Run(strconv.Itoa(row.Tick), func(t *testing.T) {
			g.replay, g.demo, g.replayView = true, false, inCarView
			g.cars = [33]car{}
			for i, a := range row.Actors {
				index := a.ID - 33
				g.cars[index] = car{active: true, position: float64((trackLength - int(a.Distance)) % trackLength), lateral: float64(a.Lateral), yaw: a.Heading}
				g.soundPrevious[index] = sound.MovingSource{Distance: a.PreviousDistance, Lateral: a.PreviousLateral}
				g.trafficModel.Cars[index].Next = row.Actors[(i+1)%len(row.Actors)].ID - 33
				g.trafficModel.Cars[index].Number = 0
				if a.LowEngine {
					g.trafficModel.Cars[index].Number = 128
				}
			}
			g.cars[0].physics.FrontGaugeSpeed = row.Listener.Motion << 8
			g.soundController, g.soundSources = row.Before, row.BeforeSources
			g.updateSoundSources()
			if !reflect.DeepEqual(g.soundController, row.After) || !reflect.DeepEqual(g.soundSources, row.AfterSources) {
				t.Fatal("played original replay sound differs through the game adapter")
			}
		})
	}
}

func TestMutedAudioKeepsTheSharedGameRandomSequence(t *testing.T) {
	g, err := New(Options{Mute: true, Practice: true})
	if err != nil {
		t.Fatal(err)
	}
	start := g.trafficModel.Random
	for range 29 {
		g.advanceSound()
	}
	if g.trafficModel.Random != start {
		t.Fatal("noise variation advanced before its original interval")
	}
	g.advanceSound()
	if g.trafficModel.Random != start*0x6255+0x3619 {
		t.Fatal("muted audio lost the original shared random update")
	}
	g.Pause()
	before := g.soundController
	for range 60 {
		g.advanceSound()
	}
	if g.soundController != before {
		t.Fatal("paused audio advanced its controller")
	}
}

// The audio controller shares random updates with the surrounding drivers.
// Exercise the actual sixty-Hz Update path rather than advancing physics alone.
func TestDemonstrationContinuesWithOriginalSoundClock(t *testing.T) {
	g, err := New(Options{Mute: true, Demo: true})
	if err != nil {
		t.Fatal(err)
	}
	for range 12000 {
		if err = g.Update(); err != nil {
			t.Fatal(err)
		}
	}
	player := &g.cars[0]
	if g.tick != 3000 || player.progress < 39600 || player.speed < 100 {
		t.Fatalf("sound-clock demonstration stalled: ticks%d progress%.0f speed%.1f", g.tick, player.progress, player.speed)
	}
}
