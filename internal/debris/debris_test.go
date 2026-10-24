package debris

import (
	"encoding/json"
	"os"
	"reflect"
	"strconv"
	"testing"

	gameassets "github.com/olivierh59500/indianapolis500/assets"
	"github.com/olivierh59500/indianapolis500/internal/scene"
)

type originalPool struct {
	Count    int
	Origin   scene.Point
	Camera   scene.Camera
	Requests [4]uint8
	Groups   []struct {
		Slot int
		Effect
	}
}

func (s originalPool) supply(m *Model) {
	m.Pool = Pool{Count: s.Count, Origin: s.Origin}
	for _, e := range s.Groups {
		m.Groups[e.Slot] = e.Effect
		m.Groups[e.Slot].Active = true
	}
}
func (s originalPool) compare(t *testing.T, m *Model, requests [4]uint8) {
	t.Helper()
	if m.Count != s.Count || m.Origin != s.Origin || requests != s.Requests {
		t.Fatalf("pool summary differs count%d/%d origin%+v/%+v sound%v/%v", m.Count, s.Count, m.Origin, s.Origin, requests, s.Requests)
	}
	var active [18]bool
	for _, want := range s.Groups {
		active[want.Slot] = true
		want.Effect.Active = true
		if !reflect.DeepEqual(m.Groups[want.Slot], want.Effect) {
			t.Fatalf("original slot%d differs: native%+v original%+v", want.Slot, m.Groups[want.Slot], want.Effect)
		}
	}
	for i, e := range m.Groups {
		if e.Active != active[i] {
			t.Fatalf("slot%d activity differs", i)
		}
	}
}

func originalModel(t testing.TB) (Model, *scene.World) {
	t.Helper()
	read := func(n string) []byte {
		data, err := gameassets.Original.ReadFile("original/" + n)
		if err != nil {
			t.Fatal(err)
		}
		return data
	}
	m, err := New(read("debris-templates.json"))
	if err != nil {
		t.Fatal(err)
	}
	w, err := scene.New(read("world-vertices.bin"), read("world-data.bin"), read("world-trigonometry.bin"), read("world-horizon.bin"), read("world-lod.bin"))
	if err != nil {
		t.Fatal(err)
	}
	if err = w.LoadTrack(read("track-cosine.bin"), read("track-bank.bin")); err != nil {
		t.Fatal(err)
	}
	if err = w.LoadCameras(read("camera-arctangent.bin"), read("camera-bank-gradient.bin"), read("camera-trackside.bin")); err != nil {
		t.Fatal(err)
	}
	return m, w
}

func TestOriginalCollisionEffectCreation(t *testing.T) {
	data, err := os.ReadFile("testdata/emission.json")
	if err != nil {
		t.Fatal(err)
	}
	var rows []struct {
		Kind          string
		Input         Input
		Before, After originalPool
	}
	if err = json.Unmarshal(data, &rows); err != nil {
		t.Fatal(err)
	}
	m, w := originalModel(t)
	for _, row := range rows {
		t.Run(row.Kind, func(t *testing.T) {
			row.Before.supply(&m)
			requests := row.Before.Requests
			m.Spawn(row.Input, w, row.Before.Camera, &requests)
			row.After.compare(t, &m, requests)
		})
	}
}

func TestOriginalCollisionEffectMotion(t *testing.T) {
	data, err := os.ReadFile("testdata/lifecycle.json")
	if err != nil {
		t.Fatal(err)
	}
	var rows []struct {
		Severity, Tick int
		State          originalPool
	}
	if err = json.Unmarshal(data, &rows); err != nil {
		t.Fatal(err)
	}
	m, _ := originalModel(t)
	var requests [4]uint8
	for _, row := range rows {
		if row.Tick == 0 {
			row.State.supply(&m)
			requests = row.State.Requests
		} else {
			m.Update(row.State.Camera, &requests)
		}
		t.Run(strconv.Itoa(row.Severity)+"/"+strconv.Itoa(row.Tick), func(t *testing.T) { row.State.compare(t, &m, requests) })
	}
}
