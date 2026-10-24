package vehicle

// All wheel arrays use rear-left, rear-right, front-left, front-right order.
type SuspensionConfig struct {
	RollStiffness, PitchStiffness       uint16
	RollDistribution, PitchDistribution int16
	Springs                             [4]uint16
	Camber                              [4]int16
}

type SuspensionState struct {
	Surface     [4]int16
	Roll, Pitch [2]int16
}

type SuspensionInput struct {
	Progress                        uint16
	Lateral                         int16
	BaseLoads, Grip, Rolling        [4]uint16
	LateralForce, LongitudinalForce int16
}

type SuspensionOutput struct {
	Loads, BaseLoads, Grip, Rolling [4]uint16
	Progress                        [4]uint16
	Lateral, HeightDelta, Camber    [4]int16
	WheelBob                        [2]byte
	Pitch, Roll                     int16
	OffRoad                         byte
}

type Suspension struct {
	Config SuspensionConfig
	State  SuspensionState
}

func q15Product(a, b int16) int16 { return int16((int32(a) * int32(b) * 2) >> 16) }
func decayUnit(value int16) int16 {
	if value > 0 {
		return value - 1
	}
	if value < 0 {
		return value + 1
	}
	return 0
}
func positiveLoad(value uint16) uint16 {
	if int16(value) < 0 {
		return 0
	}
	return value
}

// Step translates $2824-$2dac: wheel supports, load transfer, chassis tilt
// and wheel presentation. Heading coefficients are original Q14 values.
func (suspension *Suspension) Step(input SuspensionInput, cosine, sine int16, surface *TrackSurface) SuspensionOutput {
	s, config := &suspension.State, &suspension.Config
	out := SuspensionOutput{BaseLoads: input.BaseLoads, Grip: input.Grip, Rolling: input.Rolling}
	c9, s9, c10, s10 := cosine>>9, sine>>9, cosine>>10, sine>>10
	out.Progress = [4]uint16{
		input.Progress + uint16(-(c9+s10)>>1),
		input.Progress + uint16((s10-c9)>>1),
		input.Progress + uint16((c9-s10)>>1),
		input.Progress + uint16((c9+s10)>>1),
	}
	out.Lateral = [4]int16{input.Lateral + s9 - c10, input.Lateral + s9 + c10, input.Lateral - s9 - c10, input.Lateral - s9 + c10}
	for i := range out.Progress {
		sample := surface.Sample(out.Progress[i], out.Lateral[i])
		if sample.OffRoad {
			out.OffRoad |= 1 << i
			out.Grip[i] = highProduct(out.Grip[i], 0x6666)
			out.Rolling[i] += 0x8f5
		}
		out.HeightDelta[i] = sample.Height - s.Surface[i]
		s.Surface[i] = sample.Height
		out.BaseLoads[i] += uint16(out.HeightDelta[i])
	}
	left := out.HeightDelta[0] + out.HeightDelta[2]
	right := out.HeightDelta[1] + out.HeightDelta[3]
	rear := out.HeightDelta[0] + out.HeightDelta[1]
	front := out.HeightDelta[2] + out.HeightDelta[3]

	rollError := input.LateralForce/5 - (s.Roll[0] + s.Roll[1])
	frontRoll := q15Product(rollError, config.RollDistribution)
	rearRoll := rollError - frontRoll
	rate := int16((config.RollStiffness >> 1) + 16383)
	s.Roll[0] += q15Product(frontRoll, rate)
	s.Roll[1] += q15Product(rearRoll, rate)
	forceRoll := int16(int32(s.Roll[0]+s.Roll[1]) * 2048 / int32(int16(config.RollStiffness)))
	out.Roll = forceRoll + int16(int32(right-left)*512/int32(int16(uint16(0x8fff)-config.RollStiffness)))

	pitchError := (input.LongitudinalForce >> 3) - (s.Pitch[0] + s.Pitch[1])
	leftPitch := q15Product(pitchError, config.PitchDistribution)
	rightPitch := pitchError - leftPitch
	rate = int16((config.PitchStiffness >> 1) + 16383)
	s.Pitch[0] += q15Product(leftPitch, rate)
	s.Pitch[1] += q15Product(rightPitch, rate)
	forcePitch := int16(int32(s.Pitch[0]+s.Pitch[1]) * 6144 / int32(int16(config.PitchStiffness)))
	out.Pitch = forcePitch + int16(int32(front-rear)*2048/int32(int16(uint16(0x8fff)-config.PitchStiffness)))

	out.Loads = [4]uint16{
		positiveLoad(out.BaseLoads[0] + uint16(s.Roll[1]) + uint16(s.Pitch[0])),
		positiveLoad(out.BaseLoads[1] - uint16(s.Roll[1]) + uint16(s.Pitch[1])),
		positiveLoad(out.BaseLoads[2] + uint16(s.Roll[0]) - uint16(s.Pitch[0])),
		positiveLoad(out.BaseLoads[3] - uint16(s.Roll[0]) - uint16(s.Pitch[1])),
	}
	for i := range 2 {
		s.Roll[i] = decayUnit(s.Roll[i])
		s.Pitch[i] = decayUnit(s.Pitch[i])
	}
	var compression [4]int16
	for i, load := range out.BaseLoads {
		deflection := int16(int32(int16(load-1000)) / int32(config.Springs[i]>>8))
		if i%2 == 0 {
			compression[i] = -(deflection + ((forceRoll - forcePitch) >> 1) - 256)
		} else {
			compression[i] = ((forceRoll + forcePitch) >> 1) - deflection + 256
		}
		compression[i] = max(int16(0), min(int16(574), compression[i]))
		out.Camber[i] = compression[i] + config.Camber[i]
		if i%2 == 0 {
			out.Camber[i] += out.Roll
		} else {
			out.Camber[i] -= out.Roll
		}
	}
	out.WheelBob = [2]byte{byte(compression[2] >> 5), byte(compression[3] >> 5)}
	return out
}
