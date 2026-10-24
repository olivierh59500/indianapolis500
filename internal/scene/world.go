// Package scene renders the recovered Indianapolis 500 world in native Go.
package scene

import (
	"encoding/binary"
	"fmt"
)

const (
	Width               = 320
	Height              = 124
	dataReferenceBase   = 0x09c0
	vertexReferenceBase = 0x6652
)

// Point stores the original signed, fifteen-bit world coordinates.
type Point struct{ X, Y, Z int16 }

// Camera uses the original 16-bit full-turn angle convention.
type Camera struct {
	X, Y, Z              int16
	Heading, Pitch, Roll int16
}

// Matrix contains the three transformed contributions of each world axis.
// Depth and horizontal coefficients use Q14; vertical coefficients use Q13.
type Matrix struct {
	Depth, Horizontal, Vertical [3]int16
}

// Projected is a cached view-space point and its original viewport outcodes.
type Projected struct {
	Depth, Horizontal, Vertical int16
	X, Y                        int
	Flags                       uint8
}

// World owns recovered scene and track tables. It contains no processor code
// or Amiga device state: graphics records select native drawing operations.
type World struct {
	Points           []Point
	data             []byte
	sine             [1024]int16
	horizon          []int16
	lod              [100]int16
	trackCosine      [3961]uint16
	trackBank        [42]uint16
	arctangent       [2049]int16
	bankGradient     [88]int16
	cameraAnchors    [17]cameraAnchor
	innerLimits      [496]int16
	outerLimits      [496]int16
	barriers         [5][]Wall
	collisionDamage  [2][8]byte
	carContactDamage [17]byte
	crowd            []byte
}

func New(vertices, data, trigonometry, horizon, lod []byte) (*World, error) {
	if len(vertices)%6 != 0 || len(trigonometry) != 2048 || len(horizon) != 178 || len(lod) != 200 {
		return nil, fmt.Errorf("recovered world data has invalid dimensions")
	}
	w := &World{data: data, Points: make([]Point, len(vertices)/6), horizon: make([]int16, len(horizon)/2)}
	for i := range w.Points {
		offset := i * 6
		w.Points[i] = Point{int16(binary.BigEndian.Uint16(vertices[offset:])), int16(binary.BigEndian.Uint16(vertices[offset+2:])), int16(binary.BigEndian.Uint16(vertices[offset+4:]))}
	}
	for i := range w.sine {
		w.sine[i] = int16(binary.BigEndian.Uint16(trigonometry[i*2:]))
	}
	for i := range w.horizon {
		w.horizon[i] = int16(binary.BigEndian.Uint16(horizon[i*2:]))
	}
	for i := range w.lod {
		w.lod[i] = int16(binary.BigEndian.Uint16(lod[i*2:]))
	}
	return w, nil
}

func (w *World) cosSin(angle int16) (int16, int16) {
	index := int(uint16(angle+31) >> 6)
	return w.sine[index], w.sine[(index+768)&1023]
}

func q14(a, b int16) int16 { return int16((int32(a) * int32(b)) >> 14) }

// Matrix follows the original fixed-point camera calculation at $b740.
func (w *World) Matrix(camera Camera) Matrix {
	ch, sh := w.cosSin(-camera.Heading)
	cr, sr := w.cosSin(camera.Roll)
	cp, sp := w.cosSin(camera.Pitch)
	first, second := q14(-sp, sr), q14(-sp, cr)
	m := Matrix{
		Depth:      [3]int16{q14(ch, cp), q14(sh, cp), sp},
		Horizontal: [3]int16{q14(first, ch) - q14(sh, cr), q14(ch, cr) + q14(first, sh), q14(cp, sr)},
		Vertical:   [3]int16{q14(q14(second, ch)+q14(sh, sr), 17617), q14(q14(second, sh)-q14(ch, sr), 17617), q14(q14(cp, cr), 17617)},
	}
	pitch := int(camera.Pitch)
	if pitch < 0 {
		pitch = -pitch
	}
	if pitch >= 0x0d48 {
		for axis := range 3 {
			m.Depth[axis] >>= 1
			m.Horizontal[axis] >>= 1
			m.Vertical[axis] >>= 1
		}
	}
	return m
}

// Transform truncates each component separately, just as the original
// component cache does. Combining products before rounding changes pixels.
func (m Matrix) Transform(point Point, camera Camera) Projected {
	delta := [3]int16{point.X - camera.X, point.Y - camera.Y, point.Z - camera.Z}
	var p Projected
	for axis, value := range delta {
		p.Depth += q14(value, m.Depth[axis])
		p.Horizontal += q14(value, m.Horizontal[axis])
		p.Vertical += int16((int32(value) * int32(m.Vertical[axis])) >> 13)
	}
	p.project()
	return p
}

func (p *Projected) project() {
	d, h, v := int32(p.Depth), int32(p.Horizontal), int32(p.Vertical)
	p.Flags = 0
	if v <= -d {
		p.Flags |= 16
	}
	if h < -d {
		p.Flags |= 8
	}
	if d < 0 {
		p.Flags |= 1
	}
	if v > d {
		p.Flags |= 4
	}
	if h >= d {
		p.Flags |= 2
	}
	if p.Flags == 0 && d != 0 {
		p.X, p.Y = 160+int(h*160/d), 62-int(v*62/d)
	}
}

func (w *World) word(reference int) (uint16, bool) {
	offset := reference - dataReferenceBase
	if offset < 0 || offset+2 > len(w.data) {
		return 0, false
	}
	return binary.BigEndian.Uint16(w.data[offset:]), true
}

func (w *World) point(reference uint16) (Point, bool) {
	offset := int(reference) - vertexReferenceBase
	if offset < 0 || offset%8 != 0 || offset/8 >= len(w.Points) {
		return Point{}, false
	}
	return w.Points[offset/8], true
}

func (w *World) literalPoint(reference uint16) (Point, bool) {
	x, ok := w.word(int(reference) + 2)
	if !ok {
		return Point{}, false
	}
	y, ok := w.word(int(reference) + 4)
	if !ok {
		return Point{}, false
	}
	z, ok := w.word(int(reference) + 6)
	if !ok {
		return Point{}, false
	}
	return Point{int16(x), int16(y), int16(z)}, true
}

// Direct-vector records round once per dot product. Sky vectors are
// directions, so their transform omits the camera's world translation.
func (m Matrix) TransformDirect(point Point, camera Camera, translate bool) Projected {
	delta := [3]int16{point.X, point.Y, point.Z}
	if translate {
		delta[0] -= camera.X
		delta[1] -= camera.Y
		delta[2] -= camera.Z
	}
	var depth, horizontal, vertical int32
	for axis, value := range delta {
		depth += int32(value) * int32(m.Depth[axis])
		horizontal += int32(value) * int32(m.Horizontal[axis])
		vertical += int32(value) * int32(m.Vertical[axis])
	}
	p := Projected{Depth: int16(depth >> 14), Horizontal: int16(horizontal >> 14), Vertical: int16(vertical >> 13)}
	p.project()
	return p
}
