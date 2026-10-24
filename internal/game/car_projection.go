package game

import (
	"encoding/binary"
	"fmt"

	gameassets "github.com/olivierh59500/indianapolis500/assets"
	worldscene "github.com/olivierh59500/indianapolis500/internal/scene"
)

type carScaler struct {
	skip, rows [125]byte
	x          [256]int16
}

type carProjection struct {
	scalers           [8]carScaler
	bearings          [641]int16
	distances         [21]uint16
	overheadDistances [11]uint16
	angles            [16]uint16
	overheadAngles    [32]uint16
	mirrorPixels      [80 * 11]byte
	mirrorsReady      bool
}

func (g *Game) loadCarProjection() error {
	read := func(name string, size int) ([]byte, error) {
		data, err := gameassets.Original.ReadFile("original/" + name)
		if err != nil {
			return nil, err
		}
		if len(data) != size {
			return nil, fmt.Errorf("recovered %s has invalid dimensions", name)
		}
		return data, nil
	}
	scalers, err := read("car-scalers.bin", 6096)
	if err != nil {
		return err
	}
	for i := range g.carProjection.scalers {
		s := &g.carProjection.scalers[i]
		data := scalers[i*762 : (i+1)*762]
		copy(s.skip[:], data[:125])
		copy(s.rows[:], data[125:250])
		for x := range s.x {
			s.x[x] = int16(binary.BigEndian.Uint16(data[250+x*2:]))
		}
	}
	bearings, err := read("car-bearings.bin", 1282)
	if err != nil {
		return err
	}
	for i := range g.carProjection.bearings {
		g.carProjection.bearings[i] = int16(binary.BigEndian.Uint16(bearings[i*2:]))
	}
	visibility, err := read("car-visibility.bin", 166)
	if err != nil {
		return err
	}
	for i := range g.carProjection.distances {
		g.carProjection.distances[i] = binary.BigEndian.Uint16(visibility[i*2:])
	}
	for i := range g.carProjection.angles {
		g.carProjection.angles[i] = binary.BigEndian.Uint16(visibility[44+i*2:])
	}
	for i := range g.carProjection.overheadDistances {
		g.carProjection.overheadDistances[i] = binary.BigEndian.Uint16(visibility[78+i*2:])
	}
	for i := range g.carProjection.overheadAngles {
		g.carProjection.overheadAngles[i] = binary.BigEndian.Uint16(visibility[102+i*2:])
	}
	return nil
}

// carSize translates the original distance buckets and eight intermediate
// scale tables. Their nonlinear X mapping also expands compressed wide spans.
func carSize(depth int, thresholds []uint16) (tier, scale int) {
	previous := 4096
	for tier < len(thresholds)-1 && depth < int(thresholds[tier]) {
		previous = int(thresholds[tier])
		tier++
	}
	threshold := int(thresholds[tier])
	if depth < threshold {
		depth = threshold
	}
	if previous > threshold {
		scale = min(7, max(0, (depth-threshold)*8/(previous-threshold)))
	}
	return tier, scale
}

// carFacing uses the original half-turn angle bins and mirrors the other
// half. The overhead view blends the screen-bearing correction by pitch.
func (p *carProjection) carFacing(heading int16, offsetX int, pitch int16, overhead bool) (angle int, mirrored bool) {
	bearing := int(p.bearings[max(0, min(640, offsetX+320))])
	thresholds := p.angles[:]
	if overhead {
		inclination := max(0, min(16383, -int(pitch)))
		bearing = (bearing * (16383 - inclination)) >> 14
		thresholds = p.overheadAngles[:]
	}
	value := (int(uint16(heading)>>6) - bearing) & 1023
	if value >= 512 {
		value, mirrored = 1024-value, true
	}
	angle = len(thresholds) - 1
	for i, threshold := range thresholds {
		if value >= int(threshold) {
			angle = len(thresholds) - 1 - i
			break
		}
	}
	return angle, mirrored
}

// paintCarShape translates the source scanline artwork compositor. A row
// skip counter retains the exact scaling phase separately for each strip.
func (g *Game) paintCarShape(anchorX, anchorY, tier, angle, scale, primary, secondary int, overhead, mirrored bool) {
	var shape *carShape
	if overhead {
		shape = &g.overheadCarShapes[tier][angle]
	} else {
		shape = &g.carShapes[tier][angle]
	}
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
		y := anchorY - 62 + int(scaler.rows[strip.y])
		if y < 0 || y >= worldscene.Height {
			continue
		}
		counter := scaler.skip[strip.y]
		// The source clips the number of input rows before applying skips.
		count := min(len(strip.spans), worldscene.Height-y)
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
			g.worldRenderer.PaintSpan(anchorX+left, anchorX+right, y, color)
			y++
		}
	}
}

type projectedCar struct {
	index    int
	point    worldscene.Projected
	heading  int16
	position uint16
	lateral  int16
	order    int
	drawn    bool
	rear     bool
}

func (g *Game) prepareProjectedCars(camera worldscene.Camera, context worldscene.Context, view int) {
	matrix := g.world.Matrix(camera)
	g.projectedCarCount = 0
	g.carProjectionOverhead = camera.Z > 200
	g.carProjectionPitch = camera.Pitch
	g.carProjection.mirrorsReady = view == inCarView
	if g.carProjection.mirrorsReady {
		for y := range 11 {
			for x := range 80 {
				g.carProjection.mirrorPixels[y*80+x] = 7
				if y < 2 {
					g.carProjection.mirrorPixels[y*80+x] = 11
				}
			}
		}
	}
	for i := range g.cars {
		if view == inCarView && i == 0 {
			continue
		}
		vehicle := g.presentedCar(i)
		if !vehicle.present() {
			continue
		}
		pose := g.carWorldPose(vehicle.position, vehicle.lateral)
		if vehicle.worldPose {
			pose.X, pose.Y, pose.Z = vehicle.worldX, vehicle.worldY, vehicle.worldZ
		}
		point := matrix.TransformDirect(pose.Point, camera, true)
		rear := point.Depth < 0
		if rear {
			if view != inCarView || g.carProjectionOverhead {
				continue
			}
			point.Depth = -point.Depth
			point.Horizontal >>= 1
		}
		depth, horizontal, vertical := int(point.Depth), int(point.Horizontal), int(point.Vertical)
		// The source permits an anchor outside the viewport when the car's
		// silhouette still overlaps it; point outcodes would hide those cars.
		if depth <= 0 || absInt(horizontal) >= depth*2 || absInt(horizontal) >= depth+62 {
			continue
		}
		point.X, point.Y = 160+horizontal*160/depth, 62-vertical*62/depth
		if rear {
			point.X, point.Y = 36+horizontal*36/depth, 2
		}
		if absInt(int(camera.Pitch)) >= 3400 {
			point.Depth *= 2
		}
		if point.Depth >= 4000 || rear && point.Depth >= 800 {
			continue
		}
		distance := int(wrappedDistance(vehicle.position - g.cars[0].position))
		if absInt(distance) > 2000 {
			continue
		}
		order := absInt(distance)
		if order <= 200 {
			order += absInt(int(int16(vehicle.lateral)>>4)-int(context.Lateral)) / 2
		} else {
			order += 32768
		}
		position := (trackLength - int(vehicle.position)) % trackLength
		g.projectedCars[g.projectedCarCount] = projectedCar{index: i, point: point, heading: pose.Heading + vehicle.yaw - camera.Heading, position: uint16(position), lateral: int16(vehicle.lateral), order: order, rear: rear}
		g.projectedCarCount++
	}
	for i := 1; i < g.projectedCarCount; i++ {
		item, j := g.projectedCars[i], i
		for j > 0 && g.projectedCars[j-1].order < item.order {
			g.projectedCars[j] = g.projectedCars[j-1]
			j--
		}
		g.projectedCars[j] = item
	}
}

func (g *Game) drawProjectedCarPass(filter worldscene.CarFilter) {
	for i := 0; i < g.projectedCarCount; i++ {
		item := &g.projectedCars[i]
		selected, suppressRear := filter.SelectCar(item.position, item.lateral)
		if item.drawn || !selected {
			continue
		}
		item.drawn = true
		vehicle := &g.cars[item.index]
		primary, secondary := carColors(vehicle)
		if item.rear {
			if !suppressRear {
				tier, scale := mirrorCarSize(int(item.point.Depth))
				angle, mirrored := g.carProjection.mirrorFacing(item.heading, item.point.X-36)
				g.paintMirrorCarShape(item.point.X, tier, angle, scale, primary, secondary, mirrored)
			}
			continue
		}
		thresholds := g.carProjection.distances[:]
		if g.carProjectionOverhead {
			thresholds = g.carProjection.overheadDistances[:]
		}
		tier, scale := carSize(int(item.point.Depth), thresholds)
		angle, mirrored := g.carProjection.carFacing(item.heading, item.point.X-160, g.carProjectionPitch, g.carProjectionOverhead)
		g.paintCarShape(item.point.X, item.point.Y, tier, angle, scale, primary, secondary, g.carProjectionOverhead, mirrored)
	}
}

func absInt(v int) int {
	if v < 0 {
		return -v
	}
	return v
}
