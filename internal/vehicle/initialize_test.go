package vehicle

import (
	"encoding/json"
	"os"
	"testing"
)

// Complete original initializer calls distinguish reset fields from history
// that survives switching between Practice, Qualify and race rule variants.
func TestOriginalModeInitialization(t *testing.T) {
	data, err := os.ReadFile("testdata/initialization.json")
	if err != nil {
		t.Fatal(err)
	}
	var rows []struct {
		Index, Mode, FuelSetting int
		Config                   Config
		Before, After            State
		Car0                     struct {
			Distance uint16
			Lateral  int16
			Speed    uint16
		}
	}
	if err = json.Unmarshal(data, &rows); err != nil {
		t.Fatal(err)
	}
	world, _ := suspensionGeometry(t)
	for _, source := range rows {
		progress := uint16((6930 - int(source.Car0.Distance) + 39600) % 39600)
		got := Initialize(source.Before, source.Config, world, SessionStart{Progress: progress, Lateral: source.Car0.Lateral, Speed: source.Car0.Speed, FuelUnits: uint16(source.FuelSetting), WarmTires: source.Mode != 0})
		if got != source.After {
			t.Fatalf("original initializer%d mode%d native%+v, original%+v", source.Index, source.Mode, got, source.After)
		}
	}
}
