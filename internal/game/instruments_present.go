package game

func (g *Game) currentInstruments() instrumentInput {
	player := &g.cars[0]
	engine := &player.physics.Engine
	front, rear := g.frontRollBar, g.rearRollBar
	if g.demo {
		front, rear = 5, 1
	}
	return instrumentInput{
		RPM: engine.RPM, Temperature: engine.Temperature, Pressure: engine.Pressure,
		Speed: player.dashboardSpeed, Gear: engine.Gear, Fuel: uint16(engine.Fuel >> 16),
		Front: uint16(1024 + 896*front), Rear: uint16(1024 + 896*rear),
		Boost: g.physicalConfig().BoostTarget, Warning: engine.Warning,
	}
}

func (g *Game) updateInstruments() {
	d := &g.dashboard
	input := g.currentInstruments()
	if d.validPixels && d.tick == g.raceTicks && d.team == g.team && d.input == input {
		return
	}
	d.render(g.team, input, d.tick != g.raceTicks)
	d.tick, d.team, d.input, d.validPixels = g.raceTicks, g.team, input, true
}
