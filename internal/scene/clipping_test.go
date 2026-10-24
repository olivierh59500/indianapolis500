package scene

import (
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"os"
	"testing"
)

// Each hash is a complete indexed framebuffer from an isolated original
// $1557c polygon draw in FS-UAE. These fixtures exercise the eye-plane tie,
// both sides of the wall, its endpoints, camera height and a small yaw.
func TestOriginalEyePlanePolygonClipping(t *testing.T) {
	cases := []struct {
		camera Camera
		digest string
	}{
		{Camera{4110, 7022, -1, 0, 0, 0}, "2eef5a416ba143d00492a887cdd2adfca9ad82f7c29b979e515589c47f27dd9e"},
		{Camera{4110, 7022, 0, 0, 0, 0}, "2d3fb696dbc958403b3b5f8dafc209a08503477e07abeb8ac3ef1c6588f2cf54"},
		{Camera{4110, 7022, 1, 0, 0, 0}, "983db52f7c1b43888b932cd9b4f95d1256399b45fe267bfc1217bb978c7c6ee1"},
		{Camera{4110, 7022, 8, 0, 0, 0}, "983db52f7c1b43888b932cd9b4f95d1256399b45fe267bfc1217bb978c7c6ee1"},
		{Camera{4110, 7022, 14, 0, 0, 0}, "983db52f7c1b43888b932cd9b4f95d1256399b45fe267bfc1217bb978c7c6ee1"},
		{Camera{4110, 7022, 15, 0, 0, 0}, "a05c272c8eb1459a7b37f4d762f51bf9e9a8738b368d9894b8b0413f29a18fbc"},
		{Camera{4110, 7022, 16, 0, 0, 0}, "3c12d137b8a6e722f372847260c56a3ddf37ebe0a6dc004fa175dfbcf38f56e2"},
		{Camera{4110, 7022, 30, 0, 0, 0}, "3c12d137b8a6e722f372847260c56a3ddf37ebe0a6dc004fa175dfbcf38f56e2"},
		{Camera{-4300, 7022, 15, 0, 0, 0}, "3f49a8731178083f1b3f604f08328958e65448e15cd6c368b20cf5a678fd7fb8"},
		{Camera{-4260, 7022, 15, 0, 0, 0}, "a05c272c8eb1459a7b37f4d762f51bf9e9a8738b368d9894b8b0413f29a18fbc"},
		{Camera{0, 7022, 15, 0, 0, 0}, "a05c272c8eb1459a7b37f4d762f51bf9e9a8738b368d9894b8b0413f29a18fbc"},
		{Camera{8580, 7022, 15, 0, 0, 0}, "135f71e566897688b91945b16debbb4dc218bd0ad00f5fa6e70f19c26b6901d2"},
		{Camera{8600, 7022, 15, 0, 0, 0}, "135f71e566897688b91945b16debbb4dc218bd0ad00f5fa6e70f19c26b6901d2"},
		{Camera{4110, 7021, 15, 0, 0, 0}, "19fa86edbc460bd2c88cd6e795bf5372cdf6815e52226450ba1cdb861ad2dbe6"},
		{Camera{4110, 7023, 15, 0, 0, 0}, "28217b25bc75103137bd9f6c67653110ef531bd113cf337b220a0129dcf902a2"},
		{Camera{4110, 7022, 15, -32768, 0, 0}, "a05c272c8eb1459a7b37f4d762f51bf9e9a8738b368d9894b8b0413f29a18fbc"},
		{Camera{4110, 7022, 15, 16, 0, 0}, "a05c272c8eb1459a7b37f4d762f51bf9e9a8738b368d9894b8b0413f29a18fbc"},
		{Camera{4110, 7022, 15, 31, 0, 0}, "a05c272c8eb1459a7b37f4d762f51bf9e9a8738b368d9894b8b0413f29a18fbc"},
		{Camera{4110, 7022, 15, 32, 0, 0}, "28217b25bc75103137bd9f6c67653110ef531bd113cf337b220a0129dcf902a2"},
		{Camera{4110, 7022, 15, -32, 0, 0}, "a05c272c8eb1459a7b37f4d762f51bf9e9a8738b368d9894b8b0413f29a18fbc"},
	}
	w := recoveredWorld(t)
	r := w.NewRenderer()
	for i, c := range cases {
		clear(r.Pixels)
		r.camera, r.matrix = c.camera, w.Matrix(c.camera)
		for j, p := range w.Points {
			r.projected[j] = r.matrix.Transform(p, c.camera)
		}
		if err := r.groups(15328, true, 0); err != nil {
			t.Fatal(err)
		}
		if actual := fmt.Sprintf("%x", sha256.Sum256(r.Pixels)); actual != c.digest {
			t.Fatalf("original eye-plane polygon %d: got %s, want %s", i, actual, c.digest)
		}
	}
}

// The original viewport is asymmetric: its rightmost visible X is319,
// while the projection center is160. These independent original wall
// framebuffers exercise that border across pitches, heights and endpoints.
func TestOriginalClippedOuterWall(t *testing.T) {
	data, err := os.ReadFile("testdata/walls.json")
	if err != nil {
		t.Fatal(err)
	}
	var cases []struct {
		Camera [6]int16
		SHA256 string
	}
	if err = json.Unmarshal(data, &cases); err != nil {
		t.Fatal(err)
	}
	w := recoveredWorld(t)
	r := w.NewRenderer()
	for i, c := range cases {
		camera := Camera{c.Camera[0], c.Camera[1], c.Camera[2], c.Camera[3], c.Camera[4], c.Camera[5]}
		clear(r.Pixels)
		r.camera, r.matrix = camera, w.Matrix(camera)
		for j, p := range w.Points {
			r.projected[j] = r.matrix.Transform(p, camera)
		}
		if err = r.groups(15352, true, 0); err != nil {
			t.Fatal(err)
		}
		if actual := fmt.Sprintf("%x", sha256.Sum256(r.Pixels)); actual != c.SHA256 {
			t.Errorf("original clipped wall%d camera%+v native%s, original%s", i, camera, actual, c.SHA256)
		}
	}
}
