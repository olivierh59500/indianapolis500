package game

import (
	"compress/gzip"
	"encoding/json"
	"os"
	"testing"
)

// The original callback emits several limits, including zero and unrestricted
// throttle. The live engine configuration must consume that stored result.
func TestOriginalPlayerCautionLimitReachesEngine(t *testing.T) {
	file, err := os.Open("../traffic/testdata/player-caution.json.gz")
	if err != nil {
		t.Fatal(err)
	}
	defer file.Close()
	reader, err := gzip.NewReader(file)
	if err != nil {
		t.Fatal(err)
	}
	defer reader.Close()
	var cases []struct {
		Kind  string
		After struct{ PlayerThrottleLimit uint16 }
	}
	if err = json.NewDecoder(reader).Decode(&cases); err != nil {
		t.Fatal(err)
	}
	g, err := New(Options{Mute: true, Race: true})
	if err != nil {
		t.Fatal(err)
	}
	g.trafficModel.Yellow = true
	for _, source := range cases {
		g.trafficModel.PlayerThrottleLimit = source.After.PlayerThrottleLimit
		for _, demonstration := range []bool{false, true} {
			g.demo = demonstration
			if got := g.physicalConfig().ThrottleLimit; got != source.After.PlayerThrottleLimit {
				t.Fatalf("%s demonstration %v: engine limit %d, original %d", source.Kind, demonstration, got, source.After.PlayerThrottleLimit)
			}
		}
	}
}
