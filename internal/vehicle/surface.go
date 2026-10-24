package vehicle

import (
	"encoding/binary"
	"fmt"
)

// TrackSurface contains the recovered repeating off-road support-height wave.
// It is independent of preferred driving lanes and physical wall collisions.
type TrackSurface struct{ wave [256]int16 }
type SurfaceSample struct {
	Height  int16
	OffRoad bool
}

func NewTrackSurface(data []byte) (TrackSurface, error) {
	var surface TrackSurface
	if len(data) != len(surface.wave)*2 {
		return surface, fmt.Errorf("recovered track surface wave has invalid dimensions")
	}
	for i := range surface.wave {
		surface.wave[i] = int16(binary.BigEndian.Uint16(data[i*2:]))
	}
	return surface, nil
}

// grassAt translates the original $5122 infield and pit-island boundaries.
// Distances are source track progress; lateral distance is in world units.
func grassAt(progress, lateral uint16) bool {
	if lateral >= 296 || progress >= 37620 {
		return false
	}
	p, l := int(progress), int(lateral)
	switch {
	case p < 2820:
		return false
	case p < 3120:
		return l > 300-(p-2820)/4
	case p < 8940:
		return l > 225
	case p < 9240:
		return l > 225+(p-8940)/4
	case p < 9550:
		return false
	case p < 9900:
		return l < 133
	case p < 10395:
		return l < 133+(p-9900)*77/495
	case p < 19800:
		return l < 210
	case p < 21240:
		return l < 210+(p-19800)/16
	case p < 28260:
		return l < 296
	case p < 29700:
		return l < 210+(29700-p)/16
	case p < 36960:
		return l < 210
	default:
		return l < 210-(p-36960)*210/660
	}
}

// Sample reproduces the original off-road flag and height, including the
// alternating pit rumble and the distinct smooth main-track surface.
func (surface *TrackSurface) Sample(progress uint16, lateral int16) SurfaceSample {
	lateral = max(int16(0), lateral)
	if grassAt(progress, uint16(lateral)) {
		return SurfaceSample{surface.wave[byte(progress+uint16(lateral))], true}
	}
	if lateral > 239 {
		return SurfaceSample{}
	}
	height := int16(128)
	if byte(progress)&128 != 0 {
		height = -128
	}
	return SurfaceSample{Height: height}
}
