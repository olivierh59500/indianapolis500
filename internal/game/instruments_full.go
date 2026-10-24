package game

import "encoding/binary"

type instrumentInput struct {
	RPM, Temperature, Pressure uint16
	Speed                      int16
	Gear, Fuel                 uint16
	Front, Rear, Boost         uint16
	Warning                    bool
}

var gearInstrumentArt = [...]int{0x392, 0x4ea, 0x642, 0x79a}
var frontInstrumentArt = [...]int{0x8f2, 0x9ea, 0xae2, 0xbda, 0xcd2, 0xdca, 0xec2, 0xfba, 0x10b2}
var rearInstrumentArt = [...]int{0x11aa, 0x12a2, 0x139a, 0x1492, 0x158a, 0x1682, 0x177a, 0x1872, 0x196a}

// blitInstrument copies the original four-plane widget artwork into ordinary
// indexed pixels. Each variant keeps its recovered plane stride and screen
// byte position; it does not execute the original graphics routines.
func (d *dashboard) blitInstrument(team, source, destination, width, height, stride int) {
	x0, y0 := destination%40*8, destination/40
	for y := range height {
		for x := range width * 8 {
			offset, bit := source+y*width+x/8, byte(1<<uint(7-x%8))
			index := byte(0)
			for plane := range 4 {
				if d.art[team][offset+plane*stride]&bit != 0 {
					index |= 1 << plane
				}
			}
			d.pixels[(y0+y)*320+x0+x] = index
		}
	}
}

func (d *dashboard) whiteDigit(value, destination int) {
	for y := range 5 {
		pattern := d.displays[value*5+y]
		for x := range 8 {
			index := byte(0)
			if pattern&(1<<uint(7-x)) != 0 {
				index = 15
			}
			d.pixels[(destination/40+y)*320+destination%40*8+x] = index
		}
	}
}

func (d *dashboard) digitalNumber(value, destination, digits, font, height, spacing int) {
	for range digits {
		digit := value % 10
		value /= 10
		for y := range height {
			pattern := d.displays[font+digit*spacing+y]
			for x := range 8 {
				index := byte(7)
				if pattern&(1<<uint(7-x)) != 0 {
					index = 8
				}
				d.pixels[(destination/40+y)*320+destination%40*8+x] = index
			}
		}
		destination--
	}
}

func (d *dashboard) boostDigit(value int) {
	destination := 0x1cf1
	for y := range 4 {
		pattern := d.displays[0xaf+(value-1)*4+y]
		for x := range 8 {
			if pattern&(1<<uint(7-x)) != 0 {
				d.pixels[(destination/40+y)*320+destination%40*8+x] |= 11
			}
		}
	}
}

func (d *dashboard) rpmBar(value int) {
	for offset, x := 0xd4, 106; offset+4 <= len(d.displays); offset, x = offset+4, x+1 {
		threshold := int(binary.BigEndian.Uint16(d.displays[offset:]))
		if value < threshold {
			break
		}
		y0, y1 := int(d.displays[offset+2]), int(d.displays[offset+3])
		if y0 == 255 {
			continue
		}
		for y := min(y0, y1) + 24; y <= max(y0, y1)+24; y++ {
			d.pixels[y*320+x] = 8
		}
	}
}

// full renders all eleven original instruments. Flash counters advance only
// while their corresponding fuel or oil condition is active.
func (d *dashboard) full(team int, input instrumentInput) {
	d.render(team, input, true)
}

func (d *dashboard) render(team int, input instrumentInput, advance bool) {
	clear(d.pixels[:])
	copy(d.pixels[148*320:], d.panels[team][:])
	v := &d.values
	v[0], v[1] = input.RPM*2, uint16(uint32(max(int16(0), input.Speed))*873>>16)
	v[2], v[3] = input.Temperature>>5-459, uint16(uint32(input.Pressure)*53>>16)
	v[4], v[5] = input.Gear, (input.Fuel+16)>>5
	v[7], v[8], v[9] = input.Front, input.Rear, (input.Boost-0xbfff)>>11+1
	v[6], v[10] = 0, 0
	if input.Fuel <= 160 {
		if advance {
			d.fuelFlash--
			if d.fuelFlash == 0 {
				d.fuelFlash = 8
			}
		}
		if d.fuelFlash <= 4 {
			v[6] = 1
		}
	}
	if input.Warning {
		if advance {
			d.oilFlash--
			if d.oilFlash == 0 {
				d.oilFlash = 10
			}
		}
		if d.oilFlash <= 5 {
			v[10] = 1
		}
	}
	if team == 0 {
		d.needle(193, 146, 0x12314+(max(1000, min(11039, int(v[0])))>>5)*4)
		d.needle(127, 144, 0x12878+max(0, min(234, int(v[1]))-55)*8)
		d.needle(249, 147, 0x12210+min(64, int(uint16(v[2]-174)))*4)
		d.needle(68, 154, 0x12e18+(int(v[3])-10)*16)
		d.needle(249, 153, 0x1216c+int(v[5])*4)
		if v[6] != 0 {
			d.blitInstrument(team, 0x1a62, 0x1da4, 2, 11, 29)
		}
		if v[10] != 0 {
			d.blitInstrument(team, 0x1d10, 0x173c, 2, 11, 29)
		}
		d.whiteDigit(int(v[9])-1, 0x1d68)
	} else if team == 1 {
		d.needle(159, 140, 0x12314+(max(0, min(13039, int(v[0]))-2000)>>5)*4)
		d.needle(219, 148, 0x12ffc+max(0, min(32, int(v[2])-200))*8)
		d.needle(100, 148, 0x12e18+(int(v[3])-10)*16)
		if v[6] != 0 {
			d.blitInstrument(team, 0x1ad8, 0x1c3f, 2, 16, 42)
		}
		if v[10] != 0 {
			d.blitInstrument(team, 0x1d86, 0x1c27, 2, 16, 42)
		}
		d.whiteDigit(int(v[9])-1, 0x1c7c)
	} else {
		d.digitalNumber(int(v[0]), 0x1bea, 5, 0x5f, 7, 8)
		d.rpmBar(int(v[0]))
		d.digitalNumber(int(v[1]), 0x1be4, 3, 0x5f, 7, 8)
		d.digitalNumber(int(v[2]), 0x1da0, 3, 0x2d, 5, 5)
		d.digitalNumber(int(v[3])+10, 0x1d96, 2, 0x2d, 5, 5)
		d.digitalNumber(int(v[5]), 0x1d9b, 2, 0x2d, 5, 5)
		if v[6] != 0 {
			d.blitInstrument(team, 0x1b80, 0x19bd, 3, 28, 100)
		}
		if v[10] != 0 {
			d.blitInstrument(team, 0x1e2e, 0x1930, 3, 13, 47)
		}
		d.boostDigit(int(v[9]))
	}
	d.blitInstrument(team, gearInstrumentArt[v[4]], 0x1ba4, 3, 24, 86)
	d.blitInstrument(team, frontInstrumentArt[(v[7]-1024)/896], 0x1b80, 2, 24, 62)
	d.blitInstrument(team, rearInstrumentArt[(v[8]-1024)/896], 0x1b82, 2, 24, 62)
}
