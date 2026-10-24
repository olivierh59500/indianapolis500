package scene

import "fmt"

func (r *Renderer) quadStrip(reference int) error {
	header, ok := r.world.word(reference)
	if !ok {
		return fmt.Errorf("invalid quad strip %x", reference)
	}
	base, ok := r.world.word(reference + 2)
	if !ok {
		return fmt.Errorf("invalid quad-strip pointer %x", reference)
	}
	var refs [4]uint16
	for i := 0; i < int(header>>8)+1; i++ {
		for j := range 4 {
			value, ok := r.world.word(int(base) + i*4 + j*2)
			if !ok {
				return fmt.Errorf("incomplete quad strip %x", reference)
			}
			refs[j] = value
		}
		if err := r.drawReferences(refs[:], uint8(header&15), true); err != nil {
			return err
		}
	}
	return nil
}

func (r *Renderer) sectionLines(operation Operation) error {
	var fields [6]uint16
	for i := range fields {
		value, ok := r.world.word(operation.Reference + i*2)
		if !ok {
			return fmt.Errorf("incomplete section-line record %x", operation.Reference)
		}
		fields[i] = value
	}
	base := operation.Reference + 12
	start := base + (operation.Section-int(int16(fields[0])))*int(int16(fields[3]))
	end := start + int(fields[2])
	if operation.Reverse {
		start -= int(fields[2])
	}
	start = max(start, base)
	end = min(end, int(fields[1]))
	if end <= start {
		return nil
	}
	if fields[4] == 0 {
		return fmt.Errorf("zero section-line stride at %x", operation.Reference)
	}
	count := (end - start) / int(fields[4])
	var refs [64]uint16
	cursor := start
	for i := 0; i < count; i++ {
		length, ok := r.world.word(cursor)
		if !ok {
			return fmt.Errorf("invalid section-line strip %x", cursor)
		}
		vertices := int(length) + 2
		cursor += 2
		if vertices > len(refs) {
			return fmt.Errorf("oversized section-line strip %x: operation=%+v fields=%x range=%x..%x base=%x", cursor, operation, fields, start, end, base)
		}
		for j := 0; j < vertices; j++ {
			value, ok := r.world.word(cursor)
			if !ok {
				return fmt.Errorf("incomplete section-line vertices %x", cursor)
			}
			refs[j] = value
			cursor += 2
		}
		if err := r.drawReferences(refs[:vertices], uint8(fields[5]&15), false); err != nil {
			return err
		}
	}
	return nil
}
