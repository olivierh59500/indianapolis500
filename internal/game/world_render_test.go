//go:build rendercheck

package game

import (
	"compress/gzip"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"os"
	"testing"
)

// The shared playable adapter must reproduce complete original banked worlds
// through its real camera, painter, palette upload and cockpit-view placement.
// Opponent cars are isolated in both source and native scenes; their painter
// and traffic integration have independent source regressions.
func TestOriginalBankedWorldsThroughGame(t *testing.T) {
	file, err := os.Open("../scene/testdata/complete-world-rasters.json.gz")
	if err != nil {
		t.Fatal(err)
	}
	defer file.Close()
	packed, err := gzip.NewReader(file)
	if err != nil {
		t.Fatal(err)
	}
	defer packed.Close()
	var cases []struct {
		Name            string
		Camera          [6]int16
		Progress        uint16
		Lateral         int16
		Mode, Phase     uint8
		Detail          int
		RelativeHeading *int16 `json:"relative_heading"`
		Pixels          []byte
	}
	if err := json.NewDecoder(packed).Decode(&cases); err != nil {
		t.Fatal(err)
	}
	g, err := New(Options{Mute: true})
	if err != nil {
		t.Fatal(err)
	}
	checked := 0
	for _, original := range cases {
		if original.RelativeHeading == nil {
			continue
		}
		if len(original.Pixels) != 320*124 {
			t.Fatal("invalid original banked world")
		}
		g.cars = [33]car{}
		g.cars[0] = car{position: float64((int(original.Progress) + trackLength - 6930) % trackLength), lateral: float64(original.Lateral) * 16, yaw: *original.RelativeHeading, active: true, player: true}
		g.detail, g.mode, g.simulationPhase = original.Detail, driveMode(original.Mode), 0
		g.presentationTick = 0
		for uint8((g.presentationTick*50/PresentationTicksPerSecond/6*12)&63) != original.Phase && g.presentationTick < 1000 {
			g.presentationTick++
		}
		if g.presentationTick == 1000 {
			t.Fatalf("unreachable original crowd phase%d", original.Phase)
		}
		camera, _ := g.originalCamera(inCarView)
		if got := [6]int16{camera.X, camera.Y, camera.Z, camera.Heading, camera.Pitch, camera.Roll}; got != original.Camera {
			t.Fatalf("%s actual game camera%v original%v", original.Name, got, original.Camera)
		}
		pixels := make([]byte, len(original.Pixels)*4)
		for i, index := range original.Pixels {
			copy(pixels[i*4:i*4+4], g.worldColors[index][:])
		}
		headerRenderRequests <- &headerRenderCheck{game: g, worldDigest: fmt.Sprintf("%x", sha256.Sum256(pixels))}
		if err := <-headerRenderResults; err != nil {
			t.Fatalf("%s: %v", original.Name, err)
		}
		checked++
	}
	if checked != 16 {
		t.Fatalf("actual game world cases%d, want16", checked)
	}
}
