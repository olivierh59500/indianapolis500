package vehicle

// TireConfig contains one compound's recovered thermal and pressure settings.
// Temperatures and pressures keep the original fixed-point units.
type TireConfig struct {
	HeatGain, OptimumTemperature, ColdPressure uint16
	Cooling, CamberDivisor, PressureDivisor    int16
	ReferencePressure, GripReference           uint16
}

type TireState struct {
	Life                                         uint32
	HeatTick, PressureTick                       int16
	TemperatureGrip, RollingResistance, Pressure uint16
	PressureError                                int16
	Bands                                        [3]uint16
	Failed                                       bool
}

type TireInput struct {
	Energy               uint16
	Camber               int16
	Ambient, Speed, Load uint16
}

type TireResult struct {
	Grip      uint16
	Punctured bool
}

type Tire struct {
	Config TireConfig
	State  TireState
}

// Step translates the four identical tire updates into one common routine.
// Thermal diffusion, grip and pressure retain the source's ten/forty-tick
// phases; grip uses the pressure before that tick's pressure adjustment.
func (tire *Tire) Step(input TireInput) TireResult {
	s, config := &tire.State, &tire.Config
	var result TireResult
	grip := LoadGrip(input.Load)
	s.HeatTick--
	if s.HeatTick == 0 {
		s.HeatTick = 10
		heat := highProduct(input.Energy, config.HeatGain)
		camberHeat := int16(int32(int16(heat)) * int32(input.Camber) / int32(config.CamberDivisor))
		s.PressureError = int16(s.Pressure - config.ReferencePressure)
		pressureHeat := int16(int32(s.PressureError>>2) * int32(int16(heat)) / int32(config.PressureDivisor))
		s.Bands[0] += heat - uint16(camberHeat) - uint16(pressureHeat)
		s.Bands[1] += heat + uint16(pressureHeat*2)
		s.Bands[2] += heat + uint16(camberHeat) - uint16(pressureHeat)

		// Diffuse the third band first, then the first, as in each original
		// unrolled update. Reordering them changes the weighted mean.
		diffusion := int16(s.Bands[2]-s.Bands[1]) >> 7
		if diffusion == 0 {
			diffusion = 1
		}
		s.Bands[2] -= uint16(diffusion)
		s.Bands[1] += uint16(diffusion)
		diffusion = int16(s.Bands[0]-s.Bands[1]) >> 7
		if diffusion == 0 {
			diffusion = 1
		}
		s.Bands[0] -= uint16(diffusion)
		s.Bands[1] += uint16(diffusion)
		average := ((s.Bands[0] >> 1) + (s.Bands[2] >> 1) + s.Bands[1]) >> 1
		cooling := int16(int32(int16(input.Ambient-average)) * int32(config.Cooling) >> 16)
		cooling = int16((int32(cooling) * int32(int16((input.Speed>>2)+711)) * 16) >> 16)
		for i := range s.Bands {
			s.Bands[i] += uint16(cooling)
			s.Bands[i] = max(uint16(18848), s.Bands[i])
		}
		s.TemperatureGrip = 65535 - unsignedAbs(int16(average-config.OptimumTemperature))*2

		peak := max(s.Bands[0], s.Bands[1], s.Bands[2]) + 3200
		excess := uint16(0)
		if peak >= config.OptimumTemperature {
			excess = peak - config.OptimumTemperature
		}
		wear := uint32(highProduct(excess, excess)) * uint32(heat)
		previousLife := s.Life
		s.Life -= wear
		if wear > previousLife && !s.Failed {
			s.Failed = true
			s.Life = 0
			result.Punctured = true
		}
	}
	grip = highProduct(s.TemperatureGrip, grip)
	pressureDifference := int16(s.Pressure - config.GripReference)
	camberPenalty := int16((int32(int16(unsignedAbs(input.Camber)*2)) * int32(pressureDifference)) >> 8)
	penalty := uint16(65535 - unsignedAbs(s.PressureError) - uint16(camberPenalty))
	result.Grip = highProduct(penalty, grip)

	s.PressureTick--
	if s.PressureTick == 0 {
		s.PressureTick = 40
		s.RollingResistance = uint16(0x12c0000 / uint32(s.Pressure))
		target := uint16(uint32(config.ColdPressure) * uint32(s.Bands[1]) / uint32(input.Ambient))
		adjustment := (int16(target-s.Pressure) >> 4) | 1
		s.Pressure += uint16(adjustment)
	}
	return result
}
