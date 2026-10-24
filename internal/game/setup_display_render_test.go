//go:build rendercheck

package game

import (
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"os"
	"testing"
)

// The actual native game adapter must paint the original observed headers,
// including the stored physical tire order and live setup selection.
func TestOriginalSetupDisplaysThroughGame(t *testing.T) {
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
	g, err := New(Options{Mute: true, Practice: true})
	if err != nil {
		t.Fatal(err)
	}
	for _, c := range cases {
		v := c.Values
		g.mode, g.informationPage, g.informationSelection = v.Mode, v.Page, v.Selection
		g.setup = persistentToSetup(v.Setup)
		g.cars[0].physics, g.cars[0].dashboardSpeed = v.State, v.RearGaugeSpeed
		g.noticeInPit = v.AtPit
		g.flags.Hidden = true
		g.informationBar.valid = false
		pixels := make([]byte, len(c.Header)*4)
		for i, color := range c.Header {
			copy(pixels[i*4:i*4+4], g.worldColors[color][:])
		}
		headerRenderRequests <- &headerRenderCheck{game: g, digest: fmt.Sprintf("%x", sha256.Sum256(pixels))}
		if err := <-headerRenderResults; err != nil {
			t.Fatalf("actual game setup page%d: %v", c.OriginalPage, err)
		}
	}
}
