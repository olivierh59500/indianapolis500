package traffic

import (
	"encoding/binary"
	"fmt"
)

func (m *Model) LoadDriverCurves(spinStrength, trigonometry, retirementReasons []byte) error {
	if len(spinStrength) != 512 || len(trigonometry) != 2048 || len(retirementReasons) != 64 {
		return fmt.Errorf("invalid recovered driver lifecycle curves")
	}
	for i := range m.spinStrength {
		m.spinStrength[i] = binary.BigEndian.Uint16(spinStrength[i*2:])
	}
	for i := range m.trigonometry {
		m.trigonometry[i] = int16(binary.BigEndian.Uint16(trigonometry[i*2:]))
	}
	copy(m.retirementReasons[:], retirementReasons)
	return nil
}

// UpdateDriver implements ordinary source driver callbacks. Spin and stopped
// callbacks bypass the lane steering law, as their original return paths do.
func (m *Model) UpdateDriver(index int) {
	c := &m.Cars[index]
	switch c.Action {
	case Spinning:
		m.updateSpin(c)
	case Stopped:
		c.Acceleration, c.LateralAcceleration, c.Speed, c.LateralVelocity, c.YawRate = 0, 0, 0, 0, 0
	case Pitting, PittingUnderCaution:
		oldFlag := c.PitFlag
		target, steer := m.pitTarget(index)
		if !steer {
			return
		}
		if c.Action == PittingUnderCaution {
			if oldFlag != c.PitFlag {
				m.changeCautionMembership(index, c.PitFlag)
			}
			acceleration := c.Acceleration
			target = m.cautionTarget(index, target)
			c.Acceleration = min(acceleration, c.Acceleration)
		}
		m.steer(c, target)
	case FollowingCaution:
		target := m.racingTarget(index)
		m.steer(c, m.cautionTarget(index, target))
	default:
		if m.UpdateRacing(index) {
			c.YawRate += int16(m.random()&511) - 100
			m.registerSpin(index)
			m.updateSpin(c)
		}
	}
}

func (m *Model) updateSpin(c *Car) {
	oldYaw := c.Yaw
	angle := uint16(oldYaw)
	if oldYaw < 0 {
		angle = uint16(-oldYaw)
	}
	if angle > 16384 {
		angle = 16384 - angle
	}
	strength := m.spinStrength[angle>>8]
	speedSample := c.Speed >> 8
	delta := (int16(50) - int16(speedSample)) * 4
	if c.YawRate >= 0 {
		delta = -delta
	}
	c.YawRate += delta
	force := int16(uint32(strength) * uint32(max(m.yaw[speedSample], 1500)) >> 16)
	angleIndex := uint16(oldYaw+31) >> 6
	cosine := m.trigonometry[angleIndex]
	sine := m.trigonometry[(angleIndex+768)&1023]
	forward := int16(int32(force) * int32(cosine) >> 14)
	sideways := int16(int32(force) * int32(sine) >> 14)
	friction := abs16(sideways) + uint16(uint32(m.drag[speedSample])*uint32(c.Drag)>>16)
	friction = max(friction, 800) >> 1
	if c.Speed < friction {
		c.Speed, c.YawRate, c.LateralVelocity, c.LateralAcceleration = 0, 0, 0, 0
		c.TargetLateral = c.Lateral
		return
	}
	c.Speed -= friction
	component := int16(int8(uint16(-forward*4) >> 8))
	magnitude := min(abs16(component), c.Speed>>6)
	component = int16(magnitude)
	if int8(uint16(-forward*4)>>8) < 0 {
		component = -component
	}
	if oldYaw < 0 {
		component = -component
	}
	c.LateralAcceleration = component
	c.TargetLateral = c.Lateral
}

func (m *Model) cautionTarget(index int, target int16) int16 {
	c := &m.Cars[index]
	if c.CautionAhead < 0 {
		c.Acceleration >>= 2
		if c.Speed >= 6760 {
			c.Acceleration = 0
		}
		return target
	}
	c.TargetLateral = target
	other := m.Cars[c.CautionAhead]
	difference := int(c.Distance) - int(other.Distance)
	borrowed := difference < 0
	if borrowed {
		difference += TrackLength
	}
	lapDifference := int8(other.Laps - c.Laps)
	if borrowed {
		lapDifference--
	}
	difference += int(lapDifference) * TrackLength
	if difference >= 4000 {
		return c.TargetLateral
	}
	difference = max(difference, -19800)
	c.Acceleration >>= 1
	reduction := int16(c.Speed-other.Speed) - int16(difference-360)
	if reduction >= 0 {
		if c.Speed < 4880 {
			reduction = 0
		}
		c.Acceleration = max(c.Acceleration-reduction, -1100)
	}
	return c.TargetLateral
}

func (m *Model) changeCautionMembership(index int, leaving bool) {
	c := &m.Cars[index]
	ahead := c.CautionAhead
	if leaving {
		other := c.Previous
		for scanned := 0; other != index && scanned < len(m.Cars); scanned++ {
			car := &m.Cars[other]
			if car.CautionAhead == index {
				car.CautionAhead = ahead
				if ahead < 0 && !car.PitFlag {
					m.CautionLeader = other
				}
			}
			other = car.Previous
		}
		return
	}
	if ahead < 0 {
		ahead = m.CautionLeader
		if ahead < 0 || m.Cars[ahead].PitFlag || m.cautionGap(index, ahead) < 0 {
			if ahead >= 0 && !m.Cars[ahead].PitFlag {
				m.Cars[ahead].CautionAhead = index
			}
			m.CautionLeader = index
			return
		}
	}
	if c.CautionAhead < 0 || m.cautionGap(index, c.CautionAhead) >= 0 {
		other := index
		for scanned := 0; scanned < len(m.Cars); scanned++ {
			other = m.Cars[other].Previous
			car := &m.Cars[other]
			if car.PitFlag {
				if car.Speed == 0 || int16(car.Distance) >= 0 || car.Distance >= 37290 {
					continue
				}
				car.PitFlag = false
				m.changeCautionMembership(other, false)
			}
			ahead, c.CautionAhead, c.Laps = other, other, car.Laps
			if c.Distance < car.Distance {
				c.Laps--
			}
			break
		}
	}
	other := c.Previous
	for scanned := 0; other != index && scanned < len(m.Cars); scanned++ {
		car := &m.Cars[other]
		if car.CautionAhead == ahead {
			car.CautionAhead = index
		}
		other = car.Previous
	}
}

func (m *Model) pitTarget(index int) (int16, bool) {
	c := &m.Cars[index]
	d := c.Distance
	if int8(c.PitLaps) <= 0 {
		if c.PitLaps == c.ServiceWait {
			if d >= 9000 {
				if m.UpdateRacing(index) {
					c.YawRate += int16(m.random()&511) - 100
					m.registerSpin(index)
					m.updateSpin(c)
				}
				return 0, false
			}
			random := m.random()
			wait := random&0xff00 | uint16(uint8(random)&127+128)
			if m.Rules != 0 {
				wait = wait&0xff00 | uint16(uint8(wait)>>1)
				if wait >= 6656 {
					wait = uint16(uint8(wait)&15) + 24
				}
			}
			c.ServiceWait = uint8(wait)
		}
		difference := gap(*c, Car{Distance: c.PitDistance})
		if difference <= 1 {
			c.Speed, c.Yaw, c.YawRate, c.LateralVelocity = 0, 0, 0, 0
			random := m.random()
			delay := uint8(random & 31)
			if c.ServiceWait < 64 {
				delay = uint8(random&3) + 29
			}
			c.PitLaps += delay + 2
			return m.pitWait(index), true
		}
		if difference <= 4000 {
			sample := c.Speed >> 8
			brake := int16(uint32(m.drag[sample]) * 840 / uint32(uint16(difference)))
			if brake <= 500 {
				brake = -300
			}
			c.Acceleration = -brake
			if difference <= 150 {
				if int8(c.Speed>>8) <= 0 {
					return m.parkAtPit(index), true
				}
				return 624, true
			}
			if int16(c.Speed) <= 6000 && difference <= 1000 {
				return m.pitExitTarget(c, uint16(difference)), true
			}
		} else {
			c.Acceleration = c.BaseAcceleration
			if !c.PitFlag {
				c.PreferredLane = 0
				target := m.racingTarget(index)
				if c.Lane != 0 {
					if c.Acceleration >= 0 {
						c.Acceleration = int16(uint16(c.Acceleration) >> 1)
					}
					return target, true
				}
				c.PitFlag = true
			}
		}
	} else {
		return m.pitWait(index), true
	}
	return m.avoidPitTraffic(index, m.pitRoute(c.Distance)), true
}

func (m *Model) parkAtPit(index int) int16 {
	c := &m.Cars[index]
	c.Speed, c.Yaw, c.YawRate, c.LateralVelocity = 0, 0, 0, 0
	random := m.random()
	delay := uint8(random & 31)
	if c.ServiceWait < 64 {
		delay = uint8(random&3) + 29
	}
	c.PitLaps += delay + 2
	return m.pitWait(index)
}

func (m *Model) pitWait(index int) int16 {
	c := &m.Cars[index]
	if c.ServiceWait != 0 {
		if m.Rules != 0 {
			if c.Flags&1 != 0 {
				c.Action = Stopped
			} else if c.ServiceWait >= 96 {
				c.RetiredReason = m.retirementReasons[m.random()&63]
				c.Action, c.Acceleration = Stopped, 0
				return c.TargetLateral
			}
		}
		c.ServiceWait--
		c.Acceleration = 0
		return c.TargetLateral
	}
	c.Acceleration = c.BaseAcceleration
	if int16(c.Distance) < 0 && c.Distance < 37290 {
		c.PitFlag = false
	}
	if int16(c.Speed) <= 4500 {
		return m.pitExitTarget(c, c.Distance)
	}
	if c.Distance >= 25000 && c.Distance < 27000 {
		c.Action = Racing
		if m.Yellow {
			c.Action = FollowingCaution
		}
		c.Lane, c.PreferredLane = 0, c.SavedLane
		return m.avoidPitTraffic(index, 4880)
	}
	return m.avoidPitTraffic(index, m.pitRoute(c.Distance))
}

func (m *Model) pitExitTarget(c *Car, distance uint16) int16 {
	if distance < 4028 || distance >= 37484 {
		return 1808
	}
	return c.TargetLateral
}

func (m *Model) pitRoute(distance uint16) int16 {
	switch {
	case distance >= 37484:
		return 3008
	case distance >= 36460:
		return int16(3008 - (distance - 37484))
	case distance >= 25000:
		return 4032
	case distance >= 7900:
		return 4880
	case distance >= 6028:
		return int16(distance - 3020)
	default:
		return 3008
	}
}

func (m *Model) avoidPitTraffic(index int, target int16) int16 {
	c := m.Cars[index]
	if abs16(target-c.TargetLateral) >= 832 {
		next := c.Next
		for next != index {
			other := m.Cars[next]
			d := int16(gap(c, other) - 1)
			if uint16(d) < 0xf830 {
				break
			}
			shifted := d + 50
			relative := int16(c.Speed - other.Speed)
			if shifted < 0 && (relative >= 0 || shifted < relative>>3) {
				next = other.Next
				continue
			}
			if abs16(target-other.TargetLateral) < 832 {
				target = c.TargetLateral
				break
			}
			next = other.Next
		}
	}
	next := c.Previous
	for next != index {
		other := m.Cars[next]
		d := uint16(gap(c, other))
		if d >= 1000 {
			break
		}
		if abs16(target-other.TargetLateral) < 832 {
			reduction := int16(c.Speed-other.Speed) + int16((int16(d)-180)*-2)
			if reduction > 0 {
				braking := max(-reduction, -1800)
				if braking <= m.Cars[index].Acceleration {
					m.Cars[index].Acceleration = braking
				}
				break
			}
		}
		next = other.Previous
	}
	return target
}
