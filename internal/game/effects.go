package game

import (
	"github.com/olivierh59500/indianapolis500/internal/debris"
	"github.com/olivierh59500/indianapolis500/internal/traffic"
)

// Wall effects use the original prepared car record's motion interval rather
// than deriving their velocity from a corrected world-space contact point.
func (g *Game) wallCollisionEffect(severity uint16) debris.Input {
	state := &g.cars[0].physics
	old := &g.trafficModel.Cars[0]
	distance := uint16(6930 - state.Progress)
	if int16(distance) < 0 {
		distance += 39600
	}
	motion := int16(old.Distance>>1) - int16(distance>>1)
	if motion <= -200 {
		motion += 19800
	} else if motion >= 200 {
		motion -= 19800
	}
	lateral := state.Lateral * 16
	return debris.Input{Severity: severity, Colors: uint16(old.Primary)<<8 | uint16(old.Secondary), Sideways: lateral - old.Lateral, Forward: motion << 8, Lateral: lateral, Distance: distance}
}

func (g *Game) emitWallCollisionEffects(severity uint16) {
	camera, _ := g.originalCamera(g.soundView())
	g.effects.Spawn(g.wallCollisionEffect(severity), g.world, camera, &g.soundController.Requests)
}

func (g *Game) emitTrafficCollisionEffects(effect traffic.CollisionEffect) {
	camera, _ := g.originalCamera(g.soundView())
	g.effects.Spawn(debris.Input(effect), g.world, camera, &g.soundController.Requests)
}
