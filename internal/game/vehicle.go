package game

import (
	"encoding/json"
	"math"

	gameassets "github.com/olivierh59500/indianapolis500/assets"
	"github.com/olivierh59500/indianapolis500/internal/pilot"
	worldscene "github.com/olivierh59500/indianapolis500/internal/scene"
	"github.com/olivierh59500/indianapolis500/internal/vehicle"
)

// One source world unit is two inches. The original advances the vehicle
// fifteen times per second; track progress uses two world units per step.
const (
	worldPerMPHTick    = 44.0 / 75
	progressPerMPHTick = worldPerMPHTick / 2
	speedPerMPH        = 128 * worldPerMPHTick
)

func (g *Game) loadVehicle() error {
	data, err := gameassets.Original.ReadFile("original/vehicle-defaults.json")
	if err != nil {
		return err
	}
	if err := json.Unmarshal(data, &g.vehiclePreset); err != nil {
		return err
	}
	data, err = gameassets.Original.ReadFile("original/tire-slip-curve.bin")
	if err != nil {
		return err
	}
	g.vehicleCurve, err = vehicle.NewContactCurve(data)
	if err != nil {
		return err
	}
	data, err = gameassets.Original.ReadFile("original/track-surface-wave.bin")
	if err != nil {
		return err
	}
	g.vehicleSurface, err = vehicle.NewTrackSurface(data)
	if err != nil {
		return err
	}
	g.vehicleModel = vehicle.Vehicle{World: g.world, Curve: &g.vehicleCurve, Surface: &g.vehicleSurface}
	return nil
}

func (g *Game) initializePlayerVehicle(previous vehicle.State) {
	player := &g.cars[0]
	fuel := g.setup.fuel * 32
	if g.demo {
		fuel = 40 * 32
	}
	player.physical, player.driver = true, vehicle.DriverState{}
	player.physics = vehicle.Initialize(previous, g.physicalConfig(), g.world, vehicle.SessionStart{
		Progress: uint16(originalTrackProgress(player.position)), Lateral: int16(player.lateral),
		Speed: uint16(math.Round(player.speed * speedPerMPH)), FuelUnits: uint16(fuel), WarmTires: g.mode != practiceMode,
	})
	player.dashboardSpeed = int16(player.physics.Body.Speed)
	g.syncPlayerVehicle()
}

// placePlayerVehicle initializes a new session's ordinary world pose. During
// driving only the translated force integrator changes position and velocity.
func (g *Game) placePlayerVehicle(position, lateral, speed float64) {
	player := &g.cars[0]
	s := &player.physics
	point := g.carWorldPose(position, lateral)
	s.Body.PositionX, s.Body.PositionY = int32(point.X)<<16, int32(point.Y)<<16
	s.Body.Heading, s.Body.TravelHeading = point.Heading, point.Heading
	s.Body.Pitch, s.Body.Roll, s.Body.YawVelocity, s.Body.PreviousYawVelocity = 0, 0, 0, 0
	velocity := g.world.VelocityToWorld(int16(math.Round(speed*speedPerMPH)), 0, point.Heading)
	s.Body.VelocityX, s.Body.VelocityY = velocity.X, velocity.Y
	s.Body.Speed = uint16(math.Round(math.Abs(speed) * speedPerMPH))
	location := g.world.LocateTrack(worldscene.Vector2{X: point.X, Y: point.Y})
	s.Progress, s.Lateral, s.Steering = location.Progress, location.Lateral, 0
	physicalConfig := g.physicalConfig()
	config := &physicalConfig.Engine
	s.Engine.Gear = 0
	for s.Engine.Gear < 3 && uint32(s.Body.Speed)*uint32(config.Gears[s.Engine.Gear])>>16 >= uint32(config.ShiftUp) {
		s.Engine.Gear++
	}
	s.Engine.Ratio = config.Gears[s.Engine.Gear]
	s.Engine.RPM = max(uint16(1000), uint16(uint32(s.Body.Speed)*uint32(s.Engine.Ratio)>>16))
}

func (g *Game) stepPlayerVehicle(input drivingInput) {
	player := &g.cars[0]
	state := &player.physics
	driver := vehicle.Driver{State: player.driver}
	controls := driver.Step(vehicle.DriverContext{
		Steering: state.Steering, Damping: state.SteeringDamping,
		Increment: state.SteeringIncrement, Speed: state.Body.Speed, Failures: state.Engine.Failures,
	}, vehicle.DriverIntent{
		Steering: input.steering, Accelerate: input.throttle, Brake: input.brake,
		Reverse: input.reverse, CruiseIncrease: input.cruiseIncrease,
		DamageEnabled: !g.demo && (g.mode == qualifyMode || g.mode == raceMode && g.raceLaps >= 60),
	})
	player.driver = driver.State
	if !g.demo {
		g.reportCruiseNotification(driver.State.CruiseTarget != 0)
	}
	config := g.physicalConfig()
	if config.Engine.Gears != g.vehicleModel.Config.Engine.Gears {
		state.Engine.Ratio = config.Engine.Gears[state.Engine.Gear]
	}
	g.vehicleModel.Config = config
	g.vehicleModel.State = *state
	events := g.vehicleModel.Step(controls)
	if !g.demo {
		g.reportVehicleNotifications(events, g.noticeInPit)
		g.reportCollisionNotification(events.Impact, controls.DamageEnabled)
	}
	player.dashboardSpeed = events.DashboardSpeed
	*state = g.vehicleModel.State
	g.syncPlayerVehicle()
	if controls.Throttle != 0 || controls.Reverse {
		g.pitBoard = false
	} else if !g.demo && input.brake && state.Body.Speed == 0 && uint16(state.Lateral) < 75 && state.Progress > 6930 && state.Progress < 7080 {
		g.pitBoard = true
	}
	if events.Impact != 0 {
		g.emitWallCollisionEffects(events.Impact)
		player.driver.CruiseTarget = 0
	}
	if events.Engine.EngineFailure || events.TireFailures != 0 {
		g.soundController.Requests[0] = 127
	}
	g.soundController.NoisePitch = events.SkidPitch
	g.soundController.NoiseStep = int8(events.Skid - g.soundController.NoiseLevel)

}

func (g *Game) syncPlayerVehicle() {
	player := &g.cars[0]
	s := &player.physics
	previous := player.position
	player.position = float64((int(s.Progress) + trackLength - 6930) % trackLength)
	player.progress += wrappedDistance(player.position - previous)
	player.lateral = float64(s.Lateral) * 16
	point := g.world.TrackPlacement(s.Progress, s.Lateral<<4)
	forward, lateral := g.world.VelocityOnTrack(worldscene.Vector2{X: s.Body.VelocityX, Y: s.Body.VelocityY}, point.Heading)
	player.speed, player.lateralVelocity = float64(forward)/speedPerMPH, float64(lateral)/8
	player.yaw, player.angularVelocity = s.Body.Heading-point.Heading, s.Body.YawVelocity
	player.steering, player.gear = -float64(s.Steering)/3200, int(s.Engine.Gear)+1
	player.damageFlags = s.Engine.Failures
	player.worldX, player.worldY, player.worldZ = int16(s.Body.PositionX>>16), int16(s.Body.PositionY>>16), point.Z
	player.worldVX, player.worldVY, player.worldPose = s.Body.VelocityX, s.Body.VelocityY, true
	g.syncVehicleCondition()
}

// automaticPlayerControls follows a visible racing line through the same
// steering and pedal ramps as manual input. It never writes vehicle motion.
func (g *Game) automaticPlayerControls() drivingInput {
	if g.demoPilot == nil {
		g.demoPilot = pilot.New(g.world, &g.vehicleCurve, &g.vehicleSurface)
	}
	input := g.demoPilot.Step(pilot.Frame{
		State: g.cars[0].physics, Config: g.physicalConfig(), Driver: g.cars[0].driver,
		Cars: g.trafficModel.Cars, Tick: g.raceTicks, Traffic: g.mode != qualifyMode,
		DamageEnabled: !g.demo && (g.mode == qualifyMode || g.mode == raceMode && g.raceLaps >= 60),
	})
	return drivingInput{steering: input.Steering, throttle: input.Accelerate, brake: input.Brake, reverse: input.Reverse}
}
