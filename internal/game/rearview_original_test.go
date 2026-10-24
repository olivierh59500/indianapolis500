package game

import (
	"compress/gzip"
	"encoding/json"
	"fmt"
	"os"
	"testing"

	worldscene "github.com/olivierh59500/indianapolis500/internal/scene"
)

func readMirrorCases(t *testing.T, name string, target any) {
	t.Helper()
	f, err := os.Open("testdata/rearview-" + name + ".gz")
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	z, err := gzip.NewReader(f)
	if err != nil {
		t.Fatal(err)
	}
	defer z.Close()
	if err := json.NewDecoder(z).Decode(target); err != nil {
		t.Fatal(err)
	}
}

func resetNativeMirror(g *Game) {
	for i := range g.carProjection.mirrorPixels {
		g.carProjection.mirrorPixels[i] = 7
		if i < 160 {
			g.carProjection.mirrorPixels[i] = 11
		}
	}
}

// These complete eighty-by-eleven rear images are independent controlled
// calls to the original mirror scanline painter, including clipping and flips.
func TestOriginalMirrorScanlines(t *testing.T) {
	var rows []struct {
		Tier, Angle, Scale, Offset, Primary, Secondary int
		Mirrored                                       bool
		Pixels                                         []byte
	}
	readMirrorCases(t, "raster-cases.json", &rows)
	if len(rows) != 12 {
		t.Fatal("incomplete independent original mirror rasters")
	}
	g, err := New(Options{Mute: true})
	if err != nil {
		t.Fatal(err)
	}
	for n, row := range rows {
		t.Run(fmt.Sprint(n), func(t *testing.T) {
			resetNativeMirror(g)
			g.paintMirrorCarShape(36+row.Offset, row.Tier, row.Angle, row.Scale, row.Primary, row.Secondary, row.Mirrored)
			if len(row.Pixels) != len(g.carProjection.mirrorPixels) {
				t.Fatal("invalid original mirror size")
			}
			for i, expected := range row.Pixels {
				if actual := g.carProjection.mirrorPixels[i]; actual != expected {
					t.Fatalf("pixel%d/%d native%d original%d", i%80, i/80, actual, expected)
				}
			}
		})
	}
}

// Original direct-vector projection followed by the real rear-car selector
// supplies expected images, not a copy of the native projection calculation.
func TestOriginalMirrorProjectedCars(t *testing.T) {
	var rows []struct {
		Point                               [3]int16
		Camera                              [6]int16
		RelativeHeading, Primary, Secondary int
		Suppress                            bool
		Pixels                              []byte
	}
	readMirrorCases(t, "projected-cases.json", &rows)
	if len(rows) != 18 {
		t.Fatal("incomplete independent original rear-car projections")
	}
	g, err := New(Options{Mute: true})
	if err != nil {
		t.Fatal(err)
	}
	for n, row := range rows {
		t.Run(fmt.Sprint(n), func(t *testing.T) {
			g.cars = [33]car{}
			camera := worldscene.Camera{X: row.Camera[0], Y: row.Camera[1], Z: row.Camera[2], Heading: row.Camera[3], Pitch: row.Camera[4], Roll: row.Camera[5]}
			g.cars[0] = car{position: 900, active: true, player: true}
			g.cars[1] = car{position: 1000, lateral: 3000, active: true, lap: 1, worldPose: true, worldX: row.Point[0], worldY: row.Point[1], worldZ: row.Point[2], color: row.Primary, secondary: row.Secondary}
			pose := g.carWorldPose(1000, 3000)
			g.cars[1].yaw = int16(row.RelativeHeading) + camera.Heading - pose.Heading
			g.prepareProjectedCars(camera, worldscene.Context{}, inCarView)
			filter := worldscene.AllCars
			if row.Suppress {
				filter = worldscene.PitAroundStart
			}
			g.drawProjectedCarPass(filter)
			if len(row.Pixels) != len(g.carProjection.mirrorPixels) {
				t.Fatal("invalid original mirror size")
			}
			for i, expected := range row.Pixels {
				if actual := g.carProjection.mirrorPixels[i]; actual != expected {
					t.Fatalf("pixel%d/%d native%d original%d", i%80, i/80, actual, expected)
				}
			}
		})
	}
}
