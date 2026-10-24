package traffic

import (
	"compress/gzip"
	"encoding/json"
	"os"
	"testing"
)

// Controlled original $220DA calls independently cover the throttle curve,
// signed word arithmetic, lap epochs and queue membership changes.
func TestOriginalPlayerCautionCallback(t *testing.T) {
	file, err := os.Open("testdata/player-caution.json.gz")
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
		Kind          string
		Before, After State
	}
	if err = json.NewDecoder(reader).Decode(&cases); err != nil {
		t.Fatal(err)
	}
	if len(cases) != 39 {
		t.Fatalf("got %d original caution observations, want 39", len(cases))
	}
	for _, source := range cases {
		t.Run(source.Kind, func(t *testing.T) {
			model := Model{State: source.Before}
			model.updatePlayerDriver()
			if model.State != source.After {
				for i, car := range model.Cars {
					if car != source.After.Cars[i] {
						t.Fatalf("car %d: native %+v, original %+v", i, car, source.After.Cars[i])
					}
				}
				t.Fatalf("native limit %d head %d RNG %x; original limit %d head %d RNG %x", model.PlayerThrottleLimit, model.CautionLeader, model.Random, source.After.PlayerThrottleLimit, source.After.CautionLeader, source.After.Random)
			}
		})
	}
}
