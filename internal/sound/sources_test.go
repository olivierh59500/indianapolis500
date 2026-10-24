package sound

import (
	"encoding/json"
	"os"
	"reflect"
	"testing"

	"github.com/olivierh59500/indianapolis500/internal/traffic"
)

func TestOriginalCockpitEngineSources(t *testing.T) {
	data, err := os.ReadFile("testdata/engine-sources.json")
	if err != nil {
		t.Fatal(err)
	}
	var rows []struct {
		Kind                        string
		Before, After               originalMixerState
		BeforeSources, AfterSources Sources
		Cars                        [33]traffic.Car
		Movement                    int16
		RPM, Throttle               uint16
		Reduced, Quiet              bool
	}
	if err = json.Unmarshal(data, &rows); err != nil {
		t.Fatal(err)
	}
	for _, row := range rows {
		t.Run(row.Kind, func(t *testing.T) {
			c, s := row.Before.Controller, row.BeforeSources
			field := &traffic.Model{State: traffic.State{Cars: row.Cars, PlayerMovement: row.Movement}}
			s.Driving(&c, field, row.RPM, row.Throttle, row.Reduced, row.Quiet)
			if !reflect.DeepEqual(c, row.After.Controller) || !reflect.DeepEqual(s, row.AfterSources) {
				t.Fatalf("source selection differs: native controller%+v sources%+v original controller%+v sources%+v", c, s, row.After.Controller, row.AfterSources)
			}
		})
	}
}
