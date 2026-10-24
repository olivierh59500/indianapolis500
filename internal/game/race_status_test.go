package game

import (
	"encoding/json"
	"os"
	"testing"

	"github.com/olivierh59500/indianapolis500/internal/traffic"
)

// Source snapshots enter the real session adapter with distinct player and
// leader finish flags. Finishing changes the information page, not the scene.
func TestOriginalRaceStatusThroughGame(t *testing.T) {
	data, err := os.ReadFile("../traffic/testdata/race-status.json")
	if err != nil {
		t.Fatal(err)
	}
	var rows []struct {
		Kind                       string
		Mode                       driveMode
		Enabled, Crossed, Finished bool
		Before, After              struct {
			Cars                                      [33]traffic.Car
			Random                                    uint16
			LastPlayerFlags, SignalFlags, YellowDelay uint8
			RaceWon, Yellow                           bool
			Leader, Tail                              int
			TowInterval, TowWait                      uint16
		}
	}
	if err = json.Unmarshal(data, &rows); err != nil {
		t.Fatal(err)
	}
	g, err := New(Options{Mute: true, Race: true})
	if err != nil {
		t.Fatal(err)
	}
	for _, source := range rows {
		s := source.Before
		g.mode, g.raceLaps, g.demo = source.Mode, 10, false
		if source.Enabled {
			g.raceLaps = 30
		}
		g.informationPage = lapInformationPage
		g.trafficModel.State = traffic.State{Cars: s.Cars, Random: s.Random, LastPlayerFlags: s.LastPlayerFlags, SignalFlags: s.SignalFlags, YellowDelay: s.YellowDelay, RaceWon: s.RaceWon, Yellow: s.Yellow, Leader: s.Leader, Tail: s.Tail, TowInterval: s.TowInterval, TowWait: s.TowWait, Rules: int(source.Mode)}
		g.clock.Crossed = source.Crossed
		g.updateRaceStatus()
		s = source.After
		if g.trafficModel.SignalFlags != s.SignalFlags || g.trafficModel.LastPlayerFlags != s.LastPlayerFlags || g.trafficModel.RaceWon != s.RaceWon {
			t.Fatalf("%s playable finish flags differ", source.Kind)
		}
		want := lapInformationPage
		if source.Finished {
			want = standingsInformationPage
		}
		if g.informationPage != want || g.scene != driveScene || !g.session {
			t.Fatalf("%s original finish interrupted driving or selected wrong page", source.Kind)
		}
	}
}
