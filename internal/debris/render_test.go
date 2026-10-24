package debris

import (
	"encoding/json"
	"github.com/olivierh59500/indianapolis500/internal/scene"
	"os"
	"testing"
)

func TestOriginalPlayedCrashPainting(t *testing.T) {
	data, err := os.ReadFile("testdata/played-crash.json")
	if err != nil {
		t.Fatal(err)
	}
	var row struct{ Before, After originalPool }
	if err = json.Unmarshal(data, &row); err != nil {
		t.Fatal(err)
	}
	before, err := os.ReadFile("testdata/played-crash-before.bin")
	if err != nil {
		t.Fatal(err)
	}
	after, err := os.ReadFile("testdata/played-crash-after.bin")
	if err != nil {
		t.Fatal(err)
	}
	m, w := originalModel(t)
	row.Before.supply(&m)
	painter := w.NewRenderer()
	if err = painter.Render(row.Before.Camera, nil); err != nil {
		t.Fatal(err)
	}
	copy(painter.Pixels, before)
	m.Paint(painter)
	different, first := 0, -1
	for i, p := range after {
		if painter.Pixels[i] != p {
			different++
			if first < 0 {
				first = i
			}
		}
	}
	if different != 0 {
		t.Fatalf("original crash painter differs by%d pixels, first%d/%d native%d original%d", different, first%320, first/320, painter.Pixels[first], after[first])
	}
	row.After.compare(t, &m, row.Before.Requests)
}

func TestOriginalDynamicVertexProjection(t *testing.T) {
	data, err := os.ReadFile("testdata/played-crash.json")
	if err != nil {
		t.Fatal(err)
	}
	var row struct{ Before originalPool }
	if err = json.Unmarshal(data, &row); err != nil {
		t.Fatal(err)
	}
	var points []struct {
		Point                       scene.Point
		Depth, Horizontal, Vertical int16
	}
	data, err = os.ReadFile("testdata/played-projections.json")
	if err != nil {
		t.Fatal(err)
	}
	if err = json.Unmarshal(data, &points); err != nil {
		t.Fatal(err)
	}
	_, w := originalModel(t)
	matrix := w.Matrix(row.Before.Camera)
	for _, p := range points {
		got := matrix.TransformDirect(p.Point, row.Before.Camera, true)
		if got.Depth != p.Depth || got.Horizontal != p.Horizontal || got.Vertical != p.Vertical {
			t.Fatalf("dynamic projection differs: point%+v native%+v original%+v", p.Point, got, p)
		}
	}
}

func TestOriginalIndependentEffectRasters(t *testing.T) {
	data, err := os.ReadFile("testdata/effect-rasters.json")
	if err != nil {
		t.Fatal(err)
	}
	var rows []struct {
		Kind          string
		Before, After originalPool
	}
	if err = json.Unmarshal(data, &rows); err != nil {
		t.Fatal(err)
	}
	pixels, err := os.ReadFile("testdata/effect-rasters.bin")
	if err != nil {
		t.Fatal(err)
	}
	m, w := originalModel(t)
	p := w.NewRenderer()
	const size = 320 * 124
	for i, row := range rows {
		t.Run(row.Kind, func(t *testing.T) {
			row.Before.supply(&m)
			if err = p.Render(row.Before.Camera, nil); err != nil {
				t.Fatal(err)
			}
			copy(p.Pixels, pixels[i*2*size:(i*2+1)*size])
			m.Paint(p)
			want := pixels[(i*2+1)*size : (i*2+2)*size]
			different, first := 0, -1
			for i, b := range want {
				if b != p.Pixels[i] {
					different++
					if first < 0 {
						first = i
					}
				}
			}
			if different != 0 {
				t.Fatalf("effect pixels differ%d, first%d/%d native%d original%d", different, first%320, first/320, p.Pixels[first], want[first])
			}
			row.After.compare(t, &m, row.Before.Requests)
		})
	}
}

// The final flash can cover incorrect fragments. Verify each primitive on its
// independently captured previous framebuffer as well as the complete result.
func TestOriginalEachPlayedCollisionPrimitive(t *testing.T) {
	data, err := os.ReadFile("testdata/played-crash.json")
	if err != nil {
		t.Fatal(err)
	}
	var row struct{ Before originalPool }
	if err = json.Unmarshal(data, &row); err != nil {
		t.Fatal(err)
	}
	frames, err := os.ReadFile("testdata/played-piece-frames.bin")
	if err != nil {
		t.Fatal(err)
	}
	first, err := os.ReadFile("testdata/played-crash-before.bin")
	if err != nil {
		t.Fatal(err)
	}
	m, w := originalModel(t)
	row.Before.supply(&m)
	r := w.NewRenderer()
	if err = r.Render(row.Before.Camera, nil); err != nil {
		t.Fatal(err)
	}
	const size = 320 * 124
	for i := 0; i < 16; i++ {
		before := first
		if i > 0 {
			before = frames[(i-1)*size : i*size]
		}
		copy(r.Pixels, before)
		e := m.Groups[i/8]
		paintPiece(r, e.Points, e.Pieces[i%8])
		want := frames[i*size : (i+1)*size]
		for p, value := range want {
			if value != r.Pixels[p] {
				t.Fatalf("primitive%d differs at%d/%d", i, p%320, p/320)
			}
		}
	}
}

// Stress the maximum pool with genuine original fragment geometry near the
// camera, where clipping and overdraw cost more than distant particles.
func BenchmarkDenseOriginalCollisionPainting(b *testing.B) {
	data, err := os.ReadFile("testdata/played-crash.json")
	if err != nil {
		b.Fatal(err)
	}
	var row struct{ Before originalPool }
	if err = json.Unmarshal(data, &row); err != nil {
		b.Fatal(err)
	}
	m, w := originalModel(b)
	row.Before.supply(&m)
	heavy := m.Groups[0]
	var pool Pool
	pool.Count = 18
	for i := range pool.Groups {
		pool.Groups[i] = heavy
	}
	r := w.NewRenderer()
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		m.Pool = pool
		r.Render(row.Before.Camera, nil)
		m.Paint(r)
	}
}
