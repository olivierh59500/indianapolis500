package game

import (
	gameassets "github.com/olivierh59500/indianapolis500/assets"
	"github.com/olivierh59500/indianapolis500/internal/traffic"
	"github.com/olivierh59500/indianapolis500/internal/vehicle"
)

func (c car) present() bool { return c.active || c.lap != 0 }

func (g *Game) loadTraffic() error {
	read := func(name string) ([]byte, error) { return gameassets.Original.ReadFile("original/" + name) }
	names := []string{"track-lines.bin", "traffic-curve-flags.bin", "traffic-yaw-curve.bin", "traffic-drag-curve.bin", "traffic-spin-strength.bin", "world-trigonometry.bin", "traffic-retirement-reasons.bin", "traffic-driver-coefficients.bin"}
	var data [8][]byte
	for i, name := range names {
		var err error
		data[i], err = read(name)
		if err != nil {
			return err
		}
	}
	var err error
	g.trafficModel, err = traffic.New(data[0], data[1], data[2], data[3])
	if err != nil {
		return err
	}
	if err = g.trafficModel.LoadDriverCurves(data[4], data[5], data[6]); err != nil {
		return err
	}
	inner, err := read("track-wall-inner.bin")
	if err != nil {
		return err
	}
	outer, err := read("track-wall-outer.bin")
	if err != nil {
		return err
	}
	if err = g.trafficModel.LoadRoadLimits(inner, outer); err != nil {
		return err
	}
	g.trafficParameters = data[7]
	g.trafficModel.Random = 0x2468
	for i := range g.trafficModel.Cars {
		g.trafficModel.Cars[i].Laps = 129
	}
	return nil
}

func (g *Game) initializeTraffic() {
	model := g.trafficModel
	model.Cars[0].Primary, model.Cars[0].Secondary = vehicle.PlayerCarColors(g.team)
	model.Cars[0].Number = vehicle.PlayerCarIdentity(g.team)
	var err error
	if g.mode == raceMode || g.demo {
		err = model.InitializeRace(g.trafficParameters, g.qualifyingPosition, g.raceLaps)
	} else {
		err = model.InitializeSolo(g.trafficParameters, g.mode == qualifyMode)
	}
	if err != nil {
		g.err, g.done = err, true
		return
	}
	for i, original := range model.Cars {
		position := float64((trackLength - int(original.Distance)) % trackLength)
		g.cars[i] = car{active: model.InField(i), position: position, progress: -float64(original.Distance), lateral: float64(original.Lateral), speed: float64(original.Speed) / speedPerMPH, gear: gearForSpeed(float64(original.Speed) / speedPerMPH), color: int(original.Primary), secondary: int(original.Secondary), number: int(original.Number & 127), player: i == 0}
		if g.mode != raceMode && !g.demo && model.InField(i) {
			g.cars[i].lap = 1
		}
	}
	if g.mode == practiceMode && !g.demo {
		g.cars[0].progress = g.cars[0].position
	}
}

// playerTrafficPose supplies the force model and clock to the source car pass.
func (g *Game) playerTrafficPose() traffic.PlayerPose {
	state := &g.cars[0].physics
	lateral := state.Lateral * 16
	point := g.world.TrackPlacement(state.Progress, lateral)
	return traffic.PlayerPose{Progress: state.Progress, Lateral: lateral, Heading: state.Body.Heading, TrackHeading: point.Heading, Crossed: g.clock.Crossed, CrossingDelta: g.clock.CrossingDelta}
}

func (g *Game) updateTraffic() {
	model := g.trafficModel
	model.Timing.ElapsedTenths = g.clock.ElapsedTenths
	model.StepRacingField(g.playerTrafficPose(), g.applyPlayerCarContact)
	g.updateRaceStatus()
	for i := 1; i < len(g.cars); i++ {
		source := model.Cars[i]
		other := &g.cars[i]
		position := float64((trackLength - int(source.Distance)) % trackLength)
		other.progress += wrappedDistance(position - other.position)
		other.position = position
		other.lateral, other.lateralVelocity = float64(source.Lateral), float64(source.LateralVelocity)
		other.speed, other.yaw, other.angularVelocity = float64(source.Speed)/speedPerMPH, source.Yaw, source.YawRate
		if g.mode == raceMode || g.demo {
			other.lap = g.raceLaps + 1 - int(source.RemainingLaps)
		} else if model.InField(i) {
			other.lap = 1
		} else {
			other.lap = 0
		}
		other.gear = gearForSpeed(other.speed)
		other.active = model.InField(i)
	}
}

func (g *Game) applyPlayerCarContact(contact traffic.ContactResult) {
	car := &g.trafficModel.Cars[0]
	state := &g.cars[0].physics
	point := g.world.TrackPlacement(state.Progress, state.Lateral<<4)
	damageEnabled := !g.demo && (g.mode == qualifyMode || g.mode == raceMode && g.raceLaps >= 60)
	impulse := vehicle.ApplyTrafficImpact(state, g.world, vehicle.TrafficImpact{
		Forward: int16(car.Speed), LateralVelocity: car.LateralVelocity, YawRate: car.YawRate,
		TrackGap: contact.TrackGap, LateralGap: contact.LateralGap, Severity: contact.Severity, TrackHeading: point.Heading,
		DamageEnabled: damageEnabled,
	}, g.trafficModel.NextRandom)
	g.contactJolt.Pitch += impulse.Pitch
	g.contactJolt.Roll += impulse.Roll
	car.YawRate = 0
	g.reportCollisionNotification(contact.Severity, damageEnabled)

	g.syncPlayerVehicle()
}

func (g *Game) updateRaceStatus() {
	if g.mode != raceMode {
		return
	}
	event := g.trafficModel.StepRaceStatus(!g.demo && g.raceLaps >= 30, g.clock.Crossed)
	if event.Finished {
		g.selectInformationPage(standingsInformationPage)
	}
	if event.Won && !g.demo {
		g.notifications.request(winnerNotification)
		g.notifications.request(escapeNotification)
	}
}
