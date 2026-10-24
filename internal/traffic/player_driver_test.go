package traffic

import (
	"encoding/json"
	"os"
	"testing"
)

// Isolated original $220AC observations cover exact word-zero speed, signed
// lateral comparisons and both pit-distance boundaries. Full car records also
// verify that the callback changes no unrelated movement, ranking or flags.
func TestOriginalRacingPlayerCallback(t *testing.T) {
	data, err := os.ReadFile("testdata/player-pit-flag.json")
	if err != nil {
		t.Fatal(err)
	}
	var cases []struct {
		Kind          string
		Before, After Car
	}
	if err = json.Unmarshal(data, &cases); err != nil {
		t.Fatal(err)
	}
	if len(cases) != 20 {
		t.Fatalf("got %d original boundary observations, want 20", len(cases))
	}
	for _, source := range cases {
		t.Run(source.Kind, func(t *testing.T) {
			model := Model{State: State{Random: 0x2468, NextDriver: 17, Tail: 19, Leader: 2, Position: 29}}
			model.Cars[0] = source.Before
			want := model.State
			want.Cars[0] = source.After
			model.updateRacingPlayer()
			if model.State != want {
				t.Fatalf("native player %+v differs from original %+v", model.Cars[0], source.After)
			}
		})
	}
}
