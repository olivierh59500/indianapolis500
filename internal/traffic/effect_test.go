package traffic

import (
	"encoding/json"
	"os"
	"slices"
	"testing"
)

func TestOriginalContactEffectArguments(t *testing.T) {
	data, err := os.ReadFile("testdata/collision-effects.json")
	if err != nil {
		t.Fatal(err)
	}
	var rows []struct {
		Kind          string
		First, Second Car
		Gap           int16
		Effects       []CollisionEffect
	}
	if err = json.Unmarshal(data, &rows); err != nil {
		t.Fatal(err)
	}
	m, _ := originalRacing(t)
	for _, row := range rows {
		t.Run(row.Kind, func(t *testing.T) {
			m.State = State{}
			m.Cars[0], m.Cars[6] = row.First, row.Second
			var effects []CollisionEffect
			m.OnCollisionEffect = func(e CollisionEffect) { effects = append(effects, e) }
			m.ResolveContact(0, 6, row.Gap)
			if !slices.Equal(effects, row.Effects) {
				t.Fatalf("native effects%+v, original%+v", effects, row.Effects)
			}
		})
	}
}

func TestOriginalRoadEffectArguments(t *testing.T) {
	data, err := os.ReadFile("testdata/road-effects.json")
	if err != nil {
		t.Fatal(err)
	}
	var rows []struct {
		Kind    string
		Car     Car
		Effects []CollisionEffect
	}
	if err = json.Unmarshal(data, &rows); err != nil {
		t.Fatal(err)
	}
	m, _ := originalRacing(t)
	for _, row := range rows {
		t.Run(row.Kind, func(t *testing.T) {
			m.State = State{WreckCount: 1, Wrecks: [33]int{5}}
			m.Cars[5] = row.Car
			var effects []CollisionEffect
			m.OnCollisionEffect = func(e CollisionEffect) { effects = append(effects, e) }
			m.ConstrainWrecks()
			if !slices.Equal(effects, row.Effects) {
				t.Fatalf("native effects%+v, original%+v", effects, row.Effects)
			}
		})
	}
}
