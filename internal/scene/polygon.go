package scene

// World and effect contours are clipped one edge at a time. Corner intersections are
// evaluated from the original edge; clipping an entire polygon
// through one fixed plane order introduces additional integer rounding.
func clipOutcode(p viewPoint) uint8 {
	var code uint8
	if !sourceInside(p, 2) {
		code |= 2
	}
	if !sourceInside(p, 4) {
		code |= 4
	}
	if !sourceInside(p, 1) {
		code |= 8
	}
	if !sourceInside(p, 3) {
		code |= 16
	}
	return code
}

func clipPolygonEndpoint(inside, outside viewPoint, code uint8) (viewPoint, bool) {
	plane := 4
	if code&2 != 0 {
		plane = 2
	} else if code&8 != 0 {
		plane = 1
	} else if code&16 != 0 {
		plane = 3
	}
	point := sourceIntersection(inside, outside, plane)
	if clipOutcode(point) == 0 {
		return point, true
	}
	// A corner is tested against the original edge again; re-clipping the
	// rounded intermediate intersection changes the final screen coordinate.
	if (plane == 1 || plane == 2) && code&20 != 0 {
		plane = 4
		if code&16 != 0 {
			plane = 3
		}
		point = sourceIntersection(inside, outside, plane)
		if clipOutcode(point) == 0 {
			return point, true
		}
	}
	return point, false
}

func clipPolygonEdge(a, b viewPoint) (viewPoint, viewPoint, bool) {
	ca, cb := clipOutcode(a), clipOutcode(b)
	if ca&cb != 0 {
		return a, b, false
	}
	var ok bool
	if ca != 0 {
		a, ok = clipPolygonEndpoint(b, a, ca)
		if !ok {
			return a, b, false
		}
	}
	if cb != 0 {
		b, ok = clipPolygonEndpoint(a, b, cb)
		if !ok {
			return a, b, false
		}
	}
	return a, b, true
}

func boundaryPosition(p pixelPoint) int {
	if p.y == 0 {
		return p.x
	}
	if p.x == Width-1 {
		return Width - 1 + p.y
	}
	if p.y == Height-1 {
		return Width - 1 + Height - 1 + Width - 1 - p.x
	}
	return 2*(Width-1) + Height - 1 + Height - 1 - p.y
}

func (r *Renderer) drawPolygon(points []viewPoint, color uint8) {
	var near [64]viewPoint
	count := 0
	// Preserve the original edge order. A closing eye-plane intersection
	// belongs after the original first vertex, not before it; scanline ties
	// at that vertex must be resolved by the closing contour last.
	for i, current := range points {
		next := points[(i+1)%len(points)]
		inside, nextInside := sourceInside(current, 0), sourceInside(next, 0)
		if inside {
			near[count] = current
			count++
		}
		if inside != nextInside {
			a, b := current, next
			if !inside {
				a, b = b, a
			}
			near[count] = sourceIntersection(a, b, 0)
			count++
		}
	}
	if count < 2 {
		return
	}
	winding := int64(0)
	for i := 1; i < count-1 && winding == 0; i++ {
		a, b, c := near[0], near[i], near[i+1]
		winding = -(int64(a.d)*(int64(b.h)*int64(c.v)-int64(c.h)*int64(b.v)) - int64(a.h)*(int64(b.d)*int64(c.v)-int64(c.d)*int64(b.v)) + int64(a.v)*(int64(b.d)*int64(c.h)-int64(c.d)*int64(b.h)))
	}
	if winding == 0 {
		// At an exact eye-plane tie, the source's boundary nudges can give
		// a flat homogeneous contour a nonzero visible winding. Its clipped
		// screen edges still determine which boundary arc joins them.
		var first, previous pixelPoint
		havePoint := false
		for i := 0; i < count; i++ {
			a, b, visible := clipPolygonEdge(near[i], near[(i+1)%count])
			if !visible {
				continue
			}
			for _, p := range [2]pixelPoint{sourcePixel(a), sourcePixel(b)} {
				if !havePoint {
					first, havePoint = p, true
				} else {
					winding += int64(previous.x)*int64(p.y) - int64(p.x)*int64(previous.y)
				}
				previous = p
			}
		}
		winding += int64(previous.x)*int64(first.y) - int64(first.x)*int64(previous.y)
	}
	var polygon [128]pixelPoint
	n := 0
	appendPoint := func(p pixelPoint) {
		if n == 0 || polygon[n-1] != p {
			polygon[n] = p
			n++
		}
	}
	const perimeter = 2 * (Width - 1 + Height - 1)
	corners := [4]pixelPoint{{0, 0}, {Width - 1, 0}, {Width - 1, Height - 1}, {0, Height - 1}}
	cornerPositions := [4]int{0, Width - 1, Width - 1 + Height - 1, 2*(Width-1) + Height - 1}
	join := func(a, b pixelPoint) {
		start, end := boundaryPosition(a), boundaryPosition(b)
		if winding >= 0 {
			if end < start {
				end += perimeter
			}
			for j := 0; j < 8; j++ {
				i := j % 4
				t := cornerPositions[i] + j/4*perimeter
				if t > start && t < end {
					appendPoint(corners[i])
				}
			}
		} else {
			if end > start {
				end -= perimeter
			}
			for j := 7; j >= 0; j-- {
				i := j % 4
				t := cornerPositions[i] + (j/4-1)*perimeter
				if t < start && t > end {
					appendPoint(corners[i])
				}
			}
		}
	}
	for i := 0; i < count; i++ {
		a, b, ok := clipPolygonEdge(near[i], near[(i+1)%count])
		if !ok {
			continue
		}
		first, last := sourcePixel(a), sourcePixel(b)
		if n > 0 && polygon[n-1] != first {
			join(polygon[n-1], first)
		}
		appendPoint(first)
		appendPoint(last)
	}
	if n < 2 {
		// With no visible original edge, a convex polygon either encloses the
		// entire view or misses it. Test the center ray using homogeneous edges.
		contains := true
		sign := int64(0)
		for i := 0; i < count; i++ {
			a, b := near[i], near[(i+1)%count]
			value := int64(a.h)*int64(b.v) - int64(b.h)*int64(a.v)
			if value == 0 {
				continue
			}
			if sign == 0 {
				sign = value
			} else if (sign < 0) != (value < 0) {
				contains = false
				break
			}
		}
		if contains && sign != 0 {
			for i := range r.Pixels {
				r.Pixels[i] = color & 15
			}
		}
		return
	}
	if polygon[n-1] != polygon[0] {
		join(polygon[n-1], polygon[0])
	}
	for y := range Height {
		r.left[y], r.right[y] = -32768, -32768
	}
	for i := 0; i < n; i++ {
		a, b := polygon[i], polygon[(i+1)%n]
		if a.y < b.y {
			r.contour(a, b, r.left[:])
		} else if a.y > b.y {
			r.contour(a, b, r.right[:])
		}
	}
	for y := range Height {
		if r.left[y] == -32768 || r.right[y] == -32768 {
			continue
		}
		left, right := max(0, min(r.left[y], r.right[y])), min(Width-1, max(r.left[y], r.right[y]))
		r.surfaceSpan(left, right, y, color)
	}
}
