package vehicle

import (
	"bytes"
	"crypto/sha256"
	"encoding/binary"
	"encoding/json"
	"fmt"
	"os"
	"testing"
)

// The numeric fixtures come from complete original $357e-$3d64 calls,
// including all wheel forces, their energy, signed motion and subpixel pose.
// No original executable code or machine state is loaded by this test.
func TestOriginalWheelForcesAndIntegration(t *testing.T) {
	var cases []struct {
		Name   string
		Config DynamicsConfig
		Input  DynamicsInput
		Before DynamicsState
		Digest string
	}
	data, err := os.ReadFile("testdata/dynamics.json")
	if err != nil {
		t.Fatal(err)
	}
	if err := json.Unmarshal(data, &cases); err != nil {
		t.Fatal(err)
	}
	world, _ := suspensionGeometry(t)
	for _, c := range cases {
		output := ApplyWheelForces(c.Before, c.Input, c.Config, world)
		var data bytes.Buffer
		if err := binary.Write(&data, binary.BigEndian, output); err != nil {
			t.Fatal(err)
		}
		if got := fmt.Sprintf("%x", sha256.Sum256(data.Bytes())); got != c.Digest {
			t.Fatalf("%s: got %+v (%s), want %s", c.Name, output, got, c.Digest)
		}
	}
}
