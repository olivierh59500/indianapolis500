package game

import "github.com/olivierh59500/indianapolis500/internal/vehicle"

func gearSetup(top int) [4]int {
	return [4]int{top << 16 / 0x6e14, top << 16 / 0xab85, top << 16 / 0xfae1, top}
}

// physicalConfig converts the original setup detents into their recovered
// integer coefficients. It adds no handling multipliers or speed limits.
// demonstrationCarSetup uses the same detents available through the original
// setup pages. The lower-revving Buick retains an ordinary longer top gear.
func demonstrationCarSetup(team int) carSetup {
	setup := defaultCarSetup()
	setup.wings = [2]int{10, 10}
	top := 440
	if team == 0 {
		top = 320
	}
	setup.gears = gearSetup(top)
	return setup
}

func (g *Game) physicalConfig() vehicle.Config {
	config := g.vehiclePreset.Config
	setup, boost := g.setup, g.boost
	frontRollBar, rearRollBar := g.frontRollBar, g.rearRollBar
	if g.demo {
		// The demonstration selects ordinary menu settings without replacing the
		// player's saved setup or changing any recovered handling coefficient.
		setup, boost = demonstrationCarSetup(g.team), 9
		frontRollBar, rearRollBar = 5, 1
		config.Engine = vehicle.EngineForTeam(config.Engine, g.team)
	}
	if g.trafficModel != nil && g.trafficModel.Yellow {
		config.ThrottleLimit = g.trafficModel.PlayerThrottleLimit
	}
	config.BoostTarget = uint16((boost-1)*0x800 + 0xbfff)
	config.Aero.FrontWingDrag = uint16(7672 + setup.wings[0]*383)
	config.Aero.RearWingDrag = uint16(12280 + setup.wings[1]*795)
	heat := [3]uint16{11034, 8220, 6930}
	optimum := [3]uint16{22368, 22688, 23008}
	rearGrip := [3]uint16{57400, 56000, 54600}
	frontGrip := [3]uint16{54460, 53200, 51940}
	for row, wheel := range informationTireOrder {
		compound := setup.rubber[row]
		tire := &config.Tires[wheel]
		tire.HeatGain, tire.OptimumTemperature = heat[compound], optimum[compound]
		config.PeakGrip[wheel] = rearGrip[compound]
		if wheel >= 2 {
			tire.HeatGain = uint16(uint32(tire.HeatGain)*0xb99a>>16) * 2
			config.PeakGrip[wheel] = frontGrip[compound]
		}
		tire.ColdPressure = uint16(setup.pressures[row])
		config.Suspension.Springs[wheel] = uint16(2047 + 256*setup.shocks[row])
		config.Suspension.Camber[wheel] = int16(setup.cambers[row]*32 - 256)
		config.Engine.Gears[row] = uint16(uint32(setup.gears[row]) << 16 / 1199)
	}
	springs := config.Suspension.Springs
	front := uint16(1024+896*frontRollBar) + springs[2] + springs[3]
	rear := uint16(1024+896*rearRollBar) + springs[0] + springs[1]
	config.Suspension.RollStiffness = front + rear
	config.Suspension.PitchStiffness = springs[0] + springs[1] + springs[2] + springs[3]
	config.Suspension.RollDistribution = int16(uint32(front) << 15 / uint32(config.Suspension.RollStiffness))
	config.Suspension.PitchDistribution = int16(uint32(springs[0]+springs[2]) << 15 / uint32(config.Suspension.PitchStiffness))
	return config
}

func (g *Game) applyLiveSetup(previous carSetup) {
	player := &g.cars[0]
	if !player.physical || g.demo {
		return
	}
	config := g.physicalConfig()
	if previous.fuel != g.setup.fuel {
		g.refuelPlayer()
	}
	for row, wheel := range informationTireOrder {
		if previous.rubber[row] != g.setup.rubber[row] {
			g.serviceTire(wheel)
		} else if previous.pressures[row] != g.setup.pressures[row] {
			g.applyTirePressure(row)
		}
	}
	player.physics.Engine.Ratio = config.Engine.Gears[player.physics.Engine.Gear]
	g.syncVehicleCondition()
}

func (g *Game) applyTirePressure(row int) {
	config := g.physicalConfig()
	wheel := informationTireOrder[row]
	tire := &g.cars[0].physics.Tires[wheel]
	tire.Pressure = uint16(uint32(config.Tires[wheel].ColdPressure) * uint32(tire.Bands[1]) / uint32(config.Ambient))
}
