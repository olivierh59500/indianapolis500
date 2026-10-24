package vehicle

import "math"

type DriverState struct{ Throttle, Brake, CruiseTarget uint16 }
type DriverContext struct {
	Steering                  int16
	Damping, Increment, Speed uint16
	Failures                  byte
}
type DriverIntent struct {
	Steering                                                  float64
	Accelerate, Brake, Reverse, CruiseIncrease, DamageEnabled bool
}

type Driver struct{ State DriverState }

func subtractPedal(value, amount uint16) uint16 {
	if value < amount {
		return 0
	}
	return value - amount
}
func increasePedal(value uint16) uint16 {
	next := value + 0x4000
	if next < value {
		return 0xe000
	}
	return next
}

// Step translates the original keyboard steering, pedal and cruise ramps.
// An analog steering intent scales the same per-tick increment, preserving
// the original digital behavior at either end of its range.
func (driver *Driver) Step(context DriverContext, input DriverIntent) Controls {
	s := &driver.State
	intent := max(-1.0, min(1.0, input.Steering))
	steering := context.Steering + int16(math.Round(-intent*float64(context.Increment)))
	amount := unsignedAbs(steering)
	amount = subtractPedal(amount, context.Damping)
	amount = min(uint16(3200), amount)
	if steering < 0 {
		steering = -int16(amount)
	} else {
		steering = int16(amount)
	}

	if input.CruiseIncrease {
		if s.CruiseTarget == 0 {
			s.CruiseTarget = context.Speed
		}
		previous := s.CruiseTarget
		s.CruiseTarget += 40
		if int16(previous) >= 0 && int16(s.CruiseTarget) < 0 {
			s.CruiseTarget = 32767
		}
	}
	if s.CruiseTarget != 0 {
		s.Brake = 0
		if int16(s.CruiseTarget-context.Speed) >= 0 {
			s.Throttle = min(uint16(0xc400), s.Throttle+0x800)
		}
		s.Throttle = subtractPedal(s.Throttle, 0x400)
	}
	// The source brake branch skips the accelerator branch entirely. Each
	// pedal also clears the other immediately, rather than waiting for decay.
	if input.Brake {
		s.CruiseTarget = 0
		s.Throttle = 0
		s.Brake = increasePedal(s.Brake)
		s.Brake = subtractPedal(s.Brake, 0x2000)
	} else {
		s.Brake = subtractPedal(s.Brake, 0x2000)
		if input.Accelerate {
			s.CruiseTarget = 0
			s.Brake = 0
			s.Throttle = increasePedal(s.Throttle)
		}
		if s.CruiseTarget == 0 {
			s.Throttle = subtractPedal(s.Throttle, 0x2000)
		}
	}
	reverse := s.Brake == 0 && s.Throttle == 0 && context.Failures&FuelEmpty == 0 && input.Reverse
	if reverse {
		s.CruiseTarget = 0
	}
	return Controls{Steering: steering, Throttle: s.Throttle, Brake: s.Brake, Reverse: reverse, DamageEnabled: input.DamageEnabled}
}
