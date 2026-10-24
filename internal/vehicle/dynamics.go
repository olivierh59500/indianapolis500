package vehicle

import "github.com/olivierh59500/indianapolis500/internal/scene"

type DynamicsConfig struct{ RearMoment, FrontMoment, AxleMoment int16 }

type DynamicsState struct {
	PositionX, PositionY                                                  int32
	VelocityX, VelocityY                                                  int16
	Speed                                                                 uint16
	TravelHeading, Heading, YawVelocity, PreviousYawVelocity, Pitch, Roll int16
}

type DynamicsInput struct {
	Mass                                                       uint16
	Steering                                                   int16
	ThrottleResponse, RearBrake, Brake                         uint16
	EngineDemand                                               [2]uint16
	Loads, Grip, PeakGrip, Rolling, BrakeDemand, LateralDemand [4]uint16
	Slip                                                       [4]int16
	WheelSpeed                                                 [4]uint16
	Motion                                                     [4]WheelMotion
	Skid, OffRoad, Failures                                    byte
	Reverse                                                    bool
}

type DynamicsOutput struct {
	State                           DynamicsState
	Longitudinal, Lateral           [4]int16
	Energy                          [4]uint16
	TotalLongitudinal, TotalLateral int16
	EngineDemand                    [2]uint16
	Skid                            byte
	FrontGaugeSpeed                 int16
}

// ApplyWheelForces translates $357e-$3d64, stopping before physical walls.
// All tire forces remain in their wheel frames until the front steering
// rotation; the original energy calculation happens before puncture forces.
func ApplyWheelForces(state DynamicsState, input DynamicsInput, config DynamicsConfig, world *scene.World) DynamicsOutput {
	out := DynamicsOutput{State: state, EngineDemand: input.EngineDemand, Skid: input.Skid, FrontGaugeSpeed: input.Motion[3].Forward}
	var negativeLong [4]bool
	for i := range 2 {
		if input.Motion[i].Forward < 0 {
			out.EngineDemand[i] += input.BrakeDemand[i]
		} else if out.EngineDemand[i] < input.BrakeDemand[i] {
			negativeLong[i] = true
			out.EngineDemand[i] = input.BrakeDemand[i] - out.EngineDemand[i]
		} else {
			out.EngineDemand[i] -= input.BrakeDemand[i]
		}
	}
	skidding := input.Skid
	for i := range 4 {
		var blend WheelBlend
		if i < 2 {
			blend, skidding = DrivenBlend(input.Motion[i].Sideways, input.Motion[i].Forward, input.RearBrake, input.ThrottleResponse, skidding)
		} else {
			blend = FrontBlend(input.Motion[i].Sideways, input.Brake)
			negativeLong[i] = -input.Motion[i].Forward < 0
		}
		longDemand := input.BrakeDemand[i]
		if i < 2 {
			longDemand = out.EngineDemand[i]
		}
		halfGrip := highProduct(input.PeakGrip[i]>>1, input.Grip[i])
		out.Lateral[i], out.Longitudinal[i] = ResolveWheelForces(input.LateralDemand[i], longDemand, input.Loads[i], halfGrip, blend, -input.Slip[i] < 0, negativeLong[i])
		if i < 2 && input.OffRoad&(1<<i) == 0 {
			out.Skid |= skidding
		}
	}
	if input.Reverse {
		out.Longitudinal[0], out.Longitudinal[1] = -500, -500
	}
	for i := range 4 {
		sideSpeed := unsignedAbs(input.Motion[i].Sideways)
		work := uint32(unsignedAbs(out.Longitudinal[i])) * uint32(sideSpeed)
		rolling := highProduct(input.Rolling[i]*2, input.Loads[i])
		drag := int16(rolling)
		if input.Motion[i].Forward < 0 {
			drag = -drag
		}
		out.Longitudinal[i] -= drag
		work += uint32(int32(int16(unsignedAbs(input.Motion[i].Forward))) * int32(int16(rolling)))
		work += uint32(int32(int16(unsignedAbs(out.Lateral[i]))) * int32(int16(sideSpeed)))
		out.Energy[i] = uint16((work << 3) >> 16)
	}
	failureBits := [4]byte{4, 2, 8, 1}
	for i, flag := range failureBits {
		if input.Failures&flag == 0 {
			continue
		}
		resistance := min(uint32(500), (uint32(input.WheelSpeed[i]>>2)*uint32(input.Mass))>>8)
		force := world.VelocityToWorld(-int16(resistance), 0, input.Slip[i])
		out.Longitudinal[i], out.Lateral[i] = force.X, -force.Y
		if i < 2 {
			out.State.Pitch += 245
		} else {
			out.State.Pitch -= 245
		}
		if i%2 == 0 {
			out.State.Roll += 245
		} else {
			out.State.Roll -= 245
		}
	}

	cosine, sine := headingCoefficients(world, input.Steering)
	for i := 2; i < 4; i++ {
		long, lat := out.Longitudinal[i], out.Lateral[i]
		out.Longitudinal[i] = int16(((int32(long)*int32(cosine) + int32(lat)*int32(sine)) * 4) >> 16)
		out.Lateral[i] = int16(((int32(lat)*int32(cosine) - int32(long)*int32(sine)) * 4) >> 16)
	}
	leftLong, rightLong := out.Longitudinal[0]+out.Longitudinal[2], out.Longitudinal[1]+out.Longitudinal[3]
	rearSide, frontSide := out.Lateral[0]+out.Lateral[1], out.Lateral[2]+out.Lateral[3]
	out.TotalLongitudinal, out.TotalLateral = leftLong+rightLong, rearSide+frontSide
	cosine, sine = headingCoefficients(world, state.Heading)
	forceX := (int32(out.TotalLongitudinal)*int32(cosine) + int32(out.TotalLateral)*int32(sine)) >> 6
	forceY := (int32(out.TotalLateral)*int32(cosine) - int32(out.TotalLongitudinal)*int32(sine)) >> 6
	out.State.VelocityX += int16(forceX / int32(int16(input.Mass)))
	out.State.VelocityY += int16(forceY / int32(int16(input.Mass)))
	moment := int32(rearSide)*int32(config.RearMoment) - int32(frontSide)*int32(config.FrontMoment) + int32(rightLong-leftLong)*int32(config.AxleMoment)
	out.State.YawVelocity += int16(moment / int32(int16(input.Mass)))
	if state.Speed < 1000 && (state.PreviousYawVelocity^out.State.YawVelocity) < 0 {
		out.State.YawVelocity = state.PreviousYawVelocity + ((out.State.YawVelocity - state.PreviousYawVelocity + 1) >> 1)
	}
	out.State.Heading += out.State.YawVelocity
	if out.State.VelocityX > 12 || out.State.VelocityX < -12 {
		out.State.PositionX += int32(out.State.VelocityX) * 512
	}
	if out.State.VelocityY > 12 || out.State.VelocityY < -12 {
		out.State.PositionY += int32(out.State.VelocityY) * 512
	}
	out.State.TravelHeading = world.Direction(out.State.VelocityX, out.State.VelocityY)
	out.State.Speed = vectorLength(unsignedAbs(out.State.VelocityX), unsignedAbs(out.State.VelocityY))
	if out.State.Speed < 13 {
		out.FrontGaugeSpeed = 0
	}
	out.State.VelocityX = decayUnit(out.State.VelocityX)
	out.State.VelocityY = decayUnit(out.State.VelocityY)
	out.State.YawVelocity = decayUnit(out.State.YawVelocity)
	out.State.PreviousYawVelocity = out.State.YawVelocity
	return out
}
