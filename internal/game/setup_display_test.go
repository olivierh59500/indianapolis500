package game

import (
	"bytes"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"os"
	"testing"
)

// All twenty-one setup selections were made through original function keys
// and Space. Full expected header pixels come from the original frontend.
func TestOriginalPlayedSetupDisplays(t *testing.T) {
	var cases []struct {
		OriginalPage int
		Values       setupDisplayValues
		Header       []byte
	}
	data, err := os.ReadFile("testdata/setup-ui-played.json")
	if err != nil {
		t.Fatal(err)
	}
	if err := json.Unmarshal(data, &cases); err != nil {
		t.Fatal(err)
	}
	if len(cases) != 21 {
		t.Fatalf("got %d original setup displays, want21", len(cases))
	}
	b := informationBar{}
	if err := b.load(); err != nil {
		t.Fatal(err)
	}
	a := setupArtwork{}
	if err := a.load(); err != nil {
		t.Fatal(err)
	}
	for _, c := range cases {
		b.setupInfo(&a, c.Values)
		if !bytes.Equal(b.pixels[:], c.Header) {
			count, first := 0, -1
			for i, v := range b.pixels {
				if v != c.Header[i] {
					count++
					if first < 0 {
						first = i
					}
				}
			}
			t.Errorf("original setup page%d: %dpixels differ, first%d/%d native%d source%d", c.OriginalPage, count, first%320, first/320, b.pixels[first], c.Header[first])
		}
	}
}

// Independent original painters exercise all setup choices with varied
// valid detents, dynamic readings and the original eligibility rules.
func TestOriginalSetupRasterMatrix(t *testing.T) {
	var cases []struct {
		OriginalPage int
		Kind         string
		Values       setupDisplayValues
		SHA256       string
	}
	data, err := os.ReadFile("testdata/setup-rasters.json")
	if err != nil {
		t.Fatal(err)
	}
	if err := json.Unmarshal(data, &cases); err != nil {
		t.Fatal(err)
	}
	if len(cases) != 138 {
		t.Fatalf("got %d original setup rasters, want138", len(cases))
	}
	b := informationBar{}
	if err := b.load(); err != nil {
		t.Fatal(err)
	}
	a := setupArtwork{}
	if err := a.load(); err != nil {
		t.Fatal(err)
	}
	for i, c := range cases {
		b.setupInfo(&a, c.Values)
		if got := fmt.Sprintf("%x", sha256.Sum256(b.pixels[:])); got != c.SHA256 {
			t.Errorf("original setup raster%d page%d %s: got%s want%s", i, c.OriginalPage, c.Kind, got, c.SHA256)
		}
	}
}
