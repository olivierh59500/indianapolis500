package traffic

import (
	"encoding/json"
	"os"
	"testing"
)

// These are observed $4946-$4a9c calls. They distinguish a player's own finish
// from an opponent's, retain the last-lap flag and prioritize finish over yellow.
func TestOriginalRaceStatusTransitions(t *testing.T) {
	data, err := os.ReadFile("testdata/race-status.json")
	if err != nil {
		t.Fatal(err)
	}
	type status struct {
		Cars                                      [33]Car
		Random                                    uint16
		LastPlayerFlags, SignalFlags, YellowDelay uint8
		RaceWon, Yellow                           bool
		Leader, Tail                              int
		TowInterval, TowWait                      uint16
	}
	var rows []struct {
		Kind                            string
		Mode                            int
		Enabled, Crossed, Finished, Won bool
		Page                            int
		Before, After                   status
	}
	if err = json.Unmarshal(data, &rows); err != nil {
		t.Fatal(err)
	}
	m, _ := originalRacing(t)
	for _, source := range rows {
		t.Run(source.Kind, func(t *testing.T) {
			s := source.Before
			m.State = State{Cars: s.Cars, Random: s.Random, LastPlayerFlags: s.LastPlayerFlags, SignalFlags: s.SignalFlags, YellowDelay: s.YellowDelay, RaceWon: s.RaceWon, Yellow: s.Yellow, Leader: s.Leader, Tail: s.Tail, TowInterval: s.TowInterval, TowWait: s.TowWait, Rules: source.Mode}
			event := m.StepRaceStatus(source.Enabled, source.Crossed)
			if event.Finished != source.Finished || event.Won != source.Won {
				t.Fatalf("native event%+v, original finish%v won%v", event, source.Finished, source.Won)
			}
			s = source.After
			if m.LastPlayerFlags != s.LastPlayerFlags || m.SignalFlags != s.SignalFlags || m.RaceWon != s.RaceWon || m.Yellow != s.Yellow || m.YellowDelay != s.YellowDelay || m.Random != s.Random || m.TowInterval != s.TowInterval || m.TowWait != s.TowWait {
				t.Fatalf("native status flags%x previous%x won%v yellow%v/%d random%x, original flags%x previous%x won%v yellow%v/%d random%x", m.SignalFlags, m.LastPlayerFlags, m.RaceWon, m.Yellow, m.YellowDelay, m.Random, s.SignalFlags, s.LastPlayerFlags, s.RaceWon, s.Yellow, s.YellowDelay, s.Random)
			}
			if m.Cars != s.Cars {
				t.Fatal("race-status processing changed the original field differently")
			}
			if event.Finished && source.Page != 1 {
				t.Fatal("source finish did not select Standings")
			}
		})
	}
}
