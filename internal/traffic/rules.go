package traffic

func (m *Model) leader() int { return m.Leader }

// SetYellow changes the original queue and driver callbacks. It preserves
// track motion and rebases lap epochs around the leader's current position.
func (m *Model) SetYellow(enabled bool) {
	if m.Yellow == enabled {
		return
	}
	m.Yellow = enabled
	if !enabled {
		m.PlayerThrottleLimit = 0xffff
		for i := range m.Cars {
			c := &m.Cars[i]
			if c.Action == FollowingCaution {
				c.Action = Racing
			} else if c.Action == PittingUnderCaution {
				c.Action = Pitting
			}
		}
		m.Cars[0].Action = Racing
		return
	}
	m.TowInterval, m.TowWait = 3000, 3000
	leader, tail := m.leader(), m.Tail
	m.CautionLeader, m.PlayerThrottleLimit = leader, 0x6000
	epoch, ahead, index := uint8(128), -1, leader
	for count := 0; count < len(m.Cars); count++ {
		c := &m.Cars[index]
		c.Laps, c.CautionAhead = epoch, ahead
		if index == tail {
			epoch = 127
		}
		ahead, index = index, c.Next
	}
	if m.Cars[leader].RemainingLaps >= 5 {
		index = leader
		for count := 0; count < len(m.Cars); count++ {
			c := &m.Cars[index]
			candidate := uint8(m.random()&15) + 5
			if candidate >= c.PitLaps && c.PitLaps < c.RemainingLaps {
				candidate &= 1
				if candidate == 0 {
					if c.Distance < 12870 {
						candidate = 1
					} else if c.Action == Racing && index != 0 {
						c.Action = Pitting
					}
				}
				if c.PitLaps != 0 {
					c.PitLaps = candidate
				}
			}
			index = c.Next
		}
	}
	index = tail
	for count := 0; count < len(m.Cars); count++ {
		c := &m.Cars[index]
		if c.Action == Racing && index != 0 {
			c.Action = FollowingCaution
		} else if c.Action == Pitting {
			c.Action = PittingUnderCaution
		}
		if c.PitFlag {
			m.changeCautionMembership(index, true)
		}
		index = c.Previous
	}
	m.Cars[0].Action = FollowingCaution
}

// TowWreck removes at most one wreck per source timer interval, after it is
// two thousand track units away from the player. The interval halves each time.
func (m *Model) TowWreck() {
	if m.WreckCount == 0 {
		return
	}
	if m.TowWait != 0 {
		m.TowWait--
		return
	}
	index := m.Wrecks[0]
	c := &m.Cars[index]
	if abs16(gap(m.Cars[0], *c)) < 2000 {
		return
	}
	c.Lateral, c.TargetLateral = -10000, -10000
	c.Action, c.Spinning = Stopped, false
	copy(m.Wrecks[:], m.Wrecks[1:m.WreckCount])
	m.WreckCount--
	m.TowInterval >>= 1
	m.TowWait = m.TowInterval
}

// StepRules runs after the field callbacks. Short races announce an incident
// without neutralizing the field; longer races queue and clear a real caution.
func (m *Model) StepRules(flagsEnabled bool) {
	if !flagsEnabled {
		if m.YellowDelay == 0 {
			m.SignalFlags = m.SignalFlags&^2 | 1
		} else {
			if m.YellowDelay == 75 {
				m.SignalFlags = m.SignalFlags&^1 | 0x12
			}
			m.YellowDelay--
		}
		return
	}
	if m.SignalFlags&2 == 0 {
		if m.YellowDelay != 0 {
			m.YellowDelay--
			if m.YellowDelay == 0 {
				m.SignalFlags = m.SignalFlags&^1 | 0x12
				m.SetYellow(true)
			}
		}
		return
	}
	if m.WreckCount != 0 {
		m.TowWreck()
		return
	}
	distance := m.Cars[m.leader()].Distance
	if distance >= 4000 && distance < 6000 {
		m.SignalFlags = m.SignalFlags&^2 | 0x11
		m.SetYellow(false)
	}
}

// RaceEvent reports the original status-page transition after player finish.
// Opponents can finish first while the player keeps driving toward the line.
type RaceEvent struct {
	Finished bool
	Won      bool
}

// StepRaceStatus translates $4946-$4a9c after the shared field update. A new
// finish announcement takes precedence over caution countdowns in that tick.
func (m *Model) StepRaceStatus(flagsEnabled, crossed bool) RaceEvent {
	var event RaceEvent
	if m.Rules != 2 {
		return event
	}
	flags := m.Cars[0].Flags
	changed := (flags^m.LastPlayerFlags)&128 != 0
	m.LastPlayerFlags = flags
	if changed {
		m.SignalFlags = m.SignalFlags&^4 | 0x18
		event.Finished = true
		if m.Leader == 0 {
			m.RaceWon = true
			event.Won = true
		}
		return event
	}
	m.StepRules(flagsEnabled)
	if crossed && m.Cars[m.Leader].RemainingLaps == 1 {
		m.SignalFlags |= 0x14
	}
	return event
}
