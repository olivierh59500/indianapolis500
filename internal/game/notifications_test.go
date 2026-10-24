package game

import (
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"os"
	"testing"

	"github.com/olivierh59500/indianapolis500/internal/traffic"
	"github.com/olivierh59500/indianapolis500/internal/vehicle"
)

func TestOriginalNotificationControllerAndPixels(t *testing.T) {
	data, err := os.ReadFile("testdata/notifications.json")
	if err != nil {
		t.Fatal(err)
	}
	var rows []struct {
		Kind              string
		Operation         string
		Now               uint16
		Request           notification
		Flags, FlagsAfter uint8
		Before, After     notificationRuntime
		SHA256            string
	}
	if err = json.Unmarshal(data, &rows); err != nil {
		t.Fatal(err)
	}
	var bar informationBar
	if err = bar.load(); err != nil {
		t.Fatal(err)
	}
	for _, row := range rows {
		t.Run(row.Kind, func(t *testing.T) {
			n := row.Before
			flags := row.Flags
			switch row.Operation {
			case "request":
				n.request(row.Request)
			case "reset":
				n.reset()
			case "step":
				flags = n.advance(row.Now, flags)
			}
			if n != row.After || flags != row.FlagsAfter {
				t.Fatalf("notification state native%+v flags%x original%+v flags%x", n, flags, row.After, row.FlagsAfter)
			}
			clear(bar.pixels[:])
			bar.text(notificationLabels[n.Visible], 24, 8, 15)
			var area [80 * 8]byte
			for y := range 8 {
				copy(area[y*80:(y+1)*80], bar.pixels[(y+8)*320+24:(y+8)*320+104])
			}
			if got := fmt.Sprintf("%x", sha256.Sum256(area[:])); got != row.SHA256 {
				t.Fatalf("notification pixels hash%s original%s", got, row.SHA256)
			}
		})
	}
}

func TestOriginalGameplayNotificationEvents(t *testing.T) {
	read := func(name string, target any) {
		t.Helper()
		data, err := os.ReadFile(name)
		if err != nil {
			t.Fatal(err)
		}
		if err = json.Unmarshal(data, target); err != nil {
			t.Fatal(err)
		}
	}
	var cases []struct {
		Family        string
		Index         int
		Before, After notificationRuntime
		Severity      uint16
		DamageEnabled bool
	}
	read("testdata/notification-events.json", &cases)
	var physics struct {
		Engines []struct {
			Config vehicle.EngineConfig
			Input  vehicle.EngineInput
			Before vehicle.EngineState
		}
		Tires []struct {
			Config vehicle.TireConfig
			Input  vehicle.TireInput
			Before vehicle.TireState
		}
	}
	read("testdata/notification-physics.json", &physics)
	var races []struct {
		Mode             driveMode
		Enabled, Crossed bool
		Before           struct {
			Cars                                      [33]traffic.Car
			Random                                    uint16
			LastPlayerFlags, SignalFlags, YellowDelay uint8
			RaceWon, Yellow                           bool
			Leader, Tail                              int
			TowInterval, TowWait                      uint16
		}
	}
	read("../traffic/testdata/race-status.json", &races)
	g, err := New(Options{Mute: true})
	if err != nil {
		t.Fatal(err)
	}
	for _, c := range cases {
		t.Run(fmt.Sprintf("%s-%d", c.Family, c.Index), func(t *testing.T) {
			g.notifications = c.Before
			switch c.Family {
			case "engine":
				s := physics.Engines[c.Index]
				engine := vehicle.Engine{Config: s.Config, State: s.Before}
				g.reportVehicleNotifications(vehicle.Events{Engine: engine.Step(s.Input)}, false)
			case "tire":
				s := physics.Tires[c.Index]
				tire := vehicle.Tire{Config: s.Config, State: s.Before}
				events := vehicle.Events{}
				if tire.Step(s.Input).Punctured {
					events.TireFailures = 1
				}
				g.reportVehicleNotifications(events, false)
			case "contact":
				g.reportCollisionNotification(c.Severity, c.DamageEnabled)
			case "race":
				r := races[c.Index]
				s := r.Before
				g.mode, g.raceLaps, g.demo = r.Mode, 10, false
				if r.Enabled {
					g.raceLaps = 30
				}
				g.trafficModel.State = traffic.State{Cars: s.Cars, Random: s.Random, LastPlayerFlags: s.LastPlayerFlags,
					SignalFlags: s.SignalFlags, YellowDelay: s.YellowDelay, RaceWon: s.RaceWon, Yellow: s.Yellow,
					Leader: s.Leader, Tail: s.Tail, TowInterval: s.TowInterval, TowWait: s.TowWait, Rules: int(r.Mode)}
				g.clock.Crossed = r.Crossed
				g.updateRaceStatus()
			}
			if g.notifications != c.After {
				t.Fatalf("playable message requests native%+v original%+v", g.notifications, c.After)
			}
		})
	}
}

func TestOriginalStoppedFailureNotification(t *testing.T) {
	data, err := os.ReadFile("testdata/notification-stopped.json")
	if err != nil {
		t.Fatal(err)
	}
	var rows []struct {
		Kind          string
		Speed         uint16
		Failures      byte
		Pit           bool
		Before, After notificationRuntime
	}
	if err = json.Unmarshal(data, &rows); err != nil {
		t.Fatal(err)
	}
	g := &Game{}
	for _, row := range rows {
		g.notifications = row.Before
		events := vehicle.Events{}
		if row.Speed < 100 {
			events.StoppedFailures = row.Failures
		}
		g.reportVehicleNotifications(events, row.Pit)
		if g.notifications != row.After {
			t.Fatalf("%s playable stopped-failure prompt differs: native%+v original%+v", row.Kind, g.notifications, row.After)
		}
	}
}

func TestOriginalCruiseNotification(t *testing.T) {
	data, err := os.ReadFile("testdata/notification-cruise.json")
	if err != nil {
		t.Fatal(err)
	}
	var rows []struct {
		Kind              string
		Previous, Current uint16
		Before, After     notificationRuntime
	}
	if err = json.Unmarshal(data, &rows); err != nil {
		t.Fatal(err)
	}
	g := &Game{}
	for _, row := range rows {
		g.notifications, g.noticeCruise = row.Before, row.Previous != 0
		g.reportCruiseNotification(row.Current != 0)
		if g.notifications != row.After || g.noticeCruise != (row.Current != 0) {
			t.Fatalf("%s playable cruise message differs: native%+v original%+v", row.Kind, g.notifications, row.After)
		}
	}
}

// Original header animation keeps running while P freezes vehicle updates.
func TestPausedHeaderKeepsOriginalFlagAnimation(t *testing.T) {
	data, err := os.ReadFile("testdata/header-flags.json")
	if err != nil {
		t.Fatal(err)
	}
	type state struct{ Signal, Previous, Remaining, Delay uint8 }
	var rows []struct {
		Before, After state
		SHA256        string
	}
	if err = json.Unmarshal(data, &rows); err != nil {
		t.Fatal(err)
	}
	g, err := New(Options{Mute: true, Race: true})
	if err != nil {
		t.Fatal(err)
	}
	g.paused = true
	g.flags = flagRuntime{Previous: rows[3].Before.Previous, Remaining: rows[3].Before.Remaining, Delay: rows[3].Before.Delay}
	physical, clock := g.cars[0].physics, g.clock
	for tick, row := range rows[3:58] {
		g.trafficModel.SignalFlags = row.Before.Signal
		for range PresentationTicksPerSecond / SimulationTicksPerSecond {
			if err = g.Update(); err != nil {
				t.Fatal(err)
			}
		}
		if (state{g.trafficModel.SignalFlags, g.flags.Previous, g.flags.Remaining, g.flags.Delay}) != row.After {
			t.Fatalf("paused header update%d differs from the original", tick)
		}
		if g.cars[0].physics != physical || g.clock != clock || g.tick != 0 {
			t.Fatal("paused header animation advanced the vehicle or lap simulation")
		}
	}
}
