package game

import "testing"

// A repeated demo must preserve finite fuel within each match, begin a new
// event after its actual finish, and restore any interrupted manual session.
func TestRepeatedDemonstrationPreservesManualSession(t *testing.T) {
	for team := 0; team < 3; team++ {
		g, err := New(Options{Mute: true, Practice: true})
		if err != nil {
			t.Fatal(err)
		}
		g.selectTeam(team)
		originalCars, originalClock, originalTraffic := g.cars, g.clock, g.trafficModel.State
		originalConfig, originalSetup := g.physicalConfig(), g.setup
		originalBoost, originalFront, originalRear := g.boost, g.frontRollBar, g.rearRollBar
		g.startDrive(true)
		restarts := 0
		previousFuel := g.cars[0].physics.Engine.Fuel
		for frame := 0; frame < 100000 && restarts < 2; frame++ {
			previous := g.raceTicks
			if err := g.Update(); err != nil {
				t.Fatal(err)
			}
			if g.cars[0].physics.Engine.Failures != 0 {
				t.Fatalf("team%d demo failed with flags%x before a legitimate new race", team, g.cars[0].physics.Engine.Failures)
			}
			if g.raceTicks < previous {
				restarts++
			} else if g.cars[0].physics.Engine.Fuel > previousFuel {
				t.Fatal("demonstration fuel increased before a real new event")
			}
			previousFuel = g.cars[0].physics.Engine.Fuel
		}
		if restarts != 2 {
			t.Fatalf("team%d completed%d ordinary demo cycles, want2", team, restarts)
		}
		g.stopDemo()
		if g.cars != originalCars || g.clock != originalClock || g.trafficModel.State != originalTraffic || g.physicalConfig() != originalConfig || g.setup != originalSetup || g.boost != originalBoost || g.frontRollBar != originalFront || g.rearRollBar != originalRear {
			t.Fatalf("team%d repeated demo changed the saved manual session", team)
		}
	}
}
