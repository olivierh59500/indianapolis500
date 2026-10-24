package game

import "github.com/olivierh59500/indianapolis500/internal/session"

func (g *Game) advanceClock(previousProgress, progress uint16) session.ClockEvent {
	g.clock.SignalFlags = g.trafficModel.SignalFlags
	event := g.clock.Step(int(g.mode), previousProgress, progress)
	state := &g.cars[0].physics
	g.noticeInPit = g.clock.AtPit(int(g.mode), progress, state.Lateral, state.Body.Speed)
	g.trafficModel.SignalFlags = g.clock.SignalFlags
	g.cars[0].lap = int(g.clock.Laps)
	if event.Qualified && !g.demo {
		g.qualifyingPosition = event.Position
		g.qualifyingSpeed = g.clock.QualifyingSpeed
		g.saveFiles()
	}
	return event
}

func clockText(value [5]byte) string {
	return string([]byte{value[0], ':', value[1], value[2], '.', value[3], value[4]})
}
