package traffic

// updatePlayerDriver runs the player's separate callback after the NPC cycle.
func (m *Model) updatePlayerDriver() {
	switch m.Cars[0].Action {
	case Racing:
		m.updateRacingPlayer()
	case FollowingCaution:
		m.updatePlayerCaution()
	}
}

// cautionGap measures the original queue separation across wrapped lap epochs.
func (m *Model) cautionGap(index, ahead int) int32 {
	c, other := m.Cars[index], m.Cars[ahead]
	difference := int32(c.Distance) - int32(other.Distance)
	laps := int8(other.Laps - c.Laps)
	if difference < 0 {
		difference += TrackLength
		laps--
	}
	return difference + int32(laps)*TrackLength
}

// updatePlayerCaution follows $220DA. Its stored throttle fraction is consumed
// by the same engine model as ordinary accelerator input on following ticks.
func (m *Model) updatePlayerCaution() {
	c := &m.Cars[0]
	previous := c.PitFlag
	m.updateRacingPlayer()
	if c.PitFlag != previous {
		m.changeCautionMembership(0, c.PitFlag)
	}
	if c.CautionAhead < 0 {
		m.PlayerThrottleLimit = 0xffff
		if m.CautionLeader == 0 {
			m.PlayerThrottleLimit = 0x3000
			if c.Speed >= 6760 {
				m.PlayerThrottleLimit = 0
			}
		}
		return
	}
	distance := m.cautionGap(0, c.CautionAhead)
	if distance < 0 {
		m.PlayerThrottleLimit = 0
		return
	}
	if distance >= 4000 {
		m.PlayerThrottleLimit = 0xffff
		return
	}
	relative := int16(c.Speed - m.Cars[c.CautionAhead].Speed)
	if relative >= 0 {
		relative <<= 2
	}
	relative >>= 1
	gapError := min(int16(0), int16(distance)-540-relative)
	penalty := min(uint16(-gapError), uint16(360))
	m.PlayerThrottleLimit = 0x6000 - uint16(uint32(penalty)*0x6000/360)
}
