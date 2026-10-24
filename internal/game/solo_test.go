package game

import (
	"encoding/json"
	"os"
	"testing"

	"github.com/olivierh59500/indianapolis500/internal/traffic"
)

func TestOriginalSoloFieldsThroughPlayableModeStart(t *testing.T) {
	data, err := os.ReadFile("../traffic/testdata/solo-factory.json")
	if err != nil {
		t.Fatal(err)
	}
	var rows []struct {
		Mode              driveMode
		Seed, RandomAfter uint16
		Before, After     [33]traffic.Car
	}
	if err := json.Unmarshal(data, &rows); err != nil {
		t.Fatal(err)
	}
	g, err := New(Options{Mute: true})
	if err != nil {
		t.Fatal(err)
	}
	for i, source := range rows {
		g.mode = source.Mode
		g.trafficModel.Cars, g.trafficModel.Random = source.Before, source.Seed
		g.startDrive(false)
		for j := range g.cars {
			if g.trafficModel.Cars[j] != source.After[j] {
				t.Fatalf("mode%d case%d car%d playable factory differs from source", source.Mode, i, j)
			}
			active := j == 0 || source.After[j].Next != j
			if g.cars[j].present() != active {
				t.Fatalf("mode%d case%d car%d presentation included a dormant record", source.Mode, i, j)
			}
		}
		if g.trafficModel.Random != source.RandomAfter {
			t.Fatal("playable mode start changed the source random sequence")
		}
	}
}
