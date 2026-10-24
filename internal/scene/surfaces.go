package scene

import "fmt"

type surfaceFill uint8

const (
	solidFill surfaceFill = iota
	crowdFill
	checkerFill
)

// LoadCrowdPatterns loads three original bitmaps converted offline to indexed
// texels. Runtime sampling uses ordinary pixels rather than device registers.
func (w *World) LoadCrowdPatterns(texels []byte) error {
	if len(texels) != 3*2880 {
		return fmt.Errorf("invalid recovered crowd bitmap dimensions")
	}
	for _, color := range texels {
		if color > 15 {
			return fmt.Errorf("invalid recovered crowd color")
		}
	}
	w.crowd = append([]byte(nil), texels...)
	return nil
}

func (r *Renderer) surfaceSpan(left, right, y int, color uint8) {
	if r.fill == solidFill {
		for x := left; x <= right; x++ {
			r.Pixels[y*Width+x] = color
		}
		return
	}
	if r.fill == checkerFill {
		for x := left; x <= right; x++ {
			if (x+y)&1 != 0 {
				r.Pixels[y*Width+x] = color
			}
		}
		return
	}
	bank := 0
	if color == 0 {
		bank = 2
	} else if color == 8 {
		bank = 1
	}
	// The original three 360-byte strips share a row phase and cover all
	// possible spans. Decoding their four color planes once avoids any
	// bitplane emulation in the native renderer.
	start := bank*2880 + ((((y+24)*40)&255)+int(r.CrowdPhase&63))*8
	for x := left; x <= right; x++ {
		r.Pixels[y*Width+x] = r.world.crowd[start+x]
	}
}
