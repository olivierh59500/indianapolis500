package traffic

// LapTiming retains the original finish-line timestamp and the gap to the
// leader at that crossing. Times are tenths of a second, with word-sized gaps.
type LapTiming struct {
	Crossing uint32
	TimeGap  uint16
	LapGap   uint8
}

// Timing also keeps the original player-relative comparison shown by Lap Info.
// These values change at finish-line crossings, rather than from track distance.
type Timing struct {
	ElapsedTenths         uint32
	Cars                  [33]LapTiming
	AheadTime, BehindTime uint16
	AheadLaps, BehindLaps uint8
	ComparedCar           int
}

func (m *Model) recordCrossing(index int) {
	c := &m.Cars[index]
	t := &m.Timing.Cars[index]
	// The source stores the low timestamp word before reading the leader's
	// timestamp, so a leader crossing always records a zero time gap.
	t.Crossing = m.Timing.ElapsedTenths
	t.TimeGap = uint16(t.Crossing) - uint16(m.Timing.Cars[m.Leader].Crossing)
	c.RemainingLaps--
	t.LapGap = c.RemainingLaps - m.Cars[m.Leader].RemainingLaps
	if index == 0 {
		if m.Leader == 0 {
			return
		}
		ahead := m.RankAhead[0]
		m.Timing.ComparedCar = ahead
		m.Timing.AheadTime = uint16(t.Crossing) - uint16(m.Timing.Cars[ahead].Crossing)
		m.Timing.BehindTime = 0
		m.Timing.AheadLaps, m.Timing.BehindLaps = t.LapGap, 0
	} else if m.Leader == 0 && m.RankAhead[index] == 0 {
		m.Timing.ComparedCar = index
		m.Timing.BehindTime, m.Timing.BehindLaps = t.TimeGap, t.LapGap
		m.Timing.AheadTime, m.Timing.AheadLaps = 0, 0
	}
}
