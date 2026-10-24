package vehicle

import "github.com/olivierh59500/indianapolis500/internal/scene"

// TrafficImpact contains the post-contact track motion of the player's car.
// Its numerical velocities are transferred to the ordinary force model.
type TrafficImpact struct {
	Forward, LateralVelocity, YawRate int16
	TrackGap, LateralGap              int16
	Severity                          uint16
	TrackHeading                      int16
	DamageEnabled                     bool
}

type CameraJolt struct{ Pitch, Roll int16 }

func ApplyTrafficImpact(s *State, world *scene.World, impact TrafficImpact, random func() uint16) CameraJolt {
	side := impact.LateralVelocity << 3
	s.Body.Speed = scene.VelocityMagnitude(scene.Vector2{X: impact.Forward, Y: side})
	s.Body.TravelHeading = world.Direction(impact.Forward, side) + impact.TrackHeading
	velocity := world.VelocityToWorld(int16(s.Body.Speed), 0, s.Body.TravelHeading)
	s.Body.VelocityX, s.Body.VelocityY = velocity.X, velocity.Y
	if impact.DamageEnabled {
		direction := world.Direction(-impact.TrackGap, impact.LateralGap>>4) + impact.TrackHeading - s.Body.Heading
		s.Engine.Failures |= world.CarContactDamage(impact.Severity, direction)
	}
	limited := func(value int16, limit uint16) int16 {
		magnitude := unsignedAbs(value)
		if magnitude >= limit {
			magnitude = limit
		}
		result := int16(magnitude)
		if value < 0 {
			result = -result
		}
		return result
	}
	jolt := CameraJolt{Roll: limited(int16(random()&1023)-512, impact.Severity*2), Pitch: limited(int16(random()&1023)-512, impact.Severity*2)}
	s.Steering += limited(int16(random()&4095)-2048, impact.Severity<<3)
	s.Steering = max(int16(-3200), min(int16(3200), s.Steering))
	s.Body.YawVelocity += impact.YawRate * 2
	s.Body.PreviousYawVelocity = s.Body.YawVelocity
	for i, flag := range [4]byte{4, 2, 8, 1} {
		s.Tires[i].Failed = s.Engine.Failures&flag != 0
	}
	return jolt
}
