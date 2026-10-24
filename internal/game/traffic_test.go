package game

import (
	"encoding/json"
	"os"
	"slices"
	"testing"

	"github.com/olivierh59500/indianapolis500/internal/traffic"
	"github.com/olivierh59500/indianapolis500/internal/vehicle"
)

// These source calls scan the complete ordered field and the linked wrecks.
// The real player callback transfers every contact through the force model.
func TestOriginalOrderedFieldContacts(t *testing.T) {
	data, err := os.ReadFile("testdata/field-contacts.json")
	if err != nil {
		t.Fatal(err)
	}
	var rows []struct {
		Kind                          string
		Movement                      int16
		Before, After                 [33]traffic.Car
		PhysicalBefore, PhysicalAfter vehicle.State
		WrecksBefore, WrecksAfter     []int
		YellowDelay                   uint8
		Random, RandomAfter           uint16
		JoltPitch, JoltRoll           int16
	}
	if err := json.Unmarshal(data, &rows); err != nil {
		t.Fatal(err)
	}
	g, err := New(Options{Mute: true, Practice: true})
	if err != nil {
		t.Fatal(err)
	}
	for _, source := range rows {
		t.Run(source.Kind, func(t *testing.T) {
			m := g.trafficModel
			m.State = traffic.State{Cars: source.Before, Random: source.Random, PlayerMovement: source.Movement, WreckCount: len(source.WrecksBefore)}
			copy(m.Wrecks[:], source.WrecksBefore)
			g.cars[0].physics = source.PhysicalBefore
			g.contactJolt = vehicle.CameraJolt{}
			m.ResolveContacts(g.applyPlayerCarContact)
			for i := range m.Cars {
				if m.Cars[i] != source.After[i] {
					t.Fatalf("car%d native%+v original%+v", i, m.Cars[i], source.After[i])
				}
			}
			if g.cars[0].physics != source.PhysicalAfter {
				t.Fatalf("native physical%+v original%+v", g.cars[0].physics, source.PhysicalAfter)
			}
			if m.Random != source.RandomAfter || g.contactJolt.Pitch != source.JoltPitch || g.contactJolt.Roll != source.JoltRoll {
				t.Fatalf("native random%x jolt%+v, original random%x jolt%d/%d", m.Random, g.contactJolt, source.RandomAfter, source.JoltPitch, source.JoltRoll)
			}
			if m.YellowDelay != source.YellowDelay || !slices.Equal(m.Wrecks[:m.WreckCount], source.WrecksAfter) {
				t.Fatalf("native wrecks%v delay%d, original wrecks%v delay%d", m.Wrecks[:m.WreckCount], m.YellowDelay, source.WrecksAfter, source.YellowDelay)
			}
		})
	}
}

func TestPlayedRaceThroughGameControlsAndTraffic(t *testing.T) {
	data, err := os.ReadFile("testdata/played-race.json")
	if err != nil {
		t.Fatal(err)
	}
	var physical []struct {
		Tick            int
		Wheel           int16
		Config          vehicle.Config
		Before, After   vehicle.State
		Throttle, Brake uint16
	}
	if err = json.Unmarshal(data, &physical); err != nil {
		t.Fatal(err)
	}
	data, err = os.ReadFile("../traffic/testdata/racing.json")
	if err != nil {
		t.Fatal(err)
	}
	var original []struct {
		Before, After         [33]traffic.Car
		Cursor                int
		Random, RandomAfter   uint16
		Ranking, RankingAfter struct {
			Leader, Tail          int
			Position              uint16
			RankAhead, RankBehind [33]int
		}
	}
	if err = json.Unmarshal(data, &original); err != nil {
		t.Fatal(err)
	}
	g, err := New(Options{Mute: true, Race: true})
	if err != nil {
		t.Fatal(err)
	}
	for _, frame := range physical {
		tick := frame.Tick
		ranking := original[tick].Ranking
		g.trafficModel.Leader, g.trafficModel.Tail, g.trafficModel.Position = ranking.Leader, ranking.Tail, ranking.Position
		g.trafficModel.RankAhead, g.trafficModel.RankBehind = ranking.RankAhead, ranking.RankBehind
		g.trafficModel.Cars = original[tick-1].After
		g.trafficModel.Random, g.trafficModel.NextDriver = original[tick].Random, original[tick].Cursor
		g.vehiclePreset.Config = frame.Config
		g.cars[0].physics = frame.Before
		g.cars[0].driver = vehicle.DriverState{Throttle: frame.Throttle, Brake: frame.Brake}
		g.syncPlayerVehicle()
		// The original race used an absolute mouse wheel position. Hold
		// that measured position through the remake's ordinary analog input,
		// countering the same recovered steering-centering force.
		steering := 0.0
		if frame.Wheel != 0 {
			desired := int(frame.Wheel)
			if desired > 0 {
				desired += int(frame.Before.SteeringDamping)
			} else {
				desired -= int(frame.Before.SteeringDamping)
			}
			steering = -float64(desired-int(frame.Before.Steering)) / float64(frame.Before.SteeringIncrement)
		}
		g.stepDrive(drivingInput{steering: steering})
		if g.cars[0].physics != frame.After {
			t.Fatalf("played race tick%d physical state differs: native%+v original%+v", tick, g.cars[0].physics, frame.After)
		}
		for i := 0; i < 33; i++ {
			if g.trafficModel.Cars[i] != original[tick].After[i] {
				t.Fatalf("played race tick%d car%d: native%+v original%+v", tick, i, g.trafficModel.Cars[i], original[tick].After[i])
			}
		}
		want := original[tick].RankingAfter
		m := g.trafficModel
		if m.Leader != want.Leader || m.Tail != want.Tail || m.Position != want.Position || m.RankAhead != want.RankAhead || m.RankBehind != want.RankBehind {
			t.Fatalf("played race tick%d standings differ from the original", tick)
		}

		if g.trafficModel.Random != original[tick].RandomAfter {
			t.Fatalf("played race tick%d random sequence differs", tick)
		}
	}
}

func TestRacingTrafficSurvivesReplayAndIdleDemonstration(t *testing.T) {
	g, err := New(Options{Mute: true, Race: true})
	if err != nil {
		t.Fatal(err)
	}
	for range 40 {
		g.stepDrive(drivingInput{throttle: true})
	}
	g.contactJolt = vehicle.CameraJolt{Pitch: 34, Roll: -70}
	g.updateInstruments()
	g.dashboard.fuelFlash, g.dashboard.oilFlash = 3, 2
	instruments := g.dashboard.instrumentRuntime
	before, cars := g.trafficModel.State, g.cars
	jolt := g.contactJolt
	g.startReplay()
	if g.contactJolt != g.replayFrames[g.replayStart].contactJolt {
		t.Fatal("replay did not use its recorded contact camera impulse")
	}
	g.updateReplay()
	g.updateInstruments()
	g.finishReplay()
	if g.trafficModel.State != before || g.cars != cars || g.contactJolt != jolt || g.dashboard.instrumentRuntime != instruments {
		t.Fatal("replay changed the live computer-driver state")
	}
	g.startDrive(true)
	for range 40 {
		g.stepDrive(drivingInput{throttle: true})
	}
	g.updateInstruments()
	g.stopDemo()
	if g.trafficModel.State != before || g.cars != cars || g.contactJolt != jolt || g.dashboard.instrumentRuntime != instruments {
		t.Fatal("idle demonstration changed the paused computer-driver state")
	}
}

// Each original contact was followed by its real player-force transfer.
// Compare all persistent physical quantities, both car records and RNG.
func TestOriginalCarContactTransfersToPlayerPhysics(t *testing.T) {
	data, err := os.ReadFile("testdata/player-contacts.json")
	if err != nil {
		t.Fatal(err)
	}
	var rows []struct {
		Kind                          string
		Gap                           int16
		Rules                         int
		Before, After                 [2]traffic.Car
		PhysicalBefore, PhysicalAfter vehicle.State
		Random, RandomAfter           uint16
		JoltPitch, JoltRoll           int16
	}
	if err = json.Unmarshal(data, &rows); err != nil {
		t.Fatal(err)
	}
	g, err := New(Options{Mute: true})
	if err != nil {
		t.Fatal(err)
	}
	for _, source := range rows {
		t.Run(source.Kind, func(t *testing.T) {
			m := g.trafficModel
			m.WreckCount, m.YellowDelay, m.Yellow = 0, 0, false
			m.Cars[0], m.Cars[6], m.Random = source.Before[0], source.Before[1], source.Random
			result := m.ResolveContact(0, 6, source.Gap)
			if !result.Hit {
				t.Fatal("original player's contact disappeared")
			}
			state := source.PhysicalBefore
			car := &m.Cars[0]
			jolt := vehicle.ApplyTrafficImpact(&state, g.world, vehicle.TrafficImpact{Forward: int16(car.Speed), LateralVelocity: car.LateralVelocity, YawRate: car.YawRate, TrackGap: result.TrackGap, LateralGap: result.LateralGap, Severity: result.Severity, TrackHeading: 0, DamageEnabled: source.Rules&2 != 0}, m.NextRandom)
			car.YawRate = 0
			if state != source.PhysicalAfter {
				t.Fatalf("native physical%+v, original%+v", state, source.PhysicalAfter)
			}
			if jolt.Pitch != source.JoltPitch || jolt.Roll != source.JoltRoll {
				t.Fatalf("native jolt%+v, original%d/%d", jolt, source.JoltPitch, source.JoltRoll)
			}
			if m.Cars[0] != source.After[0] || m.Cars[6] != source.After[1] || m.Random != source.RandomAfter {
				t.Fatal("original contact car state or random sequence differs")
			}
		})
	}
}
