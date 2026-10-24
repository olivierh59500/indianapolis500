package scene

import "fmt"

// Context selects the original track section and detail-dependent drawing
// lists. Lateral is in world units, rather than the physics field's Q4 units.
type Context struct {
	Progress     uint16
	Lateral      int16
	TrackHeading int16
	Detail       int
	Mode         uint8
}

const referenceOrigin = 0x14e6c

func (w *World) tableWord(sourceAddress int) (uint16, error) {
	value, ok := w.word(sourceAddress - referenceOrigin)
	if !ok {
		return 0, fmt.Errorf("invalid world table record %x", sourceAddress-referenceOrigin)
	}
	return value, nil
}

// Operations appends the same visible objects, in the same order, as the
// original normal driving path. The caller owns and can reuse the output.
func (w *World) Operations(output []Operation, camera Camera, context Context) ([]Operation, error) {
	start := len(output)
	result, err := w.selectOperations(output, camera, context)
	reverse := !(uint16(camera.Heading-context.TrackHeading+0x2000) < 0x4000 && camera.Z <= 200)
	for i := start; i < len(result); i++ {
		result[i].Section = int(context.Progress) / 0x14a
		result[i].Reverse = reverse
		result[i].Mode = context.Mode
	}
	return result, err
}

func (w *World) selectOperations(output []Operation, camera Camera, context Context) ([]Operation, error) {
	startIndex := len(output)
	sector := int(context.Progress) / 0x14a
	bucket := int(context.Progress) / 0x7bc
	if bucket < 4 && context.Lateral < 300 {
		bucket -= 5
	}
	bucketOffset := bucket * 2
	detail := max(0, min(2, context.Detail))
	firstRange, secondRange := uint16(40), uint16(40)
	if detail != 2 {
		index := detail + bucketOffset
		if index < -10 || index > 39 {
			return output, fmt.Errorf("invalid original detail section %d", index)
		}
		firstRange, secondRange = uint16(w.lod[10+index]), uint16(w.lod[60+index])
	}
	pitch, roll := int(int16(camera.Pitch+31)>>6), int(int16(camera.Roll+31)>>6)
	if abs(pitch) <= 30 && abs(roll) <= 27 {
		output = append(output, Operation{Kind: "horizon"})
	} else {
		for _, address := range [...]int{0x17f8a, 0x17f66} {
			primitive, err := w.tableWord(address)
			if err != nil {
				return output, err
			}
			output = append(output, Operation{Kind: "object", Reference: address - referenceOrigin + 2, Primitive: primitive})
		}
	}
	forward := uint16(camera.Heading-context.TrackHeading+0x2000) < 0x4000 && camera.Z <= 200
	far, err := w.tableWord(0x166a2 + bucketOffset)
	if err != nil {
		return output, err
	}
	if !forward {
		output, err = w.appendFirstLOD(output, far, firstRange)
		if err != nil {
			return output, err
		}
		ref, err := w.tableWord(0x1663e + bucketOffset)
		if err != nil {
			return output, err
		}
		output, err = w.appendLOD(output, ref, secondRange, false, false)
		if err != nil {
			return output, err
		}
		output, err = w.appendLOD(output, far, firstRange, true, true)
		if err != nil {
			return output, err
		}
		output = append(output, Operation{Kind: "segments", Reference: 0x3142 + sector*4, Color: 0, Count: 11})
		firstTable, secondTable := 0x15fca, 0x15ffc
		if camera.Z > 200 {
			firstTable, secondTable = 0x1634c, 0x1637e
		}
		ref, err = w.tableWord(firstTable + bucketOffset)
		if err != nil {
			return output, err
		}
		output, err = w.appendList(output, ref)
		if err != nil {
			return output, err
		}
		ref, err = w.tableWord(0x16670 + bucketOffset)
		if err != nil {
			return output, err
		}
		output, err = w.appendLOD(output, ref, secondRange, false, false)
		if err != nil {
			return output, err
		}
		ref, err = w.tableWord(0x166d4 + bucketOffset)
		if err != nil {
			return output, err
		}
		output, err = w.appendLOD(output, ref, firstRange, false, false)
		if err != nil {
			return output, err
		}
		ref, err = w.tableWord(secondTable + bucketOffset)
		if err != nil {
			return output, err
		}
		output, err = w.appendList(output, ref)
		if err != nil {
			return output, err
		}
		for i := startIndex; i < len(output); i++ {
			output[i].Reverse = true
			output[i].Section = sector
		}
		return output, nil
	}
	output, err = w.appendLOD(output, far, firstRange, false, false)
	if err != nil {
		return output, err
	}
	output = append(output, Operation{Kind: "segments", Reference: 0x315a + sector*4, Color: 0, Count: 5})
	nearTable := 0x158d4
	if sector*2 < 60 && context.Lateral < 300 {
		nearTable = 0x15898
	}
	near, err := w.tableWord(nearTable + sector*2)
	if err != nil {
		return output, err
	}
	output, err = w.appendList(output, near)
	if err != nil {
		return output, err
	}
	far, err = w.tableWord(0x166d4 + bucketOffset)
	if err != nil {
		return output, err
	}
	output, err = w.appendLOD(output, far, firstRange, false, false)
	if err != nil {
		return output, err
	}
	local := sector * 2
	if local >= 232 {
		local -= 240
	}
	if local <= 58 {
		table := 0x15a10
		if context.Lateral < 300 {
			table = 0x159cc
		}
		ref, err := w.tableWord(table + local)
		if err != nil {
			return output, err
		}
		output, err = w.appendList(output, ref)
		if err != nil {
			return output, err
		}
	}
	for i := startIndex; i < len(output); i++ {
		output[i].Section = sector
	}
	return output, nil
}

func (w *World) appendFirstLOD(output []Operation, reference uint16, limit uint16) ([]Operation, error) {
	if reference == 0 {
		return output, nil
	}
	distance, ok := w.word(int(reference) + 2)
	if !ok {
		return output, fmt.Errorf("invalid first detail entry %x", reference)
	}
	if distance >= limit {
		return output, nil
	}
	list, ok := w.word(int(reference) + 4)
	if !ok {
		return output, fmt.Errorf("invalid first object-list entry %x", reference)
	}
	return w.appendList(output, list)
}

func (w *World) appendList(output []Operation, reference uint16) ([]Operation, error) {
	if reference == 0 {
		return output, nil
	}
	count, ok := w.word(int(reference))
	if !ok {
		return output, fmt.Errorf("invalid object list %x", reference)
	}
	if count > 1024 {
		return output, fmt.Errorf("invalid object-list count %d at %x", count, reference)
	}
	for i := 0; i < int(count); i++ {
		object, ok := w.word(int(reference) + 2 + i*2)
		if !ok {
			return output, fmt.Errorf("incomplete object list %x", reference)
		}
		primitive, ok := w.word(int(object))
		if !ok {
			return output, fmt.Errorf("invalid object %x", object)
		}
		output = append(output, Operation{Kind: "object", Reference: int(object) + 2, Primitive: primitive})
	}
	return output, nil
}

func (w *World) appendLOD(output []Operation, reference uint16, limit uint16, skipFirst, absolute bool) ([]Operation, error) {
	if reference == 0 {
		return output, nil
	}
	count, ok := w.word(int(reference))
	if !ok {
		return output, fmt.Errorf("invalid detail list %x", reference)
	}
	if int16(count) < 0 {
		return output, nil
	}
	first := 0
	if skipFirst {
		first = 1
	}
	for i := first; i <= int(count); i++ {
		cursor := int(reference) + 2 + i*4
		distance, ok := w.word(cursor)
		if !ok {
			return output, fmt.Errorf("incomplete detail list %x", reference)
		}
		if absolute && int16(distance) < 0 {
			distance = uint16(-int16(distance))
		}
		if distance > limit {
			continue
		}
		list, ok := w.word(cursor + 2)
		if !ok {
			return output, fmt.Errorf("incomplete detail entry %x", cursor)
		}
		var err error
		output, err = w.appendList(output, list)
		if err != nil {
			return output, err
		}
	}
	return output, nil
}
