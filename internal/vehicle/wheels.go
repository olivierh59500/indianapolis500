package vehicle

import "github.com/olivierh59500/indianapolis500/internal/scene"

type WheelInput struct {
	VelocityX, VelocityY, YawVelocity, Heading, Steering int16
	Mass, TotalLoad, Brake                               uint16
	Grip, Rolling                                        [4]uint16
	OffRoad                                              byte
}

type WheelOutput struct {
	Slip                                                        [4]int16
	Speed                                                       [4]uint16
	Motion                                                      [4]WheelMotion
	Normalization                                               uint16
	RearAverage                                                 int16
	LateralDemand, Rolling, BrakeDemand                         [4]uint16
	RearBrake, FrontAverage, SteeringDamping, SteeringIncrement uint16
	Skid                                                        byte
	SkidPitch                                                   int16
}

// CalculateWheels translates $2dac-$3236, including local wheel motion from
// yaw, front steering, the contact law, rolling resistance and brake demand.
func CalculateWheels(input WheelInput, curve *ContactCurve, world *scene.World) WheelOutput {
	out := WheelOutput{Rolling: input.Rolling, SkidPitch: 8192}
	out.Normalization = uint16((uint32(input.Mass) << 15) / uint32(input.TotalLoad*2))
	rearRadius := int16(int32(input.YawVelocity) * 0x5699 >> 16)
	frontRadius := int16(int32(input.YawVelocity) * 0x7322 >> 16)
	radii := [4]int16{rearRadius, rearRadius, frontRadius, frontRadius}
	bearings := [4]int16{-0x5b50, -0x24b0, 0x53d0, 0x2c30}
	var longitudinalDemand [4]uint16
	for i := range 4 {
		rotation := world.VelocityToWorld(radii[i], 0, input.Heading+bearings[i])
		x, y := input.VelocityX+rotation.X, input.VelocityY+rotation.Y
		out.Slip[i] = input.Heading - world.Direction(x, y)
		if i >= 2 {
			out.Slip[i] += input.Steering
		}
		out.Speed[i] = vectorLength(unsignedAbs(x), unsignedAbs(y))
		polar := world.VelocityToWorld(int16(out.Speed[i]), 0, out.Slip[i])
		out.Motion[i] = WheelMotion{polar.X, -polar.Y}
		contact := curve.Demand(out.Motion[i], out.Slip[i], input.Grip[i], out.Normalization)
		longitudinalDemand[i], out.LateralDemand[i] = contact.Longitudinal, contact.Lateral
		if input.OffRoad&(1<<i) == 0 {
			sideSpeed := unsignedAbs(out.Motion[i].Sideways)
			if sideSpeed >= 300 {
				level := contact.Skid
				if sideSpeed-300 < 1324 {
					level = byte(uint32((sideSpeed-300)>>2) * uint32(level) >> 8)
				}
				out.Skid |= level
				out.SkidPitch += int16(out.Speed[i]>>5) - 256
			}
		}
	}
	out.RearAverage = (out.Motion[0].Forward >> 1) + (out.Motion[1].Forward >> 1)
	out.FrontAverage = (out.Speed[2] + out.Speed[3]) >> 1
	if out.FrontAverage < 2400 {
		out.SteeringDamping = (out.Speed[2] + out.Speed[3]) >> 2
	} else {
		out.SteeringDamping = uint16(0x735b40 / uint32(out.FrontAverage+3900))
	}
	out.SteeringIncrement = uint16(0x18ae16/uint32(out.FrontAverage+8087)) + out.SteeringDamping
	out.RearBrake = input.Brake - (input.Brake >> 2)
	for i := range 4 {
		out.Rolling[i] = min(out.Rolling[i], longitudinalDemand[i])
		demand := longitudinalDemand[i] - out.Rolling[i]
		brake := input.Brake
		if i < 2 {
			brake = out.RearBrake
		}
		out.BrakeDemand[i] = highProduct(demand, brake)
	}
	return out
}
