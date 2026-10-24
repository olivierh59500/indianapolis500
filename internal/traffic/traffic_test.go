package traffic

import (
	"encoding/json"
	"os"
	"testing"
)

type racingFrame struct {
	Tick, Cursor          int
	Tail, TailAfter       int
	Pose                  PlayerPose
	CursorAfter           int
	Random, RandomAfter   uint16
	Before, After         [33]Car
	Ranking, RankingAfter originalRanking
}

// This couples all opponent movement, track ordering and round-robin AI.
// The measured physical pose supplies player preparation before shared motion.
func TestOriginalNormalRacingField(t *testing.T) {
	m, frames := originalRacing(t)
	for _, frame := range frames {
		m.State = State{Cars: frame.Before, Random: frame.Random, NextDriver: frame.Cursor, Tail: frame.Tail, Rules: 2}
		frame.Ranking.supply(m)
		pit, spin := m.StepRacingField(frame.Pose, nil)
		frame.RankingAfter.compare(t, m)
		if pit != 0 || spin != 0 {
			t.Fatalf("original tick%d had no pit or spin transition", frame.Tick)
		}
		for i := range m.Cars {
			if m.Cars[i] != frame.After[i] {
				t.Fatalf("original tick%d car%d full field: native%+v, original%+v", frame.Tick, i, m.Cars[i], frame.After[i])
			}
		}
		if m.Random != frame.RandomAfter || m.NextDriver != frame.CursorAfter {
			t.Fatalf("original tick%d random/cadence: native%04x/%d, original%04x/%d", frame.Tick, m.Random, m.NextDriver, frame.RandomAfter, frame.CursorAfter)
		}
	}
}

func TestOriginalRollingGridInitialization(t *testing.T) {
	m, _ := originalRacing(t)
	data, err := os.ReadFile("testdata/factory.json")
	if err != nil {
		t.Fatal(err)
	}
	var frames []struct {
		Seed, RandomAfter     uint16
		Position              int
		Before, After         [33]Car
		Ranking, RankingAfter originalRanking
	}
	if err = json.Unmarshal(data, &frames); err != nil {
		t.Fatal(err)
	}
	parameters, err := os.ReadFile("../../assets/original/traffic-driver-coefficients.bin")
	if err != nil {
		t.Fatal(err)
	}
	for _, frame := range frames {
		m.Cars, m.Random = frame.Before, frame.Seed
		frame.Ranking.supply(m)
		if err = m.InitializeRace(parameters, frame.Position, 10); err != nil {
			t.Fatal(err)
		}
		frame.RankingAfter.compare(t, m)
		for i := range m.Cars {
			if m.Cars[i] != frame.After[i] {
				t.Fatalf("original rolling grid seed%04x position%d car%d: native%+v, original%+v", frame.Seed, frame.Position, i, m.Cars[i], frame.After[i])
			}
		}
		if m.Random != frame.RandomAfter || m.NextDriver != 1 {
			t.Fatal("the original rolling grid changed the random sequence or initial driver cadence")
		}
	}
}

func originalRacing(t *testing.T) (*Model, []racingFrame) {
	t.Helper()
	read := func(name string) []byte {
		b, e := os.ReadFile("../../assets/original/" + name)
		if e != nil {
			t.Fatal(e)
		}
		return b
	}
	m, e := New(read("track-lines.bin"), read("traffic-curve-flags.bin"), read("traffic-yaw-curve.bin"), read("traffic-drag-curve.bin"))
	if e != nil {
		t.Fatal(e)
	}
	if e = m.LoadDriverCurves(read("traffic-spin-strength.bin"), read("world-trigonometry.bin"), read("traffic-retirement-reasons.bin")); e != nil {
		t.Fatal(e)
	}
	if e = m.LoadRoadLimits(read("track-wall-inner.bin"), read("track-wall-outer.bin")); e != nil {
		t.Fatal(e)
	}
	b, e := os.ReadFile("testdata/racing.json")
	if e != nil {
		t.Fatal(e)
	}
	var frames []racingFrame
	if e = json.Unmarshal(b, &frames); e != nil {
		t.Fatal(e)
	}
	return m, frames
}

func TestOriginalPitCautionSpinAndStoppedCallbacks(t *testing.T) {
	m, _ := originalRacing(t)
	data, err := os.ReadFile("testdata/driver-modes.json")
	if err != nil {
		t.Fatal(err)
	}
	var frames []struct {
		Kind                 string
		Before, Ahead, After Car
		Random, RandomAfter  uint16
	}
	if err = json.Unmarshal(data, &frames); err != nil {
		t.Fatal(err)
	}
	for _, frame := range frames {
		t.Run(frame.Kind, func(t *testing.T) {
			m.Cars[5], m.Cars[6], m.Random, m.Rules = frame.Before, frame.Ahead, frame.Random, 2
			m.UpdateDriver(5)
			if m.Cars[5] != frame.After {
				t.Fatalf("native%+v, original%+v", m.Cars[5], frame.After)
			}
			if m.Random != frame.RandomAfter {
				t.Fatalf("random native%04x, original%04x", m.Random, frame.RandomAfter)
			}
		})
	}
}

// Each pair was observed around a real traffic IRQ during a normally started
// ten-lap Amiga race. The native movement is checked for all 32 opponents,
// including fraction borrowing, lateral inertia and finish-line wraparound.
func TestOriginalComputerCarMovement(t *testing.T) {
	_, frames := originalRacing(t)
	for _, frame := range frames {
		for i := 1; i < 33; i++ {
			c, want := frame.Before[i], frame.After[i]
			c.Advance()
			if c.Distance != want.Distance || c.Fraction != want.Fraction || c.Lateral != want.Lateral || c.Yaw != want.Yaw || c.Laps != want.Laps {
				t.Fatalf("original tick%d car%d movement: native%+v, original%+v", frame.Tick, i, c, want)
			}
		}
	}
}

// Race order is supplied from the observed post-movement context here, so this
// test isolates the four round-robin driver decisions and their steering law.
// Sorting, pit lifecycle and physical car contacts require separate checks.
func TestOriginalRacingDecisions(t *testing.T) {
	m, frames := originalRacing(t)
	for _, frame := range frames {
		m.Cars, m.Random = frame.After, frame.Random
		for i := frame.Cursor; i < frame.Cursor+4; i++ {
			c := frame.Before[i]
			c.Advance()
			c.Previous, c.Next = frame.After[i].Previous, frame.After[i].Next
			c.PitLaps = frame.After[i].PitLaps
			c.RemainingLaps, c.Flags = frame.After[i].RemainingLaps, frame.After[i].Flags
			m.Cars[i] = c
		}
		for i := frame.Cursor; i < frame.Cursor+4; i++ {
			before := m.State
			plan := m.PlanRacing(i, m.Cars[i].PreferredLane)
			if m.State != before {
				t.Fatal("previewing a racing decision changed live cars or random state")
			}
			original := frame.After[i]
			if plan.Lateral != original.TargetLateral || plan.Lane != original.Lane || plan.Acceleration != original.Acceleration {
				t.Fatalf("original tick%d driver%d plan%+v differs from the source target", frame.Tick, i, plan)
			}
			if m.UpdateRacing(i) {
				t.Fatalf("unexpected spin in original tick%d car%d", frame.Tick, i)
			}
			if m.Cars[i] != frame.After[i] {
				t.Fatalf("original tick%d car%d decision: native%+v, original%+v", frame.Tick, i, m.Cars[i], frame.After[i])
			}
		}
		if m.Random != frame.RandomAfter {
			t.Fatalf("original tick%d random sequence: native%04x, original%04x", frame.Tick, m.Random, frame.RandomAfter)
		}
	}
}
