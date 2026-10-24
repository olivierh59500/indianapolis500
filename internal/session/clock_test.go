package session

import (
	"encoding/json"
	"os"
	"testing"
)

// The observed car completed an ordinary four-lap qualification attempt. The
// clock advances continuously from its first checkpoint, with no time resets.
func TestOriginalPlayedQualification(t *testing.T) {
	var fixture struct {
		Initial Clock
		Steps   []struct {
			Progress uint16
			Clock    *Clock
			Rank     int
		}
	}
	data, err := os.ReadFile("testdata/played-qualification.json")
	if err != nil {
		t.Fatal(err)
	}
	if err := json.Unmarshal(data, &fixture); err != nil {
		t.Fatal(err)
	}
	if len(fixture.Steps) != 4100 {
		t.Fatalf("got %d original qualification steps, want 4100", len(fixture.Steps))
	}
	clock := fixture.Initial
	qualified := 0
	for tick, step := range fixture.Steps {
		if step.Clock != nil {
			actual, want := clock, *step.Clock
			// Rendering consumes the source's notice/signal bits between physical
			// ticks. Their isolated transitions have separate source fixtures.
			actual.Flags, actual.SignalFlags = want.Flags, want.SignalFlags
			if actual != want {
				t.Fatalf("played qualification tick %d: clock %+v, want %+v", tick, actual, want)
			}
		}
		if tick+1 == len(fixture.Steps) {
			break
		}
		event := clock.Step(1, step.Progress, fixture.Steps[tick+1].Progress)
		if event.Qualified {
			qualified++
			next := fixture.Steps[tick+1]
			if next.Clock == nil || event.Position != next.Rank {
				t.Fatalf("played qualification rank %d differs from source %d", event.Position, next.Rank)
			}
		}
	}
	if qualified != 1 {
		t.Fatalf("completed %d original qualification attempts, want 1", qualified)
	}
}

// Independent source calls cover the running decimal display, its cap,
// forward/reverse crossings, warm-up exclusion and a full qualification cycle.
func TestOriginalClockAndQualificationTransitions(t *testing.T) {
	data, err := os.ReadFile("testdata/clocks.json")
	if err != nil {
		t.Fatal(err)
	}
	var rows []struct {
		Kind                       string
		Mode                       int
		PreviousProgress, Progress uint16
		Before, After              Clock
		RankBefore, RankAfter      int
	}
	if err := json.Unmarshal(data, &rows); err != nil {
		t.Fatal(err)
	}
	for _, source := range rows {
		t.Run(source.Kind, func(t *testing.T) {
			clock := source.Before
			event := clock.Step(source.Mode, source.PreviousProgress, source.Progress)
			if clock != source.After {
				t.Fatalf("native clock%+v original%+v", clock, source.After)
			}
			if event.Qualified && event.Position != source.RankAfter || !event.Qualified && source.RankAfter != source.RankBefore {
				t.Fatalf("native result%+v original rank%d ->%d", event, source.RankBefore, source.RankAfter)
			}
		})
	}
}

func TestOriginalEventClockInitialization(t *testing.T) {
	data, err := os.ReadFile("testdata/initialization.json")
	if err != nil {
		t.Fatal(err)
	}
	var rows []struct {
		Mode          int
		Distance      uint16
		Before, After Clock
	}
	if err := json.Unmarshal(data, &rows); err != nil {
		t.Fatal(err)
	}
	for _, source := range rows {
		if got := Initialize(source.Before, source.Mode, source.Distance); got != source.After {
			t.Fatalf("mode%d native clock%+v original%+v", source.Mode, got, source.After)
		}
	}
}

func TestOriginalPitClockReset(t *testing.T) {
	data, err := os.ReadFile("testdata/pit.json")
	if err != nil {
		t.Fatal(err)
	}
	var rows []struct {
		Mode          int
		Progress      uint16
		Lateral       int16
		Speed         uint16
		InPit         bool
		Before, After Clock
	}
	if err := json.Unmarshal(data, &rows); err != nil {
		t.Fatal(err)
	}
	for _, source := range rows {
		clock := source.Before
		inPit := clock.AtPit(source.Mode, source.Progress, source.Lateral, source.Speed)
		if clock != source.After || inPit != source.InPit {
			t.Fatalf("mode%d at%d/%d speed%d native clock%+v original%+v pit%v/%v", source.Mode, source.Progress, source.Lateral, source.Speed, clock, source.After, inPit, source.InPit)
		}
	}
}
