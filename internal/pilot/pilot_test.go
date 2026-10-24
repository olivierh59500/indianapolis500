package pilot

import (
	"encoding/json"
	"os"
	"testing"

	gameassets "github.com/olivierh59500/indianapolis500/assets"
	worldscene "github.com/olivierh59500/indianapolis500/internal/scene"
	"github.com/olivierh59500/indianapolis500/internal/vehicle"
)

func testDriver(t *testing.T) (*Pilot, Frame) {
	t.Helper()
	read := func(name string) []byte {
		data, err := gameassets.Original.ReadFile("original/" + name)
		if err != nil {
			t.Fatal(err)
		}
		return data
	}
	world, err := worldscene.New(read("world-vertices.bin"), read("world-data.bin"), read("world-trigonometry.bin"), read("world-horizon.bin"), read("world-lod.bin"))
	if err != nil {
		t.Fatal(err)
	}
	for _, err := range []error{world.LoadTrack(read("track-cosine.bin"), read("track-bank.bin")), world.LoadCameras(read("camera-arctangent.bin"), read("camera-bank-gradient.bin"), read("camera-trackside.bin")), world.LoadTrackLimits(read("track-wall-inner.bin"), read("track-wall-outer.bin")), world.LoadBarriers(read("track-barriers.bin"), read("collision-damage.bin")), world.LoadCarContactDamage(read("car-contact-damage.bin"))} {
		if err != nil {
			t.Fatal(err)
		}
	}
	curve, err := vehicle.NewContactCurve(read("tire-slip-curve.bin"))
	if err != nil {
		t.Fatal(err)
	}
	surface, err := vehicle.NewTrackSurface(read("track-surface-wave.bin"))
	if err != nil {
		t.Fatal(err)
	}
	raw, err := os.ReadFile("../vehicle/testdata/played-qualification.json")
	if err != nil {
		t.Fatal(err)
	}
	var frame Frame
	if err := json.Unmarshal(raw, &frame); err != nil {
		t.Fatal(err)
	}
	frame.DamageEnabled = true
	return New(world, &curve, &surface), frame
}

// The initial setup and physical state were independently observed in the
// original. This native strategy test uses ordinary controls and live model
// steps, without claiming that its chosen inputs are original oracle values.
func TestDriverKeepsOriginalObservationsImmutable(t *testing.T) {
	p, frame := testDriver(t)
	model := vehicle.Vehicle{Config: frame.Config, State: frame.State, World: p.world, Curve: p.curve, Surface: p.surface}
	driver := vehicle.Driver{State: frame.Driver}
	progress := 0
	previous := frame.State.Progress
	initialFuel := frame.State.Engine.Fuel
	for tick := 0; tick < 4000; tick++ {
		frame.State, frame.Driver, frame.Tick = model.State, driver.State, tick
		before := frame
		input := p.Step(frame)
		if frame != before {
			t.Fatal("forecast changed an observed car, setup, driver or traffic state")
		}
		if input.DamageEnabled != frame.DamageEnabled {
			t.Fatal("driver changed the observed event damage rule")
		}
		if input.Steering != 0 && input.Steering != -1 && input.Steering != 1 || input.Accelerate && input.Brake {
			t.Fatalf("not an ordinary key intent:%+v", input)
		}
		control := driver.Step(vehicle.DriverContext{Steering: model.State.Steering, Damping: model.State.SteeringDamping, Increment: model.State.SteeringIncrement, Speed: model.State.Body.Speed, Failures: model.State.Engine.Failures}, input)
		event := model.Step(control)
		if event.Impact != 0 || model.State.Engine.Failures != 0 {
			t.Fatalf("driver hit a wall or failed at%d", tick)
		}
		if model.State.Engine.Fuel > initialFuel {
			t.Fatal("fuel increased during ordinary driving")
		}
		initialFuel = model.State.Engine.Fuel
		step := int(model.State.Progress) - int(previous)
		if step > trackLength/2 {
			step -= trackLength
		}
		if step < -trackLength/2 {
			step += trackLength
		}
		progress += step
		previous = model.State.Progress
		if progress >= 4*trackLength {
			return
		}
	}
	t.Fatal("driver did not complete four uninterrupted laps")
}
