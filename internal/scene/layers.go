package scene

import "fmt"

// CarFilter describes the source painter's pit-lane partitions. Cars are
// consumed by their first matching pass, between the deferred track surfaces.
type CarFilter uint8

const (
	AllCars CarFilter = iota
	PitBeforeEntrance
	PitCars
	PitAlongStraight
	PitAroundStart
	PitAfterExit
	TrackAroundStart
	BeforeEntrance
	AlongPitStraight
	AfterExit
)

// SelectCar translates the carry-result predicates at $2167e-$217f0.
// Position is the original decreasing lap-distance field, not world progress.
// The second result suppresses the close rear view behind the pit wall.
func (filter CarFilter) SelectCar(position uint16, lateral int16) (bool, bool) {
	pit := lateral < 4800
	around := position < 4110 || position >= 37290
	switch filter {
	case AllCars:
		return true, false
	case PitBeforeEntrance:
		return pit && position < 4110, false
	case PitCars:
		return pit, false
	case PitAlongStraight:
		return pit && position >= 4110 && position < 36980, false
	case PitAroundStart:
		return pit, pit && around
	case PitAfterExit:
		return pit && position > 36980, false
	case TrackAroundStart:
		return !pit, !pit && around
	case BeforeEntrance:
		return position < 4110, false
	case AlongPitStraight:
		return position >= 4110 && position < 36980, false
	case AfterExit:
		return position > 36980, false
	}
	return false, false
}

// SceneOperations adds the original second world pass to the base scenery.
// Road surfaces and pit walls must share the painter order with the cars.
func (w *World) SceneOperations(output []Operation, camera Camera, context Context) ([]Operation, error) {
	start := len(output)
	output, err := w.Operations(output, camera, context)
	if err != nil {
		return output, err
	}
	output, err = w.appendSceneLayers(output, start, camera.Z > 200, context)
	for i := start; i < len(output); i++ {
		output[i].Mode = context.Mode
	}
	return output, err
}

func (w *World) appendSceneLayers(output []Operation, start int, elevated bool, context Context) ([]Operation, error) {
	var deferred [3]Operation
	var present [3]bool
	var lists [32]uint16
	listCount := 0
	for _, operation := range output[start:] {
		if operation.Kind != "object" {
			continue
		}
		switch operation.Primitive {
		case 2:
			slot, valid := w.word(operation.Reference)
			primitive, primitiveValid := w.word(operation.Reference + 2)
			if !valid || !primitiveValid || slot > 4 || slot%2 != 0 {
				return output, fmt.Errorf("invalid deferred scene record %x", operation.Reference)
			}
			deferred[slot/2] = Operation{Kind: "object", Primitive: primitive, Reference: operation.Reference + 4, Reverse: operation.Reverse, Section: operation.Section}
			present[slot/2] = true
		case 18:
			list, valid := w.word(operation.Reference)
			if !valid || listCount == len(lists) {
				return output, fmt.Errorf("invalid deferred scene list %x", operation.Reference)
			}
			lists[listCount] = list
			listCount++
		}
	}
	flush := func(slot int) {
		if present[slot] {
			output = append(output, deferred[slot])
			present[slot] = false
		}
	}
	var deferredError error
	flushLists := func() {
		for _, list := range lists[:listCount] {
			var err error
			output, err = w.appendList(output, list)
			if err != nil {
				deferredError = err
				break
			}
		}
		listCount = 0
	}
	cars := func(filter CarFilter) { output = append(output, Operation{Kind: "cars", Filter: filter}) }
	progress := context.Progress
	if elevated {
		flush(0)
		flushLists()
		flush(1)
		cars(AllCars)
		flush(2)
	} else if progress >= 9900 {
		flushLists()
		flush(2)
		flush(1)
		flush(0)
		cars(AllCars)
	} else if context.Lateral < 300 {
		cars(TrackAroundStart)
		switch {
		case progress < 2820:
			flush(1)
			flush(2)
			flushLists()
			cars(BeforeEntrance)
			flush(0)
			cars(AllCars)
		case progress < 9550:
			cars(AlongPitStraight)
			flush(1)
			if context.Lateral < 230 {
				if progress < 8200 {
					flush(2)
				}
				flushLists()
				flush(2)
				flush(0)
				cars(AllCars)
			} else {
				flush(0)
				cars(AllCars)
				if progress < 8200 {
					flush(2)
				}
				flushLists()
				flush(2)
			}
		default:
			flush(1)
			flushLists()
			flush(2)
			cars(AfterExit)
			flush(0)
			cars(AllCars)
		}
	} else {
		switch {
		case progress < 2820:
			cars(PitBeforeEntrance)
			flush(0)
			cars(PitCars)
			flush(2)
			flushLists()
			flush(1)
		case progress < 9550:
			cars(PitAlongStraight)
			flush(0)
			cars(PitAroundStart)
			if progress < 8200 {
				flush(2)
			}
			flushLists()
			flush(2)
			flush(1)
		default:
			cars(PitAfterExit)
			flush(0)
			flushLists()
			flush(2)
			flush(1)
		}
		cars(AllCars)
	}
	return output, deferredError
}

// PaintSpan is the native indexed scanline compositor used by recovered car
// artwork. Clipping here keeps all actor pixels inside the world viewport.
func (r *Renderer) PaintSpan(left, right, y int, color uint8) {
	if y < 0 || y >= Height {
		return
	}
	if left > right {
		left, right = right, left
	}
	left, right = max(0, left), min(Width-1, right)
	for x := left; x <= right; x++ {
		r.Pixels[y*Width+x] = color & 15
	}
}

// RenderWithCars interleaves ordinary Go actor drawing with scene surfaces.
func (r *Renderer) RenderWithCars(camera Camera, operations []Operation, draw func(CarFilter)) error {
	r.drawCars = draw
	defer func() { r.drawCars = nil }()
	return r.Render(camera, operations)
}
