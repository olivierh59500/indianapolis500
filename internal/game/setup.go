package game

import (
	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/inpututil"
)

type informationPage uint8

const (
	lapInformationPage informationPage = iota
	standingsInformationPage
	fuelInformationPage
	wingsInformationPage
	rubberInformationPage
	staggerInformationPage
	pressuresInformationPage
	shocksInformationPage
	cambersInformationPage
	gearsInformationPage
)

type carSetup struct {
	fuel      int
	wings     [2]int
	rubber    [4]int
	stagger   int
	pressures [4]int
	shocks    [4]int
	cambers   [4]int
	gears     [4]int
}

func defaultCarSetup() carSetup {
	return carSetup{
		fuel:      40,
		wings:     [2]int{14, 14},
		rubber:    [4]int{1, 0, 1, 0},
		stagger:   1,
		pressures: [4]int{10240, 10240, 10240, 10240},
		shocks:    [4]int{2, 2, 2, 4},
		cambers:   [4]int{-2, 3, 2, 7},
		gears:     [4]int{1116, 716, 489, 480},
	}
}

func (g *Game) updateInformationControls() {
	if g.options.Touch {
		for _, id := range inpututil.AppendJustPressedTouchIDs(nil) {
			x, y := ebiten.TouchPosition(id)
			if x < TouchGameX && y >= 44 && y < 80 {
				page := (g.informationPage + 1) % 10
				if g.pitBoard {
					page = fuelInformationPage
				}
				g.selectInformationPage(page)
				return
			}
			if g.touchSetupVisible() {
				if x < TouchGameX && y >= 88 {
					g.cycleInformationComponent(1)
					return
				}
				if x >= TouchRightX && y >= 110 && y < 178 && g.canAdjustInformation() {
					delta := 1
					if x < TouchRightX+40 {
						delta = -1
					}
					g.adjustInformationValue(delta)
					return
				}
			}
		}
	}
	pageKeys := [...]ebiten.Key{
		ebiten.KeyF1, ebiten.KeyF2, ebiten.KeyF3, ebiten.KeyF4, ebiten.KeyF5,
		ebiten.KeyF6, ebiten.KeyF7, ebiten.KeyF8, ebiten.KeyF9, ebiten.KeyF10,
	}
	for page, key := range pageKeys {
		if inpututil.IsKeyJustPressed(key) {
			g.selectInformationPage(informationPage(page))
		}
	}
	if g.pitBoard {
		if g.options.Touch {
			for _, id := range inpututil.AppendJustPressedTouchIDs(nil) {
				x, y := ebiten.TouchPosition(id)
				switch {
				case x < TouchGameX && y >= 88:
					g.pitSelection = (g.pitSelection + 1) % 5
				case x >= TouchRightX && y >= 110 && y < 178:
					g.servicePitItem(g.pitSelection)
				}
			}
		}
		if inpututil.IsKeyJustPressed(ebiten.KeySpace) {
			g.pitSelection = (g.pitSelection + 1) % 5
		}
		if inpututil.IsKeyJustPressed(ebiten.KeyBackspace) {
			g.pitSelection = (g.pitSelection + 4) % 5
		}
		if inpututil.IsKeyJustPressed(ebiten.KeyEnter) {
			g.servicePitItem(g.pitSelection)
		}
		return
	}
	if g.informationPage < fuelInformationPage {
		if g.informationPage == standingsInformationPage {
			if inpututil.IsKeyJustPressed(ebiten.KeySpace) || inpututil.IsKeyJustPressed(ebiten.KeyTab) || inpututil.IsKeyJustPressed(ebiten.KeyO) {
				g.scrollStandings(1)
			}
			if inpututil.IsKeyJustPressed(ebiten.KeyBackspace) || inpututil.IsKeyJustPressed(ebiten.KeyI) {
				g.scrollStandings(-1)
			}
			if inpututil.IsKeyJustPressed(ebiten.KeyR) {
				g.selectInformationPage(standingsInformationPage)
			}
		}
		return
	}
	if inpututil.IsKeyJustPressed(ebiten.KeySpace) || inpututil.IsKeyJustPressed(ebiten.KeyTab) {
		g.cycleInformationComponent(1)
	}
	if inpututil.IsKeyJustPressed(ebiten.KeyBackspace) {
		g.cycleInformationComponent(-1)
	}
	if !g.canAdjustInformation() {
		return
	}
	if g.informationPage == fuelInformationPage && inpututil.IsKeyJustPressed(ebiten.KeyEnter) {
		g.refuelPlayer()
	}
	delta := 0
	if inpututil.IsKeyJustPressed(ebiten.KeyComma) {
		delta--
	}
	if inpututil.IsKeyJustPressed(ebiten.KeyPeriod) {
		delta++
	}
	if delta != 0 {
		g.adjustInformationValue(delta)
	}
}

func (g *Game) canAdjustInformation() bool {
	return g.mode == practiceMode || g.noticeInPit && (g.informationPage <= pressuresInformationPage || g.mode == qualifyMode)
}

func (g *Game) touchSetupVisible() bool {
	return g.options.Touch && !g.pitBoard && g.informationPage >= fuelInformationPage && g.cars[0].physics.Body.Speed == 0
}

func (g *Game) cycleInformationComponent(direction int) {
	g.informationBar.valid = false
	count := g.informationComponentCount()
	selection := setupClockwiseSelection(g.informationPage, g.informationSelection)
	selection = (selection + direction + count) % count
	switch g.informationPage {
	case rubberInformationPage, pressuresInformationPage, shocksInformationPage, cambersInformationPage:
		g.informationSelection = setupTireRows[selection]
	default:
		g.informationSelection = selection
	}
}

func (g *Game) informationComponentCount() int {
	switch g.informationPage {
	case fuelInformationPage, staggerInformationPage, gearsInformationPage:
		return 1
	case wingsInformationPage:
		return 2
	default:
		return 4
	}
}

func (g *Game) adjustInformationValue(delta int) {
	g.informationBar.valid = false
	selection := g.informationSelection
	previous := g.setup
	switch g.informationPage {
	case fuelInformationPage:
		g.setup.fuel = max(5, min(40, g.setup.fuel+delta*5))
	case wingsInformationPage:
		g.setup.wings[selection] = max(0, min(16, g.setup.wings[selection]+delta))
	case rubberInformationPage:
		g.setup.rubber[selection] = max(0, min(2, g.setup.rubber[selection]+delta))
	case staggerInformationPage:
		g.setup.stagger = max(-10, min(10, g.setup.stagger+delta))
	case pressuresInformationPage:
		g.setup.pressures[selection] = max(10240, min(20480, g.setup.pressures[selection]+delta*512))
	case shocksInformationPage:
		g.setup.shocks[selection] = max(0, min(8, g.setup.shocks[selection]+delta))
	case cambersInformationPage:
		g.setup.cambers[selection] = max(-12, min(12, g.setup.cambers[selection]+delta))
	case gearsInformationPage:
		g.setup.gears = gearSetup(max(280, min(490, g.setup.gears[3]+delta*10)))
	}
	g.applyLiveSetup(previous)
	// Setup callbacks still run when an adjustment reaches its limit.
	if g.cars[0].physical && !g.demo {
		switch g.informationPage {
		case pressuresInformationPage:
			if previous.pressures[selection] == g.setup.pressures[selection] {
				g.applyTirePressure(selection)
			}
		case rubberInformationPage:
			if previous.rubber[selection] == g.setup.rubber[selection] {
				g.serviceTire(informationTireOrder[selection])
			}
		case fuelInformationPage:
			if previous.fuel == g.setup.fuel {
				g.refuelPlayer()
			}
		}
	}
}

func (g *Game) drawInformation(screen *ebiten.Image) {
	switch g.informationPage {
	case standingsInformationPage:
		g.drawStandingsInformation(screen)
	case fuelInformationPage, wingsInformationPage, rubberInformationPage, staggerInformationPage,
		pressuresInformationPage, shocksInformationPage, cambersInformationPage, gearsInformationPage:
		g.drawSetupInformation(screen)
	default:
		g.drawLapInformation(screen)
	}
}

func (g *Game) drawPitBoard(screen *ebiten.Image) {
	g.drawOriginalPit(screen)
}

func (g *Game) drawSetupInformation(screen *ebiten.Image) {
	g.drawOriginalSetup(screen)
}
