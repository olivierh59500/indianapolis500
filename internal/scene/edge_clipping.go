package scene

func sourceDistance(p viewPoint, plane int) int32 {
	switch plane {
	case 0:
		return p.d
	case 1:
		return p.d + p.h
	case 2:
		return p.d - p.h
	case 3:
		return p.d + p.v
	default:
		return p.d - p.v
	}
}

func sourceInside(p viewPoint, plane int) bool {
	distance := sourceDistance(p, plane)
	if plane == 2 || plane == 3 {
		return distance > 0
	}
	return distance >= 0
}

// Source clipping uses a truncated Q15 interpolation, then moves the right
// and bottom intersections one world unit inside their planes. Their cached
// screen positions still use the exact last visible column and row.
func sourceIntersection(a, b viewPoint, plane int) viewPoint {
	distance := uint16(sourceDistance(a, plane))
	denominator := uint16(sourceDistance(a, plane) - sourceDistance(b, plane))
	weight := uint16(32767)
	if denominator > distance {
		weight = uint16((uint32(distance)<<16)/uint32(denominator)) >> 1
	}
	interpolate := func(first, last int32) int32 {
		return int32(int16(first) + int16((int32(int16(last-first))*int32(weight))>>15))
	}
	p := viewPoint{d: interpolate(a.d, b.d), h: interpolate(a.h, b.h), v: interpolate(a.v, b.v), border: a.border & b.border}
	switch plane {
	case 0:
		p.d = 0
	case 1:
		p.h = -p.d
		p.border |= 1
	case 2:
		p.h = p.d
		p.d++
		p.border |= 2
	case 3:
		p.v = -p.d
		p.d++
		p.border |= 8
	case 4:
		p.v = p.d
		p.border |= 4
	}
	return p
}

func sourcePixel(p viewPoint) pixelPoint {
	x, y := 160, 62
	if p.d != 0 {
		x += int(p.h * 160 / p.d)
		y -= int(p.v * 62 / p.d)
	}
	if p.border&1 != 0 {
		x = 0
	}
	if p.border&2 != 0 {
		x = 319
	}
	if p.border&4 != 0 {
		y = 0
	}
	if p.border&8 != 0 {
		y = 123
	}
	return pixelPoint{max(0, min(319, x)), max(0, min(123, y))}
}

func (r *Renderer) drawSegment(a, b viewPoint, color uint8) {
	for plane := 0; plane < 5; plane++ {
		ia, ib := sourceInside(a, plane), sourceInside(b, plane)
		if !ia && !ib {
			return
		}
		if !ia {
			a = sourceIntersection(b, a, plane)
		} else if !ib {
			b = sourceIntersection(a, b, plane)
		}
	}
	r.line(sourcePixel(a), sourcePixel(b), color, false)
}
