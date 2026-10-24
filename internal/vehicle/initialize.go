package vehicle

import "github.com/olivierh59500/indianapolis500/internal/scene"

// SessionStart contains ordinary track and setup quantities for a new event.
// Lateral retains the computer-car Q4 convention until the force model starts.
type SessionStart struct {
	Progress         uint16
	Lateral          int16
	Speed, FuelUnits uint16
	WarmTires        bool
}

// Initialize follows the original event reset, preserving thermal counters,
// cached grip, suspension history and fractional fuel unless reset explicitly.
func Initialize(previous State, config Config, world *scene.World, start SessionStart) State {
	s := previous
	point := world.TrackPlacement(start.Progress, start.Lateral)
	s.Body.PositionX, s.Body.PositionY = int32(point.X)<<16, int32(point.Y)<<16
	s.Body.Heading, s.Body.TravelHeading = point.Heading, point.Heading
	velocity := world.VelocityToWorld(int16(start.Speed), 0, point.Heading)
	s.Body.VelocityX, s.Body.VelocityY = velocity.X, velocity.Y
	s.Body.Speed = start.Speed
	s.Body.YawVelocity, s.Body.PreviousYawVelocity, s.Body.Pitch, s.Body.Roll = 0, 0, 0, 0
	s.Progress, s.Lateral = start.Progress, start.Lateral>>4
	s.Steering, s.TotalLongitudinal, s.TotalLateral = 0, 0, 0
	s.Suspension.Surface = [4]int16{}
	s.SteeringDamping, s.SteeringIncrement = 0, 100
	s.FrontGaugeSpeed = int16(start.Speed)
	s.Engine.Gear = 0
	for s.Engine.Gear < 3 && uint32(start.Speed)*uint32(config.Engine.Gears[s.Engine.Gear])>>16 >= uint32(config.Engine.ShiftUp) {
		s.Engine.Gear++
	}
	s.Engine.Ratio = config.Engine.Gears[s.Engine.Gear]
	s.Engine.RPM = max(uint16(1000), uint16(uint32(start.Speed)*uint32(s.Engine.Ratio)>>16))
	s.Engine.Temperature, s.Engine.Life = 21376, 150<<16
	s.Engine.Fuel = uint32(start.FuelUnits)<<16 | s.Engine.Fuel&65535
	s.Engine.Mass = 7500 + start.FuelUnits
	s.Engine.Failures &^= 63
	temperature := uint16(19488)
	if start.WarmTires {
		temperature = 22048
	}
	for i := range s.Tires {
		tire := &s.Tires[i]
		life := uint32(700)
		if i >= 2 {
			life = 850
		}
		tire.Life, tire.Failed = life<<16, false
		tire.Bands = [3]uint16{temperature, temperature, temperature}
		tire.Pressure = uint16(uint32(config.Tires[i].ColdPressure) * uint32(temperature) / uint32(config.Ambient))
	}
	return s
}
