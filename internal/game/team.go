package game

import "github.com/olivierh59500/indianapolis500/internal/vehicle"

// selectTeam is a setup callback, rather than a per-tick coefficient override.
// Existing physical state and selected gear ratios survive the original action.
func (g *Game) selectTeam(team int) {
	g.team = team
	g.vehiclePreset.Config.Engine = vehicle.EngineForTeam(g.vehiclePreset.Config.Engine, team)
	if g.trafficModel != nil {
		g.trafficModel.Cars[0].Number = vehicle.PlayerCarIdentity(team)
		g.trafficModel.Cars[0].Primary, g.trafficModel.Cars[0].Secondary = vehicle.PlayerCarColors(team)
	}
	primary, secondary := vehicle.PlayerCarColors(team)
	g.cars[0].color, g.cars[0].secondary = int(primary), int(secondary)
	g.dashboard.validPixels = false
}
