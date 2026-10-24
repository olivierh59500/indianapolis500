package game

import (
	"encoding/json"
	"os"
	"testing"

	"github.com/olivierh59500/indianapolis500/internal/session"
)

func TestOriginalClockThroughGameSession(t *testing.T) {
	data, err := os.ReadFile("../session/testdata/clocks.json")
	if err != nil {
		t.Fatal(err)
	}
	var rows []struct {
		Kind                       string
		Mode                       driveMode
		PreviousProgress, Progress uint16
		Before, After              session.Clock
		RankBefore, RankAfter      int
	}
	if err := json.Unmarshal(data, &rows); err != nil {
		t.Fatal(err)
	}
	g, err := New(Options{Mute: true})
	if err != nil {
		t.Fatal(err)
	}
	for _, source := range rows {
		g.mode, g.clock, g.qualifyingPosition = source.Mode, source.Before, source.RankBefore
		g.trafficModel.SignalFlags = source.Before.SignalFlags
		// These source checkpoints end before the separate pit-reset block.
		// Keep its ordinary moving-road predicate false in this adapter test.
		g.cars[0].physics.Lateral, g.cars[0].physics.Body.Speed = 500, 10000
		g.advanceClock(source.PreviousProgress, source.Progress)
		if g.clock != source.After || g.qualifyingPosition != source.RankAfter || g.trafficModel.SignalFlags != source.After.SignalFlags {
			t.Fatalf("%s playable clock or qualification result differs from original", source.Kind)
		}
	}
}

func TestQualifyingCompletesThroughOrdinaryControls(t *testing.T) {
	g, err := New(Options{Mute: true})
	if err != nil {
		t.Fatal(err)
	}
	g.mode = qualifyMode
	g.startDrive(false)
	var completed bool
	for tick := 0; tick < 6000; tick++ {
		before := g.clock.Laps
		g.stepDrive(g.automaticPlayerControls())
		if before == 4 && g.clock.Laps == 1 && g.clock.QualifyingSpeed != 0 {
			completed = true
			break
		}
	}
	if !completed || g.scene != driveScene || g.clock.QualifyingTime != 0 || g.cars[0].physics.Engine.Failures != 0 {
		t.Fatalf("ordinary qualifying did not complete: scene%d clock%+v failures%x", g.scene, g.clock, g.cars[0].physics.Engine.Failures)
	}
	if g.qualifyingPosition != session.QualifyingPosition(g.clock.QualifyingSpeed) {
		t.Fatal("ordinary qualifying did not publish its measured speed to the race grid")
	}
	for range 30 {
		g.stepDrive(drivingInput{throttle: true})
	}
	if g.scene != driveScene {
		t.Fatal("the original continuous qualifying session returned to a menu")
	}
}
