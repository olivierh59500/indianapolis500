package game

import (
	"encoding/json"
	"math"
	"os"

	"github.com/olivierh59500/indianapolis500/internal/debris"
	"github.com/olivierh59500/indianapolis500/internal/vehicle"
)

type persistentSetup struct {
	Fuel      int    `json:"fuel"`
	Wings     [2]int `json:"wings"`
	Rubber    [4]int `json:"rubber"`
	Stagger   int    `json:"stagger"`
	Pressures [4]int `json:"pressures"`
	Shocks    [4]int `json:"shocks"`
	Cambers   [4]int `json:"cambers"`
	Gears     [4]int `json:"gears"`
}

type persistentCar struct {
	Position        float64              `json:"position"`
	Progress        float64              `json:"progress"`
	Lateral         float64              `json:"lateral"`
	LateralVelocity float64              `json:"lateral_velocity"`
	Speed           float64              `json:"speed"`
	Steering        float64              `json:"steering"`
	Lap             int                  `json:"lap"`
	Gear            int                  `json:"gear"`
	Color           int                  `json:"color"`
	Secondary       int                  `json:"secondary"`
	Number          int                  `json:"number"`
	Player          bool                 `json:"player"`
	Active          bool                 `json:"active,omitempty"`
	Yaw             int16                `json:"yaw,omitempty"`
	AngularVelocity int16                `json:"angular_velocity,omitempty"`
	DamageFlags     byte                 `json:"damage_flags,omitempty"`
	WorldX          int16                `json:"world_x,omitempty"`
	WorldY          int16                `json:"world_y,omitempty"`
	WorldZ          int16                `json:"world_z,omitempty"`
	WorldVX         int16                `json:"world_vx,omitempty"`
	WorldVY         int16                `json:"world_vy,omitempty"`
	WorldPose       bool                 `json:"world_pose,omitempty"`
	DashboardSpeed  int16                `json:"dashboard_speed,omitempty"`
	Physics         *vehicle.State       `json:"physics,omitempty"`
	Driver          *vehicle.DriverState `json:"driver,omitempty"`
}

type persistentReplayFrame struct {
	Effects     *debris.Pool       `json:"effects,omitempty"`
	Cars        [33]persistentCar  `json:"cars"`
	ContactJolt vehicle.CameraJolt `json:"contact_jolt"`
}

type persistentFiles struct {
	Version            int                     `json:"version"`
	Setup              persistentSetup         `json:"setup"`
	SavedSetups        [3]persistentSetup      `json:"saved_setups"`
	SavedSetupValid    [3]bool                 `json:"saved_setup_valid"`
	SavedReplay        []persistentReplayFrame `json:"saved_replay,omitempty"`
	Team               int                     `json:"team"`
	Detail             int                     `json:"detail"`
	Sensitivity        int                     `json:"sensitivity"`
	QualifyingPosition int                     `json:"qualifying_position"`
	QualifyingSpeed    uint16                  `json:"qualifying_speed,omitempty"`
}

func setupToPersistent(setup carSetup) persistentSetup {
	return persistentSetup{
		Fuel: setup.fuel, Wings: setup.wings, Rubber: setup.rubber, Stagger: setup.stagger,
		Pressures: setup.pressures, Shocks: setup.shocks, Cambers: setup.cambers, Gears: setup.gears,
	}
}

func persistentToSetup(setup persistentSetup) carSetup {
	return carSetup{
		fuel: setup.Fuel, wings: setup.Wings, rubber: setup.Rubber, stagger: setup.Stagger,
		pressures: setup.Pressures, shocks: setup.Shocks, cambers: setup.Cambers, gears: setup.Gears,
	}
}

func carToPersistent(vehicle car) persistentCar {
	result := persistentCar{
		Position: vehicle.position, Progress: vehicle.progress, Lateral: vehicle.lateral,
		LateralVelocity: vehicle.lateralVelocity, Speed: vehicle.speed, Steering: vehicle.steering,
		Lap: vehicle.lap, Gear: vehicle.gear, Color: vehicle.color, Secondary: vehicle.secondary,
		Number: vehicle.number, Player: vehicle.player, Active: vehicle.active,
		Yaw: vehicle.yaw, AngularVelocity: vehicle.angularVelocity, DamageFlags: vehicle.damageFlags,
		WorldX: vehicle.worldX, WorldY: vehicle.worldY, WorldZ: vehicle.worldZ, WorldVX: vehicle.worldVX, WorldVY: vehicle.worldVY, WorldPose: vehicle.worldPose,
		DashboardSpeed: vehicle.dashboardSpeed,
	}
	if vehicle.physical {
		result.Physics, result.Driver = &vehicle.physics, &vehicle.driver
	}
	return result
}

func persistentToCar(vehicle persistentCar) car {
	result := car{
		position: vehicle.Position, progress: vehicle.Progress, lateral: vehicle.Lateral,
		lateralVelocity: vehicle.LateralVelocity, speed: vehicle.Speed, steering: vehicle.Steering,
		lap: vehicle.Lap, gear: vehicle.Gear, color: vehicle.Color, secondary: vehicle.Secondary,
		number: vehicle.Number, player: vehicle.Player, active: vehicle.Active,
		yaw: vehicle.Yaw, angularVelocity: vehicle.AngularVelocity, damageFlags: vehicle.DamageFlags,
		worldX: vehicle.WorldX, worldY: vehicle.WorldY, worldZ: vehicle.WorldZ, worldVX: vehicle.WorldVX, worldVY: vehicle.WorldVY, worldPose: vehicle.WorldPose,
		dashboardSpeed: vehicle.DashboardSpeed,
	}
	if vehicle.Physics != nil {
		result.physical, result.physics = true, *vehicle.Physics
	}
	if vehicle.Driver != nil {
		result.driver = *vehicle.Driver
	}
	return result
}

func (g *Game) installDefaultSetupSlots() {
	base := defaultCarSetup()
	g.savedSetups[0] = base
	g.savedSetups[0].wings = [2]int{7, 7}
	g.savedSetups[0].rubber = [4]int{0, 0, 0, 0}
	g.savedSetups[1] = base
	g.savedSetups[1].wings = [2]int{3, 3}
	g.savedSetups[1].gears = [4]int{500, 395, 300, 235}
	g.savedSetups[2] = base
	g.savedSetups[2].shocks = [4]int{0, 0, 1, 1}
	g.savedSetupValid = [3]bool{true, true, true}
}

func (g *Game) activateFileItem() {
	if g.selected < 6 {
		slot := g.selected / 2
		if g.selected&1 == 0 {
			if g.savedSetupValid[slot] {
				previous := g.setup
				g.setup = g.savedSetups[slot]
				g.applyLiveSetup(previous)
				if g.mode != practiceMode {
					g.session = false
				}
			}
		} else {
			g.savedSetups[slot], g.savedSetupValid[slot] = g.setup, true
			g.saveFiles()
		}
		g.menuPage, g.selected = mainMenuPage, 6
		return
	}
	if g.selected == 6 {
		g.loadSavedReplay()
		return
	}
	g.saveCurrentReplay()
	g.saveFiles()
	g.menuPage, g.selected = mainMenuPage, 6
}

func (g *Game) saveFiles() {
	if g.savePath == "" {
		return
	}
	files := persistentFiles{
		Version: 2, Setup: setupToPersistent(g.setup), SavedSetupValid: g.savedSetupValid,
		Team: g.team, Detail: g.detail, Sensitivity: g.sensitivity, QualifyingPosition: g.qualifyingPosition, QualifyingSpeed: g.qualifyingSpeed,
	}
	for index := range g.savedSetups {
		files.SavedSetups[index] = setupToPersistent(g.savedSetups[index])
	}
	files.SavedReplay = make([]persistentReplayFrame, len(g.savedReplay))
	for frameIndex, frame := range g.savedReplay {
		files.SavedReplay[frameIndex].ContactJolt = frame.contactJolt
		if frame.effects.Count > 0 {
			effects := frame.effects
			files.SavedReplay[frameIndex].Effects = &effects
		}
		for carIndex, vehicle := range frame.cars {
			files.SavedReplay[frameIndex].Cars[carIndex] = carToPersistent(vehicle)
		}
	}
	data, err := json.Marshal(files)
	if err != nil {
		return
	}
	temporary := g.savePath + ".tmp"
	if os.WriteFile(temporary, data, 0o600) == nil {
		_ = os.Rename(temporary, g.savePath)
	}
}

func (g *Game) loadFiles() {
	if g.savePath == "" {
		return
	}
	data, err := os.ReadFile(g.savePath)
	if err != nil {
		return
	}
	var files persistentFiles
	if json.Unmarshal(data, &files) != nil || files.Version < 1 || files.Version > 2 {
		return
	}
	if files.Version == 1 {
		files.Setup = setupToPersistent(upgradeSetup(persistentToSetup(files.Setup)))
		for i := range files.SavedSetups {
			files.SavedSetups[i] = setupToPersistent(upgradeSetup(persistentToSetup(files.SavedSetups[i])))
		}
		original := files.SavedReplay
		files.SavedReplay = make([]persistentReplayFrame, (len(original)*15+19)/20)
		for i := range files.SavedReplay {
			files.SavedReplay[i] = original[min(len(original)-1, (i*20+7)/15)]
		}
	}
	setup := persistentToSetup(files.Setup)
	if !validCarSetup(setup) {
		return
	}
	g.setup = setup
	for index := range g.savedSetups {
		candidate := persistentToSetup(files.SavedSetups[index])
		if files.SavedSetupValid[index] && validCarSetup(candidate) {
			g.savedSetups[index], g.savedSetupValid[index] = candidate, true
		}
	}
	if team := max(0, min(2, files.Team)); team != g.team {
		g.selectTeam(team)
	}
	g.detail = max(0, min(2, files.Detail))
	g.sensitivity = max(0, min(2, files.Sensitivity))
	g.qualifyingPosition = max(1, min(33, files.QualifyingPosition))
	g.qualifyingSpeed = files.QualifyingSpeed
	if len(files.SavedReplay) <= len(g.replayFrames) {
		g.savedReplay = make([]replayFrame, len(files.SavedReplay))
		for frameIndex, frame := range files.SavedReplay {
			g.savedReplay[frameIndex].contactJolt = frame.ContactJolt
			if frame.Effects != nil {
				g.savedReplay[frameIndex].effects = *frame.Effects
			}
			for carIndex, vehicle := range frame.Cars {
				g.savedReplay[frameIndex].cars[carIndex] = persistentToCar(vehicle)
			}
		}
	}
}

func validCarSetup(setup carSetup) bool {
	if setup.fuel < 5 || setup.fuel > 40 || setup.stagger < -10 || setup.stagger > 10 {
		return false
	}
	for _, value := range setup.wings {
		if value < 0 || value > 16 {
			return false
		}
	}
	for index := range 4 {
		if setup.rubber[index] < 0 || setup.rubber[index] > 2 || setup.pressures[index] < 10240 || setup.pressures[index] > 20480 ||
			setup.shocks[index] < 0 || setup.shocks[index] > 8 || setup.cambers[index] < -12 || setup.cambers[index] > 12 ||
			setup.gears[index] < 280 || setup.gears[index] > 1300 {
			return false
		}
	}
	return true
}

// Preserve older local files when upgrading the former approximate setup
// ranges to the recovered physical detents. New files store version two.
func upgradeSetup(old carSetup) carSetup {
	setup := defaultCarSetup()
	setup.fuel = max(5, min(40, (old.fuel+2)/5*5))
	setup.stagger = max(-10, min(10, old.stagger))
	for i := range setup.wings {
		setup.wings[i] = max(0, min(16, 14+old.wings[i]-5))
	}
	for i := range 4 {
		setup.rubber[i] = max(0, min(2, old.rubber[i]))
		setup.pressures[i] = max(10240, min(20480, 10240+(old.pressures[i]-25)*256))
		setup.shocks[i] = max(0, min(8, setup.shocks[i]+old.shocks[i]-1))
		legacyCamber := [4]int{-2, 2, -1, 1}
		setup.cambers[i] = max(-12, min(12, setup.cambers[i]+int(math.Round(float64(old.cambers[i]-legacyCamber[i])*0.4))))
	}
	setup.gears = gearSetup(max(280, min(490, 480+(old.gears[3]-260)*2)))
	return setup
}

func (g *Game) saveCurrentReplay() {
	if g.replayCount < 2 {
		return
	}
	if cap(g.savedReplay) < g.replayCount {
		g.savedReplay = make([]replayFrame, g.replayCount)
	} else {
		g.savedReplay = g.savedReplay[:g.replayCount]
	}
	start := (g.replayHead - g.replayCount + len(g.replayFrames)) % len(g.replayFrames)
	for index := range g.savedReplay {
		g.savedReplay[index] = g.replayFrames[(start+index)%len(g.replayFrames)]
	}
}

func (g *Game) loadSavedReplay() {
	if len(g.savedReplay) < 2 {
		return
	}
	count := min(len(g.savedReplay), len(g.replayFrames))
	copy(g.replayFrames[:count], g.savedReplay[len(g.savedReplay)-count:])
	g.replayCount = count
	g.replayHead = count % len(g.replayFrames)
	g.startReplay()
}
