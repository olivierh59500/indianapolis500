package game

import (
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"os"
	"testing"
)

type playedLapInfo struct {
	Kind        string
	Values      lapInformationValues
	ComparedCar int
	SHA256      string
	SHA256RGBA  string
}

func (g *Game) supplyLapInfo(source playedLapInfo) {
	v := source.Values
	g.mode, g.clock, g.raceLaps = v.Mode, v.Clock, int(v.TotalLaps)-1
	m := g.trafficModel
	m.Position = v.Position
	m.Cars[0].RemainingLaps, m.Cars[0].Flags = v.RemainingLaps, v.PlayerFlags
	m.Timing.Cars[0] = v.PlayerTiming
	m.Timing.AheadTime, m.Timing.BehindTime = v.AheadTime, v.BehindTime
	m.Timing.AheadLaps, m.Timing.BehindLaps = v.AheadLaps, v.BehindLaps
	if v.HasComparedCar {
		m.Timing.ComparedCar = source.ComparedCar
		m.Cars[source.ComparedCar].Primary, m.Cars[source.ComparedCar].Secondary = v.ComparedColors[0], v.ComparedColors[1]
	}
	g.selectInformationPage(lapInformationPage)
}

func TestNormallyPlayedLapInfo(t *testing.T) {
	data, err := os.ReadFile("testdata/played-lap-info.json")
	if err != nil {
		t.Fatal(err)
	}
	var rows []playedLapInfo
	if err = json.Unmarshal(data, &rows); err != nil {
		t.Fatal(err)
	}
	for _, source := range rows {
		t.Run(source.Kind, func(t *testing.T) {
			g, err := New(Options{Mute: true})
			if err != nil {
				t.Fatal(err)
			}
			g.supplyLapInfo(source)
			values := g.lapInformationValues()
			if values != source.Values {
				t.Fatalf("playable Lap Info values differ: native%+v original%+v", values, source.Values)
			}
			g.informationBar.lapInfo(values)
			g.informationBar.flag(0)
			if got := fmt.Sprintf("%x", sha256.Sum256(g.informationBar.pixels[:])); got != source.SHA256 {
				if os.Getenv("INDY_HEADER_DIAGNOSTIC") != "" {
					_ = os.WriteFile("/private/tmp/indy-lap-info-reference/native-played-"+source.Kind+".bin", g.informationBar.pixels[:], 0600)
				}
				t.Fatalf("normally played Lap Info hash%s original%s", got, source.SHA256)
			}
		})
	}
}

func TestOriginalLapInfoRaster(t *testing.T) {
	data, err := os.ReadFile("testdata/lap-info.json")
	if err != nil {
		t.Fatal(err)
	}
	var cases []struct {
		Kind   string
		Values lapInformationValues
		SHA256 string
	}
	if err = json.Unmarshal(data, &cases); err != nil {
		t.Fatal(err)
	}
	var b informationBar
	if err = b.load(); err != nil {
		t.Fatal(err)
	}
	for _, c := range cases {
		t.Run(c.Kind, func(t *testing.T) {
			b.lapInfo(c.Values)
			b.flag(0)
			if got := fmt.Sprintf("%x", sha256.Sum256(b.pixels[:])); got != c.SHA256 {
				if os.Getenv("INDY_HEADER_DIAGNOSTIC") != "" {
					_ = os.WriteFile("/private/tmp/indy-lap-info-reference/native-"+c.Kind+".bin", b.pixels[:], 0600)
				}
				t.Fatalf("Lap Info hash%s, original%s", got, c.SHA256)
			}
		})
	}
}
