package traffic

import (
	"encoding/json"
	"os"
	"testing"
)

// These timestamps and gaps come from the same independent source field
// captures as the physical car records, including forward and reverse epochs.
func TestOriginalFinishLineTiming(t *testing.T) {
	var frames []playerEpochFrame
	for _, name := range []string{"player-epochs.json", "finish-field.json"} {
		data, err := os.ReadFile("testdata/" + name)
		if err != nil {
			t.Fatal(err)
		}
		var rows []playerEpochFrame
		if err = json.Unmarshal(data, &rows); err != nil {
			t.Fatal(err)
		}
		frames = append(frames, rows...)
	}
	data, err := os.ReadFile("testdata/crossing-timing.json")
	if err != nil {
		t.Fatal(err)
	}
	var times []struct {
		Kind          string
		Before, After Timing
	}
	if err = json.Unmarshal(data, &times); err != nil {
		t.Fatal(err)
	}
	if len(times) != len(frames) {
		t.Fatal("source timing and physical observations have different lengths")
	}
	m, _ := originalRacing(t)
	for i, frame := range frames {
		t.Run(frame.Kind, func(t *testing.T) {
			if frame.Kind != times[i].Kind {
				t.Fatal("source timing observation belongs to another physical frame")
			}
			s := frame.Before
			m.State = State{Cars: s.Cars, Random: s.Random, NextDriver: s.Cursor, Tail: s.Tail,
				Rules: frame.Mode, Leader: s.Leader, Position: uint16(s.Position),
				RankAhead: s.RankAhead, RankBehind: s.RankBehind, Timing: times[i].Before}
			m.StepRacingField(frame.Pose, nil)
			if m.Timing != times[i].After {
				t.Fatalf("finish-line timing differs: native%+v original%+v", m.Timing, times[i].After)
			}
		})
	}
}
