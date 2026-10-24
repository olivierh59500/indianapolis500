package game

import (
	"crypto/sha256"
	"fmt"
	"math"
	"path/filepath"
	"testing"

	gameassets "github.com/olivierh59500/indianapolis500/assets"
	worldscene "github.com/olivierh59500/indianapolis500/internal/scene"
	"github.com/olivierh59500/indianapolis500/internal/session"
	"github.com/olivierh59500/indianapolis500/internal/traffic"
	"github.com/olivierh59500/indianapolis500/internal/vehicle"
)

func TestRecoveredStartupAssets(t *testing.T) {
	tests := []struct {
		name   string
		bytes  int
		digest string
	}{
		{"car-shapes.bin", 107422, "b06c92b16209574c8d1b43d62d2d711a4f2aa148d08caa115bfc90f37a16e361"},
		{"cockpit-panel.png", 3260, "10bdd0cec1951f0081e808770be82164b791cd58397014348d008c29c8666d9c"},

		{"noise-sample.bin", 4960, "4b9ec45590dad478a78deedfe716dd31c95e8d3128a0c701955014e4aa4f03f1"},
		{"collision-sample.bin", 10000, "2854c959a9d3be67da6bce865d892f4278d7732aa5d8e7b168efa3e2defa963f"},
		{"engine-speed-pitch.bin", 150, "a58fa7f760d20f44e57f9b9cbb33a23799f0eba48dfa529809844b5baddb71d7"},
		{"engine-sample.bin", 22256, "b0e79c4f275c89a97ce782352256bc3c528b6e6362055c8690fd5f00f33011e0"},
		{"font.bin", 482, "a7901d0b3eb7d1e2e2fd839dde84db23f3edd3e58b55a2eef8af42db3f953910"},
		{"indy-cam.png", 1532, "c298602d764024596f4cb4d19f77f3a8393b0fdb4114f057f352bde84b445bf9"},
		{"menu-logo.png", 1742, "77d165981bfedc9f2287c64bae265cd39d81009ff129125e8d4a69833e31595f"},
		{"palette.bin", 32, "e6a965ed5c6750ede5b13b123de5a49cfe029a896690ec3ab64eb03cbc2a88a1"},
		{"pit-art.bin", 22636, "02c98308e799d539e7f30817fffd9214f56f8360db837567e657a41cddba0061"},
		{"title.png", 12052, "a4b2f4b8734500388637eafdbebb852b574fb49fc1cedafba7dd01a10994ec18"},
		{"track-lines.bin", 3100, "d8c273d27214e034d9ed02eac215f9cf39d87cddaaa991072e7cfdc352d7bac0"},
	}
	for _, test := range tests {
		data, err := gameassets.Original.ReadFile("original/" + test.name)
		if err != nil {
			t.Fatal(err)
		}
		if len(data) != test.bytes || fmt.Sprintf("%x", sha256.Sum256(data)) != test.digest {
			t.Fatalf("%s does not match its recovered source", test.name)
		}
	}
	g, err := New(Options{Mute: true})
	if err != nil {
		t.Fatal(err)
	}
	for _, progress := range []float64{11880, 17820, 31680, 37620} {
		position := math.Mod(progress-0x1b12+trackLength, trackLength)
		if g.trackCurvature(position) <= 0 {
			t.Fatalf("curve center %.0f has no curvature", progress)
		}
	}
}

func TestAutomaticDriverCompletesRaces(t *testing.T) {
	g, err := New(Options{Mute: true})
	if err != nil {
		t.Fatal(err)
	}
	g.startDrive(true)
	finished, restarted := 0, 0
	// Include the original sound/random clock between physical ticks, as the
	// actual desktop and Android Update paths do.
	for frame := range 20000 * (PresentationTicksPerSecond / SimulationTicksPerSecond) {
		wasFinished, previousTicks := g.trafficModel.Cars[0].Flags&128 != 0, g.raceTicks
		if err := g.Update(); err != nil {
			t.Fatal(err)
		}
		if !wasFinished && g.trafficModel.Cars[0].Flags&128 != 0 {
			finished++
		}
		if g.raceTicks < previousTicks {
			restarted++
		}
		if g.raceTicks > 300 && (g.cars[0].speed < 100 || g.cars[0].physics.Engine.Failures != 0) {
			t.Fatalf("automatic driver failed at frame%d: %.1fMPH flags%02x", frame, g.cars[0].speed, g.cars[0].physics.Engine.Failures)
		}
	}
	if finished < 2 || restarted < 2 {
		t.Fatalf("automatic driver completed%d races and restarted%d, want at least2", finished, restarted)
	}
}

func TestIndianapolisTrackHasFourTurns(t *testing.T) {
	g := &Game{}
	turns := 0
	inside := false
	for step := range 1000 {
		position := float64(step) * trackLength / 1000
		turning := g.trackCurvature(position) != 0
		if turning && !inside {
			turns++
		}
		inside = turning
	}
	if turns != 4 {
		t.Fatalf("track has %d turns", turns)
	}
}

func TestRecoveredCarModelsGrowTowardsCamera(t *testing.T) {
	g, err := New(Options{Mute: true})
	if err != nil {
		t.Fatal(err)
	}
	width := func(shape carShape) int {
		left, right := 255, 0
		for _, strip := range shape.strips {
			for _, span := range strip.spans {
				left = min(left, int(min(span.left, span.right)))
				right = max(right, int(max(span.left, span.right)))
			}
		}
		return right - left + 1
	}
	farWidth := width(g.carShapes[0][8])
	nearWidth := width(g.carShapes[carShapeTiers-1][8])
	if farWidth >= nearWidth || farWidth > 8 || nearWidth < 200 {
		t.Fatalf("unexpected recovered car widths: far=%d near=%d", farWidth, nearWidth)
	}
	if len(g.overheadCarShapes[overheadCarShapeTiers-1][0].strips) == 0 {
		t.Fatal("recovered overhead car model is empty")
	}
}

func TestPauseOnlyStopsManualDriving(t *testing.T) {
	g, err := New(Options{Mute: true})
	if err != nil {
		t.Fatal(err)
	}
	g.Pause()
	if g.paused {
		t.Fatal("title screen remained paused after Android resumed")
	}
	g.startDrive(true)
	g.Pause()
	if g.paused {
		t.Fatal("automatic demonstration remained paused after Android resumed")
	}
	g.startDrive(false)
	g.Pause()
	if !g.paused {
		t.Fatal("manual driving session did not pause")
	}
}

func TestPresentationPreservesOriginalSimulationRate(t *testing.T) {
	g, err := New(Options{Mute: true, Demo: true})
	if err != nil {
		t.Fatal(err)
	}
	for range PresentationTicksPerSecond/SimulationTicksPerSecond - 1 {
		if err := g.Update(); err != nil {
			t.Fatal(err)
		}
	}
	if g.tick != 0 {
		t.Fatalf("simulation advanced after %d presentation updates", g.presentationTick)
	}
	if err := g.Update(); err != nil {
		t.Fatal(err)
	}
	if g.tick != 1 || g.presentationTick != PresentationTicksPerSecond/SimulationTicksPerSecond {
		t.Fatalf("simulation tick=%d presentation tick=%d", g.tick, g.presentationTick)
	}
}

func TestAutomaticDriverSelectsAnOpenRacingLine(t *testing.T) {
	g, err := New(Options{Mute: true})
	if err != nil {
		t.Fatal(err)
	}
	g.startDrive(true)
	model := g.trafficModel
	model.Cars[0].Distance, model.Cars[0].Speed, model.Cars[0].Lane = 1000, 15000, 1
	model.Cars[0].Lateral, model.Cars[0].TargetLateral = g.trackLines[1][0], g.trackLines[1][0]
	model.Cars[1].Distance, model.Cars[1].Speed = 880, 0
	model.Cars[1].Lateral, model.Cars[1].TargetLateral = model.Cars[0].Lateral, model.Cars[0].Lateral
	// The source scan sees a complete, ordered two-car ring here.
	model.Cars[0].Next, model.Cars[0].Previous = 1, 1
	model.Cars[1].Next, model.Cars[1].Previous = 0, 0
	if plan := model.PlanRacing(0, 1); plan.Lane == 1 {
		t.Fatal("automatic driver remained behind a slower car on the same line")
	}
}

func TestInstantReplayRestoresTheLiveSession(t *testing.T) {
	g, err := New(Options{Mute: true})
	if err != nil {
		t.Fatal(err)
	}
	g.mode = practiceMode
	g.startDrive(false)
	for frame := 1; frame <= 5; frame++ {
		g.cars[0].position = float64(frame * 100)
		g.recordReplayFrame()
	}
	liveCars := g.cars
	g.startReplay()
	if !g.replay || g.cars[0].position != 100 {
		t.Fatalf("replay opened at position %.1f", g.cars[0].position)
	}
	for range PresentationTicksPerSecond / SimulationTicksPerSecond {
		g.updateReplay()
	}
	if g.replayIndex != 1 || g.cars[0].position != 200 {
		t.Fatalf("replay frame=%d position=%.1f", g.replayIndex, g.cars[0].position)
	}
	g.finishReplay()
	if g.replay || g.cars != liveCars || g.scene != menuScene {
		t.Fatal("leaving replay did not restore the live session")
	}
}

func TestIdleDemoRestoresThePausedSession(t *testing.T) {
	g, err := New(Options{Mute: true})
	if err != nil {
		t.Fatal(err)
	}
	g.mode = qualifyMode
	g.startDrive(false)
	g.cars[0].position, g.cars[0].speed = 12345, 187
	g.clock.LapCentiseconds, g.clock.LastSpeed, g.clock.BestSpeed, g.raceTicks = 607, 21000, 22000, 1400
	liveClock := g.clock
	liveCars := g.cars
	g.startDrive(true)
	if !g.demo || g.cars == liveCars {
		t.Fatal("idle demonstration did not replace the live presentation state")
	}
	g.stopDemo()
	if g.cars != liveCars || g.mode != qualifyMode || g.clock != liveClock || g.raceTicks != 1400 {
		t.Fatal("idle demonstration did not restore the paused session")
	}
}

func TestTurboBoostChangesAcceleration(t *testing.T) {
	newCruisingGame := func(boost int) *Game {
		g, err := New(Options{Mute: true})
		if err != nil {
			t.Fatal(err)
		}
		g.mode = practiceMode
		g.startDrive(false)
		g.boost = boost
		g.placePlayerVehicle(1000, 7000, 80)
		g.syncPlayerVehicle()
		return g
	}
	low, high := newCruisingGame(1), newCruisingGame(9)
	for range 15 {
		low.stepDrive(drivingInput{throttle: true})
		high.stepDrive(drivingInput{throttle: true})
	}
	if high.cars[0].speed <= low.cars[0].speed {
		t.Fatalf("boost 9 speed %.2f did not exceed boost 1 speed %.2f", high.cars[0].speed, low.cars[0].speed)
	}
}

func TestCarSetupAffectsFuelAndClampsAdjustments(t *testing.T) {
	g, err := New(Options{Mute: true})
	if err != nil {
		t.Fatal(err)
	}
	g.mode = practiceMode
	g.startDrive(false)
	g.placePlayerVehicle(1000, 7000, 80)
	g.syncPlayerVehicle()
	for range 15 {
		g.stepDrive(drivingInput{throttle: true})
	}
	if g.fuelGallons >= 40 {
		t.Fatal("driving did not consume fuel")
	}
	g.informationPage, g.informationSelection = wingsInformationPage, 0
	g.adjustInformationValue(100)
	if g.setup.wings[0] != 16 || g.physicalConfig().Aero.FrontWingDrag != 0x35e8 {
		t.Fatalf("front wing escaped its range: %d", g.setup.wings[0])
	}
	g.cars[0].physics.Engine.Fuel = 0
	previousSpeed := g.cars[0].speed
	g.updateDrive()
	if g.cars[0].speed >= previousSpeed {
		t.Fatal("empty fuel tank continued accelerating the car")
	}
}

func TestPracticeStartsStoppedAtThePitBoard(t *testing.T) {
	g, err := New(Options{Mute: true})
	if err != nil {
		t.Fatal(err)
	}
	g.mode = practiceMode
	g.startDrive(false)
	if !g.pitBoard || !g.playerInPitLane() || g.cars[0].speed != 0 || g.fuelGallons != float64(g.setup.fuel) {
		t.Fatalf("practice start pit=%v speed=%.1f fuel=%.1f", g.pitBoard, g.cars[0].speed, g.fuelGallons)
	}
	g.stepDrive(drivingInput{throttle: true})
	if g.pitBoard {
		t.Fatal("pit board remained visible after applying throttle")
	}
}

func TestQualifyingResultSetsTheRaceGrid(t *testing.T) {
	g, err := New(Options{Mute: true})
	if err != nil {
		t.Fatal(err)
	}
	g.mode = qualifyMode
	g.startDrive(false)
	// This original four-lap completion has a measured average speed of
	// 214.95 MPH and ranks fifteenth. The original continues qualifying.
	g.clock = session.Clock{Laps: 4, CrossingDelta: -60, QualifyingTime: 12561, LapCentiseconds: 4180, Phase: 5, Current: session.FormatTime(4180), Best: session.FormatTime(4097)}
	g.trafficModel.SignalFlags = 0x15
	event := g.advanceClock(6900, 6960)
	if !event.Qualified || g.scene != driveScene || g.qualifyingPosition != 15 || g.clock.Laps != 1 {
		t.Fatalf("qualifying scene=%d position=%d", g.scene, g.qualifyingPosition)
	}
	g.mode = raceMode
	g.startDrive(false)
	if g.cars[0].number != 17 || g.cars[0].progress != -2702 || g.trafficModel.Cars[0].Distance != 2702 || g.trafficModel.Cars[0].Lane != 2 {
		t.Fatalf("player number=%d grid progress=%.0f", g.cars[0].number, g.cars[0].progress)
	}
	numbers := map[int]bool{}
	for _, vehicle := range g.cars {
		if vehicle.number < 1 || numbers[vehicle.number] {
			t.Fatalf("invalid or duplicate original car identifier %d", vehicle.number)
		}
		numbers[vehicle.number] = true
	}
}

func TestRaceContinuesWhenAnOpponentTakesTheFlag(t *testing.T) {
	g, err := New(Options{Mute: true})
	if err != nil {
		t.Fatal(err)
	}
	g.mode, g.raceLaps = raceMode, 10
	g.startDrive(false)
	g.trafficModel.Cars[1].RemainingLaps = 1
	g.trafficModel.Cars[1].Distance, g.trafficModel.Cars[1].Fraction, g.trafficModel.Cars[1].Speed = 0, 0, 512
	g.updateDrive()
	if g.scene != driveScene || !g.session || g.trafficModel.Cars[1].Flags != 255 || g.trafficModel.LastPlayerFlags != 0 {
		t.Fatalf("opponent finish interrupted the live race: scene%d session%v flags%x", g.scene, g.session, g.trafficModel.Cars[1].Flags)
	}
}

func TestThirtyLapCollisionQueuesYellowWithoutDamage(t *testing.T) {
	g, err := New(Options{Mute: true})
	if err != nil {
		t.Fatal(err)
	}
	g.mode, g.raceLaps = raceMode, 30
	g.startDrive(false)
	g.placePlayerVehicle(1000, 7000, 200)
	g.syncPlayerVehicle()
	g.trafficModel.Cars[0].Distance = trackLength - 1000
	// The physical step advances the player before checking the original
	// forty-unit contact window around its new position.
	g.trafficModel.Cars[1].Distance, g.trafficModel.Cars[1].Speed, g.trafficModel.Cars[1].Lateral = trackLength-1060, 0, 7000
	g.trafficModel.Cars[1].Action = traffic.Stopped
	g.updateDrive()
	model := g.trafficModel
	if model.YellowDelay != 74 || model.Yellow || g.damage != 0 {
		t.Fatalf("pending caution=%d active=%v damage=%.2f", model.YellowDelay, model.Yellow, g.damage)
	}
	for range 74 {
		g.stepDrive(drivingInput{})
	}
	if !model.Yellow || model.YellowDelay != 0 || g.physicalConfig().ThrottleLimit != 0x6000 {
		t.Fatalf("active caution=%v delay=%d throttle limit=%x", model.Yellow, model.YellowDelay, g.physicalConfig().ThrottleLimit)
	}
}

func TestWallContactUpdatesAuthoritativeVehicle(t *testing.T) {
	g, err := New(Options{Mute: true, Practice: true})
	if err != nil {
		t.Fatal(err)
	}
	g.placePlayerVehicle(9538.5-6930, 53*16, 24)
	g.syncPlayerVehicle()
	for range 20 {
		g.stepDrive(drivingInput{throttle: true})
	}
	player := &g.cars[0]
	if player.worldX > 9190 || player.physics.Body.PositionX>>16 != int32(player.worldX) {
		t.Fatalf("pit barrier did not constrain physical motion: X%d, modelX%d", player.worldX, player.physics.Body.PositionX>>16)
	}
	camera, _ := g.originalCamera(inCarView)
	if camera.X != player.worldX || camera.Y != player.worldY {
		t.Fatalf("cockpit ignored physical contact: %+v", camera)
	}
}

func TestEngineToneFollowsPhysicalRPM(t *testing.T) {
	g, err := New(Options{Mute: true, Practice: true})
	if err != nil {
		t.Fatal(err)
	}
	g.team = 1
	g.cars[0].physics.Engine.RPM = 6000
	g.soundController.Pitch[0] = 18000
	g.updateSoundSources()
	for range 4 {
		g.advanceSound()
	}
	high := g.soundController.Voices[0].Rate
	g.cars[0].physics.Engine.RPM = 3000
	g.updateSoundSources()
	for range 4 {
		g.advanceSound()
	}
	if g.soundController.Voices[0].Rate >= high || g.soundController.Pitch[0] != 9000 {
		t.Fatal("the recovered engine tone did not follow physical RPM")
	}
}

func TestPitServiceRepairsTheSelectedPhysicalTire(t *testing.T) {
	g, err := New(Options{Mute: true, Practice: true})
	if err != nil {
		t.Fatal(err)
	}
	player := &g.cars[0]
	player.physics.Engine.Failures |= 1
	player.physics.Tires[3].Failed = true
	player.physics.Tires[3].Life = 0
	player.physics.Tires[3].Bands = [3]uint16{24000, 25000, 26000}
	g.syncVehicleCondition()
	if g.failure != "TIRE BLEW" || g.tireWear[0] != 1 {
		t.Fatal("physical puncture was not displayed")
	}
	g.servicePitItem(1)
	if player.physics.Tires[3].Life != 850<<16 || player.physics.Tires[3].Failed || player.physics.Engine.Failures&1 != 0 || g.tireWear[0] != 0 || g.tireTemperature[0] != 150 || g.failure != "" {
		t.Fatalf("pit service did not replace the RF tire: %+v", player.physics.Tires[3])
	}
	player.physics.Engine.Fuel = 0
	g.setup.fuel = 25
	g.servicePitItem(0)
	if player.physics.Engine.Fuel != 25*32<<16 || player.physics.Engine.Mass != 8300 || g.fuelGallons != 25 {
		t.Fatal("pit fuel did not reach the physical engine")
	}
}

func TestOriginalPracticePoseAndServiceZone(t *testing.T) {
	g, err := New(Options{Mute: true, Practice: true})
	if err != nil {
		t.Fatal(err)
	}
	camera, context := g.originalCamera(inCarView)
	if camera != (worldscene.Camera{X: 4110, Y: 6761, Z: 15}) || context.Progress != 7005 || context.Lateral != 39 || !g.playerInPitLane() {
		t.Fatalf("practice start differs from actual FS-UAE Practice: %+v %+v", camera, context)
	}
	g.cars[0].position = 300
	if g.playerInPitLane() {
		t.Fatal("pit service extended beyond the original parking interval")
	}
}

func TestOriginalTrackCoordinateTranslation(t *testing.T) {
	const lateral = 7200
	tests := []struct {
		progress float64
		x        float64
		z        float64
	}{
		{0, -9900, 7172},
		{9900, 9900, 7172},
		{13860, 15092, 1980},
		{15840, 15092, -1980},
		{19800, 9900, -7172},
		{29700, -9900, -7172},
		{33660, -15092, -1980},
		{35640, -15092, 1980},
	}
	g, err := New(Options{Mute: true})
	if err != nil {
		t.Fatal(err)
	}
	for _, test := range tests {
		position := math.Mod(test.progress-0x1b12+trackLength, trackLength)
		point := g.world.TrackPlacement(uint16(originalTrackProgress(position)), lateral)
		x, z := float64(point.X), float64(point.Y)
		if math.Abs(x-test.x) > 0.01 || math.Abs(z-test.z) > 0.01 {
			t.Fatalf("progress %.0f translated to %.2f,%.2f; want %.2f,%.2f", test.progress, x, z, test.x, test.z)
		}
	}
}

func TestPracticeKeepsOriginalWallCoordinateSystem(t *testing.T) {
	g, err := New(Options{Mute: true, Practice: true})
	if err != nil {
		t.Fatal(err)
	}
	inner, outer := g.trackBounds(g.cars[0].position)
	// Source $17ee uses world lateral20..580 on the two long straights.
	// The original pit pose is inside this range and must not be clamped to
	// one of the racing-line arrays when the first simulation tick runs.
	if inner != 320 || outer != 9280 {
		t.Fatalf("wall limits %g..%g", inner, outer)
	}
	for range 5 {
		g.updateDrive()
	}
	if g.cars[0].lateral != 624 || g.damage != 0 {
		t.Fatal("the first ticks displaced or crashed the original pit pose")
	}
}

func TestFileMenuCopiesSetupsAndReplay(t *testing.T) {
	g, err := New(Options{Mute: true})
	if err != nil {
		t.Fatal(err)
	}
	g.setup.wings[0] = 9
	g.menuPage, g.selected = fileMenuPage, 1
	g.activateFileItem()
	g.setup.wings[0] = 0
	g.menuPage, g.selected = fileMenuPage, 0
	g.activateFileItem()
	if g.setup.wings[0] != 9 {
		t.Fatal("saved car setup was not copied back")
	}
	for frame := 1; frame <= 4; frame++ {
		g.cars[0].position = float64(frame * 100)
		g.recordReplayFrame()
	}
	g.saveCurrentReplay()
	g.replayHead, g.replayCount = 0, 0
	g.loadSavedReplay()
	if !g.replay || g.replayCount != 4 || g.cars[0].position != 100 {
		t.Fatalf("saved replay loaded count=%d position=%.0f", g.replayCount, g.cars[0].position)
	}
}

func TestFileSlotsPersistAcrossRuns(t *testing.T) {
	path := filepath.Join(t.TempDir(), "files.json")
	g, err := New(Options{Mute: true, SavePath: path})
	if err != nil {
		t.Fatal(err)
	}
	g.setup.wings[0], g.team, g.qualifyingPosition = 8, 2, 7
	g.qualifyingSpeed = 22314
	g.savedSetups[1] = g.setup
	g.savedSetupValid[1] = true
	for frame := 1; frame <= 3; frame++ {
		g.cars[0].position = float64(frame * 50)
		g.contactJolt = vehicle.CameraJolt{Pitch: int16(frame * 30), Roll: int16(-frame * 20)}
		g.recordReplayFrame()
	}
	g.saveCurrentReplay()
	g.saveFiles()
	reloaded, err := New(Options{Mute: true, SavePath: path})
	if err != nil {
		t.Fatal(err)
	}
	if reloaded.setup.wings[0] != 8 || reloaded.team != 2 || reloaded.qualifyingPosition != 7 || reloaded.qualifyingSpeed != 22314 || len(reloaded.savedReplay) != 3 {
		t.Fatalf("persistent files loaded setup=%d team=%d grid=%d replay=%d", reloaded.setup.wings[0], reloaded.team, reloaded.qualifyingPosition, len(reloaded.savedReplay))
	}
	if reloaded.savedReplay[2].contactJolt != (vehicle.CameraJolt{Pitch: 90, Roll: -60}) {
		t.Fatal("saved replay lost the recorded contact camera impulse")
	}
}
