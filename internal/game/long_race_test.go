package game

import (
	"compress/gzip"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"os"
	"testing"

	"github.com/olivierh59500/indianapolis500/internal/session"
	"github.com/olivierh59500/indianapolis500/internal/sound"
	"github.com/olivierh59500/indianapolis500/internal/traffic"
	"github.com/olivierh59500/indianapolis500/internal/vehicle"
)

// Original keys, all 33 car states, full physical checkpoints and timing/ranking
// are replayed continuously from one observed start without later corrections.
func TestOriginalContinuousRaceThroughGame(t *testing.T) {
	var fixture struct {
		Start   int
		Initial struct {
			Config   vehicle.Config
			Physical vehicle.State
			Driver   vehicle.DriverState
			Clock    session.Clock
			Traffic  traffic.State
			Sound    sound.Controller
		}
		Frames []struct {
			Tick              int
			Progress, Speed   uint16
			Lateral, Steering int16
			CarsSHA256        string
			Input             struct {
				Steering          float64
				Accelerate, Brake bool
			}
		}
		Checkpoints []struct {
			Tick     int
			Physical vehicle.State
			Clock    session.Clock
			Ranking  struct {
				Random                   uint16
				NextDriver, Tail, Leader int
				Position                 uint16
				RankAhead, RankBehind    [33]int
			}
		}
	}
	file, err := os.Open("testdata/played-long-race.json.gz")
	if err != nil {
		t.Fatal(err)
	}
	defer file.Close()
	compressed, err := gzip.NewReader(file)
	if err != nil {
		t.Fatal(err)
	}
	defer compressed.Close()
	if err := json.NewDecoder(compressed).Decode(&fixture); err != nil {
		t.Fatal(err)
	}
	if fixture.Start != 300 || len(fixture.Frames) != 6404 {
		t.Fatalf("unexpected original race start/count %d/%d", fixture.Start, len(fixture.Frames))
	}
	if len(fixture.Checkpoints) != 23 {
		t.Fatalf("got %d complete source checkpoints, want23", len(fixture.Checkpoints))
	}
	g, err := New(Options{Mute: true, Race: true})
	if err != nil {
		t.Fatal(err)
	}
	g.vehiclePreset.Config = fixture.Initial.Config
	g.trafficModel.State = fixture.Initial.Traffic
	g.clock, g.soundController = fixture.Initial.Clock, fixture.Initial.Sound
	g.cars = [33]car{}
	for i, s := range g.trafficModel.Cars {
		g.cars[i] = car{active: g.trafficModel.InField(i), position: float64((39600 - int(s.Distance)) % 39600), progress: -float64(s.Distance), lateral: float64(s.Lateral), speed: float64(s.Speed) / speedPerMPH, color: int(s.Primary), secondary: int(s.Secondary), number: int(s.Number & 127), player: i == 0}
	}
	g.cars[0].physical, g.cars[0].physics, g.cars[0].driver = true, fixture.Initial.Physical, fixture.Initial.Driver
	g.syncPlayerVehicle()
	g.rememberSoundPositions()
	for i, frame := range fixture.Frames {
		p := &g.cars[0].physics
		if p.Progress != frame.Progress || p.Lateral != frame.Lateral || p.Body.Speed != frame.Speed || p.Steering != frame.Steering {
			t.Fatalf("source tick%d physical native%d/%d/%d/%d source%d/%d/%d/%d", frame.Tick, p.Progress, p.Lateral, p.Body.Speed, p.Steering, frame.Progress, frame.Lateral, frame.Speed, frame.Steering)
		}
		cars, err := json.Marshal(g.trafficModel.Cars)
		if err != nil {
			t.Fatal(err)
		}
		if got := fmt.Sprintf("%x", sha256.Sum256(cars)); got != frame.CarsSHA256 {
			t.Fatalf("source tick%d complete car field digest %s, want%s", frame.Tick, got, frame.CarsSHA256)
		}
		for _, c := range fixture.Checkpoints {
			if c.Tick == frame.Tick {
				if g.cars[0].physics != c.Physical {
					t.Fatalf("source tick%d complete physical state differs: native%+v source%+v", frame.Tick, g.cars[0].physics, c.Physical)
				}
				actual, want := g.clock, c.Clock
				// Original rendering consumes these notice/signal flags between ticks;
				// their transition and compositing tests are independent.
				actual.Flags, actual.SignalFlags = want.Flags, want.SignalFlags
				if actual != want {
					t.Fatalf("source tick%d clock differs native%+v source%+v", frame.Tick, actual, want)
				}
				m, r := g.trafficModel, c.Ranking
				if m.Random != r.Random || m.NextDriver != r.NextDriver || m.Tail != r.Tail || m.Leader != r.Leader || m.Position != r.Position || m.RankAhead != r.RankAhead || m.RankBehind != r.RankBehind {
					t.Fatalf("source tick%d complete ranking/random differs", frame.Tick)
				}
			}
		}
		if i+1 == len(fixture.Frames) {
			break
		}
		g.tick++
		g.presentationTick += 4
		g.stepDrive(drivingInput{steering: frame.Input.Steering, throttle: frame.Input.Accelerate, brake: frame.Input.Brake})
		for n := 0; n < 4; n++ {
			g.advanceSound()
		}
	}
	if g.trafficModel.Cars[0].Flags != 255 || g.trafficModel.Position != 31 || g.trafficModel.Cars[0].RemainingLaps != 2 {
		t.Fatalf("original complete race finish differs: position%d remaining%d flags%x", g.trafficModel.Position, g.trafficModel.Cars[0].RemainingLaps, g.trafficModel.Cars[0].Flags)
	}
	if g.informationPage != standingsInformationPage {
		t.Fatal("original player finish did not open Standings through the actual game")
	}
}
