package traffic

import (
	"encoding/json"
	"os"
	"slices"
	"testing"
)

func TestOriginalCautionAndTowingRules(t *testing.T) {
	data, err := os.ReadFile("testdata/caution.json")
	if err != nil {
		t.Fatal(err)
	}
	type rules struct {
		Random, TowInterval, TowWait uint16
		Yellow                       bool
		YellowDelay, SignalFlags     uint8
		Wrecks                       []int
		Leader, Tail                 int
	}
	var rows []struct {
		Kind, Operation         string
		Enabled                 bool
		Before, After           [33]Car
		BeforeRules, AfterRules rules
	}
	if err := json.Unmarshal(data, &rows); err != nil {
		t.Fatal(err)
	}
	m, _ := originalRacing(t)
	for _, source := range rows {
		t.Run(source.Kind, func(t *testing.T) {
			before := source.BeforeRules
			m.State = State{Cars: source.Before, Random: before.Random, Yellow: before.Yellow,
				YellowDelay: before.YellowDelay, SignalFlags: before.SignalFlags,
				TowInterval: before.TowInterval, TowWait: before.TowWait, WreckCount: len(before.Wrecks), Leader: before.Leader, Tail: before.Tail}
			copy(m.Wrecks[:], before.Wrecks)
			switch source.Operation {
			case "begin":
				m.SetYellow(true)
			case "tow":
				m.TowWreck()
			case "rules":
				m.StepRules(source.Enabled)
			}
			for i := range m.Cars {
				if m.Cars[i] != source.After[i] {
					t.Fatalf("car%d native%+v original%+v", i, m.Cars[i], source.After[i])
				}
			}
			after := source.AfterRules
			if m.Random != after.Random || m.Yellow != after.Yellow || m.YellowDelay != after.YellowDelay || m.SignalFlags != after.SignalFlags || m.TowInterval != after.TowInterval || m.TowWait != after.TowWait || !slices.Equal(m.Wrecks[:m.WreckCount], after.Wrecks) {
				t.Fatalf("native RNG%x yellow%v/%d signal%x tow%d/%d wrecks%v, original%+v", m.Random, m.Yellow, m.YellowDelay, m.SignalFlags, m.TowInterval, m.TowWait, m.Wrecks[:m.WreckCount], after)
			}
		})
	}
}
