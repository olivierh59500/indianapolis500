//go:build rendercheck

package game

import (
	"encoding/json"
	"fmt"
	"os"
	"testing"

	"github.com/hajimehoshi/ebiten/v2"
)

// These observed original calls include pending lap updates, unrelated flag
// bits and the finished-race early return in the real Lap Info callback.
func TestOriginalLapInformationFlagConsumer(t *testing.T) {
	data, err := os.ReadFile("testdata/lap-info-consumers.json")
	if err != nil {
		t.Fatal(err)
	}
	var cases []struct {
		Mode                       driveMode
		PlayerFlags, Before, After uint8
	}
	if err = json.Unmarshal(data, &cases); err != nil {
		t.Fatal(err)
	}
	g, err := New(Options{Mute: true})
	if err != nil {
		t.Fatal(err)
	}
	for _, original := range cases {
		g.mode = original.Mode
		g.trafficModel.Cars[0].Flags = original.PlayerFlags
		g.clock.Flags = original.Before
		g.informationBar.valid = false
		headerRenderRequests <- &headerRenderCheck{game: g, renderCheck: func(screen *ebiten.Image) error {
			g.drawLapInformation(screen)
			if g.clock.Flags != original.After {
				return fmt.Errorf("mode%d player flags%x lap flags%x: native%x original%x", original.Mode, original.PlayerFlags, original.Before, g.clock.Flags, original.After)
			}
			return nil
		}}
		if err := <-headerRenderResults; err != nil {
			t.Error(err)
		}
	}
}
