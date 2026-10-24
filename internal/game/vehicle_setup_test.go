package game

import (
	"encoding/json"
	"os"
	"testing"

	"github.com/olivierh59500/indianapolis500/internal/vehicle"
)

// Every adjustment was executed independently by the original Amiga setup
// callback. Compare all physical coefficients and all persistent quantities.
func TestOriginalSetupCallbacksThroughGame(t *testing.T) {
	data, err := os.ReadFile("testdata/setup.json")
	if err != nil {
		t.Fatal(err)
	}
	var cases []struct {
		Name            string
		InformationPage informationPage
		Selection       int
		Delta           int
		Command         string
		Config          vehicle.Config
		State           vehicle.State
	}
	if err := json.Unmarshal(data, &cases); err != nil {
		t.Fatal(err)
	}
	for _, c := range cases {
		t.Run(c.Name, func(t *testing.T) {
			g, err := New(Options{Mute: true, Practice: true})
			if err != nil {
				t.Fatal(err)
			}
			// These callbacks were measured from the recovered running
			// Practice state, rather than a fresh event reset.
			g.cars[0].physics = g.vehiclePreset.State
			g.syncPlayerVehicle()
			switch c.Command {
			case "frontRoll":
				g.frontRollBar += c.Delta
			case "rearRoll":
				g.rearRollBar += c.Delta
			case "boost":
				g.boost = c.Delta
			default:
				g.informationPage, g.informationSelection = c.InformationPage, c.Selection
				g.adjustInformationValue(c.Delta)
			}
			if got := g.physicalConfig(); got != c.Config {
				t.Errorf("physical setup %+v, original %+v", got, c.Config)
			}
			if got := g.cars[0].physics; got != c.State {
				t.Errorf("physical state %+v, original %+v", got, c.State)
			}
		})
	}
}
