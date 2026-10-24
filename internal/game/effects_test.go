package game

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/olivierh59500/indianapolis500/internal/debris"
)

// These are the emitter arguments observed at the original collision reached
// with ordinary accelerator input, including the two-unit prepared motion.
func TestOriginalPlayedWallEffectArguments(t *testing.T) {
	g, err := New(Options{Mute: true, Practice: true})
	if err != nil {
		t.Fatal(err)
	}
	s := &g.cars[0].physics
	s.Progress, s.Lateral = 9537, 39
	c := &g.trafficModel.Cars[0]
	c.Distance, c.Lateral, c.Primary, c.Secondary = 36997, 624, 9, 9
	want := debris.Input{Severity: 1152, Colors: 0x0909, Sideways: 0, Forward: 512, Lateral: 624, Distance: 36993}
	if got := g.wallCollisionEffect(1152); got != want {
		t.Fatalf("native%+v original%+v", got, want)
	}
}

func TestCollisionEffectsSurviveReplayAndAttract(t *testing.T) {
	g, err := New(Options{Mute: true, Practice: true})
	if err != nil {
		t.Fatal(err)
	}
	camera, _ := g.originalCamera(inCarView)
	input := debris.Input{Severity: 800, Colors: 0x0909, Distance: 36993, Lateral: 624, Forward: 512}
	g.effects.Spawn(input, g.world, camera, &g.soundController.Requests)
	first := g.effects.Pool
	g.recordReplayFrame()
	g.effects.Update(camera, &g.soundController.Requests)
	g.recordReplayFrame()
	live := g.effects.Pool
	g.startReplay()
	if g.effects.Pool != first {
		t.Fatal("replay lost the original recorded collision state")
	}
	g.finishReplay()
	if g.effects.Pool != live {
		t.Fatal("leaving replay changed live collision effects")
	}
	g.startDrive(true)
	g.stopDemo()
	if g.effects.Pool != live {
		t.Fatal("temporary attract mode changed paused collision effects")
	}
	g.saveCurrentReplay()
	g.savePath = filepath.Join(t.TempDir(), "game.json")
	g.saveFiles()
	if _, err = os.Stat(g.savePath); err != nil {
		t.Fatal(err)
	}
	restored, err := New(Options{Mute: true, SavePath: g.savePath})
	if err != nil {
		t.Fatal(err)
	}
	if len(restored.savedReplay) != 2 || restored.savedReplay[0].effects != first || restored.savedReplay[1].effects != live {
		t.Fatal("saved replay lost collision effects")
	}
}
