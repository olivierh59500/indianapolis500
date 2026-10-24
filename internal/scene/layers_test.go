package scene

import (
	"crypto/sha256"
	"fmt"
	"strings"
	"testing"
)

// Carry results and the rear-view flag were read from all ten original
// painter predicates in FS-UAE, including every pit boundary on both sides.
func TestOriginalCarPainterPartitions(t *testing.T) {
	cases := []struct {
		position       uint16
		lateral        int16
		selected, rear uint16
	}{
		{0, -100, 0x97, 0x10},
		{0, 4799, 0x97, 0x10},
		{0, 4800, 0xc1, 0x40},
		{0, 4801, 0xc1, 0x40},
		{4109, -100, 0x97, 0x10},
		{4109, 4799, 0x97, 0x10},
		{4109, 4800, 0xc1, 0x40},
		{4109, 4801, 0xc1, 0x40},
		{4110, -100, 0x11d, 0x0},
		{4110, 4799, 0x11d, 0x0},
		{4110, 4800, 0x141, 0x0},
		{4110, 4801, 0x141, 0x0},
		{36979, -100, 0x11d, 0x0},
		{36979, 4799, 0x11d, 0x0},
		{36979, 4800, 0x141, 0x0},
		{36979, 4801, 0x141, 0x0},
		{36980, -100, 0x15, 0x0},
		{36980, 4799, 0x15, 0x0},
		{36980, 4800, 0x41, 0x0},
		{36980, 4801, 0x41, 0x0},
		{36981, -100, 0x235, 0x0},
		{36981, 4799, 0x235, 0x0},
		{36981, 4800, 0x241, 0x0},
		{36981, 4801, 0x241, 0x0},
		{37289, -100, 0x235, 0x0},
		{37289, 4799, 0x235, 0x0},
		{37289, 4800, 0x241, 0x0},
		{37289, 4801, 0x241, 0x0},
		{37290, -100, 0x235, 0x10},
		{37290, 4799, 0x235, 0x10},
		{37290, 4800, 0x241, 0x40},
		{37290, 4801, 0x241, 0x40},
		{39599, -100, 0x235, 0x10},
		{39599, 4799, 0x235, 0x10},
		{39599, 4800, 0x241, 0x40},
		{39599, 4801, 0x241, 0x40},
		{65535, -100, 0x235, 0x10},
		{65535, 4799, 0x235, 0x10},
		{65535, 4800, 0x241, 0x40},
		{65535, 4801, 0x241, 0x40},
	}
	for _, c := range cases {
		var selected, rear uint16
		for filter := AllCars; filter <= AfterExit; filter++ {
			visible, hideRear := filter.SelectCar(c.position, c.lateral)
			if visible {
				selected |= 1 << filter
			}
			if hideRear {
				rear |= 1 << filter
			}
		}
		if selected != c.selected || rear != c.rear {
			t.Fatalf("position %d, lateral %d: selected=%x rear=%x, want %x/%x", c.position, c.lateral, selected, rear, c.selected, c.rear)
		}
	}
}

func TestOriginalDeferredPitAsphalt(t *testing.T) {
	w := recoveredWorld(t)
	camera := Camera{X: 9177, Y: 6775, Z: 15, Heading: 317, Pitch: 215, Roll: -27}
	context := Context{Progress: 9538, Lateral: 53, Detail: 1}
	operations, err := w.SceneOperations(nil, camera, context)
	if err != nil {
		t.Fatal(err)
	}
	r := w.NewRenderer()
	if err := r.Render(camera, operations); err != nil {
		t.Fatal(err)
	}
	// These rows belong to the original completed world pass, after $2145e.
	// The earlier scenery pass alone left the infield green over this asphalt.
	for _, y := range []int{70, 74, 82, 94, 110, 120} {
		for _, x := range []int{0, 32, 64, 96, 128, 160, 192, 224, 256, 288, 319} {
			if pixel := r.Pixels[y*Width+x]; pixel != 8 {
				t.Fatalf("completed pit surface (%d,%d): got %d, want original asphalt 8", x, y, pixel)
			}
		}
	}
}

// These complete painter lists were recorded from the original renderer after
// its normal cache cleanup, with the car list isolated from world geometry.
func TestOriginalCompleteWorldPainterOrder(t *testing.T) {
	cases := []struct {
		camera  Camera
		context Context
		digest  string
	}{
		{Camera{9177, 6775, 15, 317, 215, -27}, Context{Progress: 9538, Lateral: 53, TrackHeading: 0, Detail: 1}, "0502f0d6f691fb6b1f54dd6b7d4e53ab2e0b48aa256d7f543b352b072ae50de5"},
		{Camera{3600, 7012, 415, 4721, -4766, 0}, Context{Progress: 6750, Lateral: 290, TrackHeading: 0, Detail: 1}, "8744a588da1c006243e1bf5a5ba150ebe7b18db401a5202fdce7f35a3103494f"},
		{Camera{4110, 7022, 15, 0, 0, 0}, Context{Progress: 7005, Lateral: 300, TrackHeading: 0, Detail: 1}, "9686d1b666c907db692d740e8f9896f32b87bb8aed45b62607b0c23faaea997b"},
		{Camera{10099, 7017, 16, 944, 0, 0}, Context{Progress: 10000, Lateral: 300, TrackHeading: 944, Detail: 1}, "9cf0183979531e6e9f72a1c50d9451c97ed7b79d09072ac6cb51dec058dba257"},
		{Camera{13465, 5545, 22, 8464, 0, 0}, Context{Progress: 11880, Lateral: 300, TrackHeading: 8464, Detail: 1}, "f78aca1d8212c032fc0b6cac14e2e70e07e2742cd51f558c5168b7b402f7c041"},
		{Camera{14942, -20, 15, 16384, 0, 0}, Context{Progress: 14860, Lateral: 300, TrackHeading: 16384, Detail: 1}, "bbad7d33cffa894a1b2a8a35747968dd8f10c9e005cd560ae3a9942b637402e4"},
		{Camera{13465, -5545, 22, 24848, 0, 0}, Context{Progress: 17820, Lateral: 300, TrackHeading: 24848, Detail: 1}, "d8b104ad20c1e68b14ed0452c274d0fc8bfaa6dae6126da651911fcda5b169aa"},
		{Camera{5500, -7022, 15, -32768, 0, 0}, Context{Progress: 22000, Lateral: 300, TrackHeading: -32768, Detail: 1}, "8879b2b4bd1d20eafb49a9c88c682e7d6f504055a5caf44736799f63cb0b88f0"},
		{Camera{-13762, -5220, 22, -23424, 0, 0}, Context{Progress: 31900, Lateral: 300, TrackHeading: -23424, Detail: 1}, "c38a779d8d45a37391dc86b45f9ad01580e429b991910fbd08d0947868d1c7e9"},
		{Camera{-14942, -180, 15, -16384, 0, 0}, Context{Progress: 34560, Lateral: 300, TrackHeading: -16384, Detail: 1}, "66a07f45def8c8cc2b1f4e9826c1abd806ae80d0dc3b806906542142f1ab7517"},
		{Camera{-13465, 5545, 22, -7920, 0, 0}, Context{Progress: 37620, Lateral: 300, TrackHeading: -7920, Detail: 1}, "75fd196be032ff5c85f07d85d60daff144de62dc6a4adf884c26b0d3ef34fd16"},
	}
	w := recoveredWorld(t)
	for i, c := range cases {
		operations, err := w.SceneOperations(nil, c.camera, c.context)
		if err != nil {
			t.Fatal(err)
		}
		var normalized strings.Builder
		for _, op := range operations {
			if op.Kind == "cars" {
				continue
			}
			ref, primitive, color, count := 0, uint16(0), uint8(0), 0
			if op.Kind == "object" || op.Kind == "segments" {
				ref = op.Reference
			}
			if op.Kind == "object" {
				primitive = op.Primitive
			}
			if op.Kind == "segments" {
				color, count = op.Color, op.Count
			}
			fmt.Fprintf(&normalized, "%s:%x:%x:%x:%d;", op.Kind, ref, primitive, color, count)
		}
		if got := fmt.Sprintf("%x", sha256.Sum256([]byte(normalized.String()))); got != c.digest {
			t.Fatalf("original complete painter %d: got %s, want %s", i, got, c.digest)
		}
	}
}
