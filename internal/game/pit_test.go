package game

import (
	"bytes"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"os"
	"testing"

	gameassets "github.com/olivierh59500/indianapolis500/assets"
	"github.com/olivierh59500/indianapolis500/internal/vehicle"
)

// These five complete displays were selected with ordinary original Space
// presses. Expected pixels come from the original frontend, not this painter.
func TestOriginalPlayedPitDisplays(t *testing.T) {
	var cases []struct {
		Selection     int
		Header, Board []byte
	}
	data, err := os.ReadFile("testdata/pit-ui-played.json")
	if err != nil {
		t.Fatal(err)
	}
	if err := json.Unmarshal(data, &cases); err != nil {
		t.Fatal(err)
	}
	art, err := gameassets.Original.ReadFile("original/pit-art.bin")
	if err != nil {
		t.Fatal(err)
	}
	b := informationBar{}
	if err := b.load(); err != nil {
		t.Fatal(err)
	}
	p := pitDisplay{art: art[4:]}
	for _, c := range cases {
		b.pit(practiceMode, p.art[184*96:])
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
			t.Errorf("original pit selection%d header: %d pixels differ, first%d/%d native%d source%d", c.Selection, count, first%320, first/320, b.pixels[first], c.Header[first])
		}
	}
}

// Original isolated painters cover every mode/selection with full, partial
// and empty tire/fuel conditions. The displayed cockpit is compared separately.
func TestOriginalPitRasters(t *testing.T) {
	var cases []struct {
		Mode                      driveMode
		Selection                 int
		Life                      [4]uint16
		Fuel                      uint16
		HeaderSHA256, BoardSHA256 string
	}
	data, err := os.ReadFile("testdata/pit-rasters.json")
	if err != nil {
		t.Fatal(err)
	}
	if err := json.Unmarshal(data, &cases); err != nil {
		t.Fatal(err)
	}
	if len(cases) != 45 {
		t.Fatalf("got %d original pit rasters, want45", len(cases))
	}
	art, err := gameassets.Original.ReadFile("original/pit-art.bin")
	if err != nil {
		t.Fatal(err)
	}
	p := pitDisplay{art: art[4:]}
	b := informationBar{}
	if err := b.load(); err != nil {
		t.Fatal(err)
	}
	for i, c := range cases {
		state := vehicle.State{}
		state.Engine.Fuel = uint32(c.Fuel) << 16
		for wheel, life := range c.Life {
			state.Tires[wheel].Life = uint32(life) << 16
		}
		age, fuel := pitCondition(&state)
		p.paint(b.font, c.Selection, age, fuel)
		b.pit(c.Mode, p.art[184*96:])
		if got := fmt.Sprintf("%x", sha256.Sum256(b.pixels[:])); got != c.HeaderSHA256 {
			t.Errorf("original pit raster%d header %s, want%s", i, got, c.HeaderSHA256)
		}
		if got := fmt.Sprintf("%x", sha256.Sum256(p.pixels[:])); got != c.BoardSHA256 {
			t.Errorf("original pit raster%d board %s, want%s", i, got, c.BoardSHA256)
		}
	}
}

func TestOriginalPitTireServices(t *testing.T) {
	var cases []struct {
		Selection     int
		Pressures     [4]int
		Before, After vehicle.State
	}
	data, err := os.ReadFile("testdata/pit-tires.json")
	if err != nil {
		t.Fatal(err)
	}
	if err := json.Unmarshal(data, &cases); err != nil {
		t.Fatal(err)
	}
	g, err := New(Options{Mute: true})
	if err != nil {
		t.Fatal(err)
	}
	for _, c := range cases {
		g.cars[0].physics = c.Before
		g.setup.pressures = c.Pressures
		g.servicePitItem(c.Selection)
		if g.cars[0].physics != c.After {
			t.Errorf("original pit tire selection%d repaired a different physical state", c.Selection)
		}
	}
}

// At an already reached compound limit, the original setup callback still
// replaces that tire. Its target follows setup storage, not the pit menu cursor.
func TestOriginalCompoundLimitServiceTargets(t *testing.T) {
	var cases []struct {
		Selection     int
		Pressures     [4]int
		Before, After vehicle.State
	}
	data, err := os.ReadFile("testdata/pit-tires.json")
	if err != nil {
		t.Fatal(err)
	}
	if err := json.Unmarshal(data, &cases); err != nil {
		t.Fatal(err)
	}
	for _, c := range cases {
		g, err := New(Options{Mute: true})
		if err != nil {
			t.Fatal(err)
		}
		wheel := pitTireOrder[c.Selection-1]
		row := 3 - wheel
		g.cars[0].physical, g.cars[0].physics = true, c.Before
		g.setup.pressures = c.Pressures
		g.setup.rubber[row] = 2
		g.informationPage, g.informationSelection = rubberInformationPage, row
		g.adjustInformationValue(1)
		if g.cars[0].physics != c.After {
			t.Errorf("compound limit for physical wheel%d changed incorrect source state", wheel)
		}
	}
}
