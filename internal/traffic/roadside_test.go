package traffic

import (
	"encoding/json"
	"os"
	"testing"
)

func TestOriginalRoadsideLimits(t *testing.T) {
	data, err := os.ReadFile("testdata/bounds.json")
	if err != nil {
		t.Fatal(err)
	}
	var rows []struct {
		Distance               uint16
		Previous, Inner, Outer int16
	}
	if err := json.Unmarshal(data, &rows); err != nil {
		t.Fatal(err)
	}
	m, _ := originalRacing(t)
	for _, source := range rows {
		inner, outer := m.RoadLimits(source.Distance, source.Previous)
		if inner != source.Inner || outer != source.Outer {
			t.Fatalf("distance%d previous%d native%d/%d original%d/%d", source.Distance, source.Previous, inner, outer, source.Inner, source.Outer)
		}
	}
}

func TestOriginalWreckRoadsideResponse(t *testing.T) {
	data, err := os.ReadFile("testdata/roadside.json")
	if err != nil {
		t.Fatal(err)
	}
	var rows []struct {
		Kind                string
		Before, After       Car
		Random, RandomAfter uint16
	}
	if err := json.Unmarshal(data, &rows); err != nil {
		t.Fatal(err)
	}
	m, _ := originalRacing(t)
	for _, source := range rows {
		t.Run(source.Kind, func(t *testing.T) {
			m.State = State{Random: source.Random, WreckCount: 1, Wrecks: [33]int{5}}
			m.Cars[5] = source.Before
			m.ConstrainWrecks()
			if m.Cars[5] != source.After || m.Random != source.RandomAfter {
				t.Fatalf("native%+v RNG%x original%+v RNG%x", m.Cars[5], m.Random, source.After, source.RandomAfter)
			}
		})
	}
}
