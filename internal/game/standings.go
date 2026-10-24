package game

import (
	"fmt"

	"github.com/hajimehoshi/ebiten/v2"
	gameassets "github.com/olivierh59500/indianapolis500/assets"
	"github.com/olivierh59500/indianapolis500/internal/traffic"
)

type standingRow struct {
	Rank, Number  int
	Colors        [2]byte
	Timing        traffic.LapTiming
	RemainingLaps uint8
	Retired       uint8
	Speed         uint16
}

// informationBar paints the original 24-row header in palette indices. The
// recovered car icon retains its two replaceable livery colors.
type informationBar struct {
	pixels    [320 * 24]byte
	font      []byte
	backdrop  []byte
	car       []byte
	flags     []byte
	valid     bool
	tick      int
	selection int
	mode      driveMode
	page      informationPage
}

func (b *informationBar) load() error {
	var err error
	b.font, err = gameassets.Original.ReadFile("original/font.bin")
	if err != nil {
		return err
	}
	art, err := gameassets.Original.ReadFile("original/standings-art.bin")
	if err != nil {
		return err
	}
	if len(b.font) != 482 || len(art) != 4+88*24+40*6+6*24*18 || string(art[:4]) != "I5HB" {
		return fmt.Errorf("invalid recovered standings header artwork")
	}
	b.backdrop = art[4 : 4+88*24]
	b.car = art[4+88*24 : 4+88*24+40*6]
	b.flags = art[4+88*24+40*6:]
	return nil
}

func (b *informationBar) text(text string, x, y int, color byte) {
	for i := range len(text) {
		glyph := (int(text[i]) - 0x1f) * 8
		if glyph < 0 || glyph+8 > len(b.font) {
			continue
		}
		for row := range 8 {
			for column := range 8 {
				px := x + i*8 + column
				if px < 0 || px >= 320 || y+row < 0 || y+row >= 24 {
					continue
				}
				value := byte(0)
				if b.font[glyph+row]&(128>>uint(column)) != 0 {
					value = color
				}
				b.pixels[(y+row)*320+px] = value
			}
		}
	}
}

func (b *informationBar) flag(index int) {
	for y := range 18 {
		copy(b.pixels[(y+4)*320:(y+4)*320+24], b.flags[index*24*18+y*24:index*24*18+(y+1)*24])
	}
}

func (b *informationBar) standings(mode driveMode, totalLaps int, rows [3]standingRow) {
	clear(b.pixels[:])
	b.text("STANDINGS", 24, 0, 15)
	labels := [...]string{"PRACTICE", "QUALIFY ", "RACE    "}
	colors := [...]byte{5, 14, 12}
	b.text(labels[mode], 24, 16, colors[mode])
	for y := range 24 {
		copy(b.pixels[y*320+104:y*320+192], b.backdrop[y*88:(y+1)*88])
	}
	for row, car := range rows {
		y := row * 8
		b.text(fmt.Sprintf("%2d", car.Rank), 136, y, 15)
		b.text(fmt.Sprintf("%2d", car.Number), 160, y, 14)
		b.smallCar(192, y, car.Colors)
		if mode != raceMode {
			b.text(fmt.Sprintf("%3d.%02d", car.Speed/100, car.Speed%100), 240, y, 15)
			b.text(" MPH", 288, y, 14)
			continue
		}
		if car.Retired != 0 {
			reasons := [...]string{"RUNNING", "ENGINE", "CLUTCH", "GEARBOX", "BEARING", "IGNITION", "VALVE", "CV JOINT", "VIBRATION", "STALLED", "RADIATOR", "OIL LEAK", "CRASH"}
			reason := int(car.Retired) / 2
			if reason < len(reasons) {
				b.text(fmt.Sprintf("  %-9s", reasons[reason]), 232, y, 15)
			}
			continue
		}
		if car.Rank == 1 {
			speed := standingsAverageSpeed(uint8(totalLaps), car)
			b.text(fmt.Sprintf(" %3d.%02d", speed/100, speed%100), 232, y, 15)
			b.text(" MPH", 288, y, 14)
			continue
		}
		gap := "   "
		if car.Timing.LapGap != 0 {
			switch laps := car.Timing.LapGap; {
			case laps < 10:
				gap = fmt.Sprintf(" -%d", laps)
			case laps < 100:
				gap = fmt.Sprintf("-%2d", laps)
			default:
				gap = fmt.Sprintf("%3d", laps)
			}
		}
		b.text(gap, 232, y, 15)
		time := min(car.Timing.TimeGap, 1799)
		b.text(fmt.Sprintf(" -%d:%02d.%d", time/600, time/10%60, time%10), 256, y, 15)
	}
}

func standingsAverageSpeed(totalLaps uint8, car standingRow) uint16 {
	laps := totalLaps - car.RemainingLaps
	if laps == 0 {
		return 0
	}
	laps--
	if laps == 0 || uint32(laps)<<16 <= car.Timing.Crossing {
		return 0
	}
	average := car.Timing.Crossing / uint32(laps)
	if average == 0 {
		return 0
	}
	return uint16(9000000 / average)
}

func (g *Game) selectInformationPage(page informationPage) {
	g.informationPage, g.informationSelection = page, 0
	g.informationBar.valid, g.pitBoard = false, false
	if page == standingsInformationPage {
		position := g.qualifyingPosition
		if g.mode == raceMode {
			position = g.racePosition()
		}
		g.informationSelection = max(2, min(32, position))
	}
}

func (g *Game) scrollStandings(direction int) {
	g.informationSelection = max(2, min(32, g.informationSelection+direction))
}

func (g *Game) standingsRows() (rows [3]standingRow) {
	first := max(1, min(31, g.informationSelection-1))
	var order [33]int
	if g.mode == raceMode {
		index := g.trafficModel.Leader
		for rank := range order {
			order[rank] = index
			if index >= 0 {
				index = g.trafficModel.RankBehind[index]
			}
		}
	} else {
		index := 32
		for rank := range order {
			if rank == g.qualifyingPosition-1 {
				order[rank] = 0
			} else {
				order[rank] = index
				index--
			}
		}
	}
	for row := range rows {
		rank := first + row
		index := order[rank-1]
		if index < 0 {
			continue
		}
		car := &g.trafficModel.Cars[index]
		speed := g.qualifyingSpeed
		if index != 0 {
			speed = uint16(19512 + (index-1)*112)
		}
		rows[row] = standingRow{Rank: rank, Number: int(car.Number & 127), Colors: [2]byte{car.Primary, car.Secondary},
			Timing: g.trafficModel.Timing.Cars[index], RemainingLaps: car.RemainingLaps, Retired: car.RetiredReason, Speed: speed}
	}
	return rows
}

func (g *Game) drawStandingsInformation(screen *ebiten.Image) {
	b := &g.informationBar
	if !b.valid || b.page != standingsInformationPage || b.mode != g.mode || b.selection != g.informationSelection || g.tick-b.tick >= 30 {
		// The original race count includes the initial finish-line crossing.
		b.standings(g.mode, g.raceLaps+1, g.standingsRows())
		g.uploadInformationBar()
		b.valid, b.tick, b.selection, b.mode = true, g.tick, g.informationSelection, g.mode
		b.page = standingsInformationPage
	}
	screen.DrawImage(g.informationImage, nil)
	g.drawRaceFlags(screen)
}

// flagRuntime advances once per original header update. A new signal requests
// nine blink phases of five updates before its flags remain visible.
type flagRuntime struct {
	Previous, Remaining, Delay uint8
	Index                      int
	Hidden                     bool
	phase                      int
}

func (f *flagRuntime) step(signal byte) byte {
	if f.Remaining == 0 && signal == f.Previous {
		return signal &^ 16
	}
	f.Previous = signal
	if signal&16 != 0 {
		f.Delay, f.Remaining = 5, 9
	}
	index := 0
	if signal&2 != 0 {
		index = 1
		if signal&8 != 0 {
			index = 5
		} else if signal&4 != 0 {
			index = 4
		}
	} else if signal&8 != 0 {
		index = 3
	} else if signal&4 != 0 {
		index = 2
	}
	if f.Remaining != 0 {
		f.Delay--
		if f.Delay != 0 {
			return signal &^ 16
		}
		f.Delay = 5
		f.Remaining--
		if f.Remaining&1 != 0 {
			f.Hidden = true
			return signal &^ 16
		}
	}
	f.Index, f.Hidden = index, false
	return signal &^ 16
}
