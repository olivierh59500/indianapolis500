package scene

import (
	"encoding/binary"
	"fmt"
)

// TrackPoint is the original track-relative placement, including banking and
// the approximate tangent heading used by the replay cameras.
type TrackPoint struct {
	Point
	Heading  int16
	Progress uint16
	Lateral  int16
}

func (w *World) LoadTrack(cosine, bank []byte) error {
	if len(cosine) != len(w.trackCosine)*2 || len(bank) != len(w.trackBank)*2 {
		return fmt.Errorf("recovered track placement tables have invalid dimensions")
	}
	for i := range w.trackCosine {
		w.trackCosine[i] = binary.BigEndian.Uint16(cosine[i*2:])
	}
	for i := range w.trackBank {
		w.trackBank[i] = binary.BigEndian.Uint16(bank[i*2:])
	}
	return nil
}

// TrackPlacement translates $291c6-$2939e. Progress increases in the driving
// direction; the source routine's lap-distance argument runs the other way.
func (w *World) TrackPlacement(progress uint16, lateralQ4 int16) TrackPoint {
	p := int(progress) % 39600
	lat := int(lateralQ4 >> 4)
	result := TrackPoint{Point: Point{Z: 15}, Progress: uint16(p), Lateral: int16(lat)}
	quarter := func(t int) (int, int, int16) {
		radius := uint32(lat + 0x1286)
		sine := int(uint32(w.trackCosine[3960-t]) * radius >> 15)
		cosine := int(uint32(w.trackCosine[t]) * radius >> 15)
		factor := uint32(0x28d7)
		if t < 0x294 || t > 0xce4 {
			near := min(t, 3960-t)
			factor = uint32(w.trackBank[((near>>3)&^1)/2])
		}
		if lat > 252 {
			result.Z += int16(uint32(lat-252) * factor >> 16)
		}
		return sine, cosine, int16((t + 136) * 4)
	}
	switch {
	case p <= 0x26ac:
		result.X, result.Y = int16(p*2-0x26ac), int16(0x1a42+lat)
	case p < 0x3624:
		s, c, h := quarter(p - 0x26ac)
		result.X, result.Y, result.Heading = int16(0x26ac+s), int16(0x7bc+c), h
	case p <= 0x3de0:
		result.X, result.Y, result.Heading = int16(0x3932+lat), int16(0x7bc-(p-0x3624)*2), 0x4000
	case p < 0x4d58:
		s, c, h := quarter(p - 0x3de0)
		result.X, result.Y, result.Heading = int16(0x26ac+c), int16(-0x7bc-s), int16(0x4000+int(h))
	case p <= 0x7404:
		result.X, result.Y, result.Heading = int16(0x26ac-(p-0x4d58)*2), int16(-0x1a42-lat), -32768
	case p < 0x837c:
		s, c, h := quarter(p - 0x7404)
		result.X, result.Y, result.Heading = int16(-0x26ac-s), int16(-0x7bc-c), int16(-32768+int(h))
	case p <= 0x8b38:
		result.X, result.Y, result.Heading = int16(-0x3932-lat), int16(-0x7bc+(p-0x837c)*2), -0x4000
	default:
		s, c, h := quarter(p - 0x8b38)
		result.X, result.Y, result.Heading = int16(-0x26ac-c), int16(0x7bc+s), int16(-0x4000+int(h))
	}
	return result
}

func (w *World) LoadTrackLimits(inner, outer []byte) error {
	if len(inner) != len(w.innerLimits)*2 || len(outer) != len(w.outerLimits)*2 {
		return fmt.Errorf("recovered track limits have invalid dimensions")
	}
	for i := range w.innerLimits {
		w.innerLimits[i] = int16(binary.BigEndian.Uint16(inner[i*2:]))
		w.outerLimits[i] = int16(binary.BigEndian.Uint16(outer[i*2:]))
	}
	return nil
}

// TrackLimits returns the original radial wall limits in world units. The
// preferred driving lanes are separate data and must never serve as walls.
func (w *World) TrackLimits(progress uint16) (inner, outer int16) {
	p := int(progress) % 39600
	switch {
	case p <= 9900 || p >= 19800 && p <= 29700:
		return 20, 580
	case p >= 13860 && p <= 15840 || p >= 33660 && p <= 35640:
		return 20, 640
	}
	t := p - 9900
	switch {
	case p > 35640:
		t = p - 35640
	case p > 29700:
		t = p - 29700
	case p > 15840:
		t = p - 15840
	}
	// Opposite sides of the oval mirror the same quarter-turn wall profile.
	if p >= 15840 && p < 19800 || p >= 35640 {
		t = 3960 - t
	}
	i := min(len(w.innerLimits)-1, max(0, t>>3))
	return w.innerLimits[i], w.outerLimits[i]
}

// TrackLocation is the inverse world placement and the original wall limits,
// measured in world units before the physics lateral Q4 conversion.
type TrackLocation struct {
	Progress              uint16
	Lateral, Inner, Outer int16
}

// LocateTrack translates the original world-to-track calculation at
// $3d64-$3f6e. Replaying TrackPlacement alone loses its integer-rounding rules.
func (w *World) LocateTrack(point Vector2) TrackLocation {
	x, y := point.X, point.Y
	if x >= -9900 && x <= 9900 {
		if y < 0 {
			return TrackLocation{uint16(24750 - (x >> 1)), -6722 - y, 20, 580}
		}
		return TrackLocation{uint16((x >> 1) + 4950), y - 6722, 20, 580}
	}
	if y >= -1980 && y <= 1980 {
		if x < 0 {
			return TrackLocation{uint16(int(y>>1) + 34650), -14642 - x, 20, 640}
		}
		return TrackLocation{uint16(14850 - (y >> 1)), x - 14642, 20, 640}
	}
	var sine, cosine int16
	base := 0
	switch {
	case x > 0 && y > 0:
		sine, cosine, base = x-9900, y-1980, 9900
	case x > 0 && y < 0:
		sine, cosine, base = -y-1980, x-9900, 15840
	case x < 0 && y < 0:
		sine, cosine, base = -x-9900, -y-1980, 29700
	default:
		sine, cosine, base = y-1980, -x-9900, 35640
	}
	radius := int16(integerSquareRoot(uint32(int32(sine)*int32(sine) + int32(cosine)*int32(cosine))))
	angle := uint16(w.Direction(cosine, -sine))
	quarter := int(uint32(angle) * 15840 >> 16)
	progress := min(39599, base+quarter)
	profile := quarter
	if (x ^ y) < 0 {
		profile = 3960 - profile
	}
	index := min(len(w.innerLimits)-1, max(0, profile>>3))
	return TrackLocation{uint16(progress), radius - 4742, w.innerLimits[index], w.outerLimits[index]}
}
