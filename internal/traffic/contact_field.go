package traffic

// ResolveContacts follows the original player scan and then the wreck
// stack. Healthy computer drivers avoid each other; they are not all fed to
// a new all-pairs collision pass. Each attempted pair is visited once per tick.
func (m *Model) ResolveContacts(playerContact func(ContactResult)) {
	var pairs [32][2]int
	pairCount := 0
	visited := func(a, b int) bool {
		if a > b {
			a, b = b, a
		}
		for _, pair := range pairs[:pairCount] {
			if pair == [2]int{a, b} {
				return true
			}
		}
		if pairCount == len(pairs) {
			return true
		}
		pairs[pairCount] = [2]int{a, b}
		pairCount++
		return false
	}
	scan := func(index int, reverse bool, notify func(ContactResult)) {
		limit := int16(40)
		if reverse {
			limit = 80
		}
		next := m.Cars[index].Next
		for count := 0; next != index && count < 32; count++ {
			other := next
			next = m.Cars[other].Next
			distance := -gap(m.Cars[index], m.Cars[other])
			if distance < 0 || distance >= limit {
				break
			}
			var result ContactResult
			if reverse {
				// The reverse scan keeps a negative high byte even when an
				// earlier impact has already made the forward speed positive.
				playerStep := int16(m.Cars[index].Speed>>8) - 256
				predicted := distance + int16(m.Cars[other].Speed>>8) - playerStep
				if (distance ^ predicted) < 0 {
					distance = 1
					if predicted < 0 {
						distance = -1
					}
				} else if distance >= 40 {
					break
				}
				if visited(index, other) {
					continue
				}
				result = m.resolveImpact(index, other, distance)
			} else {
				if visited(index, other) {
					continue
				}
				result = m.ResolveContact(index, other, distance)
			}
			if result.Hit && notify != nil {
				notify(result)
			}
		}
		next = m.Cars[index].Previous
		for count := 0; next != index && count < 32; count++ {
			other := next
			next = m.Cars[other].Previous
			separation := gap(m.Cars[index], m.Cars[other])
			distance := -separation
			if separation < 0 || distance <= -limit {
				break
			}
			var result ContactResult
			if reverse {
				playerStep := int16(m.Cars[index].Speed>>8) - 256
				predicted := distance + int16(m.Cars[other].Speed>>8) - playerStep
				if (distance ^ predicted) < 0 {
					distance = 1
					if predicted < 0 {
						distance = -1
					}
				} else if distance <= -40 {
					break
				}
				if visited(index, other) {
					continue
				}
				result = m.resolveImpact(index, other, distance)
			} else {
				if visited(index, other) {
					continue
				}
				result = m.ResolveContact(index, other, distance)
			}
			if result.Hit && notify != nil {
				notify(result)
			}
		}
	}
	reverse := m.PlayerMovement < 0
	if reverse {
		m.Cars[0].Speed = uint16(m.PlayerMovement)
	}
	scan(0, reverse, playerContact)
	if reverse {
		m.Cars[0].Speed = 0
	}
	// Contacts can prepend wrecks. Continue with the current wreck's existing
	// successor, leaving newly prepended entries for the next simulation tick.
	if m.WreckCount == 0 {
		return
	}
	selected := m.Wrecks[0]
	for count := 0; count < 33; count++ {
		scan(selected, false, nil)
		position := 0
		for position < m.WreckCount && m.Wrecks[position] != selected {
			position++
		}
		if position+1 >= m.WreckCount {
			break
		}
		selected = m.Wrecks[position+1]
	}
}
