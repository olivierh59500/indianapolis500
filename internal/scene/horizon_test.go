package scene

import (
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"os"
	"testing"
)

// These complete indexed frames were drawn by the original horizon routine
// at supported pitch/roll bins, including both slopes and exact flat ties.
func TestOriginalTiltedHorizonRasters(t *testing.T) {
	data, err := os.ReadFile("testdata/horizon-rasters.json")
	if err != nil {
		t.Fatal(err)
	}
	var cases []struct {
		Pitch, Roll int
		SHA256      string
	}
	if err := json.Unmarshal(data, &cases); err != nil {
		t.Fatal(err)
	}
	if len(cases) == 0 {
		t.Fatal("missing independently observed original horizons")
	}
	renderer := recoveredWorld(t).NewRenderer()
	for _, original := range cases {
		renderer.background(Camera{Pitch: int16(original.Pitch * 64), Roll: int16(original.Roll * 64)})
		if got := fmt.Sprintf("%x", sha256.Sum256(renderer.Pixels)); got != original.SHA256 {
			t.Errorf("original pitch%d roll%d: native%s original%s", original.Pitch, original.Roll, got, original.SHA256)
		}
	}
}
