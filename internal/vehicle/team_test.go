package vehicle

import (
	"encoding/json"
	"os"
	"testing"
)

func TestOriginalTeamEngineSelection(t *testing.T) {
	data, err := os.ReadFile("testdata/teams.json")
	if err != nil {
		t.Fatal(err)
	}
	var rows []struct {
		Team                          int
		Before, After                 Config
		PhysicalBefore, PhysicalAfter State
		Number                        uint8
	}
	if err := json.Unmarshal(data, &rows); err != nil {
		t.Fatal(err)
	}
	for _, source := range rows {
		config := source.Before
		config.Engine = EngineForTeam(config.Engine, source.Team)
		if config != source.After || PlayerCarIdentity(source.Team) != source.Number {
			t.Fatalf("team%d native%+v number%x original%+v number%x", source.Team, config.Engine, PlayerCarIdentity(source.Team), source.After.Engine, source.Number)
		}
		if source.PhysicalBefore != source.PhysicalAfter {
			t.Fatal("original team callback unexpectedly changed the physical state")
		}
	}
}
