package game

import "testing"

// The demonstrator's configuration must be attainable through the ordinary
// team, setup and boost controls, without altering the user's chosen setup.
func TestDemonstrationUsesOrdinarySetupValues(t *testing.T) {
	for team := 0; team < 3; team++ {
		manual, err := New(Options{Mute: true, Practice: true})
		if err != nil {
			t.Fatal(err)
		}
		manual.selectTeam(team)
		manual.informationPage = wingsInformationPage
		manual.informationSelection = 0
		manual.adjustInformationValue(-4)
		manual.informationSelection = 1
		manual.adjustInformationValue(-4)
		manual.informationPage = gearsInformationPage
		manual.informationSelection = 0
		delta := -4
		if team == 0 {
			delta = -16
		}
		manual.adjustInformationValue(delta)
		manual.boost = 9
		expected := manual.physicalConfig()
		demo, err := New(Options{Mute: true, Practice: true})
		if err != nil {
			t.Fatal(err)
		}
		demo.selectTeam(team)
		before, boost, front, rear := demo.setup, demo.boost, demo.frontRollBar, demo.rearRollBar
		demo.startDrive(true)
		if actual := demo.physicalConfig(); actual != expected {
			t.Fatalf("team%d demonstration configuration differs from ordinary setup controls", team)
		}
		if demo.setup != before || demo.boost != boost || demo.frontRollBar != front || demo.rearRollBar != rear {
			t.Fatal("demonstration replaced the user's car setup")
		}
	}
}
