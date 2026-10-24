package game

import (
	"bytes"
	"crypto/sha256"
	"encoding/binary"
	"encoding/json"
	"fmt"
	"os"
	"testing"

	"github.com/olivierh59500/indianapolis500/internal/vehicle"
)

// Ordinary accelerator input must reach the same complete physical states as
// the original Practice session, including repeated pit barrier impacts.
func TestPlayedVehicleSequenceThroughGameControls(t *testing.T) {
	data, err := os.ReadFile("../vehicle/testdata/vehicle.json")
	if err != nil {
		t.Fatal(err)
	}
	var original struct {
		Config vehicle.Config
		State  vehicle.State
		Steps  []struct{ Digest string }
	}
	if err := json.Unmarshal(data, &original); err != nil {
		t.Fatal(err)
	}
	g, err := New(Options{Mute: true, Practice: true})
	if err != nil {
		t.Fatal(err)
	}
	// The original continuous input fixture begins from this observed
	// running Practice state, independently of a fresh event initializer.
	g.vehiclePreset.Config, g.cars[0].physics = original.Config, original.State
	g.syncPlayerVehicle()
	if SimulationTicksPerSecond != 15 {
		t.Fatal("the original advanced thirty physical ticks per hundred PAL vertical blanks")
	}
	for tick, reference := range original.Steps {
		g.stepDrive(drivingInput{throttle: true})
		var state bytes.Buffer
		if err := binary.Write(&state, binary.BigEndian, g.cars[0].physics); err != nil {
			t.Fatal(err)
		}
		if got := fmt.Sprintf("%x", sha256.Sum256(state.Bytes())); got != reference.Digest {
			t.Fatalf("game control tick%d: state hash%s, original%s", tick, got, reference.Digest)
		}
		camera, _ := g.originalCamera(inCarView)
		body := g.cars[0].physics.Body
		if camera.X != int16(body.PositionX>>16) || camera.Y != int16(body.PositionY>>16) || camera.Heading != body.Heading {
			t.Fatalf("camera departed from physical state at tick%d: %+v", tick, camera)
		}
	}
	live := g.cars
	g.startReplay()
	g.finishReplay()
	if g.cars != live {
		t.Fatal("replay changed the live physical or driver state")
	}
	var encoded bytes.Buffer
	if err := json.NewEncoder(&encoded).Encode(carToPersistent(g.cars[0])); err != nil {
		t.Fatal(err)
	}
	var stored persistentCar
	if err := json.NewDecoder(&encoded).Decode(&stored); err != nil {
		t.Fatal(err)
	}
	if restored := persistentToCar(stored); restored != g.cars[0] {
		t.Fatal("saved replay lost complete physical or driver state")
	}
}

func TestReverseAndCruiseUseIndependentControls(t *testing.T) {
	g, err := New(Options{Mute: true, Practice: true})
	if err != nil {
		t.Fatal(err)
	}
	for range 30 {
		g.stepDrive(drivingInput{reverse: true})
	}
	if g.cars[0].speed >= 0 || g.cars[0].driver.Brake != 0 {
		t.Fatal("reverse applied the brake instead of driving backwards")
	}
	g.startDrive(false)
	g.placePlayerVehicle(1000, 7000, 80)
	g.syncPlayerVehicle()
	g.stepDrive(drivingInput{cruiseIncrease: true})
	if g.cars[0].driver.CruiseTarget == 0 {
		t.Fatal("cruise did not capture the physical speed")
	}
	g.stepDrive(drivingInput{brake: true})
	if g.cars[0].driver.CruiseTarget != 0 {
		t.Fatal("brake did not cancel cruise")
	}
	g.cars[0].physics.Engine.Failures |= vehicle.FuelEmpty
	g.cars[0].driver = vehicle.DriverState{}
	g.stepDrive(drivingInput{reverse: true})
	if g.cars[0].driver.Throttle != 0 {
		t.Fatal("empty fuel created reverse throttle")
	}
}

func TestQualifyingWallImpactCanDamageThePhysicalTires(t *testing.T) {
	for _, mode := range []driveMode{practiceMode, qualifyMode} {
		g, err := New(Options{Mute: true, Practice: true})
		if err != nil {
			t.Fatal(err)
		}
		g.mode = mode
		// The independently verified pit barrier has an endpoint at 9180/6722.
		// Approach it through the complete physical integrator and game controls.
		g.placePlayerVehicle(9538.5-6930, 53*16, 200)
		g.syncPlayerVehicle()
		for range 20 {
			g.stepDrive(drivingInput{throttle: true})
		}
		punctured := g.cars[0].physics.Engine.Failures&15 != 0
		if punctured != (mode == qualifyMode) {
			t.Fatalf("mode%d physical impact tire flags%02x", mode, g.cars[0].physics.Engine.Failures)
		}
	}
}

func TestOriginalModeInitializationThroughGame(t *testing.T) {
	data, err := os.ReadFile("../vehicle/testdata/initialization.json")
	if err != nil {
		t.Fatal(err)
	}
	var cases []struct {
		Index, Mode, Laps, FuelSetting int
		Before, After                  vehicle.State
		Config                         vehicle.Config
	}
	if err = json.Unmarshal(data, &cases); err != nil {
		t.Fatal(err)
	}
	g, err := New(Options{Mute: true})
	if err != nil {
		t.Fatal(err)
	}
	for _, source := range cases {
		g.cars[0].physical, g.cars[0].physics = true, source.Before
		g.vehiclePreset.Config = source.Config
		g.mode, g.raceLaps = driveMode(source.Mode), source.Laps
		g.setup.fuel = source.FuelSetting / 32
		g.trafficModel.Random = 0x2468
		g.startDrive(false)
		if g.cars[0].physics != source.After {
			t.Fatalf("actual game initializer%d mode%d native%+v, original%+v", source.Index, source.Mode, g.cars[0].physics, source.After)
		}
		if g.mode == qualifyMode && (g.cars[0].position != 33600 || g.cars[0].lateral != 8320) {
			t.Fatal("actual qualifying game did not use its original flying start")
		}
	}
}
