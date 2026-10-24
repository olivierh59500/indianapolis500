package scene

import (
	"encoding/binary"
	"fmt"
)

// Vector2 stores signed world coordinates. Collision velocities use Q7 world
// units per simulation tick, while headings use the full sixteen-bit turn.
type Vector2 struct{ X, Y int16 }
type Wall struct{ Start, Delta Vector2 }

// Contact describes an oriented wall crossing and the original two-pixel
// separation added to the normal correction.
type Contact struct {
	Point                  Vector2
	Depth, Tangent, Normal int16
}

// CollisionState contains only vehicle quantities, with no processor or
// hardware state. Fractions belong to the world-position accumulator.
type CollisionState struct {
	Position, Velocity                  Vector2
	FractionX, FractionY                uint16
	Heading, YawVelocity, TravelHeading int16
	Speed, Impact                       uint16
	PitchImpulse, RollImpulse, Steering int16
	DamageEnabled                       bool
	DamageFlags                         byte
}

func (w *World) LoadBarriers(geometry, damage []byte) error {
	if len(geometry) != 146 || len(damage) != 16 {
		return fmt.Errorf("recovered collision tables have invalid dimensions")
	}
	cursor := 0
	for set := range w.barriers {
		groups := int(binary.BigEndian.Uint16(geometry[cursor:]))
		cursor += 2
		for range groups {
			count := int(binary.BigEndian.Uint16(geometry[cursor:]))
			cursor += 2
			previous := Vector2{int16(binary.BigEndian.Uint16(geometry[cursor:])), int16(binary.BigEndian.Uint16(geometry[cursor+2:]))}
			cursor += 4
			for range count {
				next := Vector2{int16(binary.BigEndian.Uint16(geometry[cursor:])), int16(binary.BigEndian.Uint16(geometry[cursor+2:]))}
				cursor += 4
				w.barriers[set] = append(w.barriers[set], Wall{previous, Vector2{next.X - previous.X, next.Y - previous.Y}})
				previous = next
			}
		}
	}
	if cursor != len(geometry) {
		return fmt.Errorf("recovered barrier records have inconsistent dimensions")
	}
	copy(w.collisionDamage[0][:], damage[:8])
	copy(w.collisionDamage[1][:], damage[8:])
	return nil
}

// PitBarriers selects the original source $17ee barrier set using the pose
// before movement. It is independent of the preferred driving-line tables.
func (w *World) PitBarriers(progress uint16, lateral int16) []Wall {
	set := 0
	if progress >= 39400 {
		if lateral >= 300 {
			return nil
		}
		set = 1
	} else {
		if progress > 9699 {
			return nil
		}
		if lateral < 300 {
			set = 1
			if progress > 7999 {
				set = 2
			}
			if progress > 8399 {
				set = 3
			}
			if progress > 9399 {
				set = 4
			}
		}
	}
	return w.barriers[set]
}

func withinMotion(delta, back int16) bool {
	near := func(v int16) bool { return uint16(v+2) <= 4 }
	return near(delta) || near(back-delta) || int16((back-delta)^delta) >= 0
}

// WallContact translates the original swept intersection at $5544, including
// its Q16 intersection quantization, endpoint tolerance and one-sided test.
func (w *World) WallContact(current, previous Vector2, travelHeading int16, wall Wall) (Contact, bool) {
	back := Vector2{previous.X - current.X, previous.Y - current.Y}
	denominator := uint32(int32(back.X)*int32(wall.Delta.Y) - int32(back.Y)*int32(wall.Delta.X))
	if denominator == 0 {
		return Contact{}, false
	}
	numerator := uint32(int32(wall.Start.X-current.X)*int32(back.Y) + int32(current.Y-wall.Start.Y)*int32(back.X))
	if int32(denominator^numerator) < 0 && numerator != 0 {
		return Contact{}, false
	}
	if int32(denominator) < 0 {
		denominator = 0 - denominator
		numerator = 0 - numerator
	}
	fraction := uint16(65535)
	if numerator != denominator {
		if numerator >= denominator {
			return Contact{}, false
		}
		for {
			before := denominator
			numerator <<= 1
			denominator <<= 1
			if int32(before) >= 0 && int32(denominator) < 0 {
				break
			}
		}
		denominator &= 0xffff0000
		if numerator < denominator {
			fraction = uint16(numerator / (denominator >> 16))
		}
	}
	fraction >>= 1
	hit := Vector2{
		wall.Start.X + int16((int32(fraction)*int32(wall.Delta.X)*2)>>16),
		wall.Start.Y + int16((int32(fraction)*int32(wall.Delta.Y)*2)>>16),
	}
	if !withinMotion(hit.X-current.X, back.X) || !withinMotion(hit.Y-current.Y, back.Y) {
		return Contact{}, false
	}
	edgeHeading := w.Direction(wall.Delta.X, wall.Delta.Y)
	relative := travelHeading - edgeHeading
	if relative < 0 {
		return Contact{}, false
	}
	dx, dy := int32(hit.X-current.X), int32(hit.Y-current.Y)
	distance := int16(integerSquareRoot(uint32(dx*dx + dy*dy)))
	cosine, sine := w.cosSin(relative)
	return Contact{hit, q14(distance, sine) + 2, q14(distance, cosine), edgeHeading - 16384}, true
}

// ApplyContact translates $534e: a normal push, limited tangential sliding,
// the collision's yaw impulse and the original severity-dependent damage.
func (w *World) ApplyContact(state *CollisionState, contact Contact) Vector2 {
	tangent := contact.Tangent
	if tangent < 0 {
		if uint16(-tangent) >= uint16(contact.Depth) {
			tangent = -contact.Depth
		}
	} else if uint16(tangent) >= uint16(contact.Depth) {
		tangent = contact.Depth
	}
	cosine, sine := w.cosSin(contact.Normal)
	correction := Vector2{q14(contact.Depth, cosine), -q14(contact.Depth, sine)}
	cosine, sine = w.cosSin(contact.Normal - 16384)
	correction.X += q14(tangent>>1, cosine)
	correction.Y -= q14(tangent>>1, sine)
	severity := uint16(abs(int(correction.X))+abs(int(correction.Y))) << 4
	state.Impact = max(state.Impact, severity)
	relative := state.Heading - contact.Normal
	tangentSide := int16(16384)
	if relative < 0 {
		tangentSide = -16384
	}
	tangentAngle := contact.Normal + tangentSide - state.Heading
	cosine, _ = w.cosSin(relative * 2)
	weight := q14(128, cosine)
	yaw := int16(int32(weight+512)*int32(tangent)) >> 3
	depth := contact.Depth
	if tangentAngle < 0 {
		depth = -depth
	}
	yaw += int16(int32(depth)*int32(256-weight*2)) >> 3
	state.Heading += yaw
	state.YawVelocity += yaw
	state.Position.X += correction.X
	state.Position.Y += correction.Y
	state.Velocity.X += correction.X << 7
	state.Velocity.Y += correction.Y << 7
	state.FractionX, state.FractionY = 0, 0
	state.TravelHeading = w.Direction(state.Velocity.X, state.Velocity.Y)
	vx, vy := int32(state.Velocity.X), int32(state.Velocity.Y)
	state.Speed = uint16(integerSquareRoot(uint32(vx*vx + vy*vy)))
	state.PitchImpulse += int16(int8(relative)) * 4
	state.RollImpulse += int16(int8(contact.Normal)) * 4
	if state.DamageEnabled && state.Impact >= 250 {
		table := 0
		if state.Impact >= 600 {
			table = 1
		}
		index := uint8(0x8f-uint8(uint16(relative)>>8)) >> 5
		state.DamageFlags |= w.collisionDamage[table][index]
	}
	state.Steering = 0
	return correction
}

// SurfaceContact translates the radial inner/outer-wall branch of $17ee.
// Longitudinal travel supplies the tangential part of the impact, including
// the original wrap at the start/finish line.
func (w *World) SurfaceContact(previous, current TrackLocation, trackHeading int16) (Contact, bool) {
	depth, normal, inside := int16(0), trackHeading, false
	if current.Lateral < current.Inner {
		depth = current.Inner - current.Lateral + 2
		normal -= 16384
		inside = true
	} else if current.Lateral > current.Outer {
		depth = current.Lateral - current.Outer + 2
		normal += 16384
	} else {
		return Contact{}, false
	}
	oldProgress, newProgress := int(previous.Progress), int(current.Progress)
	if newProgress < 39400 {
		if oldProgress >= 39400 && newProgress < 200 {
			oldProgress -= 39600
		}
	} else if oldProgress < 200 {
		newProgress -= 39600
	}
	lateral := int16(abs(int(previous.Lateral - current.Lateral)))
	if lateral < depth {
		lateral = depth >> 1
	}
	tangent := int16(int32(depth*2) * int32(int16(oldProgress-newProgress)) / int32(lateral))
	if inside {
		tangent = -tangent
	}
	return Contact{Depth: depth, Tangent: tangent, Normal: normal}, true
}

// ResolveWalls applies the same radial walls and oriented pit polylines as
// the original vehicle loop. Each corrected pose is checked again so a corner
// cannot push the car through the adjacent wall.
func (w *World) ResolveWalls(state *CollisionState, previous Vector2, previousLocation TrackLocation, trackHeading int16) (TrackLocation, bool) {
	hit := false
	location := w.LocateTrack(state.Position)
	for attempt := 0; attempt < 8; attempt++ {
		contact, found := w.SurfaceContact(previousLocation, location, trackHeading)
		if !found {
			for _, wall := range w.PitBarriers(previousLocation.Progress, previousLocation.Lateral) {
				if c, ok := w.WallContact(state.Position, previous, state.TravelHeading, wall); ok {
					contact, found = c, true
					break
				}
			}
		}
		if !found {
			return location, hit
		}
		w.ApplyContact(state, contact)
		hit = true
		location = w.LocateTrack(state.Position)
	}
	return location, hit
}

// VelocityToWorld and VelocityOnTrack change basis without changing the Q7
// velocity unit used by the original collision response.
func (w *World) VelocityToWorld(forward, lateral, heading int16) Vector2 {
	c, s := w.cosSin(heading)
	return Vector2{q14(forward, c) + q14(lateral, s), q14(lateral, c) - q14(forward, s)}
}
func (w *World) VelocityOnTrack(velocity Vector2, heading int16) (int16, int16) {
	c, s := w.cosSin(heading)
	return q14(velocity.X, c) - q14(velocity.Y, s), q14(velocity.X, s) + q14(velocity.Y, c)
}

// VelocityMagnitude retains the source integer length, including the unsigned
// product sum used by a contact that transfers signed track velocities.
func VelocityMagnitude(velocity Vector2) uint16 {
	x, y := int32(velocity.X), int32(velocity.Y)
	return uint16(integerSquareRoot(uint32(x*x + y*y)))
}

func (w *World) LoadCarContactDamage(data []byte) error {
	if len(data) != 17 {
		return fmt.Errorf("invalid recovered car-contact damage table")
	}
	copy(w.carContactDamage[:], data)
	return nil
}

func (w *World) CarContactDamage(severity uint16, relative int16) byte {
	if severity < 250 {
		return 0
	}
	offset := 0
	if severity >= 600 {
		offset = 8
	}
	index := (uint16(relative)>>8 + 15) >> 5
	return w.carContactDamage[offset+int(index)]
}
