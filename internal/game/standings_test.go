package game

import (
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"os"
	"testing"

	"github.com/olivierh59500/indianapolis500/internal/traffic"
)

// Each expected hash covers all 7,680 original header pixels, including car
// liveries, retirement labels, boundary positions and race/solo formatting.
func TestOriginalStandingsHeaderRaster(t *testing.T) {
	data, err := os.ReadFile("testdata/standings-headers.json")
	if err != nil {
		t.Fatal(err)
	}
	var cases []struct {
		Kind      string
		Mode      driveMode
		TotalLaps int
		Rows      [3]standingRow
		SHA256    string
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
			b.standings(c.Mode, c.TotalLaps, c.Rows)
			b.flag(0)
			if got := fmt.Sprintf("%x", sha256.Sum256(b.pixels[:])); got != c.SHA256 {
				if os.Getenv("INDY_HEADER_DIAGNOSTIC") != "" {
					_ = os.WriteFile("/private/tmp/indy-header-reference/native-"+c.Kind+".bin", b.pixels[:], 0600)
				}
				t.Fatalf("standings header hash%s, original%s", got, c.SHA256)
			}
		})
	}
}

func TestOriginalFlagAnimation(t *testing.T) {
	data, err := os.ReadFile("testdata/header-flags.json")
	if err != nil {
		t.Fatal(err)
	}
	type state struct {
		Signal, Previous, Remaining, Delay uint8
	}
	var rows []struct {
		Before, After state
		SHA256        string
	}
	if err = json.Unmarshal(data, &rows); err != nil {
		t.Fatal(err)
	}
	var b informationBar
	if err = b.load(); err != nil {
		t.Fatal(err)
	}
	f := flagRuntime{Hidden: true}
	for tick, row := range rows {
		signal := f.step(row.Before.Signal)
		if (state{signal, f.Previous, f.Remaining, f.Delay}) != row.After {
			t.Fatalf("flag state differs at original update%d: native%+v original%+v", tick, f, row.After)
		}
		clear(b.pixels[:])
		if !f.Hidden {
			b.flag(f.Index)
		}
		var area [24 * 24]byte
		for y := range 24 {
			copy(area[y*24:(y+1)*24], b.pixels[y*320:y*320+24])
		}
		if got := fmt.Sprintf("%x", sha256.Sum256(area[:])); got != row.SHA256 {
			t.Fatalf("flag pixels differ at original update%d: hash%s original%s", tick, got, row.SHA256)
		}
	}
}

func TestOriginalStandingsScroll(t *testing.T) {
	data, err := os.ReadFile("testdata/standings-selection.json")
	if err != nil {
		t.Fatal(err)
	}
	var cases []struct{ Center, Direction, After int }
	if err = json.Unmarshal(data, &cases); err != nil {
		t.Fatal(err)
	}
	g := &Game{}
	for _, c := range cases {
		g.informationSelection = c.Center
		g.scrollStandings(c.Direction)
		if g.informationSelection != c.After {
			t.Fatalf("source scroll%d from%d: native%d original%d", c.Direction, c.Center, g.informationSelection, c.After)
		}
	}
}

func TestNormallyPlayedOriginalStandings(t *testing.T) {
	data, err := os.ReadFile("testdata/played-standings.json")
	if err != nil {
		t.Fatal(err)
	}
	var source struct {
		Mode           driveMode
		TotalLaps      int
		Center, Leader int
		Position       uint16
		Cars           [33]traffic.Car
		RankBehind     [33]int
		Timing         [33]traffic.LapTiming
		Rows           [3]standingRow
		SHA256         string
	}
	if err = json.Unmarshal(data, &source); err != nil {
		t.Fatal(err)
	}
	g, err := New(Options{Mute: true})
	if err != nil {
		t.Fatal(err)
	}
	g.mode, g.raceLaps = source.Mode, source.TotalLaps-1
	g.trafficModel.State = traffic.State{Cars: source.Cars, Leader: source.Leader, Position: source.Position, RankBehind: source.RankBehind}
	g.trafficModel.Timing.Cars = source.Timing
	g.selectInformationPage(standingsInformationPage)
	rows := g.standingsRows()
	// Qualification scores are outside the race standings display.
	for i := range rows {
		rows[i].Speed = source.Rows[i].Speed
	}
	if g.informationSelection != source.Center || rows != source.Rows {
		t.Fatalf("ordinary F2 selected different original drivers: native%+v original%+v", rows, source.Rows)
	}
	g.informationBar.standings(g.mode, g.raceLaps+1, rows)
	g.informationBar.flag(0)
	if got := fmt.Sprintf("%x", sha256.Sum256(g.informationBar.pixels[:])); got != source.SHA256 {
		t.Fatalf("ordinary F2 header hash%s, original%s", got, source.SHA256)
	}
}
