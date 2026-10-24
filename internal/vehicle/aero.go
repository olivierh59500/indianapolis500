package vehicle

import "github.com/olivierh59500/indianapolis500/internal/scene"

type AeroConfig struct {
	RearWingDrag, FrontWingDrag uint16
	YawScale                    int16
}

type AeroInput struct {
	Mass, Speed                                       uint16
	VelocityX, VelocityY, YawVelocity                 int16
	Heading, TravelHeading, TrackHeading, Grade, Bank int16
}

type AeroOutput struct {
	VelocityX, VelocityY, YawVelocity int16
	GravityBank, GravityGrade         int16
	PreviousSpeed                     uint16
	PreviousTravel                    int16
	Drag, TotalLoad                   uint16
	BaseLoads                         [4]uint16
}

func q14Product(a, b int16) int16 { return int16(int32(a) * int32(b) >> 14) }
func headingCoefficients(world *scene.World, angle int16) (int16, int16) {
	vector := world.VelocityToWorld(16384, 0, angle)
	return vector.X, -vector.Y
}

// ApplyGravityAndAero translates $1bf8-$1dac. The original heading tables
// provide track gravity, yaw-dependent downforce and aerodynamic resistance.
func ApplyGravityAndAero(input AeroInput, config AeroConfig, world *scene.World) AeroOutput {
	_, bankSine := headingCoefficients(world, input.Bank)
	_, gradeSine := headingCoefficients(world, input.Grade)
	out := AeroOutput{
		VelocityX: input.VelocityX, VelocityY: input.VelocityY, YawVelocity: input.YawVelocity,
		GravityBank: int16(int32(bankSine) * 440 >> 16), GravityGrade: int16(int32(gradeSine) * 440 >> 16),
		PreviousSpeed: input.Speed, PreviousTravel: input.TravelHeading,
	}
	out.TotalLoad = uint16(int32(int16(input.Mass)) * 110 >> 8)
	rear := highProduct(out.TotalLoad, 0x999a)
	front := out.TotalLoad - rear
	out.BaseLoads = [4]uint16{rear >> 1, rear >> 1, front >> 1, front >> 1}
	cosine, sine := headingCoefficients(world, input.TrackHeading)
	out.VelocityX -= q14Product(out.GravityGrade, cosine) + q14Product(out.GravityBank, sine)
	out.VelocityY += q14Product(out.GravityGrade, sine) - q14Product(out.GravityBank, cosine)

	alignment, yawSine := headingCoefficients(world, input.Heading-input.TravelHeading)
	liftScale := uint16(max(int16(0), alignment)) * 2
	squaredSpeed := highProduct(input.Speed, input.Speed)
	rearDrag := highProduct(squaredSpeed, config.RearWingDrag)
	frontDrag := highProduct(squaredSpeed, config.FrontWingDrag)
	out.Drag = highProduct(squaredSpeed, 0x25e4) + rearDrag + frontDrag
	rearLift := highProduct((rearDrag>>1)+rearDrag*4, liftScale)
	frontLift := highProduct((frontDrag>>1)+frontDrag*4, liftScale)
	out.BaseLoads[0] += rearLift
	out.BaseLoads[1] += rearLift
	out.BaseLoads[2] += frontLift
	out.BaseLoads[3] += frontLift
	out.TotalLoad += rearLift*2 + frontLift*2
	out.YawVelocity += int16(int32(q14Product(-int16(out.Drag), yawSine)) * int32(config.YawScale) / int32(int16(input.Mass)))
	deceleration := int16((int32(out.Drag) << 8) / int32(int16(input.Mass)))
	resistance := world.VelocityToWorld(deceleration, 0, input.TravelHeading)
	out.VelocityX -= resistance.X
	out.VelocityY -= resistance.Y
	return out
}
