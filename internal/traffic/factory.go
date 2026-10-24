package traffic

import (
	"encoding/binary"
	"fmt"
)

// InitializeRace creates the original staggered rolling field from recovered
// per-driver coefficients. Existing lap epochs and the player's selected
// colors survive the reset, as they do in the original session initializer.
func (m *Model) InitializeRace(parameters []byte, qualifyingPosition, laps int) error {
	if len(parameters) != 33*16 || qualifyingPosition < 1 || qualifyingPosition > 33 || laps < 1 || laps > 254 {
		return fmt.Errorf("invalid original rolling-grid parameters")
	}
	m.initializeDrivers(parameters)
	return m.initializeGrid(qualifyingPosition, laps)
}

// initializeDrivers preserves the source pose and lap history for records
// which the selected mode leaves outside its active circular field.
func (m *Model) initializeDrivers(parameters []byte) {
	m.Cars[0].Distance, m.Cars[0].Lateral = 39525, 624
	pit := uint16(39525)
	for i := range m.Cars {
		previous := m.Cars[i]
		c := Car{
			Distance: previous.Distance, Lateral: previous.Lateral, TargetLateral: previous.TargetLateral,
			RemainingLaps: previous.RemainingLaps, Laps: previous.Laps, CautionAhead: previous.CautionAhead,
			PreferredLane: 2, Lane: 2, SavedLane: 2, RandomTick: uint8(33 - i), PitDistance: pit,
			Primary: previous.Primary, Secondary: previous.Secondary, Number: previous.Number,
			Previous: i, Next: i, Action: Stopped,
		}
		p := parameters[i*16:]
		word := func(offset int) uint16 { return binary.BigEndian.Uint16(p[offset:]) }
		c.BaseAcceleration = int16(word(4) + uint16(uint32(m.random())*uint32(word(6))>>16))
		c.Acceleration = c.BaseAcceleration
		c.Drag = word(8) + uint16(uint32(m.random())*uint32(word(10))>>16)
		c.TurnDrag = word(12) + uint16(uint32(m.random())*uint32(word(14))>>16)
		if i > 0 {
			c.Primary, c.Secondary, c.Number = p[0], p[1], p[2]
		}
		m.Cars[i] = c
		m.RankAhead[i], m.RankBehind[i] = i, i
		pit -= 150
		if pit == 37725 {
			pit = 3075
		}
	}
	m.Cars[0].Action, m.Cars[0].PitFlag = Racing, true
	m.PlayerThrottleLimit = 0xffff
	m.NextDriver = 1
	m.Tail, m.Leader = 0, 0
	m.Yellow, m.WreckCount, m.YellowDelay, m.PlayerMovement = false, 0, 0, 0
	m.SignalFlags, m.TowInterval, m.TowWait = 1, 0, 0
	m.LastPlayerFlags, m.RaceWon = 0, false
	m.Timing.Cars = [33]LapTiming{}
	m.Timing.AheadTime, m.Timing.BehindTime = 0, 0
	m.Timing.AheadLaps, m.Timing.BehindLaps = 0, 0
}

func (m *Model) initializeGrid(qualifyingPosition, laps int) error {
	var order [33]int
	opponent := 32
	for rank := range order {
		if rank == qualifyingPosition-1 {
			order[rank] = 0
		} else {
			order[rank] = opponent
			opponent--
		}
	}
	distance, lane, speed := uint16(300), uint8(0), uint16(13000)
	for rank, index := range order {
		c := &m.Cars[index]
		c.Action, c.PitFlag = Racing, false
		c.Distance, c.Lane, c.RemainingLaps = distance, lane, uint8(laps+1)
		lane++
		distance++
		if lane == 3 {
			lane = 0
			speed -= 567
			distance += 597
		}
		// The original samples the line using the next grid distance, then
		// assigns the current row's already adjusted rolling speed.
		c.Lateral = int16(m.lines[c.Lane][distance>>7])
		c.TargetLateral = c.Lateral
		random := m.random()
		c.Speed = speed + uint16(int16(random&511)-256)
		c.PitLaps = uint8(random&31) + 2
		if random >= 7936 {
			c.PitLaps = uint8(random&3) + 31
		}
		c.Previous = order[(rank+32)%33]
		c.Next = order[(rank+1)%33]
		m.RankAhead[index], m.RankBehind[index] = -1, -1
		if rank > 0 {
			m.RankAhead[index] = order[rank-1]
		}
		if rank+1 < len(order) {
			m.RankBehind[index] = order[rank+1]
		}
	}
	m.NextDriver = 1
	m.Tail = order[len(order)-1]
	m.Leader, m.Position = order[0], uint16(qualifyingPosition)
	m.Rules, m.Yellow = 2, false
	m.WreckCount, m.YellowDelay, m.PlayerMovement = 0, 0, 0
	m.SignalFlags, m.TowInterval, m.TowWait = 1, 0, 0
	return nil
}

// InitializeSolo creates the original Practice or Qualification field. In
// Practice, selected computer cars are scattered around the oval. Unselected
// records remain self-linked and cannot participate in contacts or rendering.
func (m *Model) InitializeSolo(parameters []byte, qualifying bool) error {
	if len(parameters) != 33*16 {
		return fmt.Errorf("invalid original solo-mode driver parameters")
	}
	m.initializeDrivers(parameters)
	m.Rules = 0
	if qualifying {
		m.Rules = 1
		c := &m.Cars[0]
		c.Distance, c.Lateral, c.TargetLateral, c.Speed = 6000, 8320, 8320, 15000
		return nil
	}
	head := 0
	distance := uint16(39600)
	for i := 1; i < len(m.Cars); i++ {
		distance -= 1200
		if m.random() >= 15000 {
			continue
		}
		c := &m.Cars[i]
		c.Distance, c.Speed, c.Action = distance, 15000, Racing
		c.Lateral = int16(m.lines[c.Lane][distance>>7])
		c.TargetLateral = c.Lateral
		c.PitLaps = uint8(m.random()&31) + 2
		c.Previous, c.Next = 0, head
		m.Cars[0].Next, m.Cars[head].Previous = i, i
		head = i
	}
	return nil
}
