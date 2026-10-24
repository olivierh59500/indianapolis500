package game

import (
	"github.com/hajimehoshi/ebiten/v2"
	"github.com/olivierh59500/indianapolis500/internal/vehicle"
)

type notification uint8

const (
	noNotification notification = iota
	cruiseOffNotification
	cruiseOnNotification
	soundOnNotification
	soundOffNotification
	soundLowNotification
	winnerNotification
	fuelNotification
	engineNotification
	tireNotification
	escapeNotification
	collisionNotification
	pausedNotification
)

var notificationLabels = [...]string{
	"          ", "CRUISE OFF", "CRUISE ON ", "SOUND ON  ", "SOUND OFF ", "SOUND LOW ",
	"YOU WON!  ", "NO FUEL   ", "ENG BLEW  ", "TIRE BLEW ", "PRESS ESC ", "COLLISION ", "PAUSED    ",
}

// notificationRuntime keeps the original timed message priorities. Short
// control messages interrupt long alerts, whose remaining time is retained.
type notificationRuntime struct {
	Timers    [12]uint16
	Active    notification
	LastClock uint16
	Clears    uint16
	Visible   notification
}

func (n *notificationRuntime) reset() {
	visible := n.Visible
	*n = notificationRuntime{Clears: 2, Visible: visible}
}

func (n *notificationRuntime) request(message notification) {
	for i := 1; i <= 5; i++ {
		n.Timers[i] = 0
	}
	duration := uint16(300)
	if message >= winnerNotification {
		duration = 600
	}
	if message == escapeNotification {
		duration = 3600
	}
	n.Timers[message] = duration
}

// advance uses the original sixty-hertz notification clock, including its
// word wrap, strict-negative expiry and independent pause/resume handling.
func (n *notificationRuntime) advance(now uint16, flags uint8) uint8 {
	if n.Active != noNotification {
		elapsed := int16(now - n.LastClock)
		if elapsed < 0 {
			elapsed = -elapsed
		}
		remaining := int16(n.Timers[n.Active]) - elapsed
		n.Timers[n.Active] = uint16(remaining)
		if remaining < 0 {
			n.Timers[n.Active], n.Active, n.Clears = 0, noNotification, 2
		}
	}
	for message := cruiseOffNotification; message < winnerNotification; message++ {
		if n.Timers[message] != 0 {
			n.Active, n.Clears = message, 0
			n.paint(now)
			return flags
		}
	}
	if flags&9 != 0 {
		resume := flags&8 != 0
		flags &^= 8
		n.Active = noNotification
		if !resume {
			n.paint(now)
			return flags
		}
		n.Clears = 2
	}
	if n.Active == noNotification {
		for message := winnerNotification; message <= collisionNotification; message++ {
			if n.Timers[message] != 0 {
				n.Active, n.Clears = message, 0
				break
			}
		}
		if n.Active == noNotification {
			if n.Clears != 0 {
				n.Clears--
				n.Visible = noNotification
			}
			return flags
		}
	}
	n.paint(now)
	return flags
}

func (n *notificationRuntime) paint(now uint16) {
	n.Visible = n.Active
	if n.Active == noNotification && now&32 != 0 {
		n.Visible = pausedNotification
	}
	n.LastClock = now
}

func (g *Game) updateNotifications() {
	if g.scene != driveScene || g.demo || g.replay {
		return
	}
	flags := uint8(0)
	if g.paused != g.noticeWasPaused {
		g.notifications.request(noNotification)
		if !g.paused {
			flags |= 8
		}
		g.noticeWasPaused = g.paused
	}
	if g.paused {
		flags |= 1
	}
	g.notifications.advance(g.noticeClock, flags)
}

func (g *Game) updateHeaderTimers() {
	if g.scene != driveScene || g.demo || g.replay {
		return
	}
	g.flags.phase++
	if g.flags.phase == PresentationTicksPerSecond/SimulationTicksPerSecond {
		g.flags.phase = 0
		g.trafficModel.SignalFlags = g.flags.step(g.trafficModel.SignalFlags)
	}
	g.updateNotifications()
}

func (g *Game) drawNotification(screen *ebiten.Image) {
	if message := g.notifications.Visible; message != noNotification {
		g.drawText(screen, notificationLabels[message], 24, 8, 15)
	}
}

func (g *Game) reportVehicleNotifications(events vehicle.Events, wasInPit bool) {
	if events.TireFailures != 0 {
		g.notifications.request(tireNotification)
	}
	if events.Engine.FuelExhausted {
		g.notifications.request(fuelNotification)
	}
	if events.Engine.EngineFailure {
		g.notifications.request(engineNotification)
	}
	if failed := events.StoppedFailures; failed != 0 && (failed&31 != 0 || !wasInPit) {
		g.notifications.request(escapeNotification)
	}
}

func (g *Game) reportCollisionNotification(severity uint16, damageEnabled bool) {
	if damageEnabled && severity >= 250 {
		g.notifications.request(collisionNotification)
	}
}

func (g *Game) reportCruiseNotification(active bool) {
	if active != g.noticeCruise {
		message := cruiseOffNotification
		if active {
			message = cruiseOnNotification
		}
		g.notifications.request(message)
		g.noticeCruise = active
	}
}
