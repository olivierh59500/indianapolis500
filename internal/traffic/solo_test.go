package traffic

import (
	"encoding/json"
	"os"
	"testing"
)

func TestOriginalPracticeAndQualificationFactory(t *testing.T) {
	data, err := os.ReadFile("testdata/solo-factory.json")
	if err != nil {
		t.Fatal(err)
	}
	var rows []struct {
		Mode                  int
		Seed, RandomAfter     uint16
		Before, After         [33]Car
		Ranking, RankingAfter originalRanking
	}
	if err := json.Unmarshal(data, &rows); err != nil {
		t.Fatal(err)
	}
	parameters, err := os.ReadFile("../../assets/original/traffic-driver-coefficients.bin")
	if err != nil {
		t.Fatal(err)
	}
	m, _ := originalRacing(t)
	for i, source := range rows {
		m.Cars, m.Random = source.Before, source.Seed
		source.Ranking.supply(m)
		if err := m.InitializeSolo(parameters, source.Mode == 1); err != nil {
			t.Fatal(err)
		}
		source.RankingAfter.compare(t, m)
		for j := range m.Cars {
			if m.Cars[j] != source.After[j] {
				t.Fatalf("case%d mode%d car%d native%+v original%+v", i, source.Mode, j, m.Cars[j], source.After[j])
			}
		}
		if m.Random != source.RandomAfter || m.NextDriver != 1 || m.Rules != source.Mode {
			t.Fatalf("case%d source random sequence or mode context differs", i)
		}
	}
}

func TestOriginalCompletePracticeField(t *testing.T) {
	data, err := os.ReadFile("testdata/practice-field.json")
	if err != nil {
		t.Fatal(err)
	}
	var rows []racingFrame
	if err := json.Unmarshal(data, &rows); err != nil {
		t.Fatal(err)
	}
	m, _ := originalRacing(t)
	for _, source := range rows {
		m.State = State{Cars: source.Before, Random: source.Random, NextDriver: source.Cursor, Tail: source.Tail}
		source.Ranking.supply(m)
		m.StepRacingField(source.Pose, nil)
		source.RankingAfter.compare(t, m)
		for i := range m.Cars {
			if m.Cars[i] != source.After[i] {
				t.Fatalf("Practice tick%d car%d native%+v original%+v", source.Tick, i, m.Cars[i], source.After[i])
			}
		}
		if m.Random != source.RandomAfter || m.NextDriver != source.CursorAfter {
			t.Fatalf("Practice tick%d random/cadence native%x/%d original%x/%d", source.Tick, m.Random, m.NextDriver, source.RandomAfter, source.CursorAfter)
		}
	}
}
