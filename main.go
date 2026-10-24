// Indianapolis 500 runs the native Go game with Ebitengine presentation.
package main

import (
	"flag"
	"log"
	"os"
	"path/filepath"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/olivierh59500/indianapolis500/internal/game"
)

func main() {
	fullscreen := flag.Bool("fullscreen", false, "start in fullscreen mode")
	touch := flag.Bool("touch", false, "preview the Android touch layout")
	demo := flag.Bool("demo", false, "start the automatic demonstration")
	practice := flag.Bool("practice", false, "start directly in practice mode")
	race := flag.Bool("race", false, "start directly in a ten-lap race")
	mute := flag.Bool("mute", false, "disable sound")
	scale := flag.Int("scale", 3, "initial window scale from 1 to 8")
	smoke := flag.Int("smoke", 0, "run the selected scene for this many ticks")
	capture := flag.String("capture", "", "write the final smoke frame as PNG")
	dataDirectory := flag.String("data-dir", "", "override the settings and replay directory")
	flag.Parse()
	if *scale < 1 || *scale > 8 || *smoke < 0 || *capture != "" && *smoke == 0 || *demo && (*practice || *race) || *practice && *race {
		log.Fatal("invalid scale or smoke capture options")
	}
	if *smoke > 0 {
		if !*practice && !*race {
			*demo = true
		}
		ebiten.SetRunnableOnUnfocused(true)
		ebiten.SetWindowVisible(false)
	}
	savePath := ""
	if *smoke == 0 {
		directory := *dataDirectory
		if directory == "" {
			if configDirectory, configErr := os.UserConfigDir(); configErr == nil {
				directory = filepath.Join(configDirectory, "indianapolis500")
			}
		}
		if directory != "" {
			if err := os.MkdirAll(directory, 0o755); err != nil {
				log.Fatal(err)
			}
			savePath = filepath.Join(directory, "files.json")
		}
	}
	application, err := game.New(game.Options{Touch: *touch, Demo: *demo, Practice: *practice, Race: *race, Mute: *mute, Smoke: *smoke, Capture: *capture, SavePath: savePath})
	if err != nil {
		log.Fatal(err)
	}
	width := game.ScreenWidth
	if *touch {
		width = game.TouchWidth
	}
	ticksPerSecond := game.PresentationTicksPerSecond
	if *smoke > 0 {
		ticksPerSecond = 6000
	}
	ebiten.SetTPS(ticksPerSecond)
	ebiten.SetScreenFilterEnabled(false)
	ebiten.SetWindowTitle("Indianapolis 500 — Go / Ebitengine")
	ebiten.SetWindowSize(width**scale, game.ScreenHeight**scale)
	ebiten.SetWindowResizingMode(ebiten.WindowResizingModeEnabled)
	ebiten.SetFullscreen(*fullscreen)
	if err := ebiten.RunGame(application); err != nil {
		log.Fatal(err)
	}
}
