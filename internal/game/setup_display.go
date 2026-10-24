package game

import (
	"encoding/json"
	"fmt"

	"github.com/hajimehoshi/ebiten/v2"
	gameassets "github.com/olivierh59500/indianapolis500/assets"
	"github.com/olivierh59500/indianapolis500/internal/vehicle"
)

// Setup storage retains its existing physical order. The original information
// choices instead travel clockwise: right front, right rear, left rear, left front.
var setupTireRows = [4]int{0, 2, 3, 1}
var setupTireWheels = [4]int{3, 1, 0, 2}

type setupTextSpan struct {
	X, Y, Color int
	Text        string
}
type setupArtwork struct {
	Pixels []byte
	Text   map[string][]setupTextSpan
}
type setupDisplayValues struct {
	Mode           driveMode
	Page           informationPage
	Selection      int
	Setup          persistentSetup
	Config         vehicle.Config
	State          vehicle.State
	RearGaugeSpeed int16
	AtPit          bool
}

func (a *setupArtwork) load() error {
	data, err := gameassets.Original.ReadFile("original/setup-art.bin")
	if err != nil {
		return err
	}
	if len(data) != 5084 || string(data[:4]) != "I5SU" {
		return fmt.Errorf("invalid recovered setup artwork")
	}
	a.Pixels = data[4:]
	data, err = gameassets.Original.ReadFile("original/setup-text.json")
	if err != nil {
		return err
	}
	return json.Unmarshal(data, &a.Text)
}

func (b *informationBar) setupText(a *setupArtwork, name string, inherit byte) {
	for _, span := range a.Text[name] {
		color := inherit
		if span.Color >= 0 {
			color = byte(span.Color)
		}
		b.text(span.Text, span.X, span.Y, color)
	}
}

func (b *informationBar) setupBitmap(data []byte, width, height, x, y int) {
	for row := 0; row < height; row++ {
		for column := 0; column < width; column++ {
			if x+column >= 0 && x+column < 320 && y+row >= 0 && y+row < 24 {
				b.pixels[(y+row)*320+x+column] = data[row*width+column]
			}
		}
	}
}

func (b *informationBar) setupMask(offset int, mask byte, color byte) {
	y, x := offset/40, offset%40*8
	for bit := 0; bit < 8; bit++ {
		if mask&(128>>uint(bit)) != 0 && x+bit >= 0 && x+bit < 320 && y >= 0 && y < 24 {
			b.pixels[y*320+x+bit] = color
		}
	}
}

func (b *informationBar) setupLine(x0, y0, x1, y1 int, color byte) {
	dx, dy := x1-x0, y1-y0
	sx, sy := 1, 1
	if dx < 0 {
		dx = -dx
		sx = -1
	}
	if dy < 0 {
		dy = -dy
		sy = -1
	}
	err := dx - dy
	for {
		if x0 >= 0 && x0 < 320 && y0 >= 0 && y0 < 24 {
			b.pixels[y0*320+x0] = color
		}
		if x0 == x1 && y0 == y1 {
			return
		}
		e := err * 2
		if e >= -dy {
			err -= dy
			x0 += sx
		}
		if e <= dx {
			err += dx
			y0 += sy
		}
	}
}

// Original numeric patterns are stored least-significant character first.
// '#' always prints a digit; '*' suppresses a leading zero, including its space.
func (b *informationBar) setupNumber(value uint16, pattern string, column, row int, color byte) {
	buffer := make([]byte, len(pattern))
	for i, c := range []byte(pattern) {
		out := c
		if c == '#' || c == '*' {
			if c == '*' && value == 0 {
				out = ' '
			} else {
				out = '0' + byte(value%10)
				value /= 10
			}
		}
		buffer[len(buffer)-1-i] = out
	}
	b.text(string(buffer), column*8, row*8, color)
}

func setupClockwiseSelection(page informationPage, selection int) int {
	switch page {
	case rubberInformationPage, pressuresInformationPage, shocksInformationPage, cambersInformationPage:
		for i, row := range setupTireRows {
			if row == selection {
				return i
			}
		}
	}
	return selection
}

func (b *informationBar) setupFrontTires(a *setupArtwork, side int, left, right int) {
	x := 104 + side*48
	b.setupBitmap(a.Pixels[128*23:128*23+48*17], 48, 17, x, 4)
	for i, compound := range []int{right, left} {
		if compound >= 2 {
			continue
		}
		color := byte(7)
		if compound == 1 {
			color = 8
		}
		offset := 0x32d + side*6 + i*5
		b.setupMask(offset, 126, color)
		for row := 1; row <= 11; row++ {
			b.setupMask(offset-row*40, 255, color)
		}
		b.setupMask(offset-12*40, 126, color)
	}
}

func (b *informationBar) setupRearTires(a *setupArtwork, side, left, right, leftHeight, rightHeight int) {
	x := 104 + side*48
	b.setupBitmap(a.Pixels[128*23+48*17:128*23+2*48*17], 48, 17, x, 4)
	patterns := [3]uint32{0xffe0ffe0, 0xffe0ffe0, 0xd560aaa0}
	for tire, compound := range []int{left, right} {
		color := byte(7)
		if compound != 0 {
			color = 8
		}
		offset, mask, height := 0x32e+side*6, uint32(0x7fc07fc0), leftHeight
		if tire == 1 {
			offset += 4
			mask = 0x03fe03fe
			height = rightHeight
		}
		pattern := patterns[compound]
		if tire == 1 {
			pattern = (pattern&0xffff)>>5 | ((pattern>>16)>>5)<<16
		}
		for column := 0; column < 3; column++ {
			b.setupMask(offset-column, byte(mask), color)
			for row := 1; row <= height; row++ {
				bits := byte(pattern)
				if row%2 == 0 {
					bits = byte(pattern >> 16)
				}
				b.setupMask(offset-column-row*40, bits, color)
			}
			b.setupMask(offset-column-(height+1)*40, byte(mask), color)
			if column == 0 {
				pattern = (pattern >> 8) & 0x00ff00ff
				mask = (mask >> 8) & 0x00ff00ff
			} else {
				pattern >>= 8
				mask >>= 8
			}
		}
	}
}

func (b *informationBar) setupTireGraphic(a *setupArtwork, which int, s persistentSetup) {
	if which == 0 || which == 3 {
		side := 1
		if which == 3 {
			side = 0
		}
		b.setupFrontTires(a, side, s.Rubber[1], s.Rubber[0])
		return
	}
	side := 0
	if which == 2 {
		side = 1
	}
	b.setupRearTires(a, side, s.Rubber[3], s.Rubber[2], 12, 12)
}

func (b *informationBar) setupWings(a *setupArtwork, s persistentSetup, selected int) {
	b.setupBitmap(a.Pixels[:128*23], 128, 23, 152, 0)
	front, rear := s.Wings[0], s.Wings[1]
	for wing, level := range []int{front, rear} {
		color := byte(15)
		if wing != selected {
			color = 7
		}
		offset, shaft := 0x63+level*40, 0x3a
		if wing == 1 {
			offset = 0x72 + level*40
			shaft = 0x4b
		}
		first, middle := byte(30), byte(31)
		if wing == 1 {
			first, middle = 120, 248
		}
		b.setupMask(offset, first, color)
		b.setupMask(offset+80, first, color)
		b.setupMask(offset+40, middle, color)
		end := offset - 1
		if wing == 1 {
			end = offset + 1
		}
		for shaft < end {
			b.setupMask(shaft, 56, color)
			shaft += 40
		}
		for i, mask := range []byte{254, 124, 56, 16} {
			b.setupMask(end+i*40, mask, color)
		}
	}
	tilt := max(-10, min(10, ((front-rear)*40)>>5))
	for _, row := range []int{11, 12} {
		b.setupLine(168, row+tilt, 264, row-tilt, 15)
	}
	length := (front*383 + rear*795) >> 9
	b.setupLine(216, 2, 220+length, 2, 14)
	b.setupLine(216, 3, 221+length, 3, 14)
	b.setupLine(216, 4, 220+length, 4, 14)
	b.setupLine(218+length, 6, 218+length, 0, 14)
	b.setupLine(219+length, 1, 219+length, 5, 14)
}

func (b *informationBar) setupShock(a *setupArtwork, detent int) {
	b.setupBitmap(a.Pixels[128*23+2*48*17:], 24, 21, 288, 1)
	offset := 0x24 + (detent+1)*80
	b.setupMask(offset, 30, 15)
	b.setupMask(offset+80, 30, 15)
	b.setupMask(offset+40, 31, 15)
}

func (b *informationBar) setupCamber(which, detent int) {
	x, offset, side := 148, 0x11, which == 1 || which == 3
	if !side {
		x = 156
		offset = 0x12
		detent = -detent
	}
	first, last, bottom, tail := byte(3), byte(228), byte(2), byte(36)
	if !side {
		first, last, bottom, tail = 19, 224, 18, 32
	}
	b.setupMask(offset, first, 15)
	b.setupMask(offset+1, 255, 15)
	b.setupMask(offset+2, last, 15)
	b.setupMask(offset+40, 58, 15)
	b.setupMask(offset+41, 170, 15)
	b.setupMask(offset+42, 174, 15)
	b.setupMask(offset+80, bottom, 15)
	b.setupMask(offset+82, tail, 15)
	b.setupLine(x, 20, x+(detent>>1), 5, 15)
}

func (b *informationBar) setupInfo(a *setupArtwork, v setupDisplayValues) {
	clear(b.pixels[:])
	labels := [3]string{"PRACTICE", "QUALIFY ", "RACE    "}
	colors := [3]byte{5, 14, 12}
	b.text(labels[v.Mode], 24, 16, colors[v.Mode])
	editable := v.Mode == practiceMode || v.AtPit
	if v.Page >= shocksInformationPage {
		editable = v.Mode == practiceMode || v.Mode == qualifyMode && v.AtPit
	}
	pen := byte(8)
	if editable {
		pen = 15
	}
	selected := setupClockwiseSelection(v.Page, v.Selection)
	names := [4]string{"rightFront", "rightRear", "leftRear", "leftFront"}
	text := func(name string) { b.setupText(a, name, pen) }
	number := func(value uint16, pattern string, x, y int) { b.setupNumber(value, pattern, x, y, 15) }
	temperature := func(value uint16, x, y int) { number((value>>5)-459, "#**", x, y) }
	wheel, row := 0, 0
	if v.Page == rubberInformationPage || v.Page == pressuresInformationPage || v.Page == shocksInformationPage || v.Page == cambersInformationPage {
		wheel, row = setupTireWheels[selected], setupTireRows[selected]
	}
	switch v.Page {
	case fuelInformationPage:
		text("fuel")
		number(uint16(v.Setup.Fuel), "#*", 38, 0)
		fuel := (uint16(v.State.Engine.Fuel>>16) + 16) >> 5
		number(fuel, "#*", 25, 0)
		mileage := v.State.Engine.FuelUsed
		if mileage != 0 {
			mileage = uint16(uint32(max(int16(0), v.RearGaugeSpeed)) * 43 / uint32(mileage))
		}
		number(mileage, "##.#*", 22, 1)
		number(uint16(uint32(mileage)*uint32(fuel)/250), "#*", 25, 2)
	case wingsInformationPage:
		text("wings")
		if selected == 0 {
			b.setupText(a, "frontWing", 15)
		} else {
			b.setupText(a, "rearWing", 15)
		}
		b.setupWings(a, v.Setup, selected)
	case rubberInformationPage:
		text("rubber")
		for i, color := range []byte{7, 8, 8} {
			for y := 0; y < 7; y++ {
				mask := byte(255)
				if i == 2 {
					mask = 170
					if y%2 != 0 {
						mask = 85
					}
				}
				for x := 0; x < 3; x++ {
					b.setupMask(i*320+y*40+37+x, mask, color)
				}
			}
		}
		text(names[selected])
		b.setupTireGraphic(a, selected, v.Setup)
		text([3]string{"soft", "medium", "hard"}[v.Setup.Rubber[row]])
		tire := v.State.Tires[wheel]
		average := uint16(uint16((tire.Bands[0]+tire.Bands[2])>>1)+tire.Bands[1]) >> 1
		x := 13
		if selected == 1 || selected == 3 {
			x = 20
		}
		temperature(average, x, 2)
	case staggerInformationPage:
		text("stagger")
		for y, value := range []int16{int16(v.Setup.Stagger * 100), int16(v.Setup.Stagger * 100)} {
			pattern := "##.#+"
			magnitude := value
			if magnitude < 0 {
				magnitude = -magnitude
				pattern = "##.#-"
			}
			number(uint16(magnitude/10), pattern, 28, y+1)
		}
		height := abs(v.Setup.Stagger*100) / 100
		left, right := 12, 12
		if v.Setup.Stagger < 0 {
			left += (height + 1) / 2
			right -= height / 2
		} else {
			left -= (height + 1) / 2
			right += height / 2
		}
		b.setupRearTires(a, 0, v.Setup.Rubber[3], v.Setup.Rubber[2], left, right)
	case pressuresInformationPage, cambersInformationPage:
		prefix := "pressureTire"
		if v.Page == cambersInformationPage {
			prefix = "camberTire"
		}
		if v.Page == pressuresInformationPage {
			text("pressures")
		} else {
			text("cambers")
		}
		text(prefix + fmt.Sprint(selected))
		b.setupTireGraphic(a, selected, v.Setup)
		x := 13
		if selected == 1 || selected == 3 {
			x = 20
		}
		if v.Page == pressuresInformationPage {
			pressure := v.State.Tires[wheel].Pressure
			if v.State.Engine.Failures&tireFailureFlags[wheel] != 0 {
				pressure = 0
			}
			number(pressure/680, "#*", x, 2)
		} else {
			value := v.Setup.Cambers[row]
			pattern := "##.#+"
			if value < 0 {
				value = -value
				pattern = "##.#-"
			}
			number(uint16(value*25), pattern, x, 2)
			b.setupCamber(selected, v.Setup.Cambers[row])
		}
		for band, value := range v.State.Tires[wheel].Bands {
			temperature(value, 35, band)
		}
	case shocksInformationPage:
		text("shocks")
		text("shockTire" + fmt.Sprint(selected))
		b.setupTireGraphic(a, selected, v.Setup)
		b.setupShock(a, v.Setup.Shocks[row])
	case gearsInformationPage:
		text("gears")
		for i, value := range v.Setup.Gears {
			number(uint16(value), "##.#*", 14+i*6, 1)
		}
	}
}

func (g *Game) drawOriginalSetup(screen *ebiten.Image) {
	b := &g.informationBar
	if !b.valid || b.page != g.informationPage || b.selection != g.informationSelection || b.mode != g.mode || b.tick != g.tick {
		b.setupInfo(&g.setupArtwork, setupDisplayValues{Mode: g.mode, Page: g.informationPage, Selection: g.informationSelection, Setup: setupToPersistent(g.setup), Config: g.physicalConfig(), State: g.cars[0].physics, RearGaugeSpeed: g.cars[0].dashboardSpeed, AtPit: g.noticeInPit})
		g.uploadInformationBar()
		b.valid, b.page, b.selection, b.mode, b.tick = true, g.informationPage, g.informationSelection, g.mode, g.tick
	}
	screen.DrawImage(g.informationImage, nil)
	g.drawRaceFlags(screen)
}
