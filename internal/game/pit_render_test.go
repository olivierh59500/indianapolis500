//go:build rendercheck

package game

import (
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"os"
	"testing"
)

// The actual original cockpit outline covers the bottom of the pit board.
// Compare the complete composited area through the real Ebitengine painter.
func TestOriginalPlayedPitCompositing(t *testing.T) {
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
	g, err := New(Options{Mute: true, Practice: true})
	if err != nil {
		t.Fatal(err)
	}
	rgba := func(indices []byte) []byte {
		out := make([]byte, len(indices)*4)
		for i, c := range indices {
			copy(out[i*4:i*4+4], g.worldColors[c][:])
		}
		return out
	}
	for _, c := range cases {
		g.pitSelection = c.Selection
		g.pitBoard = true
		g.flags.Hidden = true
		headerRenderRequests <- &headerRenderCheck{game: g, digest: fmt.Sprintf("%x", sha256.Sum256(rgba(c.Header))), pitBodyDigest: fmt.Sprintf("%x", sha256.Sum256(rgba(c.Board)))}
		if err := <-headerRenderResults; err != nil {
			t.Fatalf("original played pit selection%d: %v", c.Selection, err)
		}
	}
}
