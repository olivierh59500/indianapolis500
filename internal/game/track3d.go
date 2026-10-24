package game

import (
	"math"
)

func (g *Game) presentedCar(index int) car {
	vehicle := g.cars[index]
	if !vehicle.present() {
		return vehicle
	}
	fraction := float64(g.simulationPhase) / (PresentationTicksPerSecond / SimulationTicksPerSecond)
	vehicle.position = math.Mod(vehicle.position+vehicle.speed*progressPerMPHTick*fraction+trackLength, trackLength)
	vehicle.lateral += vehicle.lateralVelocity * fraction
	vehicle.yaw += int16(float64(vehicle.angularVelocity) * fraction)
	if vehicle.worldPose {
		vehicle.worldX += int16(math.Round(float64(vehicle.worldVX) * fraction / 128))
		vehicle.worldY += int16(math.Round(float64(vehicle.worldVY) * fraction / 128))
		if vehicle.physical {
			point := g.world.TrackPlacement(uint16(originalTrackProgress(vehicle.position)), int16(vehicle.lateral))
			vehicle.yaw = vehicle.physics.Body.Heading + int16(float64(vehicle.angularVelocity)*fraction) - point.Heading
		}
	}
	return vehicle
}

// Native forward distance increases along the source world. The original
// lap-distance field decreases, so it must not be added with the same sign.
func originalTrackProgress(position float64) float64 {
	p := math.Mod(6930+position, trackLength)
	if p < 0 {
		p += trackLength
	}
	return p
}

func (g *Game) trackBounds(position float64) (float64, float64) {
	inner, outer := g.world.TrackLimits(uint16(originalTrackProgress(position)))
	return float64(inner) * 16, float64(outer) * 16
}

// The source pit-service flag uses this measured parking interval, not a
// widened preferred driving lane ($17ee, original source around $4110).
func (g *Game) playerInPitLane() bool {
	player := &g.cars[0]
	p := originalTrackProgress(player.position)
	return p > 6930 && p < 7080 && player.lateral < 75*16
}

// Recovered racing lines are indexed by decreasing lap distance. Native
// forward position increases, so using it directly reverses the bend samples.
func racingLineSample(position float64) int {
	distance := math.Mod(-position, trackLength)
	if distance < 0 {
		distance += trackLength
	}
	return int(distance) >> 7
}
