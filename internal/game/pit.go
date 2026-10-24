package game

import (
	"fmt"

	"github.com/hajimehoshi/ebiten/v2"
	gameassets "github.com/olivierh59500/indianapolis500/assets"
	"github.com/olivierh59500/indianapolis500/internal/vehicle"
)

const pitInformationPage informationPage = 255

// Pit choices go clockwise around the car, independently of setup storage.
var pitTireOrder = [4]int{3, 1, 0, 2}

type pitDisplay struct {
	art       []byte
	pixels    [184 * 96]byte
	rgba      [184 * 96 * 4]byte
	image     *ebiten.Image
	valid     bool
	selection int
	age       [4]uint16
	fuel      uint16
}

func (p *pitDisplay) load() error {
	data, err := gameassets.Original.ReadFile("original/pit-art.bin")
	if err != nil {
		return err
	}
	if len(data) != 4+184*96+216*23 || string(data[:4]) != "I5PT" {
		return fmt.Errorf("invalid recovered pit artwork")
	}
	p.art = data[4:]
	p.image = ebiten.NewImage(184, 96)
	return nil
}

func (p *pitDisplay) pixel(x, y int, color byte) {
	x -= 24
	y -= 30
	if x >= 0 && x < 184 && y >= 0 && y < 96 {
		p.pixels[y*184+x] = color
	}
}

func (p *pitDisplay) text(font []byte, text string, x, y int, color byte) {
	for i := range len(text) {
		glyph := (int(text[i]) - 0x1f) * 8
		if glyph < 0 || glyph+8 > len(font) {
			continue
		}
		for row := 0; row < 8; row++ {
			for column := 0; column < 8; column++ {
				if font[glyph+row]&(128>>uint(column)) != 0 {
					p.pixel(x+i*8+column, y+row, color)
				}
			}
		}
	}
}

func (p *pitDisplay) paint(font []byte, selection int, age [4]uint16, fuel uint16) {
	copy(p.pixels[:], p.art[:184*96])
	// The original draws its outline through viewport coordinates, with the
	// screen's twenty-four header rows added to the vertical positions.
	rectangles := [5][4]int{{74, 47, 88, 60}, {96, 19, 109, 36}, {93, 71, 109, 89}, {55, 71, 71, 89}, {55, 19, 67, 36}}
	r := rectangles[selection]
	for x := r[0]; x <= r[2]; x++ {
		p.pixel(x, r[1]+24, 15)
		p.pixel(x, r[3]+24, 15)
	}
	for y := r[1]; y <= r[3]; y++ {
		p.pixel(r[0], y+24, 15)
		p.pixel(r[2], y+24, 15)
	}
	// Tank fill is rounded to whole gallons, then painted in five-gallon rows.
	for row := uint16(0); row < ((fuel+16)>>5)/5; row++ {
		for x := 77; x <= 85; x++ {
			p.pixel(x, 81-int(row), 14)
		}
	}
	// The uppermost row retains the original two masks for the remainder.
	partial := [5][2]byte{{0, 0}, {6, 0}, {7, 128}, {7, 224}, {7, 248}}
	gallons := (fuel + 16) >> 5
	for part, mask := range partial[gallons%5] {
		for bit := 0; bit < 8; bit++ {
			if mask&(128>>uint(bit)) != 0 {
				p.pixel(72+part*8+bit, 81-int(gallons/5), 14)
			}
		}
	}
	tires := [4][3]int{{58, 98, 66}, {96, 98, 104}, {58, 46, 63}, {98, 46, 104}}
	for wheel, box := range tires {
		for row := uint16(0); row < age[wheel]; row++ {
			for x := box[0]; x <= box[2]; x++ {
				p.pixel(x, box[1]+int(row), 8)
			}
		}
	}
	labels := [5]string{"FUEL", "RF TIRE", "RR TIRE", "LR TIRE", "LF TIRE"}
	for i, label := range labels {
		color := byte(8)
		if i == selection {
			color = 15
		}
		p.text(font, label, 144, 40+i*8, color)
	}
	p.text(font, "SPACE:", 144, 88, 14)
	p.text(font, " CHOOSE", 144, 96, 8)
	p.text(font, "ENTER:", 144, 104, 14)
	p.text(font, " CHANGE", 144, 112, 8)
}

func pitCondition(s *vehicle.State) (age [4]uint16, fuel uint16) {
	for wheel := range age {
		life, divisor := uint16(700), uint16(58)
		if wheel >= 2 {
			life, divisor = 850, 77
		}
		age[wheel] = (life - uint16(s.Tires[wheel].Life>>16)) / divisor
	}
	return age, uint16(s.Engine.Fuel >> 16)
}

func (b *informationBar) pit(mode driveMode, help []byte) {
	clear(b.pixels[:])
	for y := 0; y < 23; y++ {
		copy(b.pixels[y*320+104:(y+1)*320], help[y*216:(y+1)*216])
	}
	b.text("IN PIT   ", 24, 0, 15)
	labels := [3]string{"PRACTICE", "QUALIFY ", "RACE    "}
	colors := [3]byte{5, 14, 12}
	b.text(labels[mode], 24, 16, colors[mode])
}

func (g *Game) drawOriginalPit(screen *ebiten.Image) {
	b, p := &g.informationBar, &g.pitDisplay
	if !b.valid || b.page != pitInformationPage || b.mode != g.mode {
		b.pit(g.mode, p.art[184*96:])
		g.uploadInformationBar()
		b.valid, b.page, b.mode = true, pitInformationPage, g.mode
	}
	screen.DrawImage(g.informationImage, nil)
	g.drawRaceFlags(screen)
	age, fuel := pitCondition(&g.cars[0].physics)
	if !p.valid || p.selection != g.pitSelection || p.age != age || p.fuel != fuel {
		p.paint(b.font, g.pitSelection, age, fuel)
		for i, c := range p.pixels {
			copy(p.rgba[i*4:i*4+4], g.worldColors[c][:])
		}
		p.image.WritePixels(p.rgba[:])
		p.valid, p.selection, p.age, p.fuel = true, g.pitSelection, age, fuel
	}
	op := &ebiten.DrawImageOptions{}
	op.GeoM.Translate(24, 30)
	screen.DrawImage(p.image, op)
}
