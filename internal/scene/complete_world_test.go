package scene

import (
	"bytes"
	"compress/gzip"
	"encoding/json"
	"os"
	"testing"
)

// Pixels and painter lists are independently observed original output. The
// banked cases come from original camera observations and actual original
// visibility selection, in both Practice and Race with changing crowd phases.
func TestOriginalCompleteWorldRasters(t *testing.T) {
	file, err := os.Open("testdata/complete-world-rasters.json.gz")
	if err != nil {
		t.Fatal(err)
	}
	defer file.Close()
	packed, err := gzip.NewReader(file)
	if err != nil {
		t.Fatal(err)
	}
	defer packed.Close()
	var cases []struct {
		Name         string
		Camera       [6]int16
		Progress     uint16
		Lateral      int16
		TrackHeading int16 `json:"track_heading"`
		Mode, Phase  uint8
		Detail       int
		Pixels       []byte
		Operations   []struct {
			Kind      string
			Reference int `json:"data_ref"`
			Primitive uint16
			D0, D2    uint16
		}
	}
	if err := json.NewDecoder(packed).Decode(&cases); err != nil {
		t.Fatal(err)
	}
	if len(cases) != 27 {
		t.Fatalf("complete original world cases%d, want27", len(cases))
	}
	w := recoveredWorld(t)
	r := w.NewRenderer()
	for _, original := range cases {
		t.Run(original.Name, func(t *testing.T) {
			if len(original.Pixels) != Width*Height {
				t.Fatal("invalid original world framebuffer")
			}
			v := original.Camera
			camera := Camera{v[0], v[1], v[2], v[3], v[4], v[5]}
			ops, err := w.SceneOperations(nil, camera, Context{Progress: original.Progress, Lateral: original.Lateral, TrackHeading: original.TrackHeading, Detail: original.Detail, Mode: original.Mode})
			if err != nil {
				t.Fatal(err)
			}
			index := 0
			for _, op := range ops {
				if op.Kind == "cars" {
					continue
				}
				if index >= len(original.Operations) {
					t.Fatal("extra native world operation")
				}
				want := original.Operations[index]
				if op.Kind != want.Kind || op.Kind == "object" && (op.Reference != want.Reference || op.Primitive != want.Primitive) || op.Kind == "segments" && (op.Reference != want.Reference || op.Color != uint8(want.D0)&15 || op.Count != int(want.D2)+1) {
					t.Fatalf("operation%d native%+v original%+v", index, op, want)
				}
				index++
			}
			if index != len(original.Operations) {
				t.Fatalf("native world operations%d original%d", index, len(original.Operations))
			}
			r.CrowdPhase = original.Phase
			if err := r.Render(camera, ops); err != nil {
				t.Fatal(err)
			}
			if !bytes.Equal(r.Pixels, original.Pixels) {
				different, first := 0, -1
				for i, want := range original.Pixels {
					if r.Pixels[i] != want {
						different++
						if first < 0 {
							first = i
						}
					}
				}
				t.Fatalf("%d original world pixels differ, first%d/%d native%d original%d", different, first%Width, first/Width, r.Pixels[first], original.Pixels[first])
			}
		})
	}
}
