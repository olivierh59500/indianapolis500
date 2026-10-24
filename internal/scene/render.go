package scene

import "fmt"

// Operation selects one native scene drawing operation. References address
// graphics records in the recovered data section, never executable code.
type Operation struct {
	Kind      string    `json:"kind"`
	Reference int       `json:"data_ref"`
	Primitive uint16    `json:"primitive"`
	Color     uint8     `json:"color"`
	Count     int       `json:"count"`
	Reverse   bool      `json:"reverse"`
	Section   int       `json:"section"`
	Filter    CarFilter `json:"filter,omitempty"`
	Mode      uint8     `json:"mode,omitempty"`
}

type viewPoint struct {
	d, h, v int32
	border  uint8
}
type pixelPoint struct{ x, y int }

// Renderer reuses its indexed framebuffer and projection cache across frames.
type Renderer struct {
	world        *World
	Pixels       []uint8
	projected    []Projected
	left, right  [Height]int
	clipA, clipB [64]viewPoint
	polygon      [64]pixelPoint
	camera       Camera
	matrix       Matrix
	drawCars     func(CarFilter)
	fill         surfaceFill
	CrowdPhase   uint8
}

func (w *World) NewRenderer() *Renderer {
	return &Renderer{world: w, Pixels: make([]uint8, Width*Height), projected: make([]Projected, len(w.Points))}
}

func (r *Renderer) Render(camera Camera, operations []Operation) error {
	matrix := r.world.Matrix(camera)
	r.camera, r.matrix = camera, matrix
	for i, point := range r.world.Points {
		r.projected[i] = matrix.Transform(point, camera)
	}
	for i := range r.Pixels {
		r.Pixels[i] = 7
	}
	for _, operation := range operations {
		r.fill = solidFill
		switch operation.Kind {
		case "horizon":
			r.background(camera)
		case "segments":
			cursor := operation.Reference
			for i := 0; i < operation.Count; i++ {
				first, ok := r.world.word(cursor)
				if !ok {
					return fmt.Errorf("invalid segment record %x", cursor)
				}
				second, ok := r.world.word(cursor + 2)
				if !ok {
					return fmt.Errorf("invalid segment record %x", cursor+2)
				}
				if err := r.drawReferences([]uint16{first, second}, operation.Color&15, false); err != nil {
					return err
				}
				cursor += 4
			}
		case "cars":
			if r.drawCars != nil {
				r.drawCars(operation.Filter)
			}
		case "object":
			if operation.Primitive == 0x878 && operation.Mode != 0 || operation.Primitive == 0x886 && operation.Mode == 2 {
				r.fill = crowdFill
				if len(r.world.crowd) == 0 {
					return fmt.Errorf("missing recovered crowd patterns")
				}
			} else if operation.Primitive == 0x90a {
				r.fill = checkerFill
			}
			switch operation.Primitive {
			case 2, 18, 0xa28:
				// The original registers these records for a later rendering pass.
			case 0x570:
				r.pitDashes(camera, matrix, operation.Reverse)
			case 0x2e, 0x710, 0x878, 0x886, 0x90a:
				if err := r.groups(operation.Reference, operation.Primitive != 0x2e, 0); err != nil {
					return err
				}
			case 0x28a, 0x800:
				if err := r.groups(operation.Reference, operation.Primitive == 0x800, 2); err != nil {
					return err
				}
			case 0x4f0:
				if err := r.quadStrip(operation.Reference); err != nil {
					return err
				}
			case 0x3b8:
				if err := r.sectionLines(operation); err != nil {
					return err
				}
			default:
				return fmt.Errorf("unsupported original scene operation %x at %x", operation.Primitive, operation.Reference)
			}
		default:
			return fmt.Errorf("unknown scene operation %q", operation.Kind)
		}
	}
	return nil
}

func (r *Renderer) background(camera Camera) {
	pitch := int(int16(camera.Pitch+31) >> 6)
	roll := int(int16(camera.Roll+31) >> 6)
	pitch = max(-30, min(30, pitch))
	roll = max(-27, min(27, roll))
	center := int(r.world.horizon[pitch+30])
	slope := int(r.world.horizon[61+abs(roll)])
	if roll < 0 {
		slope = -slope
	}
	top, bottom := center-abs(slope), center+abs(slope)
	for y := 0; y < Height; y++ {
		value := uint8(11)
		if y >= bottom {
			value = 7
		}
		for x := 0; x < Width; x++ {
			r.Pixels[y*Width+x] = value
		}
	}
	if slope == 0 {
		return
	}
	// The original tilted contour emits only its first dot on each row.
	// Its rising version starts at X304; final samples wrap to the previous
	// row and leave isolated edge dots outside the filled horizon band.
	horizon, error := bottom-1, (Width-1)/2
	delta, previousY := 2*abs(slope), -32768
	for sample := 0; sample < Width; sample++ {
		if horizon != previousY {
			x, y := sample, horizon
			if slope > 0 {
				x = Width - 16 - sample
			}
			if x < 0 {
				x += Width
				y--
			}
			if x >= 0 && x < Width && y >= 0 && y < Height {
				r.Pixels[y*Width+x] = 7
			}
			previousY = horizon
		}
		error -= delta
		if error < 0 {
			horizon--
			error += Width - 1
		}
	}
	for y := max(0, top); y < min(Height, bottom); y++ {
		inside := slope < 0
		for x := Width - 1; x >= 0; x-- {
			edge := r.Pixels[y*Width+x] == 7
			if edge {
				inside = !inside
			}
			if edge || inside {
				r.Pixels[y*Width+x] = 7
			} else {
				r.Pixels[y*Width+x] = 11
			}
		}
	}

}

func (r *Renderer) groups(reference int, filled bool, projection int) error {
	header, ok := r.world.word(reference)
	if !ok {
		return fmt.Errorf("invalid scene record %x", reference)
	}
	groups, color, cursor := int(header>>8)+1, uint8(header&15), reference+2
	var refs [64]uint16
	for group := 0; group < groups; group++ {
		count, ok := r.world.word(cursor)
		if !ok {
			return fmt.Errorf("invalid scene group %x", cursor)
		}
		vertices := int(count) + 2
		cursor += 2
		if vertices > len(refs) {
			return fmt.Errorf("original polygon has %d vertices", vertices)
		}
		for i := 0; i < vertices; i++ {
			ref, ok := r.world.word(cursor)
			if !ok {
				return fmt.Errorf("invalid vertex reference at %x", cursor)
			}
			refs[i] = ref
			cursor += 2
		}
		if err := r.drawReferencesMode(refs[:vertices], color, filled, projection); err != nil {
			return err
		}
	}
	return nil
}

func (r *Renderer) drawReferences(refs []uint16, color uint8, filled bool) error {
	return r.drawReferencesMode(refs, color, filled, 0)
}

func (r *Renderer) drawReferencesMode(refs []uint16, color uint8, filled bool, projection int) error {
	for i, ref := range refs {
		if projection != 0 {
			point, ok := r.world.literalPoint(ref)
			if !ok {
				return fmt.Errorf("invalid direct-vector reference %x", ref)
			}
			p := r.matrix.TransformDirect(point, r.camera, projection == 1)
			r.clipA[i] = viewPoint{d: int32(p.Depth), h: int32(p.Horizontal), v: int32(p.Vertical)}
			continue
		}
		index := (int(ref) - vertexReferenceBase) / 8
		if int(ref) < vertexReferenceBase || (int(ref)-vertexReferenceBase)%8 != 0 || index >= len(r.projected) {
			return fmt.Errorf("unsupported world vertex reference %x", ref)
		}
		p := r.projected[index]
		r.clipA[i] = viewPoint{d: int32(p.Depth), h: int32(p.Horizontal), v: int32(p.Vertical)}
	}
	if filled {
		r.drawPolygon(r.clipA[:len(refs)], color)
	} else {
		for i := 1; i < len(refs); i++ {
			r.drawSegment(r.clipA[i-1], r.clipA[i], color)
		}
	}
	return nil
}

// contour translates the original scanline-edge writer. Shallow edges use
// the first crossing of each row; long shallow edges use an integer quotient
// and remainder with a half-step start. Keeping one X per edge per row is
// essential: taking the full Bresenham boundary enlarges filled polygons.
func (r *Renderer) contour(a, b pixelPoint, values []int) {
	if a.y == b.y {
		return
	}
	if a.y > b.y {
		a, b = b, a
	}
	dy := b.y - a.y
	dx := b.x - a.x
	step := 1
	if dx < 0 {
		dx = -dx
		step = -1
	}
	x, y := a.x, a.y
	store := func() {
		if y >= 0 && y < len(values) {
			values[y] = x
		}
	}
	store()
	if dx <= dy {
		error := dy / 2
		for y = a.y + 1; y <= b.y; y++ {
			error -= dx
			if error < 0 {
				x += step
				error += dy
			}
			store()
		}
		return
	}
	if dx < 10 {
		error := dx / 2
		for i := 0; i < dx; i++ {
			x += step
			error -= dy
			if error < 0 {
				y++
				error += dx
				store()
			}
		}
		return
	}
	quotient, remainder := dx/dy, dx%dy
	x += (step * quotient) >> 1
	y++
	store()
	error := dy / 2
	for y++; y <= b.y; y++ {
		x += step * quotient
		// The original subtracts the remainder; reversing this recurrence
		// changes the exact tie rows even when most pixels are identical.
		error -= remainder
		if error < 0 {
			x += step
			error += dy
		}
		store()
	}
}

func (r *Renderer) line(a, b pixelPoint, color uint8, boundary bool) {
	dx, dy := abs(b.x-a.x), -abs(b.y-a.y)
	sx, sy := 1, 1
	if a.x > b.x {
		sx = -1
	}
	if a.y > b.y {
		sy = -1
	}
	err := dx + dy
	for {
		if a.y >= 0 && a.y < Height {
			if boundary {
				r.left[a.y] = min(r.left[a.y], a.x)
				r.right[a.y] = max(r.right[a.y], a.x)
			} else if a.x >= 0 && a.x < Width {
				r.Pixels[a.y*Width+a.x] = color
			}
		}
		if a == b {
			return
		}
		twice := 2 * err
		if twice >= dy {
			err += dy
			a.x += sx
		}
		if twice <= dx {
			err += dx
			a.y += sy
		}
	}
}

func (r *Renderer) pitDashes(camera Camera, matrix Matrix, reverse bool) {
	start := int(int16(uint16(camera.X) & 0xff00))
	end := start + 0x300
	if reverse {
		start -= 0x300
	}
	start = max(-0x1000, start)
	end = min(0x1f00, end)
	points := [4]Point{{int16(start + 256), 6873, 0}, {int16(start), 6873, 0}, {int16(start), 6870, 0}, {int16(start + 256), 6870, 0}}
	var quad [4]viewPoint
	for i, p := range points {
		q := matrix.TransformDirect(p, camera, true)
		quad[i] = viewPoint{d: int32(q.Depth), h: int32(q.Horizontal), v: int32(q.Vertical)}
	}
	// $153dc projects the full interval once, halves its view-space delta
	// for the gap, then advances the same rounded delta for every dash.
	// Reprojecting new world coordinates accumulates different rounding.
	delta := viewPoint{d: int32(int16(quad[3].d - quad[2].d)), h: int32(int16(quad[3].h - quad[2].h)), v: int32(int16(quad[3].v - quad[2].v))}
	advance := func(p *viewPoint, step viewPoint) {
		p.d = int32(int16(p.d + step.d))
		p.h = int32(int16(p.h + step.h))
		p.v = int32(int16(p.v + step.v))
	}
	half := viewPoint{d: delta.d >> 1, h: delta.h >> 1, v: delta.v >> 1}
	advance(&quad[1], half)
	advance(&quad[2], half)
	for x := start; x <= end; x += 256 {
		r.drawPolygon(quad[:], 8)
		for i := range quad {
			advance(&quad[i], delta)
		}
	}
}

func abs(value int) int {
	if value < 0 {
		return -value
	}
	return value
}
