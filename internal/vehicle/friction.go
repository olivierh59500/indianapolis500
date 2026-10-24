package vehicle

import "math"

// WheelBlend gives the unsigned Q16 direction of the combined tire demand.
// The source saturates an axial component at 65535, including its zero vector.
type WheelBlend struct{ Lateral, Longitudinal uint16 }

func unsignedAbs(value int16) uint16 {
	if value < 0 {
		return uint16(-value)
	}
	return uint16(value)
}

func vectorLength(x, y uint16) uint16 {
	// The integer inputs have at most 32 significant squared bits. A binary64
	// square root retains enough precision for the original integer floor.
	return uint16(math.Sqrt(float64(uint32(x)*uint32(x) + uint32(y)*uint32(y))))
}

func normalizedComponent(value, length uint16) uint16 {
	if value >= length {
		return 65535
	}
	return uint16((uint32(value) << 16) / uint32(length))
}

func demandBlend(lateral, longitudinal uint16) WheelBlend {
	length := vectorLength(lateral, longitudinal)
	return WheelBlend{normalizedComponent(lateral, length), normalizedComponent(longitudinal, length)}
}

// DrivenBlend translates $5242. Brake and throttle demands combine with the
// source bitwise rule. Skid volume changes only inside its wheel-speed band.
func DrivenBlend(slip, speed int16, rearBrake, throttle uint16, skid byte) (WheelBlend, byte) {
	lateral, wheelSpeed := unsignedAbs(slip), unsignedAbs(speed)
	longitudinal := (rearBrake | (throttle >> 1)) >> 6
	combined := rearBrake | throttle
	if wheelSpeed >= 300 && wheelSpeed < 4000 && combined > 16384 {
		skid = byte((combined - 16385) >> 8)
		if longitudinal < wheelSpeed>>3 {
			skid = byte(((uint32(longitudinal)<<16)/uint32(wheelSpeed>>3))>>8) >> 1
		}
	}
	return demandBlend(lateral, longitudinal), skid
}

// FrontBlend translates the unpowered-wheel demand normalizer at $52a8.
func FrontBlend(slip int16, brake uint16) WheelBlend {
	return demandBlend(unsignedAbs(slip), brake>>6)
}

// ResolveWheelForces translates $52e6: apply the original friction circle,
// multiply by normal load and restore the two force directions. Grip is half
// the Q15 coefficient, as stored by the tire model before this calculation.
func ResolveWheelForces(lateral, longitudinal, load, grip uint16, blend WheelBlend, lateralNegative, longitudinalNegative bool) (int16, int16) {
	if grip < vectorLength(lateral>>1, longitudinal>>1) {
		lateral = min(lateral, highProduct(blend.Lateral, grip*2))
		longitudinal = min(longitudinal, highProduct(blend.Longitudinal, grip*2))
	}
	lat := uint16((uint32(lateral) * uint32(load) * 2) >> 16)
	long := uint16((uint32(longitudinal) * uint32(load) * 2) >> 16)
	if lateralNegative {
		lat = 0 - lat
	}
	if longitudinalNegative {
		long = 0 - long
	}
	return int16(lat), int16(long)
}

// LoadGrip translates $6fc2, the load-sensitive tire-grip factor. The caller
// supplies the previous wheel load, before the next suspension-load update.
func LoadGrip(load uint16) uint16 {
	shifted := load + 700
	if shifted >= 11585 {
		return 0
	}
	return ^uint16((uint32(shifted) * uint32(shifted) * 16) >> 16)
}
