package game

import (
	"fmt"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/olivierh59500/indianapolis500/internal/session"
	"github.com/olivierh59500/indianapolis500/internal/traffic"
)

type lapInformationValues struct {
	Mode                     driveMode
	Clock                    session.Clock
	Position                 uint16
	TotalLaps, RemainingLaps uint8
	PlayerFlags              uint8
	PlayerTiming             traffic.LapTiming
	AheadTime, BehindTime    uint16
	AheadLaps, BehindLaps    uint8
	HasComparedCar           bool
	ComparedColors           [2]byte
}

func headerInteger(value uint16, digits int) string {
	modulus := uint16(10)
	for i := 1; i < digits; i++ {
		modulus *= 10
	}
	return fmt.Sprintf("%*d", digits, value%modulus)
}

func (b *informationBar) lapInfo(v lapInformationValues) {
	clear(b.pixels[:])
	b.text("LAP INFO", 24, 0, 15)
	labels := [...]string{"PRACTICE", "QUALIFY ", "RACE    "}
	colors := [...]byte{5, 14, 12}
	b.text(labels[v.Mode], 24, 16, colors[v.Mode])
	if v.Mode == raceMode {
		b.raceLapInfo(v)
		return
	}
	first, time, speed := "BEST", v.Clock.Best, v.Clock.BestSpeed
	if v.Mode == qualifyMode {
		first, time, speed = "AVG ", v.Clock.Qualifying, v.Clock.QualifyingSpeed
	}
	b.text(first, 112, 0, 14)
	b.text("LAST", 112, 8, 14)
	b.text("CURR", 112, 16, 14)
	b.text(clockText(time), 160, 0, 15)
	b.text(clockText(v.Clock.Last), 160, 8, 15)
	b.text(clockText(v.Clock.Current), 160, 16, 15)
	b.headerSpeed(speed, 240, 0)
	b.headerSpeed(v.Clock.LastSpeed, 240, 8)
	b.text("LAP", 256, 16, 14)
	b.text(headerInteger(v.Clock.Laps, 3), 288, 16, 15)
}

func (b *informationBar) headerSpeed(value uint16, x, y int) {
	b.text(fmt.Sprintf("%3d.%02d", value/100, value%100), x, y, 15)
	b.text(" MPH", 288, y, 14)
}

func (b *informationBar) raceLapInfo(v lapInformationValues) {
	b.text("LAP", 112, 0, 14)
	b.text(headerInteger(uint16(v.TotalLaps-v.RemainingLaps), 3), 144, 0, 15)
	b.text("OF", 176, 0, 14)
	b.text(headerInteger(uint16(v.TotalLaps)-1, 3), 200, 0, 15)
	if v.PlayerTiming.LapGap == 0 {
		b.text("LEAD LAP", 256, 0, 14)
	} else {
		b.text("    DOWN", 256, 0, 14)
		b.text(headerInteger(uint16(v.PlayerTiming.LapGap), 3), 256, 0, 15)
	}
	if v.PlayerFlags&1 == 0 {
		b.text("LAST", 112, 8, 14)
		b.text(clockText(v.Clock.Last), 160, 8, 15)
		b.headerSpeed(v.Clock.LastSpeed, 240, 8)
	} else {
		b.text("RACE", 112, 8, 14)
		b.text(finishedRaceTime(v), 160, 8, 15)
		speed := standingsAverageSpeed(v.TotalLaps, standingRow{Timing: v.PlayerTiming, RemainingLaps: v.RemainingLaps})
		b.text(fmt.Sprintf(" %3d.%02d", speed/100, speed%100), 232, 8, 15)
		b.text(" MPH", 288, 8, 14)
	}
	b.text("POS", 272, 16, 14)
	b.text(headerInteger(v.Position, 2), 304, 16, 15)
	if v.HasComparedCar {
		b.smallCar(112, 16, v.ComparedColors)
	}
	x := 152
	laps, time, sign := v.AheadLaps, v.AheadTime, byte('-')
	if v.BehindLaps != 0 || v.BehindTime != 0 {
		laps, time, sign = v.BehindLaps, v.BehindTime, '+'
	}
	if laps != 0 {
		text := headerInteger(uint16(laps), 3)
		if laps < 10 {
			text = fmt.Sprintf(" %c%d", sign, laps)
		} else if laps < 100 {
			text = fmt.Sprintf("%c%d", sign, laps)
		}
		b.text(text, x, 16, 15)
		x += 24
	}
	time = min(time, 1799)
	b.text(fmt.Sprintf(" %c%d:%02d.%d", sign, time/600, time/10%60, time%10), x, 16, 15)
}

func finishedRaceTime(v lapInformationValues) string {
	if v.RemainingLaps != 0 || v.PlayerTiming.Crossing > 0x90000 {
		return "FLAGGED"
	}
	seconds := v.PlayerTiming.Crossing / 10
	// The source has a single hour character rather than a variable-width field.
	return string([]byte{'0' + byte(seconds/3600%10), ':', '0' + byte(seconds/600%6), '0' + byte(seconds/60%10), ':', '0' + byte(seconds/10%6), '0' + byte(seconds%10)})
}

func (b *informationBar) smallCar(x, y int, colors [2]byte) {
	for row := range 6 {
		for column := range 40 {
			value := b.car[row*40+column]
			if value == 12 || value == 13 {
				value = colors[value-12]
			}
			b.pixels[(y+row)*320+x+column] = value
		}
	}
}

func (g *Game) lapInformationValues() lapInformationValues {
	m := g.trafficModel
	v := lapInformationValues{Mode: g.mode, Clock: g.clock, Position: m.Position,
		TotalLaps: uint8(g.raceLaps + 1), RemainingLaps: m.Cars[0].RemainingLaps,
		PlayerFlags: m.Cars[0].Flags, PlayerTiming: m.Timing.Cars[0],
		AheadTime: m.Timing.AheadTime, BehindTime: m.Timing.BehindTime,
		AheadLaps: m.Timing.AheadLaps, BehindLaps: m.Timing.BehindLaps}
	if index := m.Timing.ComparedCar; index >= 0 && index < len(m.Cars) {
		v.HasComparedCar = true
		v.ComparedColors = [2]byte{m.Cars[index].Primary, m.Cars[index].Secondary}
	}
	return v
}

func (g *Game) uploadInformationBar() {
	for i, index := range g.informationBar.pixels {
		color := g.worldColors[index]
		copy(g.informationPixels[i*4:i*4+4], color[:])
	}
	g.informationImage.WritePixels(g.informationPixels[:])
}

func (g *Game) drawLapInformation(screen *ebiten.Image) {
	b := &g.informationBar
	if !b.valid || b.page != lapInformationPage || b.mode != g.mode || b.tick != g.tick {
		b.lapInfo(g.lapInformationValues())
		g.uploadInformationBar()
		b.valid, b.tick, b.mode, b.page = true, g.tick, g.mode, lapInformationPage
	}
	screen.DrawImage(g.informationImage, nil)
	// The source Lap Info callback consumes its pending lap update. Finished
	// races return before that operation and retain all clock flag bits.
	if g.mode != raceMode || g.trafficModel.Cars[0].Flags&1 == 0 {
		g.clock.Flags &^= 1
	}
	g.drawRaceFlags(screen)
}
