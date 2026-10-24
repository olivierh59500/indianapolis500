package sound

import (
	"encoding/json"
	"os"
	"reflect"
	"strconv"
	"testing"

	gameassets "github.com/olivierh59500/indianapolis500/assets"
	"github.com/olivierh59500/indianapolis500/internal/scene"
	"github.com/olivierh59500/indianapolis500/internal/vehicle"
)

func TestOriginalInstantReplaySoundSources(t *testing.T) {
	data, err := os.ReadFile("testdata/exterior-sources.json")
	if err != nil {
		t.Fatal(err)
	}
	var rows []struct {
		Tick, ClockBefore, ClockAfter int
		Before, After                 originalMixerState
		BeforeSources, AfterSources   Sources
		Actors                        []MovingSource
		Listener                      Listener
	}
	if err = json.Unmarshal(data, &rows); err != nil {
		t.Fatal(err)
	}
	read := func(name string) []byte {
		data, err := gameassets.Original.ReadFile("original/" + name)
		if err != nil {
			t.Fatal(err)
		}
		return data
	}
	world, err := scene.New(read("world-vertices.bin"), read("world-data.bin"), read("world-trigonometry.bin"), read("world-horizon.bin"), read("world-lod.bin"))
	if err != nil {
		t.Fatal(err)
	}
	if err = world.LoadCameras(read("camera-arctangent.bin"), read("camera-bank-gradient.bin"), read("camera-trackside.bin")); err != nil {
		t.Fatal(err)
	}
	surface, err := vehicle.NewTrackSurface(read("track-surface-wave.bin"))
	if err != nil {
		t.Fatal(err)
	}
	for _, row := range rows {
		t.Run(strconv.Itoa(row.Tick), func(t *testing.T) {
			c, s := row.Before.Controller, row.BeforeSources
			s.Exterior(&c, row.Actors, row.Listener, world, &surface)
			c.Voices = [4]Voice{}
			if !reflect.DeepEqual(c, row.After.Controller) || !reflect.DeepEqual(s, row.AfterSources) {
				t.Fatalf("replay sound differs clocks%d/%d: native pitch%v steps%v levels%v noise%d/%d selected%v original pitch%v steps%v levels%v noise%d/%d selected%v", row.ClockBefore, row.ClockAfter, c.Pitch, c.PitchStep, c.EngineLevel, c.NoisePitch, c.NoiseStep, s.Previous, row.After.Pitch, row.After.PitchStep, row.After.EngineLevel, row.After.NoisePitch, row.After.NoiseStep, row.AfterSources.Previous)
			}
		})
	}
}
