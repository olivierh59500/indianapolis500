package game

import (
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"os"
	"testing"
)

// Original calls draw all eleven widgets over each recovered team panel.
// Compare both derived values and every framebuffer pixel, including warning
// sequences and overlapping digital graphics.
func TestOriginalCompleteTeamDashboards(t *testing.T) {
	data, err := os.ReadFile("testdata/dashboards.json")
	if err != nil {
		t.Fatal(err)
	}
	var rows []struct {
		instrumentInput
		Team, Case          int
		Values              [11]uint16
		FuelFlash, OilFlash byte
		SHA256              string
	}
	if err := json.Unmarshal(data, &rows); err != nil {
		t.Fatal(err)
	}
	var d dashboard
	for _, source := range rows {
		if source.Case == 0 {
			d = dashboard{}
			if err := d.load(); err != nil {
				t.Fatal(err)
			}
		}
		d.full(source.Team, source.instrumentInput)
		if d.values != source.Values || d.fuelFlash != source.FuelFlash || d.oilFlash != source.OilFlash {
			t.Fatalf("team%d case%d native values%v flashes%d/%d, original%v %d/%d", source.Team, source.Case, d.values, d.fuelFlash, d.oilFlash, source.Values, source.FuelFlash, source.OilFlash)
		}
		digest := fmt.Sprintf("%x", sha256.Sum256(d.pixels[:]))
		if digest != source.SHA256 {
			t.Fatalf("team%d case%d complete dashboard hash%s original%s", source.Team, source.Case, digest, source.SHA256)
		}
	}
}
