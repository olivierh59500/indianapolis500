package game

import (
	"encoding/binary"
	"fmt"

	gameassets "github.com/olivierh59500/indianapolis500/assets"
)

type dashboard struct {
	instrumentRuntime
	vectors     [1994]int16
	pixels      [ScreenWidth * ScreenHeight]byte
	panels      [3][320 * 52]byte
	art         [3][]byte
	displays    []byte
	validPixels bool
}

type instrumentRuntime struct {
	fuelFlash, oilFlash byte
	values              [11]uint16
	input               instrumentInput
	tick, team          int
}

func newInstrumentRuntime() instrumentRuntime {
	return instrumentRuntime{fuelFlash: 8, oilFlash: 10, tick: -1}
}

func (d *dashboard) load() error {
	data, err := gameassets.Original.ReadFile("original/dashboard-vectors.bin")
	if err != nil {
		return err
	}
	if len(data) != len(d.vectors)*2 {
		return fmt.Errorf("recovered dashboard vectors have invalid dimensions")
	}
	for i := range d.vectors {
		d.vectors[i] = int16(binary.BigEndian.Uint16(data[i*2:]))
	}
	panels, err := gameassets.Original.ReadFile("original/dashboard-panels.bin")
	if err != nil {
		return err
	}
	art, err := gameassets.Original.ReadFile("original/dashboard-art.bin")
	if err != nil {
		return err
	}
	d.displays, err = gameassets.Original.ReadFile("original/dashboard-displays.bin")
	if err != nil {
		return err
	}
	if len(panels) != 3*8320 || len(art) != 3*7916 || len(d.displays) != 594 {
		return fmt.Errorf("invalid recovered team instrument artwork")
	}
	for team := range 3 {
		d.art[team] = art[team*7916 : (team+1)*7916]
		for y := range 52 {
			for x := range 320 {
				offset, bit := y*40+x/8, byte(1<<uint(7-x%8))
				for plane := range 4 {
					if panels[team*8320+plane*2080+offset]&bit != 0 {
						d.panels[team][y*320+x] |= 1 << plane
					}
				}
			}
		}
	}
	d.instrumentRuntime = newInstrumentRuntime()
	return nil
}

func (d *dashboard) line(x, y, tx, ty int) {
	dx, dy := abs(tx-x), -abs(ty-y)
	sx, sy := 1, 1
	if x > tx {
		sx = -1
	}
	if y > ty {
		sy = -1
	}
	err := dx + dy
	for {
		if x >= 0 && x < ScreenWidth && y >= 0 && y < ScreenHeight {
			d.pixels[y*ScreenWidth+x] = 15
		}
		if x == tx && y == ty {
			return
		}
		twice := err * 2
		if twice >= dy {
			err += dy
			x += sx
		}
		if twice <= dx {
			err += dx
			y += sy
		}
	}
}

func abs(n int) int {
	if n < 0 {
		return -n
	}
	return n
}

// needle translates $ef40: two parallel sides meet at a recovered integer
// endpoint. The original drawing origin is twenty-four screen rows lower.
func (d *dashboard) needle(x, y, address int) {
	i := (address - 0x1216c) / 2
	dx, dy := int(d.vectors[i]), int(d.vectors[i+1])
	y += 24
	ox, oy := 0, 1
	if abs(dx) < abs(dy) {
		ox, oy = 1, 0
	}
	d.line(x-ox, y, x+dx, y-dy)
	d.line(x+dx, y-dy, x, y)
	d.line(x+dx+ox, y-dy+oy, x+ox, y+oy)
}

func (d *dashboard) lola(rpm, temperature, pressure uint16) {
	clear(d.pixels[:])
	value := max(0, min(13039, int(rpm)*2)-2000) >> 5
	d.needle(159, 140, 0x12314+value*4)
	value = max(0, min(32, (int(temperature)>>5)-459-200))
	d.needle(219, 148, 0x12ffc+value*8)
	value = int(uint32(pressure)*53>>16) - 10
	d.needle(100, 148, 0x12e18+value*16)
}
