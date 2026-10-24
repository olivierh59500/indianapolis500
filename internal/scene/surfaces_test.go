package scene

import (
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"os"
	"testing"
)

// These full indexed frame hashes come from independent original crowd and
// checker drawing calls. They include partial words, single rows, long slopes,
// all three bitmap banks, four animation phases and untouched background pixels.
func TestOriginalCrowdAndTransparentCheckerSurfaces(t *testing.T) {
	data, err := os.ReadFile("testdata/surfaces.json")
	if err != nil {
		t.Fatal(err)
	}
	var cases []struct {
		Fill                string
		Color, Phase, Shape int
		Spans               [][3]int
		SHA256              string
	}
	if err = json.Unmarshal(data, &cases); err != nil {
		t.Fatal(err)
	}
	r := recoveredWorld(t).NewRenderer()
	for _, original := range cases {
		clear(r.Pixels)
		r.fill = crowdFill
		if original.Fill == "checker" {
			r.fill = checkerFill
		}
		r.CrowdPhase = uint8(original.Phase)
		for _, span := range original.Spans {
			r.surfaceSpan(span[1], span[2], span[0], uint8(original.Color))
		}
		if digest := fmt.Sprintf("%x", sha256.Sum256(r.Pixels)); digest != original.SHA256 {
			t.Errorf("%s color%d phase%d shape%d native%s, original%s", original.Fill, original.Color, original.Phase, original.Shape, digest, original.SHA256)
		}
	}
}
