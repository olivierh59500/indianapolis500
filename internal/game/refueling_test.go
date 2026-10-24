package game

import (
	"encoding/json"
	"os"
	"testing"

	"github.com/olivierh59500/indianapolis500/internal/vehicle"
)

// Original refueling callbacks replace the whole fuel units while retaining
// fractional consumption, engine condition and the other physical quantities.
func TestOriginalRefueling(t *testing.T) {
	var cases []struct {
		Gallons       int
		Before, After vehicle.State
	}
	data, err := os.ReadFile("testdata/refueling.json")
	if err != nil {
		t.Fatal(err)
	}
	if err := json.Unmarshal(data, &cases); err != nil {
		t.Fatal(err)
	}
	if len(cases) != 16 {
		t.Fatalf("got %d source refueling cases, want 16", len(cases))
	}
	g, err := New(Options{Mute: true})
	if err != nil {
		t.Fatal(err)
	}
	for i, source := range cases {
		g.cars[0].physical, g.cars[0].physics = true, source.Before
		g.setup.fuel = source.Gallons
		g.refuelPlayer()
		if g.cars[0].physics != source.After {
			t.Fatalf("source refueling %d: native fuel %08x mass%d failures%02x, want %08x mass%d failures%02x", i, g.cars[0].physics.Engine.Fuel, g.cars[0].physics.Engine.Mass, g.cars[0].physics.Engine.Failures, source.After.Engine.Fuel, source.After.Engine.Mass, source.After.Engine.Failures)
		}
	}
}
