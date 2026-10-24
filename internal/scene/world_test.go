package scene

import (
	"crypto/sha256"
	"encoding/binary"
	"fmt"
	"testing"

	gameassets "github.com/olivierh59500/indianapolis500/assets"
)

func recoveredWorld(t testing.TB) *World {
	t.Helper()
	read := func(name string) []byte {
		data, err := gameassets.Original.ReadFile("original/" + name)
		if err != nil {
			t.Fatal(err)
		}
		return data
	}
	w, err := New(read("world-vertices.bin"), read("world-data.bin"), read("world-trigonometry.bin"), read("world-horizon.bin"), read("world-lod.bin"))
	if err != nil {
		t.Fatal(err)
	}
	if err := w.LoadTrack(read("track-cosine.bin"), read("track-bank.bin")); err != nil {
		t.Fatal(err)
	}
	if err := w.LoadCameras(read("camera-arctangent.bin"), read("camera-bank-gradient.bin"), read("camera-trackside.bin")); err != nil {
		t.Fatal(err)
	}
	if err := w.LoadTrackLimits(read("track-wall-inner.bin"), read("track-wall-outer.bin")); err != nil {
		t.Fatal(err)
	}
	if err := w.LoadBarriers(read("track-barriers.bin"), read("collision-damage.bin")); err != nil {
		t.Fatal(err)
	}
	if err := w.LoadCrowdPatterns(read("crowd-patterns.bin")); err != nil {
		t.Fatal(err)
	}
	return w
}

func TestOriginalCameraMatrix(t *testing.T) {
	w := recoveredWorld(t)
	tests := []struct {
		camera Camera
		matrix Matrix
	}{
		{Camera{X: 4110, Y: 6761, Z: 15}, Matrix{Depth: [3]int16{16384, 0, 0}, Horizontal: [3]int16{0, 16384, 0}, Vertical: [3]int16{0, 0, 17617}}},
		{Camera{X: 4324, Y: 6761, Z: 15, Heading: -4, Pitch: 236}, Matrix{Depth: [3]int16{16379, 0, 402}, Horizontal: [3]int16{0, 16384, 0}, Vertical: [3]int16{-433, 0, 17611}}},
	}
	for _, test := range tests {
		// Values are the original coefficient words captured just after $b740
		// in FS-UAE Practice, before drawing either stationary/driving frame.
		if actual := w.Matrix(test.camera); actual != test.matrix {
			t.Fatalf("camera %+v: matrix %+v, want %+v", test.camera, actual, test.matrix)
		}
	}
}

func TestRecoveredWorldVertices(t *testing.T) {
	w := recoveredWorld(t)
	if len(w.Points) != 2282 || w.Points[0] != (Point{9200, 6855, 0}) || w.Points[3] != (Point{-9900, 7322, 0}) {
		t.Fatal("the original signed fifteen-bit scene coordinates were not recovered")
	}
}

func TestOriginalProjectedVerticesDuringDriving(t *testing.T) {
	w := recoveredWorld(t)
	camera := Camera{X: 4361, Y: 6761, Z: 15, Heading: -6, Pitch: 238}
	matrix := w.Matrix(camera)
	// These records were read from the original projected-vertex cache at
	// the end of the driven Practice frame, before the next physics update.
	tests := []struct {
		reference uint16
		point     Projected
	}{
		{0x80a2, Projected{Depth: 9747, Horizontal: -633, Vertical: -181, X: 150, Y: 63}},
		{0x80aa, Projected{Depth: 9961, Horizontal: -421, Vertical: -1, X: 154, Y: 62}},
		{0x8092, Projected{Depth: 8099, Horizontal: 823, Vertical: 98, X: 176, Y: 62}},
		{0x808a, Projected{Depth: 7981, Horizontal: 541, Vertical: -96, X: 170, Y: 62}},
		{0x809a, Projected{Depth: 9592, Horizontal: -786, Vertical: -343, X: 147, Y: 64}},
		{0x8072, Projected{Depth: 5899, Horizontal: 1011, Vertical: -102, X: 187, Y: 63}},
	}
	for _, test := range tests {
		point, ok := w.point(test.reference)
		if !ok {
			t.Fatalf("missing original vertex %x", test.reference)
		}
		if actual := matrix.Transform(point, camera); actual != test.point {
			t.Fatalf("vertex %x %+v: got %+v, want %+v", test.reference, point, actual, test.point)
		}
	}
}

func TestOriginalVisibleOperationSequence(t *testing.T) {
	w := recoveredWorld(t)
	operations, err := w.Operations(nil, Camera{X: 4444, Y: 6761, Z: 15, Heading: -10, Pitch: 282}, Context{Progress: 7172, Lateral: 39, TrackHeading: 0, Detail: 1})
	if err != nil {
		t.Fatal(err)
	}
	var encoded []byte
	for _, op := range operations {
		encoded = append(encoded, op.Kind[0])
		switch op.Kind {
		case "object":
			encoded = binary.BigEndian.AppendUint16(encoded, uint16(op.Reference))
			encoded = binary.BigEndian.AppendUint16(encoded, op.Primitive)
		case "segments":
			encoded = binary.BigEndian.AppendUint16(encoded, uint16(op.Reference))
			encoded = append(encoded, op.Color)
			encoded = binary.BigEndian.AppendUint16(encoded, uint16(op.Count))
		}
	}
	// Hash of the 46 actual operation entries recorded while FS-UAE rendered
	// this Practice frame, independent of the native selection implementation.
	if len(operations) != 46 || fmt.Sprintf("%x", sha256.Sum256(encoded)) != "7ac3763829a0f8eb0fff8dca08b0d9c858ef5bf9e6180c08818a144a3e56b6a8" {
		t.Fatal("native visibility/order differs from the played original frame")
	}
}

func TestOriginalContourScanlines(t *testing.T) {
	tests := []struct {
		a, b   pixelPoint
		digest string
	}{
		{pixelPoint{10, 10}, pixelPoint{300, 100}, "f6b350dda6e0ea0f7ac43630c95820745069a7438280396000c567faa917fd7f"},
		{pixelPoint{10, 10}, pixelPoint{300, 11}, "df8f17a1e4e9717d6421adc1b57028403ff3c8ec7d73bea9191feb6c1e1bc423"},
		{pixelPoint{300, 10}, pixelPoint{10, 100}, "0286914c6b0deb639f97a22b47f34baf4c46962cebc27d6cecf1eb67a5b9c09e"},
		{pixelPoint{10, 10}, pixelPoint{50, 100}, "fb873562e73891753d03e053813bcf92a500ac66dbcb817c1aff5315a1ece9f9"},
		{pixelPoint{100, 10}, pixelPoint{10, 100}, "d132e1557640efb1b7e0ed6d6070725dcd43d63be8f0b30fc732842ad6df3fc8"},
		{pixelPoint{10, 10}, pixelPoint{18, 13}, "209a0d69614feb5433997349e50c55af2d3c7197a7f9cd24830f882cf221eaf3"},
		{pixelPoint{18, 10}, pixelPoint{10, 13}, "e255db29a465d4dbd6429ab3b29586023379cde3c11a96393e7679cc4752ef8e"},
		{pixelPoint{10, 10}, pixelPoint{10, 100}, "2b71ef35614c7a7f2ee3ce88c96b4813bd83a9fca3315f0c25df7715c24a1558"},
	}
	renderer := &Renderer{}
	for _, test := range tests {
		var values [Height]int
		for i := range values {
			values[i] = -999
		}
		renderer.contour(test.a, test.b, values[:])
		var encoded []byte
		for _, value := range values {
			encoded = binary.BigEndian.AppendUint16(encoded, uint16(int16(value)))
		}
		if got := fmt.Sprintf("%x", sha256.Sum256(encoded)); got != test.digest {
			t.Fatalf("original contour %+v -> %+v mismatch: %s", test.a, test.b, got)
		}
	}
}

func TestAllOriginalVisibilityTables(t *testing.T) {
	w := recoveredWorld(t)
	for detail := 0; detail < 3; detail++ {
		for lateral := int16(50); lateral <= 400; lateral += 350 {
			for progress := 0; progress < 39600; progress += 330 {
				for view := 0; view < 3; view++ {
					camera := Camera{X: 4444, Y: 6761, Z: 15}
					if view == 1 {
						camera.Heading = -32768
					}
					if view == 2 {
						camera.Z = 4000
					}
					if _, err := w.Operations(nil, camera, Context{Progress: uint16(progress), Lateral: lateral, Detail: detail}); err != nil {
						t.Fatalf("detail%d lateral%d progress%d view%d: %v", detail, lateral, progress, view, err)
					}
				}
			}
		}
	}
}

func TestAllOriginalSceneOperationsRender(t *testing.T) {
	w := recoveredWorld(t)
	renderer := w.NewRenderer()
	operations := make([]Operation, 0, 256)
	for detail := 0; detail < 3; detail++ {
		for lateral := int16(50); lateral <= 400; lateral += 350 {
			for progress := 0; progress < 39600; progress += 330 {
				for view := 0; view < 4; view++ {
					camera := Camera{X: 4444, Y: 6761, Z: 15}
					if view == 1 {
						camera.Heading = -32768
					}
					if view == 2 {
						camera.Z = 4000
					}
					if view == 3 {
						camera.Pitch = 8000
					}
					var err error
					operations, err = w.Operations(operations[:0], camera, Context{Progress: uint16(progress), Lateral: lateral, Detail: detail})
					if err == nil {
						err = renderer.Render(camera, operations)
					}
					if err != nil {
						t.Fatalf("detail%d lateral%d progress%d view%d: %v", detail, lateral, progress, view, err)
					}
				}
			}
		}
	}
}

func TestOriginalTrackPlacement(t *testing.T) {
	w := recoveredWorld(t)
	// These thirty poses were captured from the original $291c6 routine,
	// covering both straights, all four turns and three lateral positions.
	tests := []struct {
		progress uint16
		lateral  int16
		point    Point
		heading  int16
	}{
		{7005, 624, Point{4110, 6761, 15}, 0},
		{7005, 4800, Point{4110, 7022, 15}, 0},
		{7005, 8000, Point{4110, 7222, 15}, 0},
		{7172, 624, Point{4444, 6761, 15}, 0},
		{7172, 4800, Point{4444, 7022, 15}, 0},
		{7172, 8000, Point{4444, 7222, 15}, 0},
		{10000, 624, Point{10089, 6757, 15}, 944},
		{10000, 4800, Point{10099, 7017, 16}, 944},
		{10000, 8000, Point{10107, 7217, 20}, 944},
		{11880, 624, Point{13280, 5360, 15}, 8464},
		{11880, 4800, Point{13465, 5545, 22}, 8464},
		{11880, 8000, Point{13606, 5686, 54}, 8464},
		{14860, 624, Point{14681, -20, 15}, 16384},
		{14860, 4800, Point{14942, -20, 15}, 16384},
		{14860, 8000, Point{15142, -20, 15}, 16384},
		{17820, 624, Point{13280, -5360, 15}, 24848},
		{17820, 4800, Point{13465, -5545, 22}, 24848},
		{17820, 8000, Point{13606, -5686, 54}, 24848},
		{22000, 624, Point{5500, -6761, 15}, -32768},
		{22000, 4800, Point{5500, -7022, 15}, -32768},
		{22000, 8000, Point{5500, -7222, 15}, -32768},
		{31900, 624, Point{-13562, -5053, 15}, -23424},
		{31900, 4800, Point{-13762, -5220, 22}, -23424},
		{31900, 8000, Point{-13915, -5349, 54}, -23424},
		{34560, 624, Point{-14681, -180, 15}, -16384},
		{34560, 4800, Point{-14942, -180, 15}, -16384},
		{34560, 8000, Point{-15142, -180, 15}, -16384},
		{37620, 624, Point{-13280, 5360, 15}, -7920},
		{37620, 4800, Point{-13465, 5545, 22}, -7920},
		{37620, 8000, Point{-13606, 5686, 54}, -7920},
	}
	for _, test := range tests {
		got := w.TrackPlacement(test.progress, test.lateral)
		if got.Point != test.point || got.Heading != test.heading {
			t.Fatalf("progress %d lateral %d: got %+v, want %+v heading %d", test.progress, test.lateral, got, test.point, test.heading)
		}
	}
}

func TestOriginalReplayCameraMath(t *testing.T) {
	w := recoveredWorld(t)
	// Independent calls of the original $28ee4 routine, with all reference
	// registers and temporary data restored afterward. Modes cover an in-car
	// yaw, a banked look-at camera and an elevated look-at camera.
	tests := []struct {
		progress uint16
		lateral  int16
		mode     int
		target   uint16
		camera   Camera
	}{
		{7005, 624, 0, 7255, Camera{4110, 6761, 15, 512, 0, 0}},
		{7005, 624, 1, 7255, Camera{4110, 6761, 15, 0, 0, 0}},
		{7005, 624, 2, 7255, Camera{4110, 6761, 415, 0, -5279, 0}},
		{7005, 4800, 0, 7255, Camera{4110, 7022, 15, 512, 0, 0}},
		{7005, 4800, 1, 7255, Camera{4110, 7022, 15, 0, 0, 0}},
		{7005, 4800, 2, 7255, Camera{4110, 7022, 415, 0, -5279, 0}},
		{7005, 8000, 0, 7255, Camera{4110, 7222, 15, 512, 0, 0}},
		{7005, 8000, 1, 7255, Camera{4110, 7222, 15, 0, 0, 0}},
		{7005, 8000, 2, 7255, Camera{4110, 7222, 415, 0, -5279, 0}},
		{10000, 624, 0, 10250, Camera{10089, 6757, 15, 1456, 0, 0}},
		{10000, 624, 1, 10250, Camera{10089, 6757, 15, 924, 0, 0}},
		{10000, 624, 2, 10250, Camera{10089, 6757, 415, 924, -5490, 0}},
		{10000, 4800, 0, 10250, Camera{10099, 7017, 16, 1456, -12, 236}},
		{10000, 4800, 1, 10250, Camera{10099, 7017, 16, 914, 0, 237}},
		{10000, 4800, 2, 10250, Camera{10099, 7017, 416, 914, -5267, 0}},
		{10000, 8000, 0, 10250, Camera{10107, 7217, 20, 1456, 137, 243}},
		{10000, 8000, 1, 10250, Camera{10107, 7217, 20, 919, 150, 237}},
		{10000, 8000, 2, 10250, Camera{10107, 7217, 420, 919, -4995, 0}},
		{11880, 624, 0, 12130, Camera{13280, 5360, 15, 8976, 0, 0}},
		{11880, 624, 1, 12130, Camera{13280, 5360, 15, 8707, 0, 0}},
		{11880, 624, 2, 12130, Camera{13280, 5360, 415, 8707, -5490, 0}},
		{11880, 4800, 0, 12130, Camera{13465, 5545, 22, 8976, -82, 1661}},
		{11880, 4800, 1, 12130, Camera{13465, 5545, 22, 8724, -41, 1663}},
		{11880, 4800, 2, 12130, Camera{13465, 5545, 422, 8724, -5285, 0}},
		{11880, 8000, 0, 12130, Camera{13606, 5686, 54, 8976, -82, 1661}},
		{11880, 8000, 1, 12130, Camera{13606, 5686, 54, 8705, -41, 1663}},
		{11880, 8000, 2, 12130, Camera{13606, 5686, 454, 8705, -5136, 0}},
		{14860, 624, 0, 15110, Camera{14681, -20, 15, 16896, 0, 0}},
		{14860, 624, 1, 15110, Camera{14681, -20, 15, 16384, 0, 0}},
		{14860, 624, 2, 15110, Camera{14681, -20, 415, 16384, -5279, 0}},
		{14860, 4800, 0, 15110, Camera{14942, -20, 15, 16896, 0, 0}},
		{14860, 4800, 1, 15110, Camera{14942, -20, 15, 16384, 0, 0}},
		{14860, 4800, 2, 15110, Camera{14942, -20, 415, 16384, -5279, 0}},
		{14860, 8000, 0, 15110, Camera{15142, -20, 15, 16896, 0, 0}},
		{14860, 8000, 1, 15110, Camera{15142, -20, 15, 16384, 0, 0}},
		{14860, 8000, 2, 15110, Camera{15142, -20, 415, 16384, -5279, 0}},
		{17820, 624, 0, 18070, Camera{13280, -5360, 15, 25360, 0, 0}},
		{17820, 624, 1, 18070, Camera{13280, -5360, 15, 25091, 0, 0}},
		{17820, 624, 2, 18070, Camera{13280, -5360, 415, 25091, -5490, 0}},
		{17820, 4800, 0, 18070, Camera{13465, -5545, 22, 25360, -82, 1661}},
		{17820, 4800, 1, 18070, Camera{13465, -5545, 22, 25108, -41, 1663}},
		{17820, 4800, 2, 18070, Camera{13465, -5545, 422, 25108, -5285, 0}},
		{17820, 8000, 0, 18070, Camera{13606, -5686, 54, 25360, -82, 1661}},
		{17820, 8000, 1, 18070, Camera{13606, -5686, 54, 25089, -41, 1663}},
		{17820, 8000, 2, 18070, Camera{13606, -5686, 454, 25089, -5136, 0}},
		{22000, 624, 0, 22250, Camera{5500, -6761, 15, -32256, 0, 0}},
		{22000, 624, 1, 22250, Camera{5500, -6761, 15, -32768, 0, 0}},
		{22000, 624, 2, 22250, Camera{5500, -6761, 415, -32768, -5279, 0}},
		{22000, 4800, 0, 22250, Camera{5500, -7022, 15, -32256, 0, 0}},
		{22000, 4800, 1, 22250, Camera{5500, -7022, 15, -32768, 0, 0}},
		{22000, 4800, 2, 22250, Camera{5500, -7022, 415, -32768, -5279, 0}},
		{22000, 8000, 0, 22250, Camera{5500, -7222, 15, -32256, 0, 0}},
		{22000, 8000, 1, 22250, Camera{5500, -7222, 15, -32768, 0, 0}},
		{22000, 8000, 2, 22250, Camera{5500, -7222, 415, -32768, -5279, 0}},
		{31900, 624, 0, 32150, Camera{-13562, -5053, 15, -22912, 0, 0}},
		{31900, 624, 1, 32150, Camera{-13562, -5053, 15, -23138, 0, 0}},
		{31900, 624, 2, 32150, Camera{-13562, -5053, 415, -23138, -5483, 0}},
		{31900, 4800, 0, 32150, Camera{-13762, -5220, 22, -22912, -82, 1661}},
		{31900, 4800, 1, 32150, Camera{-13762, -5220, 22, -23154, -41, 1663}},
		{31900, 4800, 2, 32150, Camera{-13762, -5220, 422, -23154, -5285, 0}},
		{31900, 8000, 0, 32150, Camera{-13915, -5349, 54, -22912, -82, 1661}},
		{31900, 8000, 1, 32150, Camera{-13915, -5349, 54, -23151, -41, 1663}},
		{31900, 8000, 2, 32150, Camera{-13915, -5349, 454, -23151, -5136, 0}},
		{37620, 624, 0, 37870, Camera{-13280, 5360, 15, -7408, 0, 0}},
		{37620, 624, 1, 37870, Camera{-13280, 5360, 15, -7677, 0, 0}},
		{37620, 624, 2, 37870, Camera{-13280, 5360, 415, -7677, -5490, 0}},
		{37620, 4800, 0, 37870, Camera{-13465, 5545, 22, -7408, -82, 1661}},
		{37620, 4800, 1, 37870, Camera{-13465, 5545, 22, -7660, -41, 1663}},
		{37620, 4800, 2, 37870, Camera{-13465, 5545, 422, -7660, -5285, 0}},
		{37620, 8000, 0, 37870, Camera{-13606, 5686, 54, -7408, -82, 1661}},
		{37620, 8000, 1, 37870, Camera{-13606, 5686, 54, -7679, -41, 1663}},
		{37620, 8000, 2, 37870, Camera{-13606, 5686, 454, -7679, -5136, 0}},
	}
	for _, test := range tests {
		got, _ := w.trackCamera(test.progress, test.lateral, 512, test.target, test.lateral, test.mode != 0, test.mode == 2, test.mode != 2)
		if got != test.camera {
			t.Fatalf("progress%d lateral%d mode%d got%+v want%+v", test.progress, test.lateral, test.mode, got, test.camera)
		}
	}
}

func BenchmarkOriginalWorldFrame(b *testing.B) {
	w := recoveredWorld(b)
	r := w.NewRenderer()
	camera, context := w.CameraForCar(11880, 8000, 0, InCar)
	context.Detail = 2
	operations := make([]Operation, 0, 256)
	var err error
	operations, err = w.Operations(operations, camera, context)
	if err != nil {
		b.Fatal(err)
	}
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		if err := r.Render(camera, operations); err != nil {
			b.Fatal(err)
		}
	}
}

func TestRecoveredCameraViewsAroundTheOval(t *testing.T) {
	w := recoveredWorld(t)
	renderer := w.NewRenderer()
	operations := make([]Operation, 0, 256)
	for progress := 0; progress < 39600; progress += 330 {
		for view := InCar; view <= Crash; view++ {
			for _, lateral := range []int16{624, 8000} {
				camera, context := w.CameraForCar(uint16(progress), lateral, 0, view)
				for detail := 0; detail < 3; detail++ {
					context.Detail = detail
					var err error
					operations, err = w.Operations(operations[:0], camera, context)
					if err == nil {
						err = renderer.Render(camera, operations)
					}
					if err != nil {
						t.Fatalf("source view%d progress%d lateral%d detail%d: %v", view, progress, lateral, detail, err)
					}
				}
			}
		}
	}
}

func TestOriginalPlayedReplayViewSelection(t *testing.T) {
	w := recoveredWorld(t)
	// These views were selected through Space in the actual Instant Replay
	// after a driven Practice run, with the replay following its first pose.
	tests := []struct {
		view     View
		camera   Camera
		progress uint16
		lateral  int16
	}{
		{InCar, Camera{4110, 6761, 15, 0, 0, 0}, 7005, 39},
		{Behind, Camera{3970, 6956, 15, 9891, 0, 0}, 6935, 234},
		{Trackside, Camera{3600, 7012, 15, 4767, 0, 0}, 6750, 290},
		{Television, Camera{3600, 7012, 415, 4767, -4800, 0}, 6750, 290},
	}
	for _, test := range tests {
		got, context := w.CameraForCar(7005, 624, 0, test.view)
		if got != test.camera || context.Progress != test.progress || context.Lateral != test.lateral {
			t.Fatalf("played view%d got%+v %+v, want%+v at%d/%d", test.view, got, context, test.camera, test.progress, test.lateral)
		}
	}
}
