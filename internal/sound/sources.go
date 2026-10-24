package sound

import (
	"encoding/binary"
	"fmt"

	"github.com/olivierh59500/indianapolis500/internal/traffic"
)

// Sources keeps the original moving averages and voice identities as cars pass
// the listener. Curve contains recovered speed-to-pitch values.
type Sources struct {
	Previous [3]int
	History  [3][8]uint16
	Sums     [3]uint16
	Cursor   [3]uint8
	Curve    [75]uint16
}

func NewSources(curve []byte) (Sources, error) {
	s := Sources{Previous: [3]int{-1, -1, -1}}
	if len(curve) != len(s.Curve)*2 {
		return s, fmt.Errorf("invalid recovered engine pitch curve")
	}
	for i := range s.Curve {
		s.Curve[i] = binary.BigEndian.Uint16(curve[i*2:])
	}
	return s, nil
}

func soundTrackGap(player, other uint16) int16 {
	difference := int32(player) - int32(other)
	if difference < -19800 {
		difference += 39600
	}
	if difference >= 19800 {
		difference -= 39600
	}
	return int16(difference)
}

type engineNeighbor struct {
	index          int
	distance       uint16
	ahead, lateral int16
}

// Driving selects the two nearest audible engines in the original track ring.
// Pitch changes over four audio updates; no extra vehicle motion is simulated.
func (s *Sources) Driving(c *Controller, field *traffic.Model, rpm, throttle uint16, reducedPitch, quiet bool) {
	target := rpm * 3
	if reducedPitch {
		target -= target >> 2
	}
	c.PitchStep[0] = int16(target-c.Pitch[0]) >> 2
	c.EngineLevel[0] = uint8(throttle>>12) + 107
	if quiet {
		c.EngineLevel[0] = 0
		c.Pitch[0] = 2048
	}
	neighbors := [2]engineNeighbor{{index: -1, distance: 2000}, {index: -1, distance: 2000}}
	player := &field.Cars[0]
	for pass := 0; pass < 2; pass++ {
		index := player.Previous
		if pass == 1 {
			index = player.Next
		}
		for count := 0; index != 0 && count < 32; count++ {
			other := &field.Cars[index]
			current := index
			index = other.Previous
			if pass == 1 {
				index = other.Next
			}
			if other.RetiredReason != 0 {
				continue
			}
			gap := soundTrackGap(player.Distance, other.Distance)
			if pass == 1 {
				gap--
			}
			if pass == 0 && gap < 0 || pass == 1 && gap >= 0 {
				break
			}
			lateral := int16(other.Lateral-player.Lateral) >> 5
			side := lateral
			if side < 0 {
				side = -side
			}
			distance := uint16(side) >> 1
			if pass == 0 {
				distance += uint16(gap)
			} else {
				distance -= uint16(gap)
			}
			if distance > neighbors[1].distance {
				break
			}
			candidate := engineNeighbor{index: current, distance: distance, ahead: gap, lateral: lateral}
			neighbors[1] = candidate
			if neighbors[0].distance >= distance {
				neighbors[0], neighbors[1] = neighbors[1], neighbors[0]
			}
		}
	}
	if neighbors[0].index == s.Previous[1] || neighbors[1].index == s.Previous[0] {
		neighbors[0], neighbors[1] = neighbors[1], neighbors[0]
	}
	for voice, neighbor := range neighbors {
		output := voice + 1
		level := uint8(0)
		if neighbor.index >= 0 {
			other := &field.Cars[neighbor.index]
			speed := int16((int16(other.Speed)>>1)-(field.PlayerMovement>>1)) >> 7
			sideways := int16(other.LateralVelocity-player.LateralVelocity) >> 5
			radial := int32(speed)*int32(neighbor.ahead) + int32(sideways)*int32(neighbor.lateral)
			distance := neighbor.distance + 1
			factor := uint16(int16(211 - int16(radial/int32(int16(distance)))))
			pitch := s.Curve[min(int(other.Speed>>8), 74)]
			if other.Number&128 != 0 {
				pitch = (pitch + 256) >> 1
			}
			if neighbor.index != s.Previous[voice] {
				for i := range s.History[voice] {
					s.History[voice][i] = pitch
				}
				s.Sums[voice] = pitch << 3
				c.Pitch[output] = uint16(uint32(s.Sums[voice]) * uint32(factor) >> 8)
			}
			cursor := s.Cursor[voice]
			s.Sums[voice] += pitch - s.History[voice][cursor]
			s.History[voice][cursor] = pitch
			s.Cursor[voice] = (cursor + 1) & 7
			goal := uint16(uint32(s.Sums[voice]) * uint32(factor) >> 8)
			c.PitchStep[output] = int16(goal-c.Pitch[output]) >> 2
			gain := uint8(119)
			if int16(distance) >= 80 {
				gain = uint8(9500 / uint32(distance))
			}
			level = gain - 3
			if int8(level) < 0 {
				level = 0
			}
		}
		c.EngineLevel[output] = level
		s.Previous[voice] = neighbor.index
	}
}

// Reset retains the immutable recovered curve while clearing voice histories.
func (s *Sources) Reset() { curve := s.Curve; *s = Sources{Previous: [3]int{-1, -1, -1}, Curve: curve} }

// ResetSelection follows the pause reset while retaining the original moving
// averages. A newly selected passing car replaces its history when needed.
func (s *Sources) ResetSelection() {
	s.Previous = [3]int{-1, -1, -1}
	s.Cursor = [3]uint8{}
}
