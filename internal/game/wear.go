package game

import (
	"math"

	"github.com/olivierh59500/indianapolis500/internal/vehicle"
)

// Information pages use RF, LF, RR, LR; the force model uses RL, RR, FL, FR.
var informationTireOrder = [4]int{3, 2, 1, 0}
var tireFailureFlags = [4]byte{4, 2, 8, 1}

func (g *Game) syncVehicleCondition() {
	s := &g.cars[0].physics
	g.fuelGallons = float64(s.Engine.Fuel) / (32 * 65536)
	g.engineWear = max(0, 1-float64(s.Engine.Life)/(150*65536))
	for row, wheel := range informationTireOrder {
		tire := &s.Tires[wheel]
		life := 700.0
		if wheel >= 2 {
			life = 850
		}
		g.tireWear[row] = max(0, 1-float64(tire.Life)/(life*65536))
		if tire.Failed {
			g.tireWear[row] = 1
		}
		g.tireTemperature[row] = float64(tire.Bands[1])/32 - 459
	}
	g.failure = ""
	if s.Engine.Failures&vehicle.FuelEmpty != 0 {
		g.failure = "OUT OF FUEL"
	}
	if s.Engine.Failures&15 != 0 {
		g.failure = "TIRE BLEW"
	}
	if s.Engine.Failures&vehicle.EngineFailed != 0 {
		g.engineWear = 1
		g.failure = "ENGINE BLOWN"
	}
}

func (g *Game) refuelPlayer() {
	s := &g.cars[0].physics.Engine
	// The original refill replaces only whole units, retaining the low word.
	s.Fuel = uint32(g.setup.fuel*32)<<16 | s.Fuel&65535
	s.Mass = 7500 + uint16(s.Fuel>>16)
	s.Failures &^= vehicle.FuelEmpty
	g.syncVehicleCondition()
}

func (g *Game) servicePitItem(selection int) {
	if selection == 0 {
		g.refuelPlayer()
		return
	}
	if selection < 1 || selection > 4 {
		return
	}
	g.serviceTire(pitTireOrder[selection-1])
}

func (g *Game) serviceTire(wheel int) {
	s := &g.cars[0].physics
	tire := &s.Tires[wheel]
	life := uint32(700)
	if wheel >= 2 {
		life = 850
	}
	tire.Life, tire.Failed = life<<16, false
	s.Engine.Failures &^= tireFailureFlags[wheel]
	tire.Bands = [3]uint16{19488, 19488, 19488}
	coldPressure := uint16(0)
	for row, index := range informationTireOrder {
		if index == wheel {
			coldPressure = uint16(g.setup.pressures[row])
			break
		}
	}
	tire.Pressure = uint16(uint32(coldPressure) * 19488 / uint32(g.vehiclePreset.Config.Ambient))
	g.syncVehicleCondition()
}

// Car-to-car contact is applied to authoritative world velocity, so the next
// physical tick cannot discard the contact when it copies presentation data.
func (g *Game) reducePlayerSpeed(speed float64) {
	player := &g.cars[0]
	if !player.physical {
		player.speed = speed
		return
	}
	if math.Abs(player.speed) < 0.001 {
		return
	}
	s := &player.physics.Body
	ratio := max(0, min(1, speed/player.speed))
	s.VelocityX = int16(math.Round(float64(s.VelocityX) * ratio))
	s.VelocityY = int16(math.Round(float64(s.VelocityY) * ratio))
	s.Speed = uint16(math.Round(float64(s.Speed) * ratio))
	g.syncPlayerVehicle()
}
