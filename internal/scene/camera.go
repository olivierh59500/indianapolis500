package scene

import (
	"encoding/binary"
	"fmt"
)

// View follows the original replay camera selector at $28cd8.
type View uint8

const (
	InCar View = iota
	Behind
	Trackside
	Television
	Sky
	Crash
)

type cameraAnchor struct{ until, progress, lateral uint16 }

// LoadCameras loads data tables used by the native replay-camera math.
func (w *World) LoadCameras(angles, gradient, anchors []byte) error {
	if len(angles) != len(w.arctangent)*2 || len(gradient) != len(w.bankGradient)*2 || len(anchors) != len(w.cameraAnchors)*6 {
		return fmt.Errorf("recovered camera tables have invalid dimensions")
	}
	for i := range w.arctangent {
		w.arctangent[i] = int16(binary.BigEndian.Uint16(angles[i*2:]))
	}
	for i := range w.bankGradient {
		w.bankGradient[i] = int16(binary.BigEndian.Uint16(gradient[i*2:]))
	}
	for i := range w.cameraAnchors {
		a := anchors[i*6:]
		w.cameraAnchors[i] = cameraAnchor{binary.BigEndian.Uint16(a), binary.BigEndian.Uint16(a[2:]), binary.BigEndian.Uint16(a[4:])}
	}
	return nil
}

// Direction translates the original lookup-table atan2 at $5edc. Positive
// heading turns toward decreasing world Y, as used by the track coordinates.
func (w *World) Direction(dx, dy int16) int16 {
	x, y := int32(dx), int32(dy)
	origin := int32(-0x4000)
	if x < 0 {
		if y < 0 {
			origin = 0x4000
			x, y = -y, -x
		} else {
			origin = -32768
			x = -x
		}
	} else if y < 0 {
		origin = 0
		y = -y
	} else {
		x, y = y, x
	}
	if x == 0 && y == 0 {
		return 0
	}
	if x < y {
		return int16(origin + 0x4000 - int32(w.arctangent[x*2048/y]))
	}
	return int16(origin + int32(w.arctangent[y*2048/x]))
}

func wrapProgress(p int) uint16 {
	p %= 39600
	if p < 0 {
		p += 39600
	}
	return uint16(p)
}

// BankAngles translates the track's pitch gradient and banking at $4eb4.
// The lateral argument is in world units, after the physics Q4 conversion.
func (w *World) BankAngles(progress uint16, lateral int16) (pitch, roll int16) {
	p := int(progress) % 39600
	t := -1
	switch {
	case p > 9900 && p < 13860:
		t = p - 9900
	case p > 15840 && p < 19800:
		t = p - 15840
	case p > 29700 && p < 33660:
		t = p - 29700
	case p > 35640:
		t = p - 35640
	}
	lat := int(lateral)
	if t < 0 || lat < 208 {
		return 0, 0
	}
	near := abs(t - 1980)
	if near <= 1320 {
		roll = 1664
	} else {
		edge := (1980 - near) >> 3
		factor := int(w.trackBank[(edge&^1)/2])
		roll = int16(factor * 0x28be >> 16)
		if lat >= 252 {
			index := ((lat - 252) >> 6) + (edge &^ 7)
			index = min(index, len(w.bankGradient)-1)
			pitch = w.bankGradient[index]
			if t < 1980 {
				pitch = -pitch
			}
		}
	}
	if lat-208 < 65 {
		roll = int16(int(roll) * (lat - 208) >> 6)
	}
	return pitch, roll
}

// CameraForCar translates the placement and view selection at $28cd8/$28ee4.
// Relative heading is the car's yaw against the track tangent, in full-turn
// units. Dynamic suspension motion remains part of the simulation state.
func (w *World) CameraForCar(progress uint16, lateral, relativeHeading int16, view View) (Camera, Context) {
	cameraProgress, cameraLateral := progress, lateral
	elevated, lookAt, bank := false, false, true
	switch view {
	case Behind:
		cameraProgress = wrapProgress(int(progress) - 70)
		cameraLateral = int16(4800 + ((int(lateral) - 4800) >> 2))
		lookAt = true
	case Sky:
		cameraProgress = wrapProgress(int(progress) - 280)
		cameraLateral = 4800
		elevated, lookAt, bank = true, true, false
	case Trackside, Television, Crash:
		p := uint16(int(progress) % 39600)
		anchor := w.cameraAnchors[len(w.cameraAnchors)-1]
		for _, a := range w.cameraAnchors {
			if p <= a.until {
				anchor = a
				break
			}
		}
		cameraProgress, cameraLateral = anchor.progress, int16(anchor.lateral)
		// The pit straight uses two closer cameras while following a pit-lane car.
		if anchor.progress == 6750 && lateral <= 4800 {
			cameraProgress, cameraLateral = 6750, 4640
			if p <= 6000 {
				cameraProgress = 4000
			}
		}
		lookAt = true
		elevated = view != Trackside
		bank = !elevated
	}
	return w.trackCamera(cameraProgress, cameraLateral, relativeHeading, progress, lateral, lookAt, elevated, bank)
}

func (w *World) trackCamera(progress uint16, lateral, heading int16, targetProgress uint16, targetLateral int16, lookAt, elevated, bank bool) (Camera, Context) {
	point := w.TrackPlacement(progress, lateral)
	camera := Camera{X: point.X, Y: point.Y, Z: point.Z, Heading: point.Heading + heading}
	pitch, roll := int16(0), int16(0)
	if elevated {
		camera.Z += 400
	}
	if lookAt {
		target := w.TrackPlacement(targetProgress, targetLateral)
		dx, dy := target.X-camera.X, target.Y-camera.Y
		camera.Heading = w.Direction(dx, dy)
		if elevated {
			distance := int16(integerSquareRoot(uint32(int32(dx)*int32(dx) + int32(dy)*int32(dy))))
			angle := w.Direction(distance, camera.Z-target.Z)
			// The source shifts by one, then shifts that intermediate again.
			camera.Pitch = (angle >> 1) + ((angle >> 1) >> 1)
		}
	}
	if bank {
		pitch, roll = w.BankAngles(progress, lateral>>4)
		cosine, sine := w.cosSin(camera.Heading - point.Heading)
		camera.Roll = q14(roll, cosine) + q14(pitch, sine)
		camera.Pitch += q14(pitch, cosine) + q14(roll, -sine)
	}
	return camera, Context{Progress: progress, Lateral: lateral >> 4, TrackHeading: point.Heading}
}

func integerSquareRoot(value uint32) uint32 {
	var root uint32
	bit := uint32(1) << 30
	for bit > value {
		bit >>= 2
	}
	for bit != 0 {
		if value >= root+bit {
			value -= root + bit
			root = (root >> 1) + bit
		} else {
			root >>= 1
		}
		bit >>= 2
	}
	return root
}
