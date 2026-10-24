package traffic

import (
	"encoding/binary"
	"fmt"
)

func (m *Model) LoadRoadLimits(inner, outer []byte) error {
	if len(inner) != 992 || len(outer) != 992 {
		return fmt.Errorf("invalid recovered roadside limits")
	}
	for i := range m.innerLimits {
		m.innerLimits[i] = int16(binary.BigEndian.Uint16(inner[i*2:]))
		m.outerLimits[i] = int16(binary.BigEndian.Uint16(outer[i*2:]))
	}
	return nil
}

// RoadLimits selects the original Q4 wreck limits from remaining distance.
// At the pit divider, the previous lateral position selects its road side.
func (m *Model) RoadLimits(distance uint16, previousLateral int16) (int16, int16) {
	pitLimits := func() (int16, int16) {
		if uint16(previousLateral) < 4800 {
			return 320, 4480
		}
		return 5120, 9280
	}
	profile := 0
	switch {
	case distance >= 36630:
		if distance < 37310 {
			return 320, 9280
		}
		return pitLimits()
	case distance >= 30690:
		if distance < 32670 {
			return 320, 10240
		}
		profile = 3960 - (int(distance) - 32670)
	case distance < 6930:
		if distance < 4130 {
			return pitLimits()
		}
		taper := uint32(6930 - distance)
		return 320 + int16(taper*49454>>16), 9280
	case distance < 12870:
		profile = int(distance) - 6930
		if profile >= 3960 {
			return 320, 10240
		}
	case distance < 16830:
		profile = 16830 - int(distance)
	default:
		profile = int(distance) - 16830
		if profile < 9900 {
			return 320, 9280
		}
		profile -= 9900
	}
	return m.innerLimits[profile>>3] << 4, m.outerLimits[profile>>3] << 4
}

// ConstrainWrecks runs after movement and ordering, before any car contacts.
// Wrecks retain their momentum and yaw response when hitting the road edge.
func (m *Model) ConstrainWrecks() {
	for _, index := range m.Wrecks[:m.WreckCount] {
		c := &m.Cars[index]
		previous := c.Lateral - c.LateralVelocity
		if m.flags[c.Distance>>8]&1 != 0 {
			c.LateralVelocity += int16(m.yaw[c.Speed>>8] >> 6)
		}
		speedBefore := c.Speed
		inner, outer := m.RoadLimits(c.Distance, previous)
		correction, outside := int16(0), false
		if inner > c.Lateral {
			correction = inner - c.Lateral
		} else if outer < c.Lateral {
			correction, outside = outer-c.Lateral, true
		}
		if correction != 0 {
			c.Lateral += correction
			c.LateralVelocity += correction
			force := correction * 4
			speed := int32(int16(c.Speed)) - int32(force)
			if outside {
				speed = int32(int16(c.Speed)) + int32(force)
			}
			c.Speed = uint16(max(int32(0), speed))
			if c.Yaw<<1 < 0 {
				force = -force
			}
			if outside {
				c.YawRate += force
			} else {
				c.YawRate -= force
			}
			c.LateralAcceleration = 0
			if m.OnCollisionEffect != nil {
				amount := uint16(correction * 4)
				if int16(amount) < 0 {
					amount = 0 - amount
				}
				amount >>= 2
				m.OnCollisionEffect(CollisionEffect{Severity: amount + (amount >> 1), Colors: uint16(c.Primary)<<8 | uint16(c.Secondary), Sideways: c.LateralVelocity, Forward: int16(c.Speed), Lateral: c.Lateral, Distance: c.Distance})
			}
		}
		barrier := uint16(0)
		if c.Distance >= 36900 && c.Distance < 37000 && c.Distance+speedBefore>>8 >= 36980 && c.Lateral <= 2440 {
			barrier = 36980
		} else if c.Distance >= 38230 && c.Distance < 38330 && c.Distance+speedBefore>>8 >= 38310 && c.Lateral >= 3336 && c.Lateral <= 4480 {
			barrier = 38330
		}
		if barrier != 0 {
			c.Distance = barrier
			if c.Speed != 0 && m.OnCollisionEffect != nil {
				m.OnCollisionEffect(CollisionEffect{Severity: uint16(abs16(c.LateralVelocity)) + uint16(int16(c.Speed)>>3), Colors: uint16(c.Primary)<<8 | uint16(c.Secondary), Lateral: c.Lateral, Distance: c.Distance})
			}
			c.Speed, c.LateralVelocity, c.LateralAcceleration = 0, 0, 0
			c.TargetLateral = c.Lateral
		}
	}
}
