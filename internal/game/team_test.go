package game

import (
	"encoding/json"
	"os"
	"testing"

	"github.com/olivierh59500/indianapolis500/internal/vehicle"
)

func TestOriginalTeamSelectionThroughMenu(t *testing.T) {
	data, err := os.ReadFile("../vehicle/testdata/teams.json")
	if err != nil {
		t.Fatal(err)
	}
	var rows []struct {
		Team                          int
		Before, After                 vehicle.Config
		PhysicalBefore, PhysicalAfter vehicle.State
		Number                        uint8
	}
	if err := json.Unmarshal(data, &rows); err != nil {
		t.Fatal(err)
	}
	g, err := New(Options{Mute: true, Practice: true})
	if err != nil {
		t.Fatal(err)
	}
	for _, source := range rows {
		g.vehiclePreset.Config = source.Before
		g.cars[0].physics = source.PhysicalBefore
		g.menuPage, g.selected = teamMenuPage, source.Team
		g.activateMenuItem()
		if g.team != source.Team || g.vehiclePreset.Config != source.After || g.cars[0].physics != source.PhysicalAfter || g.trafficModel.Cars[0].Number != source.Number {
			t.Fatalf("team%d menu did not reproduce the complete original setup callback", source.Team)
		}
		g.mode = raceMode
		g.startDrive(false)
		if g.cars[0].number != 17 || g.trafficModel.Cars[0].Number != source.Number {
			t.Fatal("race grid confused the engine-sound flag with the visible player number")
		}
	}
}
