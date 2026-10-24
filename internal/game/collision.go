package game

import (
	worldscene "github.com/olivierh59500/indianapolis500/internal/scene"
	"math"
)

// carWorldPose preserves the existing sixty-Hz interpolation without forcing
// odd world coordinates onto the replay routine's two-unit longitudinal grid.
func (g *Game) carWorldPose(position, lateral float64) worldscene.TrackPoint {
	progress := originalTrackProgress(position)
	first := g.world.TrackPlacement(uint16(progress), int16(lateral))
	second := g.world.TrackPlacement(uint16((int(progress)+1)%trackLength), int16(lateral))
	fraction := progress - math.Floor(progress)
	first.X += int16(math.Floor(float64(second.X-first.X) * fraction))
	first.Y += int16(math.Floor(float64(second.Y-first.Y) * fraction))
	first.Z += int16(math.Floor(float64(second.Z-first.Z) * fraction))
	return first
}
