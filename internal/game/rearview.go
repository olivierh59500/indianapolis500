package game

// These original distance buckets and bearing corrections belong to the
// narrow rear view. The normal world renderer keeps its separate tables.
var mirrorDistanceTable = [...]uint16{800, 470, 260, 163, 137, 116, 98, 83, 70, 60, 51}
var mirrorBearingTable = [...]int16{180, 180, 179, 178, 177, 176, 175, 174, 172, 171, 170, 169, 168, 167, 165, 164, 163, 162, 160, 159, 157, 156, 154, 153, 151, 149, 148, 146, 144, 142, 141, 139, 137, 135, 132, 130, 128, 126, 123, 121, 118, 116, 113, 111, 108, 105, 102, 99, 96, 93, 89, 86, 83, 79, 76, 72, 68, 64, 60, 56, 52, 48, 44, 40, 36, 31, 27, 22, 18, 14, 9, 5, 0, -5, -9, -14, -18, -22, -27, -31, -36, -40, -44, -48, -52, -56, -60, -64, -68, -72, -76, -79, -83, -86, -89, -93, -96, -99, -102, -105, -108, -111, -113, -116, -118, -121, -123, -126, -128, -130, -132, -135, -137, -139, -141, -142, -144, -146, -148, -149, -151, -153, -154, -156, -157, -159, -160, -162, -163, -164, -165, -167, -168, -169, -170, -171, -172, -174, -175, -176, -177, -178, -179, -180}

func mirrorCarSize(depth int) (tier, scale int) {
	previous := int(mirrorDistanceTable[0])
	for tier < len(mirrorDistanceTable)-2 && depth < int(mirrorDistanceTable[tier+1]) {
		previous = int(mirrorDistanceTable[tier+1])
		tier++
	}
	threshold := int(mirrorDistanceTable[tier+1])
	depth = max(depth, threshold)
	if previous > threshold {
		scale = min(7, max(0, (depth-threshold)*8/(previous-threshold)))
	}
	return
}

func (p *carProjection) mirrorFacing(heading int16, offset int) (angle int, mirrored bool) {
	value := (int(uint16(uint16(0x8000)-uint16(heading))>>6) - int(mirrorBearingTable[max(0, min(143, offset+72))])) & 1023
	if value >= 512 {
		value, mirrored = 1024-value, true
	}
	angle = len(p.angles) - 1
	for i, threshold := range p.angles {
		if value >= int(threshold) {
			angle = len(p.angles) - 1 - i
			break
		}
	}
	return
}

// The rear-view image is an ordinary eighty-pixel canvas. The original
// scanline artwork uses the same scale and color records as forward cars.
func (g *Game) paintMirrorCarShape(anchorX, tier, angle, scale, primary, secondary int, mirrored bool) {
	shape := &g.carShapes[tier][angle]
	scaler := &g.carProjection.scalers[scale]
	for _, strip := range shape.strips {
		color := strip.color & 15
		if strip.color == 12 {
			color = byte(primary)
		}
		if strip.color == 9 {
			color = byte(secondary)
		}
		if int(strip.y) >= len(scaler.rows) {
			continue
		}
		y := int(scaler.rows[strip.y]) - 60
		if y < 0 || y >= 11 {
			continue
		}
		counter := scaler.skip[strip.y]
		count := min(len(strip.spans), 11-y)
		for _, span := range strip.spans[:count] {
			counter--
			if counter == 0 {
				counter = scaler.rows[124]
				continue
			}
			left, right := int(scaler.x[span.left])/2, int(scaler.x[span.right])/2
			if mirrored {
				left, right = -right-1, -left-1
			}
			left, right = max(0, anchorX+left), min(79, anchorX+right)
			for x := left; x <= right; x++ {
				g.carProjection.mirrorPixels[y*80+x] = color
			}
			y++
		}
	}
}

// The two forty-pixel mirrors overlap the middle eight pixels of the rear
// image. They occupy existing cockpit pixels rather than another world view.
func (g *Game) copyRearViewMirrors() {
	if !g.carProjection.mirrorsReady {
		return
	}
	for y := range 11 {
		for x := range 40 {
			g.dashboard.pixels[(154+y)*320+x] = g.carProjection.mirrorPixels[y*80+x]
			g.dashboard.pixels[(154+y)*320+280+x] = g.carProjection.mirrorPixels[y*80+32+x]
		}
	}
}
