package game

import (
	"compress/gzip"
	"encoding/json"
	"os"
	"testing"

	"github.com/olivierh59500/indianapolis500/internal/pilot"
	"github.com/olivierh59500/indianapolis500/internal/session"
	"github.com/olivierh59500/indianapolis500/internal/vehicle"
)

// The source driver completes four timed laps with ordinary setup and keys.
// Import only its initial state, then compare every shared-pilot input, actual
// original driver output, physical checkpoint and mechanical clock value.
func TestOriginalPredictiveQualificationThroughGame(t *testing.T) {
	var fixture struct {
		Config vehicle.Config
		State  vehicle.State
		Driver vehicle.DriverState
		Clock  session.Clock
		Steps  []struct {
			Tick              int
			Progress, Speed   uint16
			Lateral, Steering int16
			Input             vehicle.DriverIntent
			Applied           vehicle.Controls
			FullAtDriver      *vehicle.State
			Clock             *session.Clock
		}
		Tail struct {
			Full  vehicle.State
			Clock session.Clock
			Rank  int
		}
	}
	f, err := os.Open("testdata/predictive-qualification.json.gz")
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	z, err := gzip.NewReader(f)
	if err != nil {
		t.Fatal(err)
	}
	defer z.Close()
	if err := json.NewDecoder(z).Decode(&fixture); err != nil {
		t.Fatal(err)
	}
	if len(fixture.Steps) != 2683 {
		t.Fatal("incomplete original predictive qualification")
	}
	g, err := New(Options{Mute: true, SavePath: t.TempDir() + "/state.json"})
	if err != nil {
		t.Fatal(err)
	}
	g.mode = qualifyMode
	g.startDrive(false)
	g.vehiclePreset.Config = fixture.Config
	g.setup.wings = [2]int{10, 10}
	g.setup.gears = gearSetup(440)
	g.boost = 9
	if g.physicalConfig() != fixture.Config {
		t.Fatal("observed original setup does not match its ordinary menu values")
	}
	g.cars[0].physical, g.cars[0].physics, g.cars[0].driver = true, fixture.State, fixture.Driver
	g.clock = fixture.Clock
	g.trafficModel.SignalFlags = fixture.Clock.SignalFlags
	g.syncPlayerVehicle()
	g.rememberSoundPositions()
	driver := pilot.New(g.world, &g.vehicleCurve, &g.vehicleSurface)
	full, clocks := 0, 0
	checkClock := func(actual, want session.Clock) {
		t.Helper()
		// The original renderer consumes these two notice bytes between physical
		// updates. Their transitions and drawing have separate original fixtures.
		actual.Flags, actual.SignalFlags = want.Flags, want.SignalFlags
		if actual != want {
			t.Fatalf("mechanical clock differs: native%+v source%+v", actual, want)
		}
	}
	for _, step := range fixture.Steps {
		s := g.cars[0].physics
		if s.Progress != step.Progress || s.Lateral != step.Lateral || s.Body.Speed != step.Speed || s.Steering != step.Steering {
			t.Fatalf("source tick%d basic physics differs", step.Tick)
		}
		input := driver.Step(pilot.Frame{State: s, Config: g.physicalConfig(), Driver: g.cars[0].driver, Tick: step.Tick, DamageEnabled: true})
		if input.Steering != step.Input.Steering || input.Accelerate != step.Input.Accelerate || input.Brake != step.Input.Brake || input.Reverse != step.Input.Reverse {
			t.Fatalf("source tick%d shared driver intent differs", step.Tick)
		}
		clone := vehicle.Driver{State: g.cars[0].driver}
		control := clone.Step(vehicle.DriverContext{Steering: s.Steering, Damping: s.SteeringDamping, Increment: s.SteeringIncrement, Speed: s.Body.Speed, Failures: s.Engine.Failures}, input)
		if control != step.Applied {
			t.Fatalf("source tick%d original applied control differs", step.Tick)
		}
		if step.FullAtDriver != nil {
			afterDriver := s
			afterDriver.Steering = control.Steering
			if afterDriver != *step.FullAtDriver {
				t.Fatalf("source tick%d full physical driver boundary differs", step.Tick)
			}
			full++
		}
		if step.Clock != nil {
			checkClock(g.clock, *step.Clock)
			clocks++
		}
		g.tick++
		g.stepDrive(drivingInput{steering: input.Steering, throttle: input.Accelerate, brake: input.Brake, reverse: input.Reverse})
		for range 4 {
			g.advanceSound()
		}
	}
	if g.cars[0].physics != fixture.Tail.Full {
		t.Fatal("complete final original physical state differs")
	}
	full++
	checkClock(g.clock, fixture.Tail.Clock)
	clocks++
	if full != 15 || clocks != 15 || g.qualifyingPosition != fixture.Tail.Rank || g.qualifyingPosition != 22 || g.qualifyingSpeed != 20694 || g.cars[0].physics.Engine.Failures != 0 {
		t.Fatalf("incomplete original qualification result or checkpoints: states%d clocks%d rank%d speed%d", full, clocks, g.qualifyingPosition, g.qualifyingSpeed)
	}
}
