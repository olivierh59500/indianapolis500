package sound

import (
	"github.com/olivierh59500/indianapolis500/internal/scene"
	"github.com/olivierh59500/indianapolis500/internal/vehicle"
)

// MovingSource supplies the two recorded poses used by original replay audio.
// IDs remain ordinary car identities; the caller adds33 for replay emitters.
type MovingSource struct {
	ID                                int
	Distance, PreviousDistance        uint16
	Lateral, PreviousLateral, Heading int16
	LowEngine                         bool
}

type Listener struct {
	Distance        uint16
	Lateral, Motion int16
	View            int8
	Observed        int
	Enabled         bool
}

func audioAbs(value int16) int16 {
	if value < 0 {
		return -value
	}
	return value
}

func (s *Sources) exteriorPitch(c *Controller, history, voice int, actor MovingSource, factor uint16, speed int16) {
	pitch := s.Curve[min(int(uint16(audioAbs(speed))), 74)]
	if actor.LowEngine {
		pitch = (pitch + 256) >> 1
	}
	if actor.ID != s.Previous[history] {
		for i := range s.History[history] {
			s.History[history][i] = pitch
		}
		s.Sums[history] = pitch << 3
		c.Pitch[voice] = uint16(uint32(s.Sums[history]) * uint32(factor) >> 8)
	}
	cursor := s.Cursor[history]
	s.Sums[history] += pitch - s.History[history][cursor]
	s.History[history][cursor] = pitch
	s.Cursor[history] = (cursor + 1) & 7
	target := uint16(uint32(s.Sums[history]) * uint32(factor) >> 8)
	c.PitchStep[voice] = int16(target-c.Pitch[voice]) >> 2
	s.Previous[history] = actor.ID
}

func exteriorGainDistance(distance uint16, side int16, view int8) uint16 {
	if view >= 3 {
		distance -= uint16(side)
		distance += uint16(side) >> 1
		distance -= 90
	}
	return distance
}

// Exterior follows recorded car motion, relative listener velocity and the
// original camera-distance gain, instead of assigning a stationary engine RPM
// to every view of an instant replay.
func (s *Sources) Exterior(c *Controller, actors []MovingSource, listener Listener, world *scene.World, surface *vehicle.TrackSurface) {
	if !listener.Enabled {
		s.stop(c)
		return
	}
	motion := listener.Motion
	if listener.View != 0 && listener.View != 1 && listener.View != 4 {
		motion = 0
	}
	neighbors := [2]engineNeighbor{{index: -1, distance: 2600}, {index: -1, distance: 2600}}
	primary := -1
	for i, actor := range actors {
		if actor.ID == listener.Observed {
			primary = i
			continue
		}
		if actor.Distance == actor.PreviousDistance {
			continue
		}
		ahead := soundTrackGap(listener.Distance, actor.Distance)
		lateral := int16((actor.Lateral>>4)-listener.Lateral) >> 1
		distance := uint16(audioAbs(ahead)) + (uint16(audioAbs(lateral)) >> 1)
		if distance >= neighbors[1].distance {
			continue
		}
		candidate := engineNeighbor{index: i, distance: distance, ahead: ahead, lateral: lateral}
		neighbors[1] = candidate
		if neighbors[0].distance > distance {
			neighbors[0], neighbors[1] = neighbors[1], neighbors[0]
		}
	}
	id := func(index int) int {
		if index < 0 {
			return -1
		}
		return actors[index].ID
	}
	if id(neighbors[0].index) == s.Previous[1] || id(neighbors[1].index) == s.Previous[0] {
		neighbors[0], neighbors[1] = neighbors[1], neighbors[0]
	}
	for i, neighbor := range neighbors {
		c.EngineLevel[i+1] = 0
		if neighbor.index < 0 {
			s.Previous[i] = -1
			continue
		}
		actor := actors[neighbor.index]
		speed := soundTrackGap(actor.Distance, actor.PreviousDistance)
		sideVelocity := int16(actor.PreviousLateral-actor.Lateral) >> 5
		distance := neighbor.distance
		if listener.View >= 3 {
			distance += 140
		}
		distance++
		radial := int32(int16(speed-motion))*int32(neighbor.ahead) + int32(sideVelocity)*int32(neighbor.lateral)
		factor := uint16(int16(211 - int16(radial/int32(int16(distance)))))
		s.exteriorPitch(c, i, i+1, actor, factor, speed)
		distance = exteriorGainDistance(distance, audioAbs(neighbor.lateral)>>1, listener.View)
		gain := uint16(128)
		if int16(distance) >= 78 {
			gain = uint16(10000 / uint32(distance))
		}
		gain -= 3
		if int16(gain) < 0 {
			gain = 0
		}
		c.EngineLevel[i+1] = uint8(gain)
	}
	if primary < 0 {
		return
	}
	actor := actors[primary]
	ahead := soundTrackGap(listener.Distance, actor.Distance)
	side := int16((actor.Lateral>>4)-listener.Lateral) >> 1
	sideDistance := audioAbs(side) >> 1
	distance := uint16(audioAbs(ahead)) + uint16(sideDistance)
	speed := soundTrackGap(actor.Distance, actor.PreviousDistance)
	sideVelocity := int16(actor.PreviousLateral-actor.Lateral) >> 5
	if listener.View >= 3 {
		distance += 140
	}
	distance++
	radial := int32(int16(speed-motion))*int32(ahead) + int32(sideVelocity)*int32(side)
	factor := uint16(int16(211 - int16(radial/int32(int16(distance)))))
	s.exteriorPitch(c, 2, 0, actor, factor, speed)
	distance = exteriorGainDistance(distance, sideDistance, listener.View)
	gain := uint16(130)
	if int16(distance) >= 77 {
		gain = uint16(10000 / uint32(distance))
	}
	gain -= 3
	if int16(gain) < 0 {
		gain = 0
	}
	c.EngineLevel[0] = uint8(gain)
	progress := uint16(6930 - actor.Distance)
	if int16(progress) < 0 {
		progress += 39600
	}
	skid := uint8(0)
	if !surface.Sample(progress, actor.Lateral>>4).OffRoad && uint16(audioAbs(speed)+audioAbs(sideVelocity)) > 2 {
		slip := audioAbs(int16(world.Direction(speed, sideVelocity) - actor.Heading))
		if uint16(slip) >= 16384 {
			slip = int16(32768 - uint16(slip))
		}
		if uint16(slip) >= 400 {
			skid = 127
			if uint16(slip-400) < 1778 {
				skid = uint8(uint16(slip-400) / 14)
			}
		}
		c.NoisePitch = int16(uint16(audioAbs(speed))<<5) + 7168
		if distance > 60 {
			skid = uint8(uint32(skid) * 60 / uint32(distance))
		}
	}
	c.NoiseStep = int8(skid - c.NoiseLevel)
}

func (s *Sources) stop(c *Controller) {
	c.Pitch = [3]uint16{2048, 2048, 2048}
	c.PitchStep = [3]int16{}
	c.EngineLevel = [3]uint8{}
	c.NoisePitch, c.NoiseDrift, c.NoiseAccumulator, c.NoiseLevel, c.NoiseStep = 8192, 0, 0, 0, 0
	c.RandomCountdown, c.Requests, c.EffectTicks, c.Priority = 30, [4]uint8{}, 0, 0
	c.Voices[3].Sample, c.Voices[3].Level = NoiseSample, 0
	c.Voices[3].Restart++
	s.Previous, s.Cursor = [3]int{-1, -1, -1}, [3]uint8{}
}
