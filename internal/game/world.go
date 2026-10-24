package game

import (
	"fmt"
	"github.com/hajimehoshi/ebiten/v2"
	gameassets "github.com/olivierh59500/indianapolis500/assets"
	worldscene "github.com/olivierh59500/indianapolis500/internal/scene"
	"image"
	"image/color"
)

func (g *Game) loadOriginalWorld() error {
	names := []string{"world-vertices.bin", "world-data.bin", "world-trigonometry.bin", "world-horizon.bin", "world-lod.bin", "track-cosine.bin", "track-bank.bin", "camera-arctangent.bin", "camera-bank-gradient.bin", "camera-trackside.bin", "track-wall-inner.bin", "track-wall-outer.bin", "track-barriers.bin", "collision-damage.bin"}
	data := make([][]byte, len(names))
	for i, name := range names {
		d, err := gameassets.Original.ReadFile("original/" + name)
		if err != nil {
			return err
		}
		data[i] = d
	}
	var err error
	g.world, err = worldscene.New(data[0], data[1], data[2], data[3], data[4])
	if err != nil {
		return err
	}
	if err = g.world.LoadTrack(data[5], data[6]); err != nil {
		return err
	}
	if err = g.world.LoadCameras(data[7], data[8], data[9]); err != nil {
		return err
	}
	if err = g.world.LoadTrackLimits(data[10], data[11]); err != nil {
		return err
	}
	if err = g.world.LoadBarriers(data[12], data[13]); err != nil {
		return err
	}
	crowd, err := gameassets.Original.ReadFile("original/crowd-patterns.bin")
	if err != nil {
		return err
	}
	if err = g.world.LoadCrowdPatterns(crowd); err != nil {
		return err
	}
	contactDamage, err := gameassets.Original.ReadFile("original/car-contact-damage.bin")
	if err != nil {
		return err
	}
	if err = g.world.LoadCarContactDamage(contactDamage); err != nil {
		return err
	}
	g.worldRenderer = g.world.NewRenderer()
	g.worldOperations = make([]worldscene.Operation, 0, 128)
	g.worldPixels = make([]byte, worldscene.Width*worldscene.Height*4)
	g.worldImage = ebiten.NewImage(worldscene.Width, worldscene.Height)
	for i, c := range g.palette {
		r, green, b, a := c.RGBA()
		g.worldColors[i] = [4]byte{byte(r >> 8), byte(green >> 8), byte(b >> 8), byte(a >> 8)}
	}
	return g.loadCockpitBobs()
}

func (g *Game) originalCamera(view int) (worldscene.Camera, worldscene.Context) {
	vehicle := g.presentedCar(0)
	views := [...]worldscene.View{worldscene.Television, worldscene.Sky, worldscene.Trackside, worldscene.Behind, worldscene.InCar, worldscene.Crash}
	camera, context := g.world.CameraForCar(uint16(originalTrackProgress(vehicle.position)), int16(vehicle.lateral), vehicle.yaw, views[view])
	if view == inCarView && vehicle.worldPose {
		camera.X, camera.Y, camera.Z = vehicle.worldX, vehicle.worldY, vehicle.worldZ
		if vehicle.physical {
			camera.Pitch += vehicle.physics.Body.Pitch
			camera.Roll += vehicle.physics.Body.Roll
			camera.Pitch += g.contactJolt.Pitch
			camera.Roll += g.contactJolt.Roll
		}
	}
	context.Detail = g.detail
	context.Mode = uint8(g.mode)
	return camera, context
}

func (g *Game) drawOriginalWorld(target *ebiten.Image, view int) worldscene.Camera {
	camera, context := g.originalCamera(view)
	// The source advances this bitmap phase every six PAL animation ticks.
	g.worldRenderer.CrowdPhase = uint8((g.presentationTick * 50 / PresentationTicksPerSecond / 6 * 12) & 63)
	var err error
	g.worldOperations, err = g.world.SceneOperations(g.worldOperations[:0], camera, context)
	if err == nil {
		g.prepareProjectedCars(camera, context, view)
		err = g.worldRenderer.RenderWithCars(camera, g.worldOperations, g.paintCars)
		if err == nil {
			g.effects.Paint(g.worldRenderer)
		}
	}
	if err != nil {
		g.err, g.done = err, true
		return camera
	}
	for i, index := range g.worldRenderer.Pixels {
		c := g.worldColors[index&15]
		offset := i * 4
		copy(g.worldPixels[offset:offset+4], c[:])
	}
	g.worldImage.WritePixels(g.worldPixels)
	target.DrawImage(g.worldImage, nil)
	return camera
}

// loadCockpitBobs decodes artwork copied by $deb6. Mask planes make the
// road visible around the wheels and nose without a guessed polygon outline.
func (g *Game) loadCockpitBobs() error {
	data, err := gameassets.Original.ReadFile("original/cockpit-bobs.bin")
	if err != nil {
		return err
	}
	if len(data) != 39840 {
		return fmt.Errorf("recovered cockpit bobs have invalid dimensions")
	}
	decode := func(data []byte, width, height int) *ebiten.Image {
		frame := image.NewNRGBA(image.Rect(0, 0, width, height))
		stride := width / 8
		planeBytes := stride * height
		for y := 0; y < height; y++ {
			for x := 0; x < width; x++ {
				offset, bit := y*stride+x/8, byte(1<<uint(7-x%8))
				if data[offset]&bit == 0 {
					continue
				}
				index := 0
				for plane := 0; plane < 4; plane++ {
					if data[(plane+1)*planeBytes+offset]&bit != 0 {
						index |= 1 << plane
					}
				}
				c := g.worldColors[index]
				frame.SetNRGBA(x, y, color.NRGBA{c[0], c[1], c[2], 255})
			}
		}
		return ebiten.NewImageFromImage(frame)
	}
	for i := range 2 {
		g.cockpitNose[i+1] = decode(data[i*1920:(i+1)*1920], 192, 16)
	}
	march, err := gameassets.Original.ReadFile("original/cockpit-march-nose.bin")
	if err != nil {
		return err
	}
	if len(march) != 1920 {
		return fmt.Errorf("invalid recovered March cockpit nose")
	}
	g.cockpitNose[0] = decode(march, 192, 16)
	for steering := range g.cockpitWheels {
		for side := range 2 {
			offset := 3840 + (steering*2+side)*1200
			g.cockpitWheels[steering][side] = decode(data[offset:offset+1200], 64, 30)
		}
	}
	outline := image.NewNRGBA(image.Rect(0, 0, 320, 23))
	c := g.worldColors[15]
	points := [][2]int{{0, 22}, {78, 8}, {145, 0}, {175, 0}, {241, 8}, {319, 22}}
	for i := 1; i < len(points); i++ {
		x, y := points[i-1][0], points[i-1][1]
		tx, ty := points[i][0], points[i][1]
		dx, dy := tx-x, ty-y
		sx, sy := 1, 1
		if dx < 0 {
			dx = -dx
			sx = -1
		}
		if dy < 0 {
			dy = -dy
			sy = -1
		}
		e := dx - dy
		for {
			outline.SetNRGBA(x, y, color.NRGBA{c[0], c[1], c[2], 255})
			if x == tx && y == ty {
				break
			}
			e2 := e * 2
			if e2 > -dy {
				e -= dy
				x += sx
			}
			if e2 < dx {
				e += dx
				y += sy
			}
		}
	}
	g.cockpitOutline = ebiten.NewImageFromImage(outline)
	return nil
}

func (g *Game) drawCockpitFront(screen *ebiten.Image) {
	player := &g.cars[0]
	steering, bob := 6, [2]byte{8, 8}
	if player.physical {
		steering = max(0, min(12, 6+((int(player.physics.Steering)+128)>>9)))
		bob = player.physics.WheelBob
	}
	for side := range 2 {
		op := &ebiten.DrawImageOptions{}
		op.GeoM.Translate(float64(side*256), float64(118+int(bob[side])))
		screen.DrawImage(g.cockpitWheels[steering][side], op)
	}
	op := &ebiten.DrawImageOptions{}
	op.GeoM.Translate(64, 132)
	screen.DrawImage(g.cockpitNose[g.team], op)
	op = &ebiten.DrawImageOptions{}
	op.GeoM.Translate(0, 117)
	screen.DrawImage(g.cockpitOutline, op)
}
