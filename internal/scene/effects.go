package scene

// PaintWorldTriangle uses the original direct-coordinate projection and late
// collision-effect pass. The wire primitive is an open three-point polyline.
func (r *Renderer) PaintWorldTriangle(points [3]Point, color uint8, wire bool) {
	var projected [3]viewPoint
	for i, p := range points {
		q := r.matrix.TransformDirect(p, r.camera, true)
		projected[i] = viewPoint{d: int32(q.Depth), h: int32(q.Horizontal), v: int32(q.Vertical)}
	}
	r.fill = solidFill
	if wire {
		r.drawSegment(projected[0], projected[1], color&15)
		r.drawSegment(projected[1], projected[2], color&15)
	} else {
		r.drawPolygon(projected[:], color&15)
	}
}

// PaintWorldSpark projects the original expanding, stippled collision disc.
func (r *Renderer) PaintWorldSpark(point Point, age uint16, style, color uint8) {
	q := r.matrix.TransformDirect(point, r.camera, true)
	depth := int32(q.Depth)
	if depth <= 4 {
		return
	}
	size := age
	pitch := int(r.camera.Pitch)
	if pitch < 0 {
		pitch = -pitch
	}
	if pitch >= 0xd48 {
		size >>= 1
	}
	x := 160 + int(int32(q.Horizontal)*160/depth)
	y := 62 - int(int32(q.Vertical)*62/depth)
	radius := int(int32(size) * 160 / depth)
	r.sparkDisc(x, y, radius, style, color)
}

func (r *Renderer) sparkDisc(x, y, radius int, style, color uint8) {
	if radius >= 64 || radius < 0 {
		return
	}
	top, bottom := max(0, y-radius), min(Height-1, y+radius)
	if bottom <= top || x+radius < 0 || x-radius > Width-1 {
		return
	}
	// These are source circle contour rows, centered in a small local table.
	var left [129]int
	set := func(row, value int) {
		if row >= -64 && row <= 64 {
			left[row+64] = value
		}
	}
	lower, upper, middleDown, middleUp := radius+1, -radius-1, 0, 0
	near, far := x, x-radius
	d2, d3, d4 := -4*radius, 5, 3-2*radius
	set(lower-1, near)
	set(upper+1, near)
	for d2 < 0 {
		if d4 < 0 {
			set(middleDown, far)
			middleDown++
			set(middleUp, far)
			middleUp--
			d4 += d3
			near--
			d3 += 4
			d2 += 4
		} else {
			lower--
			set(lower, near)
			upper++
			set(upper, near)
			set(middleDown, far)
			middleDown++
			set(middleUp, far)
			middleUp--
			d4 += d2 + 10
			near--
			far++
			d3 += 4
			d2 += 8
		}
	}
	if d2 == 0 {
		lower--
		set(lower, near)
		upper++
		set(upper, near)
		set(middleDown, far)
		set(middleUp, far)
	}
	patterns := [4]uint32{0xffffffff, 0xaaaa5555, 0x88882222, 0x80800808}
	pattern := patterns[min(int(style), 3)]
	for row := top; row <= bottom; row++ {
		start := max(0, left[row-y+64])
		end := min(Width-1, 2*x-left[row-y+64])
		bits := uint16(pattern)
		if row&1 != 0 {
			bits = uint16(pattern >> 16)
		}
		for column := start; column <= end; column++ {
			if bits&(1<<uint(15-column%16)) != 0 {
				r.Pixels[row*Width+column] = color & 15
			}
		}
	}
}
