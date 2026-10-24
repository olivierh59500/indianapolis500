package game

import (
	"crypto/sha256"
	"fmt"
	worldscene "github.com/olivierh59500/indianapolis500/internal/scene"
	"testing"
)

// These hashes come from the original $d3d6 graphics routine, rendered over
// a cleared framebuffer in FS-UAE. They cover both artwork families, mirrored
// angles, scale phases and clipping at the viewport edges.
func TestOriginalCarScanlineRasters(t *testing.T) {
	cases := []struct {
		overhead           bool
		tier, angle, scale int
		mirrored           bool
		x, y               int
		digest             string
	}{
		{false, 20, 0, 0, false, 160, 62, "35547445f9518b63e27632265b5b737a6a54ac4269d156d7b12cebf266956c9b"},
		{false, 20, 8, 0, false, 160, 62, "81f2fff5d4b6b8022c80c5a8a46fe46db6ee4c9fd87c112c05aee74484094ce6"},
		{false, 20, 8, 0, true, 160, 62, "7b0b3fc1d8f7cf96bccf2124d9b0f6fd610dad3fca0c0589b5edd3eb47f0194e"},
		{false, 20, 15, 7, false, 174, 58, "68b3e8cfea7eca343c85afd23baf63aa0605e6319b5a5758405f914358e35a43"},
		{false, 20, 15, 7, true, 174, 58, "e2bb07724511ac13259984f7a4bb7f41abbe3ea69acaab1fdf900f86cc7e4086"},
		{false, 15, 4, 3, false, 150, 62, "2f1fb7bcb854a5473beda7b5cf04e5d1d038eaee8dea5e0ed8e2dd44aa35c476"},
		{false, 15, 4, 3, true, 150, 62, "0f772da7ab9696777a2cd586ac8d4720b91be0b4188816586e53e9fdeb6b0cf2"},
		{false, 5, 11, 1, false, 160, 68, "2a23d0e4e465ccc85210a4963258df036c6ebede4e565cfcdc663e418de14d26"},
		{false, 0, 0, 0, false, 160, 62, "5de187c5409c5a75a8435920333b503ae7906c2ff913ad15945f755e4be92651"},
		{false, 20, 2, 4, false, 15, 64, "4f090ea976fa88e8edb759493ff800e0d5ca87cada337717eaa30951714a5446"},
		{false, 20, 2, 4, true, 305, 64, "ddcaf46fc59df9bdc9ff3dbfc699d7a6a75f67cee7db7d587b1c98307a9c397a"},
		{true, 10, 0, 0, false, 160, 62, "84e0003bd608772042a8b70cb91a837953b90ef2bfdf1460f9a36729c2625912"},
		{true, 10, 15, 7, false, 160, 62, "c6b146212c653939d08d29ee2363da2e82f71b45b974a4d5403143a903176f91"},
		{true, 10, 15, 7, true, 160, 62, "ebc19f61bf82960f775851c2a3bc7e99fbdbc86df9f8049a8f788668da126080"},
		{true, 7, 31, 4, false, 160, 80, "b5f1529c5ef5feca78ea01ea53d40b808a99943cf3e9dfca97f54bb8e9725448"},
		{true, 7, 31, 4, true, 160, 80, "9973840a8e11fbc1da701c3a93d84301dd75c132ddf34e9425d3fa1055bd650d"},
		{true, 0, 0, 0, false, 160, 62, "5de187c5409c5a75a8435920333b503ae7906c2ff913ad15945f755e4be92651"},
	}
	g, err := New(Options{Mute: true})
	if err != nil {
		t.Fatal(err)
	}
	for i, c := range cases {
		clear(g.worldRenderer.Pixels)
		g.paintCarShape(c.x, c.y, c.tier, c.angle, c.scale, 12, 9, c.overhead, c.mirrored)
		if actual := fmt.Sprintf("%x", sha256.Sum256(g.worldRenderer.Pixels)); actual != c.digest {
			t.Fatalf("original car raster %d: got %s, want %s", i, actual, c.digest)
		}
	}
}

// These complete original $21912/$21ad6 calls include track placement,
// perspective projection, distance/angle selection, scaling and car drawing.
func TestOriginalProjectedCarRasters(t *testing.T) {
	cases := []struct {
		camera       worldscene.Camera
		progress     uint16
		lateral, yaw int16
		digest       string
	}{
		{worldscene.Camera{X: 4110, Y: 6761, Z: 15, Heading: 0, Pitch: 0, Roll: 0}, 7200, 1600, 0, "fd0fc5fe8101b07fa8faa0d283e4fc5c7ec2264a6697faefd8b4da3b0e1e15db"},
		{worldscene.Camera{X: 4110, Y: 6761, Z: 15, Heading: 0, Pitch: 0, Roll: 0}, 7200, 1600, 7000, "dec4c4af5ab5f073806765ecab8561c888226c75a27b1cbc6a1d8b04b65038fa"},
		{worldscene.Camera{X: 4110, Y: 6761, Z: 15, Heading: 0, Pitch: 0, Roll: 0}, 7200, 1600, -7000, "b61097efdad470c05a32728c814063c4324bd8346ce86bb3b07eeeded3c6ccbf"},
		{worldscene.Camera{X: 4110, Y: 6761, Z: 15, Heading: 0, Pitch: 0, Roll: 0}, 7100, 800, 12000, "59d4afb7fdb9c96a732609cf109692a105c720758759e6d7c584a5761c74e531"},
		{worldscene.Camera{X: 4110, Y: 6761, Z: 15, Heading: 0, Pitch: 0, Roll: 0}, 7600, 3200, 20000, "3637fa141cd30d9aa475f49966af2eb3e0c8d1dcbc880b61493f966df5254f2f"},
		{worldscene.Camera{X: 4110, Y: 6761, Z: 15, Heading: 0, Pitch: 0, Roll: 0}, 7800, 624, -20000, "4a3dc42b340a71fd97ccb5f2327e98feea47e56493af59a688f5dac1a2995d77"},
		{worldscene.Camera{X: 4110, Y: 6761, Z: 15, Heading: 0, Pitch: 0, Roll: 0}, 8200, 1600, 16000, "ae933f6f2e3c2a540664997092ba1592c93a9efe7642381d487130dae61603ba"},
		{worldscene.Camera{X: 4110, Y: 6761, Z: 15, Heading: 0, Pitch: 0, Roll: 0}, 8500, 1000, 0, "98b3587aa0a215cab6d81352872146f009f9e14e832040ebbf093240ae2cd404"},
		{worldscene.Camera{X: 3600, Y: 7012, Z: 415, Heading: 4721, Pitch: -4766, Roll: 0}, 7005, 624, 0, "f3058c797868b9a3f8d8bfde568e3f2c1bc93f626e45ec33723e2adf0d2f4ccb"},
		{worldscene.Camera{X: 3600, Y: 7012, Z: 415, Heading: 4721, Pitch: -4766, Roll: 0}, 7100, 3000, 8000, "bf8156b648f76d4c7ccbac0a328fe76897e0d588d7b807a5690738155a0154ad"},
		{worldscene.Camera{X: 3600, Y: 7012, Z: 415, Heading: 4721, Pitch: -4766, Roll: 0}, 7200, 4800, -8000, "a32a52fe358be366fd69a059c0b8cdb83d8e5e35fa6cd608ad325bbda7e4842a"},
		{worldscene.Camera{X: 3600, Y: 7012, Z: 415, Heading: 4721, Pitch: -4766, Roll: 0}, 7300, 4800, 18000, "08d773d92c62e3d691b71cbf2841408c651e6f5d4cae571010a73a7ac83489da"},
		{worldscene.Camera{X: 3600, Y: 7012, Z: 415, Heading: 4721, Pitch: -4766, Roll: 0}, 7500, 4800, -18000, "057f2d010beaf50dd2212f6cb97f157aa4ab829fc78440d219ff726428f138ce"},
	}
	g, err := New(Options{Mute: true})
	if err != nil {
		t.Fatal(err)
	}
	for i, c := range cases {
		clear(g.worldRenderer.Pixels)
		clear(g.cars[:])
		position := float64((int(c.progress) + trackLength - 6930) % trackLength)
		g.cars[0].position = position
		g.cars[1] = car{position: position, lateral: float64(c.lateral), yaw: c.yaw, lap: 1, color: 12, secondary: 9}
		g.prepareProjectedCars(c.camera, worldscene.Context{Progress: c.progress, Lateral: c.lateral >> 4}, inCarView)
		g.drawProjectedCarPass(worldscene.AllCars)
		if actual := fmt.Sprintf("%x", sha256.Sum256(g.worldRenderer.Pixels)); actual != c.digest {
			t.Fatalf("original projected car %d: got %s, want %s", i, actual, c.digest)
		}
	}
}

func BenchmarkRecoveredSceneWithCars(b *testing.B) {
	g, err := New(Options{Mute: true, Demo: true})
	if err != nil {
		b.Fatal(err)
	}
	for range 300 {
		g.updateDrive()
	}
	camera, context := g.originalCamera(tvView)
	b.ReportAllocs()
	b.ResetTimer()
	for range b.N {
		g.worldOperations, err = g.world.SceneOperations(g.worldOperations[:0], camera, context)
		if err != nil {
			b.Fatal(err)
		}
		g.prepareProjectedCars(camera, context, tvView)
		if err := g.worldRenderer.RenderWithCars(camera, g.worldOperations, g.paintCars); err != nil {
			b.Fatal(err)
		}
	}
}

// Every supported view uses the same recovered world and live car painter.
func BenchmarkRecoveredSceneAllViews(b *testing.B) {
	g, err := New(Options{Mute: true, Demo: true})
	if err != nil {
		b.Fatal(err)
	}
	for range 300 {
		g.updateDrive()
	}
	for view, name := range demoViews {
		b.Run(name, func(b *testing.B) {
			camera, context := g.originalCamera(view)
			draw := func() {
				g.worldOperations, err = g.world.SceneOperations(g.worldOperations[:0], camera, context)
				if err != nil {
					b.Fatal(err)
				}
				g.prepareProjectedCars(camera, context, view)
				if err := g.worldRenderer.RenderWithCars(camera, g.worldOperations, g.paintCars); err != nil {
					b.Fatal(err)
				}
			}
			draw()
			b.ReportAllocs()
			b.ResetTimer()
			for range b.N {
				draw()
			}
		})
	}
}
