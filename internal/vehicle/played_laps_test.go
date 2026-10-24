package vehicle

import (
	"encoding/json"
	"os"
	"testing"

	gameassets "github.com/olivierh59500/indianapolis500/assets"
)

// Real original Practice and qualification supply ordinary keys for a pit exit
// and sustained laps. Each Go run starts once, without later state corrections.
func TestOriginalPlayedLaps(t *testing.T) {
	for _, run := range []struct {
		name, file string
		steps      int
	}{{"practice", "played-laps.json", 1800}, {"qualification", "played-qualification.json", 4100}, {"pit entry", "played-pit-entry.json", 1446}} {
		t.Run(run.name, func(t *testing.T) {
			var fixture struct {
				Config Config
				State  State
				Driver DriverState
				Steps  []struct {
					Input             DriverIntent
					Progress, Speed   uint16
					Lateral, Steering int16
					Full              *State
				}
			}
			data, err := os.ReadFile("testdata/" + run.file)
			if err != nil {
				t.Fatal(err)
			}
			if err := json.Unmarshal(data, &fixture); err != nil {
				t.Fatal(err)
			}
			if len(fixture.Steps) != run.steps {
				t.Fatalf("got %d source driving steps, want %d", len(fixture.Steps), run.steps)
			}
			world, surface := suspensionGeometry(t)
			data, err = gameassets.Original.ReadFile("original/tire-slip-curve.bin")
			if err != nil {
				t.Fatal(err)
			}
			curve, err := NewContactCurve(data)
			if err != nil {
				t.Fatal(err)
			}
			car := Vehicle{Config: fixture.Config, State: fixture.State, World: world, Curve: &curve, Surface: surface}
			driver := Driver{State: fixture.Driver}
			for tick, step := range fixture.Steps {
				s := &car.State
				if s.Progress != step.Progress || s.Lateral != step.Lateral || s.Body.Speed != step.Speed || s.Steering != step.Steering {
					t.Fatalf("played original tick %d: progress/lateral/speed/steering %d/%d/%d/%d, want %d/%d/%d/%d", tick, s.Progress, s.Lateral, s.Body.Speed, s.Steering, step.Progress, step.Lateral, step.Speed, step.Steering)
				}
				if step.Full != nil && *s != *step.Full {
					t.Fatalf("played original tick %d: full physical state differs: got %+v want %+v", tick, *s, *step.Full)
				}
				controls := driver.Step(DriverContext{Steering: s.Steering, Damping: s.SteeringDamping, Increment: s.SteeringIncrement, Speed: s.Body.Speed, Failures: s.Engine.Failures}, step.Input)
				car.Step(controls)
			}
		})
	}
}
