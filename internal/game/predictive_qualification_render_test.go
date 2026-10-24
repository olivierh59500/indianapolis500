//go:build rendercheck

package game

import (
	"compress/gzip"
	"encoding/json"
	"fmt"
	"os"
	"testing"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/olivierh59500/indianapolis500/internal/session"
	"github.com/olivierh59500/indianapolis500/internal/traffic"
	"github.com/olivierh59500/indianapolis500/internal/vehicle"
)

// Six normally driven original qualifying pictures include all world pixels,
// cockpit graphics, setup instruments and information headers. The observer
// masks IRQs only for its bounded complete-frame/DMA measurement, restores the
// original CPU/stack context and never supplies motion or resource state.
func TestOriginalPredictiveQualifyingFrames(t *testing.T) {
	file, err := os.Open("testdata/predictive-qualification-frames.json.gz")
	if err != nil {
		t.Fatal(err)
	}
	defer file.Close()
	packed, err := gzip.NewReader(file)
	if err != nil {
		t.Fatal(err)
	}
	defer packed.Close()
	var rows []struct {
		Tick, OriginalTick  int
		Config              vehicle.Config
		Physical            vehicle.State
		Clock               session.Clock
		Traffic             traffic.State
		Camera              [6]int16
		Phase               int
		InformationPage     informationPage
		Center              uint16
		DashboardSpeed      int16
		FuelFlash, OilFlash uint8
		Flags               flagRuntime
		AfterClockFlags     byte
		Pixels              []byte
		RetainedBackdrop    []struct {
			Offset   int
			Previous byte
		}
	}
	if err = json.NewDecoder(packed).Decode(&rows); err != nil {
		t.Fatal(err)
	}
	if len(rows) != 6 {
		t.Fatalf("complete original race frames%d, want6", len(rows))
	}
	totalDifferences := 0
	for _, original := range rows {
		t.Run(fmt.Sprintf("source-tick-%d", original.OriginalTick), func(t *testing.T) {
			if len(original.Pixels) != 320*200 {
				t.Fatal("invalid complete original framebuffer")
			}
			g, err := New(Options{Mute: true, Race: true})
			if err != nil {
				t.Fatal(err)
			}
			g.demo, g.pitBoard = false, false
			g.mode = driveMode(original.Traffic.Rules)
			g.setup.wings = [2]int{10, 10}
			g.setup.gears = gearSetup(440)
			g.boost = 9
			g.trafficModel.State = original.Traffic
			g.clock, g.flags = original.Clock, original.Flags
			g.informationPage, g.informationSelection = original.InformationPage, int(original.Center)
			g.vehiclePreset.Config = original.Config
			g.cars = [33]car{}
			for i, s := range original.Traffic.Cars {
				g.cars[i] = car{active: g.trafficModel.InField(i), position: float64((trackLength - int(s.Distance)) % trackLength), progress: -float64(s.Distance), lateral: float64(s.Lateral), speed: float64(s.Speed) / speedPerMPH, yaw: s.Yaw, angularVelocity: s.YawRate, color: int(s.Primary), secondary: int(s.Secondary), number: int(s.Number & 127), player: i == 0}
			}
			g.cars[0].physical, g.cars[0].physics, g.cars[0].dashboardSpeed = true, original.Physical, original.DashboardSpeed
			g.syncPlayerVehicle()
			g.dashboard.fuelFlash, g.dashboard.oilFlash = original.FuelFlash, original.OilFlash
			for (g.presentationTick*50/PresentationTicksPerSecond/6*12)&63 != original.Phase && g.presentationTick < 1000 {
				g.presentationTick++
			}
			if g.presentationTick == 1000 {
				t.Fatalf("unreachable original crowd phase%d", original.Phase)
			}
			camera, _ := g.originalCamera(inCarView)
			if got := [6]int16{camera.X, camera.Y, camera.Z, camera.Heading, camera.Pitch, camera.Roll}; got != original.Camera {
				t.Fatalf("actual game camera%v original%v", got, original.Camera)
			}
			retained := make(map[int]byte, len(original.RetainedBackdrop))
			for _, p := range original.RetainedBackdrop {
				if p.Offset < 148*320 || p.Offset >= len(original.Pixels) || original.Pixels[p.Offset] != p.Previous {
					t.Fatal("invalid retained original backdrop observation")
				}
				retained[p.Offset] = p.Previous
			}
			differences := 0
			headerRenderRequests <- &headerRenderCheck{game: g, renderCheck: func(screen *ebiten.Image) error {
				g.drawDrive(screen)
				pixels := make([]byte, 320*200*4)
				screen.ReadPixels(pixels)
				for i, index := range original.Pixels {
					actual := [4]byte{pixels[i*4], pixels[i*4+1], pixels[i*4+2], pixels[i*4+3]}
					if actual == g.worldColors[index] {
						continue
					}
					previous, known := retained[i]
					clean := g.dashboard.panels[g.team][max(0, i-148*320)]
					if !known || previous != index || actual != g.worldColors[clean] {
						return fmt.Errorf("pixel%d,%d native%v original%v", i%320, i/320, actual, g.worldColors[index])
					}
					differences++
				}
				if differences != len(retained) {
					return fmt.Errorf("retained source backdrop differences%d, want%d", differences, len(retained))
				}
				if g.clock.Flags != original.AfterClockFlags {
					return fmt.Errorf("consumed clock flags%x original%x", g.clock.Flags, original.AfterClockFlags)
				}
				return g.err
			}}
			if err := <-headerRenderResults; err != nil {
				t.Fatal(err)
			}
			totalDifferences += differences
			if differences != 0 {
				t.Logf("%d older source backdrop pixels differ from the recovered clean artwork", differences)
			}
		})
	}
	if totalDifferences != 0 {
		t.Fatalf("known retained backdrop differences%d, want0", totalDifferences)
	}
}
