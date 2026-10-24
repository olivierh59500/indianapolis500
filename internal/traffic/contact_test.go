package traffic

import (
	"encoding/json"
	"os"
	"testing"
)

// Original calls cover the strict lateral boundary, sign changes during
// crossing, light speed exchanges and the 160-unit transition to spin impacts.
func TestOriginalCarContactResponses(t *testing.T) {
	data, err := os.ReadFile("testdata/contacts.json")
	if err != nil {
		t.Fatal(err)
	}
	var rows []struct {
		Kind                 string
		Gap                  int16
		Before, After        [2]Car
		Hit                  bool
		Severity             uint16
		TrackGap, LateralGap int16
		Random, RandomAfter  uint16
	}
	if err = json.Unmarshal(data, &rows); err != nil {
		t.Fatal(err)
	}
	m, _ := originalRacing(t)
	for _, source := range rows {
		t.Run(source.Kind, func(t *testing.T) {
			m.WreckCount, m.YellowDelay, m.Yellow = 0, 0, false
			m.Cars[5], m.Cars[6], m.Random = source.Before[0], source.Before[1], source.Random
			result := m.ResolveContact(5, 6, source.Gap)
			if result.Hit != source.Hit {
				t.Fatalf("native contact%+v, original hit%v", result, source.Hit)
			}
			if result.Hit && (result.Severity != source.Severity || result.TrackGap != source.TrackGap || result.LateralGap != source.LateralGap) {
				t.Fatalf("native contact%+v, original severity%d gaps%d/%d", result, source.Severity, source.TrackGap, source.LateralGap)
			}
			for i, index := range []int{5, 6} {
				if m.Cars[index] != source.After[i] {
					t.Fatalf("car%d native%+v, original%+v", index, m.Cars[index], source.After[i])
				}
			}
			if m.Random != source.RandomAfter {
				t.Fatal("contact changed the original random sequence")
			}
		})
	}
}
