package vehicle

import "github.com/olivierh59500/indianapolis500/internal/scene"

// Config combines the recovered setup coefficients for the player's vehicle.
type Config struct {
	Engine                              EngineConfig
	Tires                               [4]TireConfig
	Suspension                          SuspensionConfig
	Aero                                AeroConfig
	Dynamics                            DynamicsConfig
	PeakGrip                            [4]uint16
	Ambient, ThrottleLimit, BoostTarget uint16
	Differential                        int16
}

// State contains only persistent physical and presentation quantities.
// World position retains Q16 fractions; velocity uses Q7 world units per tick.
type State struct {
	Body                                             DynamicsState
	Engine                                           EngineState
	Tires                                            [4]TireState
	Suspension                                       SuspensionState
	Loads                                            [4]uint16
	Camber                                           [4]int16
	Energy                                           [4]uint16
	TotalLongitudinal, TotalLateral                  int16
	Progress                                         uint16
	Lateral, Steering                                int16
	SteeringDamping, SteeringIncrement, FrontAverage uint16
	WheelBob                                         [2]byte
	FrontGaugeSpeed                                  int16
}

type Controls struct {
	Steering               int16
	Throttle, Brake        uint16
	Reverse, DamageEnabled bool
}

type Events struct {
	Engine          EngineEvents
	TireFailures    byte
	StoppedFailures byte
	Impact          uint16
	Skid            byte
	SkidPitch       int16
	DashboardSpeed  int16
}

type Preset struct {
	Config Config
	State  State
}

type Vehicle struct {
	Config  Config
	State   State
	World   *scene.World
	Curve   *ContactCurve
	Surface *TrackSurface
}

// Step composes the recovered physical tick in the same order as $17ee.
// Device input, elapsed-time scheduling and game-session rules remain outside
// this model; Controls are ordinary steering and pedal quantities.
func (vehicle *Vehicle) Step(input Controls) Events {
	config, s, world := &vehicle.Config, &vehicle.State, vehicle.World
	var events Events
	oldProgress, oldLateral := s.Progress, s.Lateral
	oldPosition := scene.Vector2{X: int16(s.Body.PositionX >> 16), Y: int16(s.Body.PositionY >> 16)}
	track := world.TrackPlacement(s.Progress, s.Lateral<<4)
	grade, bank := world.BankAngles(s.Progress, s.Lateral)
	aero := ApplyGravityAndAero(AeroInput{Mass: s.Engine.Mass, Speed: s.Body.Speed, VelocityX: s.Body.VelocityX, VelocityY: s.Body.VelocityY, YawVelocity: s.Body.YawVelocity, Heading: s.Body.Heading, TravelHeading: s.Body.TravelHeading, TrackHeading: track.Heading, Grade: grade, Bank: bank}, config.Aero, world)
	var grip, rolling [4]uint16
	failureBits := [4]byte{4, 2, 8, 1}
	for i := range 4 {
		tire := Tire{Config: config.Tires[i], State: s.Tires[i]}
		tire.State.Failed = s.Engine.Failures&failureBits[i] != 0
		result := tire.Step(TireInput{Energy: s.Energy[i], Camber: s.Camber[i], Ambient: config.Ambient, Speed: aero.PreviousSpeed, Load: s.Loads[i]})
		s.Tires[i] = tire.State
		if result.Punctured {
			s.Engine.Failures |= failureBits[i]
			events.TireFailures |= failureBits[i]
		}
		grip[i], rolling[i] = result.Grip, tire.State.RollingResistance
	}
	suspension := Suspension{Config: config.Suspension, State: s.Suspension}
	cosine, sine := headingCoefficients(world, s.Body.Heading-track.Heading)
	support := suspension.Step(SuspensionInput{Progress: s.Progress, Lateral: s.Lateral, BaseLoads: aero.BaseLoads, Grip: grip, Rolling: rolling, LateralForce: s.TotalLateral, LongitudinalForce: s.TotalLongitudinal}, cosine, sine, vehicle.Surface)
	s.Suspension, s.Loads, s.Camber, s.WheelBob = suspension.State, support.Loads, support.Camber, support.WheelBob
	s.Body.VelocityX, s.Body.VelocityY, s.Body.YawVelocity = aero.VelocityX, aero.VelocityY, aero.YawVelocity
	s.Body.Pitch, s.Body.Roll = support.Pitch, support.Roll
	wheels := CalculateWheels(WheelInput{VelocityX: s.Body.VelocityX, VelocityY: s.Body.VelocityY, YawVelocity: s.Body.YawVelocity, Heading: s.Body.Heading, Steering: input.Steering, Mass: s.Engine.Mass, TotalLoad: aero.TotalLoad, Brake: input.Brake, Grip: config.PeakGrip, Rolling: support.Rolling, OffRoad: support.OffRoad}, vehicle.Curve, world)
	events.DashboardSpeed = wheels.RearAverage
	s.SteeringDamping, s.SteeringIncrement, s.FrontAverage = wheels.SteeringDamping, wheels.SteeringIncrement, wheels.FrontAverage
	engine := Engine{Config: config.Engine, State: s.Engine}
	events.Engine = engine.Step(EngineInput{Speed: aero.PreviousSpeed, Ambient: config.Ambient, Throttle: input.Throttle, ThrottleLimit: config.ThrottleLimit, BoostTarget: config.BoostTarget, LeftLoad: s.Loads[0], RightLoad: s.Loads[1], LeftGrip: config.PeakGrip[0], RightGrip: config.PeakGrip[1], RearSpeed: wheels.RearAverage, Differential: config.Differential})
	s.Engine = engine.State
	dynamics := ApplyWheelForces(s.Body, DynamicsInput{Mass: s.Engine.Mass, Steering: input.Steering, ThrottleResponse: s.Engine.ThrottleResponse, RearBrake: wheels.RearBrake, Brake: input.Brake, EngineDemand: [2]uint16{s.Engine.LeftDemand, s.Engine.RightDemand}, Loads: s.Loads, Grip: support.Grip, PeakGrip: config.PeakGrip, Rolling: wheels.Rolling, BrakeDemand: wheels.BrakeDemand, LateralDemand: wheels.LateralDemand, Slip: wheels.Slip, WheelSpeed: wheels.Speed, Motion: wheels.Motion, Skid: wheels.Skid, OffRoad: support.OffRoad, Failures: s.Engine.Failures, Reverse: input.Reverse}, config.Dynamics, world)
	s.Body, s.Energy, s.TotalLongitudinal, s.TotalLateral = dynamics.State, dynamics.Energy, dynamics.TotalLongitudinal, dynamics.TotalLateral
	s.Engine.LeftDemand, s.Engine.RightDemand = dynamics.EngineDemand[0], dynamics.EngineDemand[1]
	s.FrontGaugeSpeed = dynamics.FrontGaugeSpeed
	if s.Body.Speed < 100 {
		events.StoppedFailures = s.Engine.Failures
	}
	s.Steering = input.Steering
	events.Skid, events.SkidPitch = dynamics.Skid, wheels.SkidPitch

	contact := scene.CollisionState{Position: scene.Vector2{X: int16(s.Body.PositionX >> 16), Y: int16(s.Body.PositionY >> 16)}, Velocity: scene.Vector2{X: s.Body.VelocityX, Y: s.Body.VelocityY}, FractionX: uint16(s.Body.PositionX), FractionY: uint16(s.Body.PositionY), Heading: s.Body.Heading, YawVelocity: s.Body.YawVelocity, TravelHeading: s.Body.TravelHeading, Speed: s.Body.Speed, PitchImpulse: s.Body.Pitch, RollImpulse: s.Body.Roll, Steering: s.Steering, DamageEnabled: input.DamageEnabled, DamageFlags: s.Engine.Failures}
	location, hit := world.ResolveWalls(&contact, oldPosition, scene.TrackLocation{Progress: oldProgress, Lateral: oldLateral}, track.Heading)
	if hit {
		s.Body.PositionX = int32(contact.Position.X)<<16 | int32(contact.FractionX)
		s.Body.PositionY = int32(contact.Position.Y)<<16 | int32(contact.FractionY)
		s.Body.VelocityX, s.Body.VelocityY = contact.Velocity.X, contact.Velocity.Y
		s.Body.Heading, s.Body.YawVelocity = contact.Heading, contact.YawVelocity
		s.Body.TravelHeading, s.Body.Speed = contact.TravelHeading, contact.Speed
		s.Body.Pitch, s.Body.Roll = contact.PitchImpulse, contact.RollImpulse
		s.Engine.Failures, s.Steering = contact.DamageFlags, contact.Steering
		events.Impact = contact.Impact
	}
	s.Progress, s.Lateral = location.Progress, location.Lateral
	for i, flag := range failureBits {
		s.Tires[i].Failed = s.Engine.Failures&flag != 0
	}
	return events
}
