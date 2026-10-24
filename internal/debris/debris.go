// Package debris implements the original collision effects with native vectors
// and a fixed pool of sparks, flashes and bouncing triangular fragments.
package debris

import (
	"encoding/json"
	"fmt"

	"github.com/olivierh59500/indianapolis500/internal/scene"
)

type Kind uint8

const (
	Sparks Kind = iota
	Flash
	Fragments
)

type DrawMode uint8

const (
	Hidden DrawMode = iota
	Spark
	Filled
	Wire
)

type Piece struct {
	Velocity     scene.Point
	Mode         DrawMode
	Color, Style uint8
	Vertices     [3]uint8
	Age          uint16
}

type Effect struct {
	Active bool
	Kind   Kind
	Life   uint8
	Fresh  bool
	Points [24]scene.Point
	Pieces [8]Piece
}

type Input struct {
	Severity, Colors           uint16
	Forward, Sideways, Lateral int16
	Distance                   uint16
}

type Pool struct {
	Count  int
	Origin scene.Point
	Groups [18]Effect
}

type Model struct {
	Templates [3]Effect
	Pool
}

func New(data []byte) (Model, error) {
	var m Model
	if err := json.Unmarshal(data, &m.Templates); err != nil {
		return m, err
	}
	for i, e := range m.Templates {
		if e.Kind != Kind(i) {
			return m, fmt.Errorf("invalid recovered collision effect kind")
		}
		for _, piece := range e.Pieces {
			for _, v := range piece.Vertices {
				if v >= 24 {
					return m, fmt.Errorf("invalid recovered collision vertex")
				}
			}
		}
	}
	return m, nil
}

func (m *Model) Reset() { m.Pool = Pool{} }

func (m *Model) allocate(kind Kind) *Effect {
	if m.Count == len(m.Groups) {
		return nil
	}
	for i := range m.Groups {
		if m.Groups[i].Active {
			continue
		}
		e := &m.Groups[i]
		*e = m.Templates[kind]
		e.Active, e.Fresh = true, true
		e.Life = [3]uint8{15, 2, 75}[kind]
		m.Count++
		return e
	}
	return nil
}

func impactLevel(level uint16, position scene.Point, camera scene.Camera) uint8 {
	distance := scene.VelocityMagnitude(scene.Vector2{X: position.X - camera.X, Y: position.Y - camera.Y})
	if distance > 120 {
		level = uint16(uint32(level) * 120 / uint32(distance))
	}
	return uint8(level)
}

func (m *Model) place(e *Effect, input Input, world *scene.World) scene.TrackPoint {
	progress := uint16(6930 - input.Distance)
	if int16(progress) < 0 {
		progress += 39600
	}
	origin := world.TrackPlacement(progress, input.Lateral)
	m.Origin = origin.Point
	for i := range e.Points {
		e.Points[i].X += origin.X
		e.Points[i].Y += origin.Y
		e.Points[i].Z += origin.Z
	}
	return origin
}

func (m *Model) create(kind Kind, input Input, world *scene.World) {
	e := m.allocate(kind)
	if e == nil {
		return
	}
	origin := m.place(e, input, world)
	if kind != Fragments {
		return
	}
	forward := input.Forward >> 3
	magnitude := int16(scene.VelocityMagnitude(scene.Vector2{X: forward, Y: input.Sideways})) >> 4
	angle := world.Direction(forward, input.Sideways) + origin.Heading
	bias := world.VelocityToWorld(magnitude, 0, angle)
	scale := int32(input.Severity >> 5)
	remaining := int16(input.Severity)
	for i := range e.Pieces {
		p := &e.Pieces[i]
		p.Velocity.X = int16(int32(p.Velocity.X)*scale>>16) + bias.X
		p.Velocity.Y = int16(int32(p.Velocity.Y)*scale>>16) + bias.Y
		p.Velocity.Z = int16(int32(p.Velocity.Z) * scale >> 16)
		remaining -= 150
		if remaining < 0 {
			p.Mode = Hidden
		}
		if p.Color == 12 {
			p.Color = uint8(input.Colors >> 8)
		} else if p.Color == 9 {
			p.Color = uint8(input.Colors)
		}
	}
}

// Spawn retains the original strength tiers and first-free-slot allocation.
// A major hit requests fragments and a separate two-update flash.
func (m *Model) Spawn(input Input, world *scene.World, camera scene.Camera, requests *[4]uint8) {
	level := uint16(127)
	switch {
	case input.Severity < 150:
		m.create(Sparks, input, world)
		level = (input.Severity >> 3) + 60
	case input.Severity < 400:
		m.create(Flash, input, world)
		level = (input.Severity >> 3) + 75
	default:
		m.create(Fragments, input, world)
		m.create(Flash, input, world)
	}
	requests[0] = max(requests[0], impactLevel(level, m.Origin, camera))
}

// Update advances the effect pool before vehicle motion. Fragment requests
// follow slot/triangle order, including the original last-bounce priority write.
func (m *Model) Update(camera scene.Camera, requests *[4]uint8) {
	for i := range m.Groups {
		e := &m.Groups[i]
		if !e.Active || e.Life == 0 {
			continue
		}
		e.Life--
		if e.Kind == Sparks {
			p := &e.Pieces[0]
			p.Age++
			p.Style = uint8(min(uint16(3), (p.Age+3)>>3))
			continue
		}
		if e.Kind != Fragments {
			continue
		}
		for j := range e.Pieces {
			p := &e.Pieces[j]
			v := &p.Velocity
			v.Z--
			first := &e.Points[p.Vertices[0]]
			first.Z += v.Z
			if first.Z < 0 {
				first.Z -= v.Z
				v.X = (v.X + 1) >> 1
				v.Y = (v.Y + 1) >> 1
				v.Z = (-v.Z + 1) >> 1
				first.Z += v.Z
				if uint16(v.Z) >= 3 {
					level := min(uint16(127), uint16(v.Z)<<4)
					request := uint16(v.Z) & 3
					if request != 0 {
						request--
					}
					requests[1+request] = impactLevel(level, *first, camera)
				}
			}
			first.X += v.X
			first.Y += v.Y
			for k := 1; k < 3; k++ {
				point := &e.Points[p.Vertices[k]]
				point.X += v.X
				point.Y += v.Y
				point.Z += v.Z
			}
		}
	}
}

// Visible includes newly created effects whose lifetime may have elapsed
// before their first render. The original releases slots in the painter.
func (e Effect) Visible() bool { return e.Active && (e.Fresh || e.Life != 0) }

func (m *Model) Presented() {
	for i := range m.Groups {
		e := &m.Groups[i]
		if !e.Active {
			continue
		}
		e.Fresh = false
		if e.Life == 0 {
			e.Active = false
			m.Count--
		}
	}
}
