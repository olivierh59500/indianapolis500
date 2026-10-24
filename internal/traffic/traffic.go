// Package traffic implements the original computer drivers with native Go
// records and fixed-point track coordinates. It contains no processor state.
package traffic

import (
	"encoding/binary"
	"fmt"
)

const TrackLength = 39600

type DriverAction uint8

const (
	Racing DriverAction = iota
	Pitting
	FollowingCaution
	PittingUnderCaution
	Stopped
	Spinning
)

// Car uses Q4 lateral positions and a Q8 remaining distance fraction. Speed
// is the original distance decrement per simulation tick, before its Q8 shift.
type Car struct {
	Distance                               uint16
	Fraction                               uint8
	Lateral, Yaw                           int16
	Speed                                  uint16
	LateralVelocity, YawRate, Acceleration int16
	Drag, TurnDrag                         uint16
	PitLaps                                uint8
	PreferredLane, Lane                    uint8
	BaseAcceleration, TargetLateral        int16
	Laps                                   uint8
	RandomTick                             uint8
	LateralAcceleration                    int16
	Previous, Next                         int
	Spinning                               bool
	RemainingLaps, Flags                   uint8
	Primary, Secondary, Number             uint8
	PitDistance                            uint16
	Action                                 DriverAction
	ServiceWait, SavedLane, RetiredReason  uint8
	PitFlag                                bool
	CautionAhead                           int
}

// State holds the live race quantities independently of recovered tables.
// It can be preserved across an attract sequence or replay without sharing
// mutable cars with the temporary sequence.
type State struct {
	Cars                  [33]Car
	Random                uint16
	NextDriver            int
	Tail                  int
	Leader                int
	CautionLeader         int    // First queue car, independently of race ranking.
	PlayerThrottleLimit   uint16 // Engine fraction retained between player callbacks.
	Position              uint16
	RankAhead, RankBehind [33]int
	Rules                 int
	Yellow                bool
	PlayerMovement        int16
	Wrecks                [33]int
	WreckCount            int
	YellowDelay           uint8
	SignalFlags           uint8
	LastPlayerFlags       uint8
	RaceWon               bool
	Timing                Timing
	TowInterval           uint16
	TowWait               uint16
}

// Model owns recovered numerical curves and the surrounding cars. Lane
// numbers are ordinary array indices, rather than original pointer offsets.
type Model struct {
	OnCollisionEffect func(CollisionEffect)
	State
	lines             [5][310]uint16
	flags             [156]byte
	yaw, drag         [86]uint16
	spinStrength      [256]uint16
	trigonometry      [1024]int16
	retirementReasons [64]byte
	innerLimits       [496]int16
	outerLimits       [496]int16
}

// StepRacingField couples the physical player pose to the shared car movement,
// track ordering, contacts and four round-robin computer-driver decisions.
func (m *Model) StepRacingField(pose PlayerPose, playerContact func(ContactResult)) (pitRequests, spinRequests uint64) {
	m.PreparePlayer(pose)
	tail := m.Tail
	index := m.Cars[tail].Next
	for count := 0; count < len(m.Cars); count++ {
		i := index
		c := &m.Cars[i]
		index = c.Next
		if c.Advance() {
			m.Tail = i
			if c.Flags&1 == 0 {
				m.recordCrossing(i)
				finished := c.RemainingLaps == 0 || m.Cars[m.Leader].Flags&1 != 0
				if finished && m.Rules == 2 {
					c.Flags, c.PitLaps = 255, 1
					c.BaseAcceleration = int16(uint16(c.BaseAcceleration) >> 1)
				}
				if finished || c.RemainingLaps >= 5 {
					c.PitLaps--
					if c.PitLaps == 0 {
						pitRequests |= uint64(1) << i
						// The player's source callback is distinct from the NPC racing
						// callback, even though both are represented by the Racing action.
						if i != 0 {
							if c.Action == Racing {
								c.Action = Pitting
							} else if c.Action == FollowingCaution {
								c.Action = PittingUnderCaution
							}
						}
					}
				}
			}
		}
		if i == tail {
			break
		}
	}
	m.orderCars()
	m.ConstrainWrecks()
	m.ResolveContacts(playerContact)
	for count := 0; count < 4; count++ {
		wasSpinning := m.Cars[m.NextDriver].Spinning
		m.UpdateDriver(m.NextDriver)
		if !wasSpinning && m.Cars[m.NextDriver].Spinning {
			spinRequests |= uint64(1) << m.NextDriver
		}
		m.NextDriver++
		if m.NextDriver == len(m.Cars) {
			m.updatePlayerDriver()
			m.NextDriver = 1
		}
	}
	return pitRequests, spinRequests
}

func (m *Model) NextRandom() uint16 { return m.random() }

// InField distinguishes live field members from the source self-linked
// dormant records. A stopped or spinning car can still belong to the field.
func (m *Model) InField(index int) bool {
	return index == 0 || m.Cars[index].Next != index
}

func (m *Model) orderCars() {
	var order [33]int
	start := m.Cars[m.Tail].Next
	index, count := start, 0
	for {
		order[count] = index
		count++
		index = m.Cars[index].Next
		if index == start || count == len(order) {
			break
		}
	}
	// Equal-distance cars retain their previous order. Fractions affect
	// motion, but do not break ties in the original ordering pass.
	for i := 1; i < count; i++ {
		car, j := order[i], i
		for j > 0 && m.Cars[order[j-1]].Distance > m.Cars[car].Distance {
			front := order[j-1]
			if m.RankBehind[front] == car && m.Cars[front].RemainingLaps >= m.Cars[car].RemainingLaps && m.Cars[car].Flags&1 == 0 {
				if car == 0 {
					m.Position--
				}
				if front == 0 {
					m.Position++
				}
				m.swapRankNeighbors(front, car)
			}
			order[j] = order[j-1]
			j--
		}
		order[j] = car
	}
	m.Tail = order[count-1]
	for i, car := range order[:count] {
		m.Cars[car].Previous = order[(i+count-1)%count]
		m.Cars[car].Next = order[(i+1)%count]
	}
}

func New(lines, flags, yaw, drag []byte) (*Model, error) {
	if len(lines) != 3100 || len(flags) != 156 || len(yaw) != 172 || len(drag) != 172 {
		return nil, fmt.Errorf("invalid recovered computer-driver curves")
	}
	m := &Model{State: State{NextDriver: 1, CautionLeader: -1, PlayerThrottleLimit: 0xffff, Timing: Timing{ComparedCar: -1}}}
	for i := range m.Cars {
		m.Cars[i].Previous = (i + len(m.Cars) - 1) % len(m.Cars)
		m.Cars[i].Next = (i + 1) % len(m.Cars)
		m.Cars[i].CautionAhead = -1
	}
	for line := range m.lines {
		for i := range m.lines[line] {
			m.lines[line][i] = binary.BigEndian.Uint16(lines[(line*310+i)*2:])
		}
	}
	copy(m.flags[:], flags)
	for i := range m.yaw {
		m.yaw[i] = binary.BigEndian.Uint16(yaw[i*2:])
		m.drag[i] = binary.BigEndian.Uint16(drag[i*2:])
	}
	return m, nil
}

func (m *Model) random() uint16 {
	m.Random = m.Random*0x6255 + 0x3619
	return m.Random
}

// Advance moves every record before the four selected computer drivers run.
// It reports a finish-line crossing so the race controller can update timing,
// remaining laps, pit countdowns and race order independently.
func (c *Car) Advance() bool {
	c.LateralVelocity += c.LateralAcceleration
	c.Lateral += c.LateralVelocity
	c.Yaw += c.YawRate
	distance := int(c.Distance)*256 + int(c.Fraction) - int(c.Speed)
	crossed := distance < 0
	if crossed {
		distance += TrackLength * 256
		c.Laps++
	}
	c.Distance, c.Fraction = uint16(distance>>8), uint8(distance)
	return crossed
}

func abs16(n int16) uint16 {
	if n < 0 {
		return uint16(-n)
	}
	return uint16(n)
}

func gap(a, b Car) int16 {
	d := int(a.Distance) - int(b.Distance)
	if d < -TrackLength/2 {
		d += TrackLength
	}
	if d >= TrackLength/2 {
		d -= TrackLength
	}
	return int16(d)
}

var laneBits = [3]uint16{4, 2, 1}
var adjacentLanes = [3]uint16{6, 7, 3}

// UpdateRacing follows the original overtaking, lane occupancy and braking
// rules. A rare spin request is returned to the race controller for its
// separate collision/spin lifecycle; this routine does not invent recovery.
func (m *Model) UpdateRacing(index int) bool {
	c := &m.Cars[index]
	c.RandomTick--
	if c.RandomTick == 0 {
		c.RandomTick = 16
		flag := uint8(-m.flags[c.Distance>>8])
		if m.random() <= uint16(flag)*16+2 {
			return true
		}
	}
	m.steer(c, m.racingTarget(index))
	return false
}

func (m *Model) racingTarget(index int) int16 {
	c := &m.Cars[index]
	sampleDistance := int(c.Distance) - int(c.Speed>>4)
	if sampleDistance < 0 {
		sampleDistance += TrackLength
	}
	sample := sampleDistance >> 7
	c.Acceleration = c.BaseAcceleration - int16(c.PitLaps)
	lower, upper, allowed := m.laneSpace(index, sample)
	penalties := m.lanePenalties(index, sample)
	eligible := allowed & adjacentLanes[c.Lane]
	cost := penalties[c.Lane]
	best := laneBits[c.Lane]
	for _, lane := range [...]int{2, 1, 0} {
		if eligible&laneBits[lane] != 0 && penalties[lane] <= cost {
			if penalties[lane] != cost {
				best = 0
			}
			cost = penalties[lane]
			best |= laneBits[lane]
		}
	}
	lane := int(c.PreferredLane)
	if best&laneBits[lane] == 0 {
		lane = 1
		if best&laneBits[lane] == 0 {
			first, second := 2, 0
			if m.random()&0x8000 != 0 {
				first, second = 0, 2
			}
			lane = int(c.Lane)
			if best&laneBits[first] != 0 {
				lane = first
			} else if best&laneBits[second] != 0 {
				lane = second
			}
		}
	}
	c.Lane = uint8(lane)
	c.Acceleration -= int16(penalties[lane])
	target := m.lines[lane][sample]
	if target < lower || target >= upper {
		c.Acceleration += int16(penalties[lane])
		if target < lower {
			target = lower
		} else {
			target = upper
		}
		if lower >= upper {
			largest := max(penalties[0], penalties[1], penalties[2])
			penalties = [3]uint16{min(largest*2, 1800), min(largest*2, 1800), min(largest*2, 1800)}
			target = (lower + upper) >> 1
		}
		lane = 0
		distance := abs16(int16(m.lines[0][sample] - target))
		for other := 1; other < 3; other++ {
			d := abs16(int16(m.lines[other][sample] - target))
			if d < distance {
				lane, distance = other, d
			}
		}
		c.Acceleration -= int16(penalties[lane])
		c.Lane = uint8(lane)
	}
	return int16(target)
}

func (m *Model) laneSpace(index, sample int) (uint16, uint16, uint16) {
	c := m.Cars[index]
	lower, upper, allowed := m.lines[3][sample], m.lines[4][sample], uint16(7)
	next := c.Previous
	for range 3 {
		if next == index {
			break
		}
		other := m.Cars[next]
		d := uint16(gap(c, other))
		if d >= 180 {
			break
		}
		dv := int16(c.Speed - other.Speed)
		if (d < 90 || dv >= 0 && uint16(dv)>>4 >= d) && !c.Spinning {
			if uint16(other.Lateral) < uint16(c.Lateral) {
				lower = max(lower, uint16(other.TargetLateral+832))
			} else {
				upper = min(upper, uint16(other.TargetLateral-832))
			}
		}
		next = other.Previous
	}
	next = c.Next
	for range 3 {
		if next == index {
			break
		}
		other := m.Cars[next]
		d := uint16(gap(c, other) - 1)
		if d < 0xff4c {
			break
		}
		dv := int16(c.Speed - other.Speed)
		if d >= 0xffa6 || dv < 0 && uint16(dv>>4) < d {
			for lane := range 3 {
				if abs16(int16(m.lines[lane][sample])-other.TargetLateral) < 832 {
					allowed &^= laneBits[lane]
				}
			}
		}
		next = other.Next
	}
	return lower, upper, allowed | laneBits[c.Lane]
}

func (m *Model) lanePenalties(index, sample int) [3]uint16 {
	var penalties [3]uint16
	c := m.Cars[index]
	next := c.Previous
	for range 3 {
		if next == index {
			break
		}
		other := m.Cars[next]
		d := uint16(gap(c, other))
		if d >= 1000 {
			break
		}
		penalty := int16(c.Speed-other.Speed) + int16((int16(d)-180)*-4)
		if penalty > 0 {
			value := min(uint16(penalty), 1800)
			for lane := range 3 {
				if abs16(int16(m.lines[lane][sample])-other.TargetLateral) < 832 {
					penalties[lane] = max(penalties[lane], value)
				}
			}
		}
		next = other.Previous
	}
	return penalties
}

// steer preserves the word-sized arithmetic of the original traffic model.
// The player has a separate force integrator; this model only steers NPCs.
func (m *Model) steer(c *Car, target int16) {
	velocity := c.LateralVelocity
	limit := c.Speed >> 3
	if abs16(velocity) >= limit {
		velocity = int16(limit)
		if c.LateralVelocity < 0 {
			velocity = -velocity
		}
		c.LateralVelocity = velocity
	}
	c.TargetLateral = target
	acceleration := (target - (c.Lateral + velocity*12)) >> 6
	limit = c.Speed >> 6
	if abs16(acceleration) >= limit {
		negative := acceleration < 0
		acceleration = int16(limit)
		if negative {
			acceleration = -acceleration
		}
	}
	c.LateralAcceleration = acceleration
	yawTarget := -(acceleration << 6)
	sample := c.Speed >> 8
	if m.flags[c.Distance>>8]&1 != 0 {
		yawTarget += int16(m.yaw[sample])
	}
	c.YawRate = ((yawTarget >> 2) - c.Yaw) >> 3
	turnLoss := uint16(uint32(abs16(yawTarget)) * uint32(c.TurnDrag) >> 16)
	dragLoss := uint16(uint32(m.drag[sample]) * uint32(c.Drag) >> 16)
	speed := int16(c.Speed-turnLoss-dragLoss) + c.Acceleration
	if speed < 0 {
		speed = 0
	}
	c.Speed = uint16(speed)
}
