package traffic

// PlayerPose supplies the physical integrator and lap clock's current values.
// Preparing this record does not integrate the player's force model again.
type PlayerPose struct {
	Progress                       uint16
	Lateral, Heading, TrackHeading int16
	Crossed                        bool
	CrossingDelta                  int16
}

// updateRacingPlayer follows the original player callback after a complete
// computer-driver cycle. Stopped cars and the pit corridor leave the queue.
func (m *Model) updateRacingPlayer() {
	c := &m.Cars[0]
	c.PitFlag = c.Speed == 0 || c.Lateral <= 4800 && (c.Distance < 4110 || c.Distance >= 37290)
}

// PreparePlayer translates $233a8. Distance fractions and the previous lateral
// position survive until the shared field pass applies the measured movement.
func (m *Model) PreparePlayer(pose PlayerPose) {
	c := &m.Cars[0]
	difference := pose.Lateral - c.Lateral
	c.LateralVelocity = difference
	c.TargetLateral = c.Lateral
	if difference >= -2000 && difference <= 2000 {
		c.TargetLateral += difference * 8
	}
	c.Yaw = pose.Heading - pose.TrackHeading
	distance := uint16(6930 - pose.Progress)
	if int16(distance) < 0 {
		distance += TrackLength
	}
	movement := int16(c.Distance - distance)
	if movement < 0 {
		c.Speed = 0
		c.Distance = distance
	} else {
		c.Speed = uint16(movement) << 8
	}
	m.PlayerMovement = movement << 8
	if !pose.Crossed {
		return
	}
	m.PlayerMovement = -pose.CrossingDelta << 8
	if m.PlayerMovement >= 0 {
		c.Speed = uint16(m.PlayerMovement)
		return
	}
	c.Speed = 0
	if c.RemainingLaps < 255 {
		c.RemainingLaps++
	}
	c.Laps--
	c.Distance = distance
	if m.Tail == 0 {
		m.Tail = c.Previous
		return
	}
	// The original rotates a reverse-crossing player through the distance ring
	// before normal field movement. The source tail itself stays in place here.
	tail := m.Tail
	for count := 0; count < len(m.Cars); count++ {
		passed := m.Cars[0].Next
		m.swapTrackNeighbors(0, passed)
		if m.RankBehind[0] == passed && m.Cars[0].RemainingLaps >= m.Cars[passed].RemainingLaps && m.Cars[passed].Flags&1 == 0 {
			// $234b8 updates the source reverse-crossing position counter
			// separately from the ordinary overtake counter.
			m.Position--
			m.swapRankNeighbors(0, passed)
		}
		if passed == tail {
			break
		}
	}
}

func (m *Model) swapTrackNeighbors(first, second int) {
	previous, next := m.Cars[first].Previous, m.Cars[second].Next
	if previous == second {
		return
	}
	m.Cars[previous].Next = second
	m.Cars[second].Previous = previous
	m.Cars[second].Next = first
	m.Cars[first].Previous = second
	m.Cars[first].Next = next
	m.Cars[next].Previous = first
}

// The standings are independent from the circular distance ring. Finished
// drivers retain their place while the cars continue moving around the oval.
func (m *Model) swapRankNeighbors(front, back int) {
	ahead, behind := m.RankAhead[front], m.RankBehind[back]
	m.RankBehind[front] = behind
	if behind >= 0 {
		m.RankAhead[behind] = front
	}
	m.RankAhead[back] = ahead
	if ahead >= 0 {
		m.RankBehind[ahead] = back
	} else {
		m.Leader = back
	}
	m.RankBehind[back] = front
	m.RankAhead[front] = back
}
