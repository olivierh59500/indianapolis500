// Package vehicle implements recovered vehicle dynamics in ordinary Go.
package vehicle

const (
	EngineFailed = byte(0x10)
	FuelEmpty    = byte(0x20)
)

// EngineConfig contains the recovered torque curve and setup coefficients.
// Ratios and fractional coefficients retain their original integer units.
type EngineConfig struct {
	Gears                               [4]uint16
	ShiftUp, ShiftDown                  uint16
	PowerLinear, PowerCubic             uint16
	Consumption, DamageRPM, DamageScale uint16
}

// EngineState stores simulation quantities, with no machine or device state.
// Fuel uses 32 units per gallon in Q16; engine life also retains Q16 precision.
// Wheel demands are Q15 traction coefficients capped by the available grip.
type EngineState struct {
	RPM, Gear, Ratio                               uint16
	ThrottleResponse, Pressure, PressureTarget     uint16
	Temperature, FuelUsed, Mass, Boost             uint16
	LeftDrive, RightDrive, LeftDemand, RightDemand uint16
	Fuel, Life                                     uint32
	FuelTick                                       int16
	Warning                                        bool
	Failures                                       byte
}

// EngineInput couples the engine to driven-wheel motion, normal load and grip.
// Throttle and boost are unsigned fractions; differential bias is signed.
type EngineInput struct {
	Speed, Ambient, Throttle, ThrottleLimit, BoostTarget uint16
	LeftLoad, RightLoad, LeftGrip, RightGrip             uint16
	RearSpeed, Differential                              int16
}

type EngineEvents struct{ FuelExhausted, EngineFailure bool }

type Engine struct {
	Config EngineConfig
	State  EngineState
}

func highProduct(a, b uint16) uint16 { return uint16(uint32(a) * uint32(b) >> 16) }

func limitedWheelDemand(drive, load, grip uint16) uint16 {
	if load <= drive>>1 {
		return grip
	}
	return min(grip, uint16((uint32(drive)<<15)/uint32(load)))
}

// Step translates the original $3236-$3584 engine block. Keeping truncation at
// the original word boundaries matters for shifts, torque, fuel and failures.
// Integration with chassis and tire forces belongs to the complete vehicle tick.
func (engine *Engine) Step(input EngineInput) EngineEvents {
	s, config := &engine.State, &engine.Config
	var events EngineEvents
	rearSpeed := uint16(max(int16(0), input.RearSpeed))
	targetRPM := highProduct(rearSpeed, s.Ratio)
	if targetRPM < 1000 && s.Failures&FuelEmpty == 0 {
		targetRPM = 1000
	}
	change := max(int16(-250), min(int16(250), int16(targetRPM-s.RPM)))
	s.RPM += uint16(change)
	if s.RPM >= config.ShiftUp && s.Gear != 3 {
		s.Gear++
		s.Ratio = config.Gears[s.Gear]
	}
	if s.RPM < config.ShiftDown && s.Gear != 0 {
		s.Gear--
		s.Ratio = config.Gears[s.Gear]
	}

	linear := uint32(uint16(uint32(config.PowerLinear)*uint32(s.RPM)>>8)) * uint32(s.RPM)
	cubic := uint32(uint16(uint32(highProduct(config.PowerCubic, s.RPM))*uint32(s.RPM)>>8)) * uint32(s.RPM)
	power := linear - cubic
	torque := uint16(6000)
	if uint16(power>>16) < rearSpeed {
		torque = min(torque, uint16(power/uint32(rearSpeed)))
	}

	if s.Failures&EngineFailed != 0 || s.Fuel == 0 {
		if s.Failures&EngineFailed != 0 {
			s.RPM = 0
		}
		s.ThrottleResponse, s.PressureTarget = 0, 0x4609
		s.Failures |= FuelEmpty
	} else {
		s.Boost += uint16(int16(input.BoostTarget-s.Boost) >> 6)
		throttle := min(input.Throttle, input.ThrottleLimit)
		s.ThrottleResponse = highProduct(s.Boost, throttle)
		s.PressureTarget = s.ThrottleResponse - (throttle >> 1) + 20000
	}
	s.Pressure += uint16(int16(s.PressureTarget-s.Pressure) >> 5)
	drive := highProduct(s.ThrottleResponse, torque)
	bias := int16(int32(int16(drive)) * int32(input.Differential) >> 16)
	s.LeftDrive = uint16(int16(drive+uint16(bias)) >> 1)
	s.RightDrive = uint16(int16(drive-uint16(bias)) >> 1)
	s.LeftDemand = limitedWheelDemand(s.LeftDrive, input.LeftLoad, input.LeftGrip)
	s.RightDemand = limitedWheelDemand(s.RightDrive, input.RightLoad, input.RightGrip)

	s.FuelTick--
	if s.FuelTick != 0 {
		return events
	}
	s.FuelTick = 15
	flow := highProduct(s.Pressure, s.RPM)
	flow = highProduct(flow, rearSpeed+2000)
	s.FuelUsed = uint16(uint32(flow) * uint32(config.Consumption) >> 8)
	burn := uint32(s.FuelUsed) * 17
	previousFuel := s.Fuel
	s.Fuel -= burn
	if burn > previousFuel || s.Fuel == 0 {
		s.Fuel = 0
		events.FuelExhausted = s.Failures&FuelEmpty == 0
	}
	s.Mass = uint16(s.Fuel>>16) + 7500
	cooling := highProduct(input.Speed+1742, s.Temperature-input.Ambient)
	cooling = uint16(uint32(cooling) * 707 >> 8)
	temperatureChange := int16(cooling-s.FuelUsed+32) >> 6
	if temperatureChange == 0 {
		temperatureChange = 1
	}
	s.Temperature -= uint16(temperatureChange)
	s.Temperature = max(uint16(0x5120), s.Temperature)

	excessRPM := uint16(max(int16(0), int16(s.RPM-config.DamageRPM)))
	excessHeat := uint16(max(int16(0), int16(s.Temperature+0xb040)))
	stress := uint16(uint32(excessRPM) * uint32(excessRPM) >> 8)
	stress = highProduct(stress, s.FuelUsed)
	stress = uint16(uint32(stress) * uint32(excessHeat) >> 8)
	wear := uint32(stress) * uint32(config.DamageScale)
	s.Warning = wear >= 65536 || uint16(wear) >= 3400
	previousLife := s.Life
	s.Life -= wear
	if wear > previousLife {
		s.Failures |= EngineFailed
		events.EngineFailure = true
		wear = 20000
	}
	remainingLife := uint16(s.Life >> 16)
	if remainingLife < 75 && uint16(wear) >= 850 || remainingLife < 30 {
		s.Warning = true
	}
	return events
}
