// Package session implements the original event timing and qualification rules
// with native track coordinates and integer time quantities.
package session

// Clock retains lap times in centiseconds. Its five-character fields hold
// the original M:SS.cc digits without punctuation, including the running
// display's separate seven-hundredths update between ten-hertz carries.
type Clock struct {
	Laps                             uint16
	CrossingDelta                    int16
	LastSpeed, BestSpeed             uint16
	QualifyingSpeed, LapCentiseconds uint16
	QualifyingTime, ElapsedTenths    uint32
	Phase                            int8
	Flags, SignalFlags               byte
	Crossed                          bool
	Current, Best, Last, Qualifying  [5]byte
}

type ClockEvent struct {
	LapCompleted, Qualified bool
	Position                int
}

// Initialize resets an event's clock while retaining its best recorded lap.
// The original seeded running time is rounded down only for the display;
// the full centisecond remainder remains available to finish-line accounting.
func Initialize(previous Clock, mode int, remainingDistance uint16) Clock {
	initial := uint16(uint32(39600-remainingDistance) * 16549 >> 16)
	zero := [5]byte{'0', '0', '0', '0', '0'}
	if previous.Best == ([5]byte{}) {
		previous.Best = zero
	}
	flags := byte(1)
	if mode != 0 {
		flags = 0x11
	}
	return Clock{
		CrossingDelta: -1, BestSpeed: previous.BestSpeed, Best: previous.Best,
		LapCentiseconds: initial, Phase: 15, SignalFlags: flags, Crossed: previous.Crossed,
		Current: FormatTime(initial / 10 * 10), Last: zero, Qualifying: zero,
	}
}

// QualifyingPosition ranks hundredths of a mile per hour against the original
// thirty-two fixed opponent scores. Equal scores place the player ahead.
func QualifyingPosition(speed uint16) int {
	position := 33
	for score := uint16(19512); score <= 22984; score += 112 {
		if speed < score {
			break
		}
		position--
	}
	return position
}

func FormatTime(value uint16) [5]byte {
	return [5]byte{'0' + byte(value/6000), '0' + byte(value/1000%6), '0' + byte(value/100%10), '0' + byte(value/10%10), '0' + byte(value%10)}
}

func speedForLap(centiseconds uint16) uint16 {
	if centiseconds == 0 || 90000000/uint32(centiseconds) > 65535 {
		// An impossibly short lap retains the original invalid-result value.
		return 0x4a80
	}
	return uint16(90000000 / uint32(centiseconds))
}

// Step advances the original fifteen-hertz clock and finish-line accounting.
// Qualification (mode one) excludes the warm lap and continuously completes
// four-lap attempts. The physical integrator supplies both track positions.
func (c *Clock) Step(mode int, previousProgress, progress uint16) ClockEvent {
	var event ClockEvent
	if c.LapCentiseconds != 17999 {
		c.Current[4] += 7
		if c.Current[4] >= '9'+1 {
			c.Current[4] -= 10
		}
	}
	c.Phase -= 10
	if c.Phase < 0 {
		c.Phase += 15
		c.LapCentiseconds += 10
		c.ElapsedTenths++
		c.Current[3]++
		if c.Current[3] == '9'+1 {
			c.Current[3] = '0'
			c.Current[2]++
			if c.Current[2] == '9'+1 {
				c.Current[2] = '0'
				c.Current[1]++
				if c.Current[1] == '6' {
					c.Current[1] = '0'
					c.Current[0]++
					if c.Current[0] == '3' {
						c.Current = [5]byte{'2', '5', '9', '9', '9'}
						c.LapCentiseconds = 17999
					}
				}
			}
		}
	}
	c.Crossed = false
	if previousProgress < 1000 || progress < 1000 || (int16(previousProgress-6931)^int16(progress-6931)) >= 0 {
		return event
	}
	delta, previousDelta := int16(previousProgress-progress), c.CrossingDelta
	c.CrossingDelta = delta
	if delta >= 0 {
		previousDelta = 0
	}
	if previousDelta >= 0 {
		c.LapCentiseconds = 0
	} else {
		c.Flags |= 1
		correction := uint16(int32(-int16(progress-6931)) * 7 / int32(delta))
		if c.LapCentiseconds != 17999 {
			if correction > c.LapCentiseconds {
				c.LapCentiseconds = 0
			} else {
				c.LapCentiseconds -= correction
			}
		}
		duration := c.LapCentiseconds
		c.LapCentiseconds = correction
		c.Laps++
		event.LapCompleted = true
		if mode == 1 && c.Laps != 1 {
			c.QualifyingTime += uint32(duration)
			average := uint16(c.QualifyingTime / uint32(c.Laps-1))
			c.Qualifying, c.QualifyingSpeed = FormatTime(average), speedForLap(average)
			if c.Laps == 2 {
				c.SignalFlags = 1
			} else if c.Laps == 4 {
				c.SignalFlags = 0x15
			}
			if c.Laps >= 5 {
				event.Qualified, event.Position = true, QualifyingPosition(c.QualifyingSpeed)
				c.SignalFlags, c.QualifyingTime, c.Laps = 0x19, 0, 1
			}
		}
		c.Last, c.LastSpeed = FormatTime(duration), speedForLap(duration)
		if c.LastSpeed >= c.BestSpeed {
			c.BestSpeed, c.Best = c.LastSpeed, c.Last
		}
	}
	c.Crossed = true
	copy(c.Current[:4], "0000")
	return event
}

// AtPit applies the original stationary pit-box reset. A qualifying attempt
// is abandoned there; an existing grid result remains available separately.
func (c *Clock) AtPit(mode int, progress uint16, lateral int16, speed uint16) bool {
	if uint16(lateral) >= 75 || progress <= 6930 || progress >= 7080 || speed != 0 {
		return false
	}
	c.Laps = 0
	if mode == 1 {
		c.Qualifying = [5]byte{'0', '0', '0', '0', '0'}
		c.QualifyingSpeed, c.QualifyingTime, c.SignalFlags = 0, 0, 1
	}
	return true
}
