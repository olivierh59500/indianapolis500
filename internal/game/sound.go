package game

import (
	worldscene "github.com/olivierh59500/indianapolis500/internal/scene"
	"github.com/olivierh59500/indianapolis500/internal/sound"
	"math"
)

// setPaused resets the driving sound controls on each real pause transition.
// Foreground muted PCM continues at the retained engine rates; Android may
// separately suspend its audio backend while the application is in background.
func (g *Game) setPaused(paused bool) {
	if g.paused == paused {
		return
	}
	g.paused = paused
	g.resetDrivingSoundControls()
}

// resetDrivingSoundControls follows AF06 for pause and menu transitions.
// Engine waveforms retain their phase; only the noise waveform restarts.
func (g *Game) resetDrivingSoundControls() {
	g.soundController.ResetDrivingControls()
	g.soundSources.ResetSelection()
}

// The original audio clock runs before its divided physics interrupt. Even a
// muted build advances the shared random sequence at the same game updates.
func (g *Game) advanceSound() {
	if g.scene == driveScene && !g.paused {
		g.soundController.Step(g.trafficModel.NextRandom)
	}
}

func (g *Game) updateSound() {
	g.audio.SetFrame(g.soundController.Voices, g.scene == driveScene && !g.paused)
}

func (g *Game) currentSoundPosition(index int) sound.MovingSource {
	c := &g.cars[index]
	distance := uint16((trackLength - int(math.Round(c.position))%trackLength) % trackLength)
	return sound.MovingSource{ID: 33 + index, Distance: distance, Lateral: int16(math.Round(c.lateral)), Heading: c.yaw, LowEngine: g.trafficModel.Cars[index].Number&128 != 0}
}

func (g *Game) rememberSoundPositions() {
	for i := range g.cars {
		g.soundPrevious[i] = g.currentSoundPosition(i)
	}
}

func (g *Game) updateSoundSources() {
	player := &g.cars[0]
	if !g.demo && !g.replay {
		g.soundSources.Driving(&g.soundController, g.trafficModel, player.physics.Engine.RPM, player.driver.Throttle, g.team == 0, player.physics.Engine.Failures&32 != 0)
	} else {
		view := g.soundView()
		_, context := g.originalCamera(view)
		distance := uint16(6930 - context.Progress)
		if int16(distance) < 0 {
			distance += 39600
		}
		views := [6]int8{3, 4, 2, 1, 0, 5}
		listener := sound.Listener{Distance: distance, Lateral: context.Lateral, Motion: player.physics.FrontGaugeSpeed >> 8, View: views[view], Observed: 33, Enabled: true}
		var actors [33]sound.MovingSource
		count, index := 0, 0
		for scanned := 0; scanned < 33; scanned++ {
			if g.cars[index].present() {
				actor := g.currentSoundPosition(index)
				actor.PreviousDistance, actor.PreviousLateral = g.soundPrevious[index].Distance, g.soundPrevious[index].Lateral
				actors[count] = actor
				count++
			}
			index = g.trafficModel.Cars[index].Next
			if index == 0 {
				break
			}
		}
		g.soundSources.Exterior(&g.soundController, actors[:count], listener, g.world, &g.vehicleSurface)
	}
	g.rememberSoundPositions()
}

func (g *Game) requestImpactSound(severity uint16, x, y int32) {
	level := uint16(127)
	if severity < 150 {
		level = (severity >> 3) + 60
	} else if severity < 400 {
		level = (severity >> 3) + 75
	}
	camera, _ := g.originalCamera(g.soundView())
	distance := worldscene.VelocityMagnitude(worldscene.Vector2{X: int16(x>>16) - camera.X, Y: int16(y>>16) - camera.Y})
	if distance > 120 {
		level = uint16(uint32(level) * 120 / uint32(distance))
	}
	g.soundController.Requests[0] = max(g.soundController.Requests[0], uint8(level))
}

func (g *Game) soundView() int {
	if g.demo {
		return demoViewForSecond(g.tick / TicksPerSecond % 60)
	}
	if g.replay {
		return g.replayView
	}
	return inCarView
}
