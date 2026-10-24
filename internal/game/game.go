// Package game implements the native Indianapolis 500 simulation.
package game

import (
	"encoding/binary"
	"fmt"
	"image"
	"image/color"
	"image/png"
	"math"
	"os"
	"strings"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/inpututil"
	"github.com/hajimehoshi/ebiten/v2/vector"
	gameassets "github.com/olivierh59500/indianapolis500/assets"
	"github.com/olivierh59500/indianapolis500/internal/debris"
	"github.com/olivierh59500/indianapolis500/internal/pilot"
	worldscene "github.com/olivierh59500/indianapolis500/internal/scene"
	"github.com/olivierh59500/indianapolis500/internal/session"
	"github.com/olivierh59500/indianapolis500/internal/sound"
	"github.com/olivierh59500/indianapolis500/internal/traffic"
	"github.com/olivierh59500/indianapolis500/internal/vehicle"
)

const (
	SimulationTicksPerSecond   = 15
	PresentationTicksPerSecond = 60
	TicksPerSecond             = SimulationTicksPerSecond
	ScreenWidth                = 320
	ScreenHeight               = 200
	TouchWidth                 = 480
	TouchGameX                 = 80
	TouchRightX                = TouchGameX + ScreenWidth
)

type scene uint8

const (
	titleScene scene = iota
	menuScene
	driveScene
)

type menuPage uint8

const (
	mainMenuPage menuPage = iota
	raceMenuPage
	teamMenuPage
	optionsMenuPage
	fileMenuPage
)

type driveMode uint8

const (
	practiceMode driveMode = iota
	qualifyMode
	raceMode
)

const (
	trackLength      = 0x9ab0
	replayFrameCount = 20 * SimulationTicksPerSecond
)

const (
	carShapeTiers          = 21
	carShapeAngles         = 16
	overheadCarShapeTiers  = 11
	overheadCarShapeAngles = 32
)

type carSpan struct {
	left  uint8
	right uint8
}

type carShapeStrip struct {
	color uint8
	y     uint8
	spans []carSpan
}

type carShape struct {
	strips []carShapeStrip
	maxY   int
}

type car struct {
	position                                 float64
	progress                                 float64
	lateral                                  float64
	lateralVelocity                          float64
	speed                                    float64
	steering                                 float64
	lap                                      int
	gear                                     int
	color                                    int
	secondary                                int
	number                                   int
	player                                   bool
	active                                   bool
	yaw, angularVelocity                     int16
	damageFlags                              byte
	worldX, worldY, worldZ, worldVX, worldVY int16
	worldPose                                bool
	physical                                 bool
	physics                                  vehicle.State
	driver                                   vehicle.DriverState
	dashboardSpeed                           int16
	automaticLateral                         float64
	automaticReverse                         int8
	automaticBraking                         bool
}

type replayFrame struct {
	effects     debris.Pool
	cars        [33]car
	contactJolt vehicle.CameraJolt
}

type driveSnapshot struct {
	menuDrivingPaused    bool
	effects              debris.Pool
	cars                 [33]car
	traffic              traffic.State
	soundController      sound.Controller
	soundSources         sound.Sources
	soundPrevious        [33]sound.MovingSource
	contactJolt          vehicle.CameraJolt
	instruments          instrumentRuntime
	flags                flagRuntime
	notifications        notificationRuntime
	noticeClock          uint16
	noticeWasPaused      bool
	noticeCruise         bool
	noticeInPit          bool
	clock                session.Clock
	mode                 driveMode
	raceLaps             int
	raceTicks            int
	fuelGallons          float64
	informationPage      informationPage
	informationSelection int
	pitBoard             bool
	pitSelection         int
	tireWear             [4]float64
	tireTemperature      [4]float64
	engineWear           float64
	damage               float64
	failure              string
}

var menuItems = [...]string{
	"RESUME DRIVING",
	"INSTANT REPLAY",
	"PRACTICE",
	"QUALIFY",
	"RACE",
	"TEAMS",
	"FILE",
	"OPTIONS",
}

const (
	tvView = iota
	skyView
	trackView
	behindView
	inCarView
	crashView
)

var demoViews = [...]string{"TV VIEW", "SKY VIEW", "TRACK VIEW", "BEHIND VIEW", "IN CAR VIEW", "CRASH"}

var raceItems = [...]string{"10 LAPS, 25 MI", "30 LAPS, 75 MI", "60 LAPS, 150 MI", "200 LAPS, 500 MI"}

var teamItems = [...]string{"MARCH/COSWORTH", "LOLA/BUICK", "PENSKE/CHEVROLET"}

var fileItems = [...]string{
	"LOAD SETUP 1", "SAVE SETUP 1",
	"LOAD SETUP 2", "SAVE SETUP 2",
	"LOAD SETUP 3", "SAVE SETUP 3",
	"LOAD REPLAY", "SAVE REPLAY",
}

// Options selects host presentation.
type Options struct {
	Touch    bool
	Demo     bool
	Practice bool
	Race     bool
	Mute     bool
	Smoke    int
	Capture  string
	SavePath string
}

// Game owns the shared desktop and Android state.
type Game struct {
	effects               debris.Model
	replayReturnEffects   debris.Pool
	clock                 session.Clock
	options               Options
	savePath              string
	title                 *ebiten.Image
	panel                 *ebiten.Image
	menuLogo              *ebiten.Image
	pitDisplay            pitDisplay
	setupArtwork          setupArtwork
	cockpitNose           [3]*ebiten.Image
	cockpitWheels         [13][2]*ebiten.Image
	cockpitOutline        *ebiten.Image
	font                  *ebiten.Image
	palette               color.Palette
	world                 *worldscene.World
	worldRenderer         *worldscene.Renderer
	worldOperations       []worldscene.Operation
	worldImage            *ebiten.Image
	worldPixels           []byte
	worldColors           [16][4]byte
	vehiclePreset         vehicle.Preset
	vehicleCurve          vehicle.ContactCurve
	vehicleSurface        vehicle.TrackSurface
	vehicleModel          vehicle.Vehicle
	contactJolt           vehicle.CameraJolt
	trafficModel          *traffic.Model
	trafficParameters     []byte
	dashboard             dashboard
	informationBar        informationBar
	informationImage      *ebiten.Image
	informationPixels     [ScreenWidth * 24 * 4]byte
	flagImages            [6]*ebiten.Image
	flags                 flagRuntime
	notifications         notificationRuntime
	noticeClock           uint16
	noticeWasPaused       bool
	noticeCruise          bool
	noticeInPit           bool
	dashboardImage        *ebiten.Image
	dashboardPixels       [ScreenWidth * 76 * 4]byte
	touchGame             *ebiten.Image
	viewport              *ebiten.Image
	cockpitView           *ebiten.Image
	gamepadIDs            []ebiten.GamepadID
	trackLines            [5][310]int16
	paintCars             func(worldscene.CarFilter)
	carProjection         carProjection
	projectedCars         [33]projectedCar
	projectedCarCount     int
	carProjectionOverhead bool
	carProjectionPitch    int16
	carShapes             [carShapeTiers][carShapeAngles]carShape
	overheadCarShapes     [overheadCarShapeTiers][overheadCarShapeAngles]carShape
	audio                 *sound.Engine
	soundController       sound.Controller
	soundSources          sound.Sources
	soundPrevious         [33]sound.MovingSource
	scene                 scene
	menuPage              menuPage
	mode                  driveMode
	tick                  int
	presentationTick      int
	simulationPhase       int
	idle                  int
	selected              int
	paused                bool
	menuDrivingPaused     bool
	demo                  bool
	demoPilot             *pilot.Pilot
	replay                bool
	session               bool
	team                  int
	raceLaps              int
	detail                int
	sensitivity           int
	boost                 int
	frontRollBar          int
	rearRollBar           int
	qualifyingPosition    int
	qualifyingSpeed       uint16
	setup                 carSetup
	savedSetups           [3]carSetup
	savedSetupValid       [3]bool
	savedReplay           []replayFrame
	fuelGallons           float64
	informationPage       informationPage
	informationSelection  int
	pitBoard              bool
	pitSelection          int
	tireWear              [4]float64
	tireTemperature       [4]float64
	engineWear            float64
	damage                float64
	failure               string
	demoAnchorKey         int
	demoAnchorPosition    float64
	demoFinishTicks       int
	replayFrames          [replayFrameCount]replayFrame
	replayReturnCars      [33]car
	replayReturnJolt      vehicle.CameraJolt
	replayInstruments     instrumentRuntime
	replayHead            int
	replayCount           int
	replayStart           int
	replayIndex           int
	replayView            int
	demoReturn            driveSnapshot
	demoReturnValid       bool
	cars                  [33]car
	raceTicks             int
	capture               bool
	done                  bool
	err                   error
}

// New loads the ADF-derived title, font and palette.
func New(options Options) (*Game, error) {
	titleFile, err := gameassets.Original.Open("original/title.png")
	if err != nil {
		return nil, err
	}
	titleImage, err := png.Decode(titleFile)
	closeErr := titleFile.Close()
	if err != nil {
		return nil, err
	}
	if closeErr != nil {
		return nil, closeErr
	}
	panelFile, err := gameassets.Original.Open("original/indy-cam.png")
	if err != nil {
		return nil, err
	}
	panelImage, err := png.Decode(panelFile)
	closeErr = panelFile.Close()
	if err != nil {
		return nil, err
	}
	if closeErr != nil {
		return nil, closeErr
	}
	menuLogoFile, err := gameassets.Original.Open("original/menu-logo.png")
	if err != nil {
		return nil, err
	}
	menuLogoImage, err := png.Decode(menuLogoFile)
	closeErr = menuLogoFile.Close()
	if err != nil {
		return nil, err
	}
	if closeErr != nil {
		return nil, closeErr
	}
	fontData, err := gameassets.Original.ReadFile("original/font.bin")
	if err != nil {
		return nil, err
	}
	trackData, err := gameassets.Original.ReadFile("original/track-lines.bin")
	if err != nil {
		return nil, err
	}
	carData, err := gameassets.Original.ReadFile("original/car-shapes.bin")
	if err != nil {
		return nil, err
	}
	engineSample, err := gameassets.Original.ReadFile("original/engine-sample.bin")
	if err != nil {
		return nil, err
	}
	noiseSample, err := gameassets.Original.ReadFile("original/noise-sample.bin")
	if err != nil {
		return nil, err
	}
	collisionSample, err := gameassets.Original.ReadFile("original/collision-sample.bin")
	if err != nil {
		return nil, err
	}
	pitchCurve, err := gameassets.Original.ReadFile("original/engine-speed-pitch.bin")
	if err != nil {
		return nil, err
	}
	paletteData, err := gameassets.Original.ReadFile("original/palette.bin")
	if err != nil {
		return nil, err
	}
	if len(fontData) != 482 || len(paletteData) != 32 || len(trackData) != 5*310*2 || len(engineSample) != 0x56f0 || len(noiseSample) != 4960 || len(collisionSample) != 10000 {
		return nil, fmt.Errorf("recovered Indianapolis 500 assets have invalid lengths")
	}
	palette := make(color.Palette, len(paletteData)/2)
	for index := range palette {
		value := binary.BigEndian.Uint16(paletteData[index*2:])
		palette[index] = color.NRGBA{
			R: uint8(value>>8&15) * 17,
			G: uint8(value>>4&15) * 17,
			B: uint8(value&15) * 17,
			A: 255,
		}
	}
	g := &Game{
		options: options, savePath: options.SavePath, title: ebiten.NewImageFromImage(titleImage), panel: ebiten.NewImageFromImage(panelImage),
		menuLogo: ebiten.NewImageFromImage(menuLogoImage),
		font:     buildFont(fontData), palette: palette,
		touchGame: ebiten.NewImage(ScreenWidth, ScreenHeight), viewport: ebiten.NewImage(320, 124), cockpitView: ebiten.NewImage(ScreenWidth, 124),
		raceLaps: 10, detail: 1, sensitivity: 1, qualifyingPosition: 33, team: 1,
		boost: 7, frontRollBar: 5, rearRollBar: 1,
		setup: defaultCarSetup(),
	}
	if err := g.loadOriginalWorld(); err != nil {
		return nil, err
	}
	if err := g.loadVehicle(); err != nil {
		return nil, err
	}
	if err := g.loadTraffic(); err != nil {
		return nil, err
	}
	if err := g.dashboard.load(); err != nil {
		return nil, err
	}
	if err := g.informationBar.load(); err != nil {
		return nil, err
	}
	g.informationImage = ebiten.NewImage(ScreenWidth, 24)
	g.notifications.reset()
	for index := range g.flagImages {
		art := image.NewPaletted(image.Rect(0, 0, 24, 18), g.palette)
		copy(art.Pix, g.informationBar.flags[index*24*18:(index+1)*24*18])
		g.flagImages[index] = ebiten.NewImageFromImage(art)
	}
	g.dashboardImage = ebiten.NewImage(ScreenWidth, 76)
	g.installDefaultSetupSlots()
	g.loadFiles()
	g.paintCars = g.drawProjectedCarPass
	for line := range g.trackLines {
		for sample := range g.trackLines[line] {
			offset := (line*len(g.trackLines[line]) + sample) * 2
			g.trackLines[line][sample] = int16(binary.BigEndian.Uint16(trackData[offset:]))
		}
	}
	g.carShapes, g.overheadCarShapes, err = parseCarShapes(carData)
	if err != nil {
		return nil, err
	}
	if err := g.loadCarProjection(); err != nil {
		return nil, err
	}
	effectData, loadErr := gameassets.Original.ReadFile("original/debris-templates.json")
	if loadErr != nil {
		return nil, loadErr
	}
	g.effects, err = debris.New(effectData)
	if err != nil {
		return nil, err
	}
	g.trafficModel.OnCollisionEffect = g.emitTrafficCollisionEffects
	g.soundSources, err = sound.NewSources(pitchCurve)
	if err != nil {
		return nil, err
	}
	g.soundController = sound.NewController()
	if err := g.pitDisplay.load(); err != nil {
		return nil, err
	}
	if err := g.setupArtwork.load(); err != nil {
		return nil, err
	}
	if !options.Mute {
		g.audio, err = sound.New(engineSample, noiseSample, collisionSample)
		if err != nil {
			return nil, err
		}
	}
	if options.Demo {
		g.startDrive(true)
	} else if options.Race {
		g.mode = raceMode
		g.raceLaps = 10
		g.startDrive(false)
	} else if options.Practice {
		g.mode = practiceMode
		g.startDrive(false)
	} else {
		g.startDrive(false)
		g.scene, g.session = titleScene, false
	}
	return g, nil
}

func parseCarShapes(data []byte) ([carShapeTiers][carShapeAngles]carShape, [overheadCarShapeTiers][overheadCarShapeAngles]carShape, error) {
	var shapes [carShapeTiers][carShapeAngles]carShape
	var overhead [overheadCarShapeTiers][overheadCarShapeAngles]carShape
	if len(data) < 16 || string(data[:8]) != "INDYCAR2" ||
		binary.BigEndian.Uint16(data[8:]) != carShapeTiers || binary.BigEndian.Uint16(data[10:]) != carShapeAngles ||
		binary.BigEndian.Uint16(data[12:]) != overheadCarShapeTiers || binary.BigEndian.Uint16(data[14:]) != overheadCarShapeAngles {
		return shapes, overhead, fmt.Errorf("recovered Indianapolis 500 car shapes have an invalid header")
	}
	offset := 16
	decode := func() (carShape, error) {
		if offset+2 > len(data) {
			return carShape{}, fmt.Errorf("recovered Indianapolis 500 car shapes are truncated")
		}
		length := int(binary.BigEndian.Uint16(data[offset:]))
		offset += 2
		if length < 1 || offset+length > len(data) {
			return carShape{}, fmt.Errorf("recovered Indianapolis 500 car shape has an invalid length")
		}
		record := data[offset : offset+length]
		offset += length
		cursor := 1
		shape := carShape{maxY: -1}
		for range int(record[0]) {
			if cursor+3 > len(record) {
				return carShape{}, fmt.Errorf("recovered Indianapolis 500 car shape is truncated")
			}
			strip := carShapeStrip{color: record[cursor], y: record[cursor+1]}
			rows := int(record[cursor+2])
			cursor += 3
			if rows == 0 || cursor+rows*2 > len(record) {
				return carShape{}, fmt.Errorf("recovered Indianapolis 500 car strip has an invalid height")
			}
			strip.spans = make([]carSpan, rows)
			for row := range rows {
				strip.spans[row] = carSpan{left: record[cursor], right: record[cursor+1]}
				cursor += 2
			}
			shape.maxY = max(shape.maxY, int(strip.y)+rows-1)
			shape.strips = append(shape.strips, strip)
		}
		if cursor != len(record) {
			return carShape{}, fmt.Errorf("recovered Indianapolis 500 car shape has trailing data")
		}
		return shape, nil
	}
	for tier := range carShapeTiers {
		for angle := range carShapeAngles {
			shape, parseErr := decode()
			if parseErr != nil {
				return shapes, overhead, parseErr
			}
			shapes[tier][angle] = shape
		}
	}
	for tier := range overheadCarShapeTiers {
		for angle := range overheadCarShapeAngles {
			shape, parseErr := decode()
			if parseErr != nil {
				return shapes, overhead, parseErr
			}
			overhead[tier][angle] = shape
		}
	}
	if offset != len(data) {
		return shapes, overhead, fmt.Errorf("recovered Indianapolis 500 car shape archive has trailing data")
	}
	return shapes, overhead, nil
}

func buildFont(data []byte) *ebiten.Image {
	font := ebiten.NewImage(16*8, 4*8)
	for character := 0x20; character <= 0x5a; character++ {
		offset := (character - 0x1f) * 8
		if offset+8 > len(data) {
			continue
		}
		index := character - 0x20
		for y := range 8 {
			for x := range 8 {
				if data[offset+y]&(1<<uint(7-x)) != 0 {
					font.Set(index%16*8+x, index/16*8+y, color.White)
				}
			}
		}
	}
	return font
}

// Update advances one original-style simulation tick.
func (g *Game) Update() error {
	if g.done {
		if g.err != nil {
			return g.err
		}
		return ebiten.Termination
	}
	if g.capture {
		return nil
	}
	g.advanceSound()
	g.noticeClock++
	defer g.updateHeaderTimers()
	defer g.updateSound()
	gamepadID, gamepadAvailable := g.firstStandardGamepad()
	gamepadBack := gamepadAvailable && g.scene == menuScene && inpututil.IsStandardGamepadButtonJustPressed(gamepadID, ebiten.StandardGamepadButtonRightRight)
	if inpututil.IsKeyJustPressed(ebiten.KeyEscape) || gamepadBack {
		g.Back()
		return nil
	}
	gamepadPause := gamepadAvailable && !g.demo && inpututil.IsStandardGamepadButtonJustPressed(gamepadID, ebiten.StandardGamepadButtonCenterRight)
	if (inpututil.IsKeyJustPressed(ebiten.KeyP) || gamepadPause) && g.scene == driveScene {
		g.setPaused(!g.paused)
	}
	if g.scene == driveScene && g.options.Touch && g.touchPausePressed() {
		g.setPaused(!g.paused)
	}
	if g.paused {
		return nil
	}
	g.presentationTick++
	switch g.scene {
	case titleScene:
		if g.confirmPressed() {
			selection := 2
			if g.session {
				selection = 0
			}
			g.scene, g.selected, g.idle = menuScene, selection, 0
			return nil
		}
		g.idle++
		if g.idle >= 15*PresentationTicksPerSecond {
			g.startDrive(true)
		}
	case menuScene:
		g.updateMenu()
	case driveScene:
		if g.replay {
			g.updateReplay()
			break
		}
		if g.demo && g.demoExitPressed() {
			g.stopDemo()
			return nil
		}
		if !g.demo {
			g.updateDrivingCommands()
		}
		g.simulationPhase++
		if g.simulationPhase >= PresentationTicksPerSecond/SimulationTicksPerSecond {
			g.simulationPhase = 0
			g.tick++
			g.updateDrive()
		}
	}
	if g.options.Smoke > 0 && g.tick >= g.options.Smoke {
		g.capture = true
	}
	return nil
}

func (g *Game) updateMenu() {
	count := g.menuItemCount()
	if inpututil.IsMouseButtonJustPressed(ebiten.MouseButtonLeft) {
		x, y := ebiten.CursorPosition()
		if g.options.Touch {
			x -= TouchGameX
		}
		if index, ok := g.menuItemAt(x, y); ok {
			g.selected = index
			g.activateMenuItem()
			g.idle = 0
			return
		}
	}
	move := 0
	if inpututil.IsKeyJustPressed(ebiten.KeyArrowUp) || inpututil.IsKeyJustPressed(ebiten.KeyBackspace) {
		move--
	}
	if inpututil.IsKeyJustPressed(ebiten.KeyArrowDown) || inpututil.IsKeyJustPressed(ebiten.KeySpace) {
		move++
	}
	if id, ok := g.firstStandardGamepad(); ok {
		if inpututil.IsStandardGamepadButtonJustPressed(id, ebiten.StandardGamepadButtonLeftTop) {
			move--
		}
		if inpututil.IsStandardGamepadButtonJustPressed(id, ebiten.StandardGamepadButtonLeftBottom) {
			move++
		}
	}
	if g.options.Touch {
		for _, id := range inpututil.AppendJustPressedTouchIDs(nil) {
			x, y := ebiten.TouchPosition(id)
			if x >= TouchGameX {
				continue
			}
			if y < ScreenHeight/2 {
				move--
			} else {
				move++
			}
		}
	}
	if move != 0 {
		g.selected = (g.selected + move + count) % count
		g.idle = 0
	}
	if g.confirmPressed() {
		g.activateMenuItem()
		g.idle = 0
		return
	}
	g.idle++
	if g.idle >= 15*PresentationTicksPerSecond {
		g.startDrive(true)
	}
}

func (g *Game) menuItemCount() int {
	switch g.menuPage {
	case raceMenuPage:
		return len(raceItems)
	case teamMenuPage:
		return len(teamItems)
	case optionsMenuPage:
		return 2
	case fileMenuPage:
		return len(fileItems)
	default:
		return len(menuItems)
	}
}

func (g *Game) activateMenuItem() {
	switch g.menuPage {
	case raceMenuPage:
		g.raceLaps = [...]int{10, 30, 60, 200}[g.selected]
		g.mode = raceMode
		g.startDrive(false)
	case teamMenuPage:
		g.selectTeam(g.selected)
		g.menuPage, g.selected = mainMenuPage, 0
		g.saveFiles()
	case optionsMenuPage:
		if g.selected == 0 {
			g.sensitivity = (g.sensitivity + 1) % 3
		} else {
			g.detail = (g.detail + 1) % 3
		}
		g.saveFiles()
	case fileMenuPage:
		g.activateFileItem()
	default:
		switch g.selected {
		case 0:
			if g.session {
				g.scene, g.paused = driveScene, g.menuDrivingPaused
				g.menuDrivingPaused = false
				if !g.paused {
					g.resetDrivingSoundControls()
				}
			}
		case 1:
			g.startReplay()
		case 2:
			g.mode = practiceMode
			g.startDrive(false)
		case 3:
			g.mode = qualifyMode
			g.startDrive(false)
		case 4:
			g.menuPage, g.selected = raceMenuPage, 0
		case 5:
			g.menuPage, g.selected = teamMenuPage, g.team
		case 6:
			g.menuPage, g.selected = fileMenuPage, 0
		case 7:
			g.menuPage, g.selected = optionsMenuPage, 0
		}
	}
}

func (g *Game) confirmPressed() bool {
	if g.scene == titleScene && inpututil.IsMouseButtonJustPressed(ebiten.MouseButtonLeft) {
		return true
	}
	if inpututil.IsKeyJustPressed(ebiten.KeyEnter) || inpututil.IsKeyJustPressed(ebiten.KeyZ) {
		return true
	}
	if id, ok := g.firstStandardGamepad(); ok && inpututil.IsStandardGamepadButtonJustPressed(id, ebiten.StandardGamepadButtonRightBottom) {
		return true
	}
	for _, id := range inpututil.AppendJustPressedTouchIDs(nil) {
		x, _ := ebiten.TouchPosition(id)
		if g.scene == titleScene || !g.options.Touch || x >= TouchRightX {
			return true
		}
	}
	return false
}

func (g *Game) touchPausePressed() bool {
	for _, id := range inpututil.AppendJustPressedTouchIDs(nil) {
		x, y := ebiten.TouchPosition(id)
		if x < TouchGameX && y < 44 {
			return true
		}
	}
	return false
}

func (g *Game) demoExitPressed() bool {
	if inpututil.IsKeyJustPressed(ebiten.KeyEnter) || inpututil.IsKeyJustPressed(ebiten.KeySpace) ||
		inpututil.IsKeyJustPressed(ebiten.KeyZ) || inpututil.IsMouseButtonJustPressed(ebiten.MouseButtonLeft) || len(inpututil.AppendJustPressedTouchIDs(nil)) > 0 {
		return true
	}
	id, ok := g.firstStandardGamepad()
	return ok && (inpututil.IsStandardGamepadButtonJustPressed(id, ebiten.StandardGamepadButtonRightBottom) ||
		inpututil.IsStandardGamepadButtonJustPressed(id, ebiten.StandardGamepadButtonCenterRight))
}

func (g *Game) firstStandardGamepad() (ebiten.GamepadID, bool) {
	g.gamepadIDs = ebiten.AppendGamepadIDs(g.gamepadIDs[:0])
	for _, id := range g.gamepadIDs {
		if ebiten.IsStandardGamepadLayoutAvailable(id) {
			return id, true
		}
	}
	return 0, false
}

func (g *Game) updateDrivingCommands() {
	boostKeys := [...]ebiten.Key{
		ebiten.KeyDigit1, ebiten.KeyDigit2, ebiten.KeyDigit3,
		ebiten.KeyDigit4, ebiten.KeyDigit5, ebiten.KeyDigit6,
		ebiten.KeyDigit7, ebiten.KeyDigit8, ebiten.KeyDigit9,
	}
	for index, key := range boostKeys {
		if inpututil.IsKeyJustPressed(key) {
			g.boost = index + 1
		}
	}
	if inpututil.IsKeyJustPressed(ebiten.KeyEqual) {
		g.frontRollBar = min(8, g.frontRollBar+1)
	}
	if inpututil.IsKeyJustPressed(ebiten.KeyMinus) {
		g.frontRollBar = max(0, g.frontRollBar-1)
	}
	if inpututil.IsKeyJustPressed(ebiten.KeyBracketRight) {
		g.rearRollBar = min(8, g.rearRollBar+1)
	}
	if inpututil.IsKeyJustPressed(ebiten.KeyBracketLeft) {
		g.rearRollBar = max(0, g.rearRollBar-1)
	}
	controlPressed := ebiten.IsKeyPressed(ebiten.KeyControl) || ebiten.IsKeyPressed(ebiten.KeyControlLeft) || ebiten.IsKeyPressed(ebiten.KeyControlRight)
	if controlPressed && inpututil.IsKeyJustPressed(ebiten.KeyR) {
		g.startDrive(false)
	}
	g.updateInformationControls()
}

func (g *Game) startDrive(demo bool) {
	previousPhysics := g.vehiclePreset.State
	if g.cars[0].physical {
		previousPhysics = g.cars[0].physics
	}
	if demo && g.session && !g.demo && !g.replay {
		g.demoReturn = driveSnapshot{
			menuDrivingPaused: g.menuDrivingPaused,
			cars:              g.cars, mode: g.mode, raceLaps: g.raceLaps, effects: g.effects.Pool,
			traffic:         g.trafficModel.State,
			soundController: g.soundController, soundSources: g.soundSources, soundPrevious: g.soundPrevious,
			contactJolt:   g.contactJolt,
			instruments:   g.dashboard.instrumentRuntime,
			flags:         g.flags,
			notifications: g.notifications, noticeClock: g.noticeClock,
			noticeWasPaused: g.noticeWasPaused, noticeCruise: g.noticeCruise, noticeInPit: g.noticeInPit,
			clock: g.clock, raceTicks: g.raceTicks,
			fuelGallons: g.fuelGallons, informationPage: g.informationPage, informationSelection: g.informationSelection,
			pitBoard: g.pitBoard, pitSelection: g.pitSelection,
			tireWear: g.tireWear, tireTemperature: g.tireTemperature,
			engineWear: g.engineWear, damage: g.damage, failure: g.failure,
		}
		g.demoReturnValid = true
	} else if !demo {
		g.demoReturnValid = false
		g.menuDrivingPaused = false
	}
	g.scene, g.demo, g.paused, g.idle, g.tick = driveScene, demo, false, 0, 0
	g.presentationTick, g.simulationPhase = 0, 0
	g.demoAnchorKey = -1
	g.demoPilot = nil
	g.demoFinishTicks = 0
	g.replay = false
	if !demo {
		g.session = true
		g.replayHead, g.replayCount = 0, 0
		g.fuelGallons = float64(g.setup.fuel)
		g.informationPage, g.informationSelection = lapInformationPage, 0
		g.pitBoard, g.pitSelection = g.mode == practiceMode, 0
	} else {
		g.fuelGallons = 40
		g.pitBoard = false
	}
	g.raceTicks = 0
	g.tireWear, g.engineWear, g.damage, g.failure = [4]float64{}, 0, 0, ""
	g.tireTemperature = [4]float64{70, 70, 70, 70}
	g.contactJolt = vehicle.CameraJolt{}
	g.dashboard.instrumentRuntime = newInstrumentRuntime()
	g.dashboard.validPixels = false
	g.informationBar.valid = false
	g.flags = flagRuntime{Hidden: true}
	g.notifications.reset()
	g.noticeWasPaused, g.noticeCruise, g.noticeInPit = false, false, false
	if !demo && g.mode != raceMode {
		model := g.trafficModel
		model.SetYellow(false)
		model.WreckCount, model.YellowDelay, model.SignalFlags = 0, 0, 1
		model.TowInterval, model.TowWait = 0, 0
	}
	if demo {
		g.mode = raceMode
		g.raceLaps = 10
	}
	g.cars = [33]car{}
	g.initializeTraffic()
	g.soundController.Reset()
	g.soundSources.Reset()
	g.effects.Reset()
	primary, secondary := vehicle.PlayerCarColors(g.team)
	g.cars[0].color, g.cars[0].secondary = int(primary), int(secondary)
	g.initializePlayerVehicle(previousPhysics)
	g.rememberSoundPositions()
	g.clock = session.Initialize(g.clock, int(g.mode), g.trafficModel.Cars[0].Distance)
	g.trafficModel.SignalFlags = g.clock.SignalFlags
	g.cars[0].lap = int(g.clock.Laps)
}

func (g *Game) stopDemo() {
	if g.demoReturnValid {
		snapshot := &g.demoReturn
		g.menuDrivingPaused = snapshot.menuDrivingPaused
		g.cars, g.mode, g.raceLaps = snapshot.cars, snapshot.mode, snapshot.raceLaps
		g.trafficModel.State = snapshot.traffic
		g.effects.Pool = snapshot.effects
		g.soundController, g.soundSources = snapshot.soundController, snapshot.soundSources
		g.soundPrevious = snapshot.soundPrevious
		g.contactJolt = snapshot.contactJolt
		g.dashboard.instrumentRuntime = snapshot.instruments
		g.dashboard.validPixels = false
		g.informationBar.valid = false
		g.flags = snapshot.flags
		g.notifications, g.noticeClock = snapshot.notifications, snapshot.noticeClock
		g.noticeWasPaused, g.noticeCruise, g.noticeInPit = snapshot.noticeWasPaused, snapshot.noticeCruise, snapshot.noticeInPit
		g.clock, g.raceTicks = snapshot.clock, snapshot.raceTicks
		g.fuelGallons = snapshot.fuelGallons
		g.informationPage, g.informationSelection = snapshot.informationPage, snapshot.informationSelection
		g.pitBoard, g.pitSelection = snapshot.pitBoard, snapshot.pitSelection
		g.tireWear, g.tireTemperature = snapshot.tireWear, snapshot.tireTemperature
		g.engineWear, g.damage, g.failure = snapshot.engineWear, snapshot.damage, snapshot.failure
	}
	g.demoReturnValid = false
	selection := 2
	if g.session {
		selection = 0
	}
	g.scene, g.menuPage, g.selected, g.demo, g.paused, g.idle = menuScene, mainMenuPage, selection, false, false, 0
}

type drivingInput struct {
	throttle, brake, reverse bool
	steering                 float64
	cruiseIncrease           bool
}

func (g *Game) drivingControls() drivingInput {
	throttle, brake, steer := false, false, 0.0
	if g.demo {
		return g.automaticPlayerControls()
	} else {
		throttle = ebiten.IsKeyPressed(ebiten.KeyArrowUp) || ebiten.IsKeyPressed(ebiten.KeyW) || ebiten.IsKeyPressed(ebiten.KeyN) ||
			ebiten.IsKeyPressed(ebiten.KeyNumpad8)
		brake = ebiten.IsKeyPressed(ebiten.KeyArrowDown) || ebiten.IsKeyPressed(ebiten.KeyS) || ebiten.IsKeyPressed(ebiten.KeyB) ||
			ebiten.IsKeyPressed(ebiten.KeyNumpad5)
		if ebiten.IsKeyPressed(ebiten.KeyArrowLeft) || ebiten.IsKeyPressed(ebiten.KeyA) || ebiten.IsKeyPressed(ebiten.KeyNumpad4) {
			steer--
		}
		if ebiten.IsKeyPressed(ebiten.KeyArrowRight) || ebiten.IsKeyPressed(ebiten.KeyD) || ebiten.IsKeyPressed(ebiten.KeyNumpad6) {
			steer++
		}
		if id, ok := g.firstStandardGamepad(); ok {
			axis := ebiten.StandardGamepadAxisValue(id, ebiten.StandardGamepadAxisLeftStickHorizontal)
			if math.Abs(axis) > 0.18 {
				steer = axis
			}
			if ebiten.IsStandardGamepadButtonPressed(id, ebiten.StandardGamepadButtonLeftLeft) {
				steer = -1
			}
			if ebiten.IsStandardGamepadButtonPressed(id, ebiten.StandardGamepadButtonLeftRight) {
				steer = 1
			}
			throttle = throttle || ebiten.IsStandardGamepadButtonPressed(id, ebiten.StandardGamepadButtonRightBottom) ||
				ebiten.IsStandardGamepadButtonPressed(id, ebiten.StandardGamepadButtonFrontBottomRight)
			brake = brake || ebiten.IsStandardGamepadButtonPressed(id, ebiten.StandardGamepadButtonRightRight) ||
				ebiten.IsStandardGamepadButtonPressed(id, ebiten.StandardGamepadButtonFrontBottomLeft)
		}
		if g.options.Touch {
			for _, id := range ebiten.AppendTouchIDs(nil) {
				x, y := ebiten.TouchPosition(id)
				if g.pitBoard || g.touchSetupVisible() {
					if x >= TouchRightX && y >= 22 && y < 90 {
						throttle = true
					}
					continue
				}
				switch {
				case x < TouchGameX && y >= 80:
					steer = float64(x-TouchGameX/2) / (TouchGameX / 2)
				case x >= TouchRightX && y < ScreenHeight/2:
					throttle = true
				case x >= TouchRightX:
					brake = true
				}
			}
		}
	}
	return drivingInput{throttle: throttle, brake: brake, reverse: ebiten.IsKeyPressed(ebiten.KeyNumpad2), steering: steer, cruiseIncrease: ebiten.IsKeyPressed(ebiten.KeyC)}
}

func (g *Game) updateDrive() {
	g.stepDrive(g.drivingControls())
}

// stepDrive advances the shared simulation from ordinary player controls.
func (g *Game) stepDrive(input drivingInput) {
	player := &g.cars[0]
	g.contactJolt = vehicle.CameraJolt{}
	if g.demo {
		key := demoCameraSequenceKey(g.tick)
		if key != g.demoAnchorKey {
			g.demoAnchorKey = key
			g.demoAnchorPosition = player.position
		}
	}
	g.raceTicks++
	camera, _ := g.originalCamera(g.soundView())
	g.effects.Update(camera, &g.soundController.Requests)
	previousProgress := player.physics.Progress
	g.stepPlayerVehicle(input)
	g.advanceClock(previousProgress, player.physics.Progress)
	g.updateTraffic()
	g.updateSoundSources()

	if !g.demo {
		g.recordReplayFrame()
	} else if g.trafficModel.Cars[0].Flags&128 != 0 {
		// A completed demonstration starts a real new session. Keep the saved
		// manual game intact rather than granting fuel to the finished race.
		g.demoFinishTicks++
		if g.demoFinishTicks >= 4*SimulationTicksPerSecond {
			g.startDrive(true)
		}
	}
}

func (g *Game) recordReplayFrame() {
	g.replayFrames[g.replayHead].cars = g.cars
	g.replayFrames[g.replayHead].contactJolt = g.contactJolt
	g.replayFrames[g.replayHead].effects = g.effects.Pool
	g.replayHead = (g.replayHead + 1) % len(g.replayFrames)
	g.replayCount = min(g.replayCount+1, len(g.replayFrames))
}

func (g *Game) startReplay() {
	if g.replayCount < 2 {
		return
	}
	g.replayReturnCars = g.cars
	g.replayReturnJolt = g.contactJolt
	g.replayReturnEffects = g.effects.Pool
	g.replayInstruments = g.dashboard.instrumentRuntime
	g.replayStart = (g.replayHead - g.replayCount + len(g.replayFrames)) % len(g.replayFrames)
	g.replayIndex, g.replayView = 0, inCarView
	g.replay, g.demo, g.paused = true, false, false
	g.scene = driveScene
	g.presentationTick, g.simulationPhase = 0, 0
	g.cars = g.replayFrames[g.replayStart].cars
	g.rememberSoundPositions()
	g.loadReplayFrame()
	g.demoAnchorKey = -1
	g.demoAnchorPosition = g.cars[0].position
}

func (g *Game) loadReplayFrame() {
	frame := (g.replayStart + g.replayIndex) % len(g.replayFrames)
	g.cars = g.replayFrames[frame].cars
	g.contactJolt = g.replayFrames[frame].contactJolt
	g.effects.Pool = g.replayFrames[frame].effects
	g.updateSoundSources()
}

func (g *Game) finishReplay() {
	g.cars = g.replayReturnCars
	g.contactJolt = g.replayReturnJolt
	g.effects.Pool = g.replayReturnEffects
	g.dashboard.instrumentRuntime = g.replayInstruments
	g.dashboard.validPixels = false
	g.replay, g.paused = false, false
	g.scene, g.menuPage, g.selected, g.idle = menuScene, mainMenuPage, 1, 0
}

func (g *Game) updateReplay() {
	move := 0
	if inpututil.IsKeyJustPressed(ebiten.KeyArrowLeft) {
		move--
	}
	if inpututil.IsKeyJustPressed(ebiten.KeyArrowRight) {
		move++
	}
	if id, ok := g.firstStandardGamepad(); ok {
		if inpututil.IsStandardGamepadButtonJustPressed(id, ebiten.StandardGamepadButtonLeftLeft) {
			move--
		}
		if inpututil.IsStandardGamepadButtonJustPressed(id, ebiten.StandardGamepadButtonLeftRight) {
			move++
		}
	}
	if g.options.Touch {
		for _, id := range inpututil.AppendJustPressedTouchIDs(nil) {
			x, y := ebiten.TouchPosition(id)
			switch {
			case x < TouchGameX && y >= 44 && x < TouchGameX/2:
				move--
			case x < TouchGameX && y >= 44:
				move++
			case x >= TouchRightX:
				g.finishReplay()
				return
			}
		}
	}
	if move != 0 {
		g.replayView = (g.replayView + move + len(demoViews)) % len(demoViews)
		g.demoAnchorPosition = g.cars[0].position
	}
	if inpututil.IsKeyJustPressed(ebiten.KeyEnter) || inpututil.IsKeyJustPressed(ebiten.KeySpace) || inpututil.IsKeyJustPressed(ebiten.KeyZ) {
		g.finishReplay()
		return
	}
	if id, ok := g.firstStandardGamepad(); ok && inpututil.IsStandardGamepadButtonJustPressed(id, ebiten.StandardGamepadButtonRightBottom) {
		g.finishReplay()
		return
	}
	g.simulationPhase++
	if g.simulationPhase < PresentationTicksPerSecond/SimulationTicksPerSecond {
		return
	}
	g.simulationPhase = 0
	g.replayIndex++
	if g.replayIndex >= g.replayCount {
		g.finishReplay()
		return
	}
	g.loadReplayFrame()
}

func gearForSpeed(speed float64) int {
	switch {
	case speed < 45:
		return 1
	case speed < 95:
		return 2
	case speed < 155:
		return 3
	default:
		return 4
	}
}

func wrappedDistance(distance float64) float64 {
	if distance > trackLength/2 {
		distance -= trackLength
	} else if distance < -trackLength/2 {
		distance += trackLength
	}
	return distance
}

func (g *Game) trackCurvature(position float64) float64 {
	progress := originalTrackProgress(position)
	turns := [...][2]float64{{0x26ac, 0x3624}, {0x3de0, 0x4d58}, {0x7404, 0x837c}, {0x8b38, trackLength}}
	for _, turn := range turns {
		if progress < turn[0] || progress >= turn[1] {
			continue
		}
		strength := min((progress-turn[0])/500, (turn[1]-progress)/500, 1)
		return strength * 0.72
	}
	return 0
}

// Draw renders the original 320 by 200 image and optional touch controls.
func (g *Game) Draw(screen *ebiten.Image) {
	target := screen
	if g.options.Touch {
		target = g.touchGame
	}
	switch g.scene {
	case titleScene:
		target.DrawImage(g.title, nil)
	case menuScene:
		g.drawMenu(target)
	case driveScene:
		g.drawDrive(target)
	}
	if g.options.Touch {
		screen.Fill(color.Black)
		op := &ebiten.DrawImageOptions{}
		op.GeoM.Translate(TouchGameX, 0)
		screen.DrawImage(target, op)
		g.drawTouch(screen)
	}
	if g.capture {
		g.capture = false
		if g.options.Capture != "" {
			bounds := screen.Bounds()
			pixels := image.NewRGBA(bounds)
			screen.ReadPixels(pixels.Pix)
			file, err := os.Create(g.options.Capture)
			if err == nil {
				err = png.Encode(file, pixels)
				if closeErr := file.Close(); err == nil {
					err = closeErr
				}
			}
			g.err = err
		}
		g.done = true
	}
}

func (g *Game) drawMenu(screen *ebiten.Image) {
	g.drawDrive(screen)
	vector.DrawFilledRect(screen, 24, 30, 264, 140, g.palette[7], false)
	vector.DrawFilledRect(screen, 26, 32, 260, 136, g.palette[15], false)
	vector.DrawFilledRect(screen, 28, 34, 256, 132, g.palette[0], false)
	switch g.menuPage {
	case mainMenuPage:
		g.drawText(screen, "INDIANAPOLIS 500", 96, 40, 14)
		op := &ebiten.DrawImageOptions{}
		op.GeoM.Translate(116, 80)
		screen.DrawImage(g.menuLogo, op)
		rows := [...]int{56, 64, 88, 96, 104, 128, 136, 144}
		for index, item := range menuItems {
			paletteIndex := 7
			if index < 2 {
				paletteIndex = 8
			}
			if index == 0 && g.session {
				paletteIndex = 7
			}
			if index == 1 && g.replayCount > 1 {
				paletteIndex = 7
			}
			g.drawMenuChoiceAt(screen, item, 40, rows[index], index, paletteIndex)
		}
	case raceMenuPage:
		g.drawText(screen, "CHOOSE A RACE", 13*8, 7*8, 14)
		rows := [...]int{9, 13, 16, 18}
		for index, item := range raceItems {
			g.drawMenuChoice(screen, item, 11, rows[index], index, 7)
		}
		g.drawText(screen, "NO YELLOW FLAGS", 14*8, 10*8, 7)
		g.drawText(screen, "NO CAR DAMAGE", 14*8, 11*8, 7)
		g.drawText(screen, "NO CAR DAMAGE", 14*8, 14*8, 7)
	case teamMenuPage:
		g.drawText(screen, "CHOOSE A TEAM", 13*8, 7*8, 14)
		rows := [...]int{10, 12, 14}
		for index, item := range teamItems {
			g.drawMenuChoice(screen, item, 12, rows[index], index, 7)
		}
	case fileMenuPage:
		g.drawText(screen, "FILES", 17*8, 5*8, 14)
		rows := [...]int{8, 9, 11, 12, 14, 15, 17, 18}
		for index, item := range fileItems {
			paletteIndex := 7
			if index == 6 && len(g.savedReplay) < 2 || index == 7 && g.replayCount < 2 {
				paletteIndex = 8
			}
			g.drawMenuChoice(screen, item, 11, rows[index], index, paletteIndex)
		}
	case optionsMenuPage:
		g.drawText(screen, "SET OPTIONS", 14*8, 7*8, 14)
		g.drawText(screen, "MOUSE SENSITIVITY", 11*8, 9*8, 7)
		g.drawText(screen, "GRAPHICS DETAIL", 11*8, 14*8, 7)
		levels := [...]string{"LOW", "MEDIUM", "HIGH"}
		for index, level := range levels {
			paletteIndex := 7
			if index == g.sensitivity {
				paletteIndex = 14
				g.drawText(screen, ">", 13*8, (10+index)*8, 14)
			}
			g.drawText(screen, level, 14*8, (10+index)*8, paletteIndex)
			paletteIndex = 7
			if index == g.detail {
				paletteIndex = 14
				g.drawText(screen, ">", 13*8, (15+index)*8, 14)
			}
			g.drawText(screen, level, 14*8, (15+index)*8, paletteIndex)
		}
		selectedRow := 9
		if g.selected == 1 {
			selectedRow = 14
		}
		g.drawText(screen, ">", 9*8, selectedRow*8, 14)
	}
}

func (g *Game) drawMenuChoice(screen *ebiten.Image, text string, column, row, index, paletteIndex int) {
	if index == g.selected {
		paletteIndex = 15
	}
	g.drawText(screen, text, column*8, row*8, paletteIndex)
}

func (g *Game) drawMenuChoiceAt(screen *ebiten.Image, text string, x, y, index, paletteIndex int) {
	if index == g.selected {
		paletteIndex = 15
	}
	g.drawText(screen, text, x, y, paletteIndex)
}

func (g *Game) drawDrive(screen *ebiten.Image) {
	if g.replay {
		g.drawReplay(screen)
		return
	}
	if g.demo {
		g.drawDemo(screen)
		return
	}
	screen.Fill(g.palette[0])
	g.drawOriginalWorld(g.cockpitView, inCarView)
	op := &ebiten.DrawImageOptions{}
	op.GeoM.Translate(0, 24)
	screen.DrawImage(g.cockpitView, op)
	if g.pitBoard {
		g.drawPitBoard(screen)
	} else {
		g.drawInformation(screen)
	}
	g.drawCockpit(screen)
	g.drawNotification(screen)
}

func (g *Game) drawReplay(screen *ebiten.Image) {
	screen.Fill(g.palette[0])
	g.drawText(screen, "CAMERA", 24, 0, 15)
	g.drawText(screen, "VIEW:", 88, 0, 14)
	name := demoViews[g.replayView]
	g.drawText(screen, name, 136, 0, 15)
	g.drawText(screen, "REPLAY", 24, 16, 12)
	g.drawOriginalWorld(g.viewport, g.replayView)
	op := &ebiten.DrawImageOptions{}
	op.GeoM.Translate(0, 24)
	screen.DrawImage(g.viewport, op)
	if g.replayView == inCarView {
		g.drawCockpit(screen)
		return
	}
	op = &ebiten.DrawImageOptions{}
	op.GeoM.Translate(0, 148)
	screen.DrawImage(g.panel, op)
}

func (g *Game) drawCockpit(screen *ebiten.Image) {
	g.drawCockpitFront(screen)
	g.updateInstruments()
	g.copyRearViewMirrors()
	clear(g.dashboardPixels[:])
	for i, index := range g.dashboard.pixels[124*ScreenWidth:] {
		if index != 0 || i >= 24*ScreenWidth {
			c := g.worldColors[index]
			copy(g.dashboardPixels[i*4:i*4+4], c[:])
		}
	}
	g.dashboardImage.WritePixels(g.dashboardPixels[:])
	op := &ebiten.DrawImageOptions{}
	op.GeoM.Translate(0, 124)
	screen.DrawImage(g.dashboardImage, op)
}

func (g *Game) drawRaceFlags(screen *ebiten.Image) {
	if !g.flags.Hidden {
		op := &ebiten.DrawImageOptions{}
		op.GeoM.Translate(0, 4)
		screen.DrawImage(g.flagImages[g.flags.Index], op)
	}
}

func (g *Game) racePosition() int {
	return int(g.trafficModel.Position)
}

func (g *Game) drawDemo(screen *ebiten.Image) {
	screen.Fill(g.palette[0])
	second := g.presentationTick / PresentationTicksPerSecond % 60
	view := demoViewForSecond(second)
	g.drawText(screen, "DEMO", 24, 0, 15)
	g.drawText(screen, "CLICK TO DRIVE", 104, 0, 14)
	name := demoViews[view]
	indent := [6]int{12, 11, 10, 10, 10, 13}
	g.drawText(screen, name, 32+indent[view]*8, 16, 15)
	g.drawDemoRoadView(g.viewport, view)
	op := &ebiten.DrawImageOptions{}
	op.GeoM.Translate(0, 24)
	screen.DrawImage(g.viewport, op)
	if view == inCarView {
		g.drawCockpit(screen)
		return
	}
	op = &ebiten.DrawImageOptions{}
	op.GeoM.Translate(0, 148)
	screen.DrawImage(g.panel, op)
}

func demoViewForSecond(second int) int {
	switch {
	case second < 10:
		return tvView
	case second < 20:
		return skyView
	case second < 30:
		return trackView
	case second < 40:
		return behindView
	case second < 50:
		return inCarView
	default:
		return crashView
	}
}

func demoCameraSequenceKey(tick int) int {
	second := tick / TicksPerSecond % 60
	view := demoViewForSecond(second)
	localSecond := second % 10
	shot := 0
	if view == tvView || view == crashView {
		shot = localSecond / 2
	} else if view == trackView {
		shot = localSecond
	}
	return view*16 + shot
}

func (g *Game) drawDemoRoadView(screen *ebiten.Image, view int) {
	g.drawOriginalWorld(screen, view)
}

func carColors(vehicle *car) (int, int) {
	return vehicle.color, vehicle.secondary
}

func integerText(value, width int) string {
	text := []byte(strings.Repeat("0", width))
	for index := len(text) - 1; index >= 0 && value > 0; index-- {
		text[index] = byte('0' + value%10)
		value /= 10
	}
	return string(text)
}

func (g *Game) drawText(target *ebiten.Image, text string, x, y, paletteIndex int) {
	if paletteIndex < 0 || paletteIndex >= len(g.palette) {
		return
	}
	r, green, b, a := g.palette[paletteIndex].RGBA()
	for _, character := range strings.ToUpper(text) {
		index := int(character) - 0x20
		if index < 0 || index >= 59 {
			x += 8
			continue
		}
		source := g.font.SubImage(imageRect(index%16*8, index/16*8, 8, 8)).(*ebiten.Image)
		op := &ebiten.DrawImageOptions{}
		op.GeoM.Translate(float64(x), float64(y))
		op.ColorScale.Scale(float32(r)/65535, float32(green)/65535, float32(b)/65535, float32(a)/65535)
		target.DrawImage(source, op)
		x += 8
	}
}

func imageRect(x, y, width, height int) image.Rectangle {
	return image.Rect(x, y, x+width, y+height)
}

func (g *Game) drawTouch(screen *ebiten.Image) {
	panel := color.NRGBA{8, 22, 24, 255}
	accent := color.NRGBA{0, 210, 220, 255}
	vector.DrawFilledRect(screen, 0, 0, TouchGameX, ScreenHeight, panel, false)
	vector.DrawFilledRect(screen, TouchRightX, 0, TouchWidth-TouchRightX, ScreenHeight, panel, false)
	if g.scene == menuScene {
		vector.DrawFilledRect(screen, 8, 16, 64, 72, color.NRGBA{0, 84, 96, 255}, false)
		vector.DrawFilledRect(screen, 8, 112, 64, 72, color.NRGBA{0, 84, 96, 255}, false)
		g.drawText(screen, "UP", 32, 48, 15)
		g.drawText(screen, "DOWN", 24, 144, 15)
		vector.DrawFilledRect(screen, TouchRightX+8, 52, 64, 96, color.NRGBA{0, 132, 0, 255}, false)
		g.drawText(screen, "SELECT", TouchRightX+8, 96, 15)
		return
	}
	if g.scene == titleScene {
		vector.DrawFilledRect(screen, TouchRightX+8, 52, 64, 96, color.NRGBA{0, 132, 0, 255}, false)
		g.drawText(screen, "START", TouchRightX+16, 96, 15)
		return
	}
	vector.DrawFilledRect(screen, 8, 8, 64, 28, color.NRGBA{70, 70, 82, 255}, false)
	g.drawText(screen, "PAUSE", 16, 18, 15)
	if g.replay {
		vector.DrawFilledRect(screen, 8, 52, 28, 112, color.NRGBA{0, 84, 96, 255}, false)
		vector.DrawFilledRect(screen, 44, 52, 28, 112, color.NRGBA{0, 84, 96, 255}, false)
		g.drawText(screen, "<", 18, 104, 15)
		g.drawText(screen, ">", 54, 104, 15)
		vector.DrawFilledRect(screen, TouchRightX+8, 52, 64, 96, color.NRGBA{170, 0, 0, 255}, false)
		g.drawText(screen, "EXIT", TouchRightX+24, 96, 15)
		return
	}
	if g.pitBoard {
		vector.DrawFilledRect(screen, 8, 44, 64, 32, color.NRGBA{0, 84, 96, 255}, false)
		g.drawText(screen, "INFO", 24, 56, 15)
		vector.DrawFilledRect(screen, 8, 88, 64, 88, color.NRGBA{0, 84, 96, 255}, false)
		vector.DrawFilledRect(screen, TouchRightX+8, 22, 64, 68, color.NRGBA{0, 132, 0, 255}, false)
		vector.DrawFilledRect(screen, TouchRightX+8, 110, 64, 68, color.NRGBA{0, 84, 96, 255}, false)
		g.drawText(screen, "NEXT", 24, 128, 15)
		g.drawText(screen, "GAS", TouchRightX+28, 52, 15)
		g.drawText(screen, "CHANGE", TouchRightX+16, 140, 15)
		return
	}
	vector.DrawFilledRect(screen, 8, 44, 64, 32, color.NRGBA{0, 84, 96, 255}, false)
	g.drawText(screen, "INFO", 24, 56, 15)
	if g.touchSetupVisible() {
		vector.DrawFilledRect(screen, 8, 88, 64, 88, color.NRGBA{0, 84, 96, 255}, false)
		g.drawText(screen, "NEXT", 24, 128, 15)
		vector.DrawFilledRect(screen, TouchRightX+8, 22, 64, 68, color.NRGBA{0, 132, 0, 255}, false)
		g.drawText(screen, "GAS", TouchRightX+28, 52, 15)
		shade := color.NRGBA{0, 84, 96, 255}
		if !g.canAdjustInformation() {
			shade = color.NRGBA{55, 55, 55, 255}
		}
		vector.DrawFilledRect(screen, TouchRightX+8, 110, 28, 68, shade, false)
		vector.DrawFilledRect(screen, TouchRightX+44, 110, 28, 68, shade, false)
		g.drawText(screen, "-", TouchRightX+18, 140, 15)
		g.drawText(screen, "+", TouchRightX+54, 140, 15)
		return
	}
	vector.StrokeCircle(screen, TouchGameX/2, 124, 30, 3, accent, false)
	vector.DrawFilledRect(screen, 7, 121, 66, 6, accent, false)
	vector.DrawFilledRect(screen, 37, 91, 6, 66, accent, false)
	g.drawText(screen, "STEER", 20, 164, 15)
	vector.DrawFilledRect(screen, TouchRightX+8, 22, 64, 68, color.NRGBA{0, 132, 0, 255}, false)
	vector.DrawFilledRect(screen, TouchRightX+8, 110, 64, 68, color.NRGBA{170, 0, 0, 255}, false)
	g.drawText(screen, "GAS", TouchRightX+28, 52, 15)
	g.drawText(screen, "BRAKE", TouchRightX+16, 140, 15)
}

// Pause freezes a manual driving session when Android suspends the surface.
func (g *Game) Pause() {
	if g.scene == driveScene && !g.demo {
		g.setPaused(true)
	}
}

// Back applies the same navigation as Escape.
func (g *Game) Back() {
	switch g.scene {
	case driveScene:
		if g.replay {
			g.finishReplay()
			return
		}
		if g.demo {
			g.stopDemo()
			return
		}
		g.menuDrivingPaused = g.paused
		g.resetDrivingSoundControls()
		g.scene, g.menuPage, g.selected, g.demo, g.paused, g.idle = menuScene, mainMenuPage, 0, false, false, 0
	case menuScene:
		if g.menuPage != mainMenuPage {
			g.menuPage, g.selected = mainMenuPage, 0
		} else {
			g.scene, g.idle = titleScene, 0
		}
	}
}

// Layout preserves native pixels and adds side controls on touch devices.
func (g *Game) Layout(_, _ int) (int, int) {
	if g.options.Touch {
		return TouchWidth, ScreenHeight
	}
	return ScreenWidth, ScreenHeight
}
