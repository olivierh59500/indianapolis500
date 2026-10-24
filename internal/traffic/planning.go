package traffic

// DriverPlan is a steering-line and acceleration decision, without motion.
type DriverPlan struct {
	Lateral      int16
	Lane         uint8
	Acceleration int16
}

// PlanRacing previews the original lane-occupancy and braking policy. The
// caller can follow it through ordinary controls without moving any records
// or consuming the live simulation's random sequence.
func (m *Model) PlanRacing(index int, preferred uint8) DriverPlan {
	preview := *m
	preview.Cars[index].PreferredLane = preferred
	lateral := preview.racingTarget(index)
	c := preview.Cars[index]
	return DriverPlan{Lateral: lateral, Lane: c.Lane, Acceleration: c.Acceleration}
}
