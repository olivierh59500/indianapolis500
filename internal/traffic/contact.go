package traffic

// CollisionEffect is the original visual/audio event emitted by a hard contact.
type CollisionEffect struct {
	Severity, Colors           uint16
	Forward, Sideways, Lateral int16
	Distance                   uint16
}

// ContactResult describes a real source contact, independently of sound,
// replay recording and transfer to the player's force-integrated body.
type ContactResult struct {
	Hit                  bool
	Severity             uint16
	TrackGap, LateralGap int16
}

// ResolveContact follows the ordinary, non-spinning contact path. A crossing
// during the next step changes the sign probe before the lateral test.
func (m *Model) ResolveContact(first, second int, distance int16) ContactResult {
	a, b := &m.Cars[first], &m.Cars[second]
	predicted := distance + int16(b.Speed>>8) - int16(a.Speed>>8)
	if (distance ^ predicted) < 0 {
		distance = 1
		if predicted < 0 {
			distance = -1
		}
	}
	return m.resolveImpact(first, second, distance)
}

func signedHigh(a, b int16) int16 { return int16(int32(a) * int32(b) >> 16) }

func (m *Model) resolveImpact(first, second int, distance int16) ContactResult {
	a, b := &m.Cars[first], &m.Cars[second]
	lateral := b.Lateral - a.Lateral
	predicted := lateral + b.LateralVelocity - a.LateralVelocity
	if (lateral ^ predicted) < 0 {
		lateral = 1
		if predicted < 0 {
			lateral = -1
		}
	}
	if abs16(lateral) >= 624 {
		return ContactResult{}
	}
	relativeSide := b.LateralVelocity - a.LateralVelocity
	halfSpeed := int16(b.Speed)>>1 - int16(a.Speed)>>1
	severity := abs16(relativeSide) + abs16(halfSpeed>>2)
	result := ContactResult{true, severity, distance, lateral}
	if severity < 160 {
		if (lateral ^ relativeSide) < 0 {
			a.LateralVelocity, b.LateralVelocity = b.LateralVelocity, a.LateralVelocity
		}
		difference := int16(b.Speed - a.Speed)
		if (distance ^ difference) < 0 {
			a.Speed, b.Speed = b.Speed, a.Speed
			m.clampNPCSpeed(first)
			m.clampNPCSpeed(second)
		}
		a.YawRate >>= 1
		if a.YawRate != 0 {
			a.YawRate++
		}
		b.YawRate >>= 1
		if b.YawRate != 0 {
			b.YawRate++
		}
		return result
	}
	if m.OnCollisionEffect != nil {
		m.OnCollisionEffect(CollisionEffect{Severity: uint16(severity), Colors: uint16(b.Primary)<<8 | uint16(a.Secondary), Sideways: (relativeSide >> 1) + a.LateralVelocity, Forward: int16(a.Speed) + ((halfSpeed >> 2) << 2), Lateral: b.Lateral, Distance: b.Distance})
	}
	yaw := (relativeSide>>3)*distance + (lateral>>5)*(halfSpeed>>5)
	a.YawRate += yaw
	b.YawRate += yaw
	along := int16((int32(distance>>1) << 16) / 41)
	across := int16((int32((-lateral)>>1) << 16) / 625)
	sideShare := signedHigh(int16(32767-abs16(along)), relativeSide) + signedHigh(int16(abs16(halfSpeed>>2)), across)
	a.LateralVelocity += sideShare
	b.LateralVelocity -= sideShare
	speedShare := signedHigh(int16(abs16(relativeSide<<3)), along) + signedHigh(int16(32767-abs16(across)), halfSpeed)*2
	a.Speed = uint16(int16(a.Speed) + speedShare)
	b.Speed = uint16(int16(b.Speed) - speedShare)
	m.clampNPCSpeed(first)
	m.clampNPCSpeed(second)
	if second != 0 && !b.Spinning {
		m.registerSpin(second)
	}
	return result
}

func (m *Model) registerSpin(index int) {
	c := &m.Cars[index]
	if index == 0 || c.Spinning {
		return
	}
	if !c.PitFlag && m.Yellow {
		m.changeCautionMembership(index, true)
	}
	c.Action, c.Spinning, c.RetiredReason, c.PitFlag = Spinning, true, 24, true
	copy(m.Wrecks[1:], m.Wrecks[:m.WreckCount])
	m.Wrecks[0] = index
	m.WreckCount = min(len(m.Wrecks), m.WreckCount+1)
	if m.YellowDelay == 0 && !m.Yellow {
		m.YellowDelay = 75
	}
}

func (m *Model) clampNPCSpeed(index int) {
	if index != 0 && int16(m.Cars[index].Speed) < 0 {
		m.Cars[index].Speed = 0
	}
}
