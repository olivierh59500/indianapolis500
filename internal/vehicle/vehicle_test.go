package vehicle

import (
	"bytes"
	"crypto/sha256"
	"encoding/binary"
	"encoding/json"
	"fmt"
	gameassets "github.com/olivierh59500/indianapolis500/assets"
	"os"
	"testing"
)

// The model runs continuously from one original physical state. These140
// independently captured complete ticks include acceleration and repeated pit
// barrier impacts; every persistent physical state quantity is compared.
func TestOriginalContinuousVehiclePhysics(t *testing.T) {
	var fixture struct {
		Config Config
		State  State
		Steps  []struct {
			Input  Controls
			Digest string
			Impact uint16
		}
	}
	data, err := os.ReadFile("testdata/vehicle.json")
	if err != nil {
		t.Fatal(err)
	}
	if err := json.Unmarshal(data, &fixture); err != nil {
		t.Fatal(err)
	}
	world, surface := suspensionGeometry(t)
	data, err = gameassets.Original.ReadFile("original/tire-slip-curve.bin")
	if err != nil {
		t.Fatal(err)
	}
	curve, err := NewContactCurve(data)
	if err != nil {
		t.Fatal(err)
	}
	vehicle := Vehicle{Config: fixture.Config, State: fixture.State, World: world, Curve: &curve, Surface: surface}
	for i, c := range fixture.Steps {
		events := vehicle.Step(c.Input)
		var state bytes.Buffer
		if err := binary.Write(&state, binary.BigEndian, vehicle.State); err != nil {
			t.Fatal(err)
		}
		if got := fmt.Sprintf("%x", sha256.Sum256(state.Bytes())); got != c.Digest || events.Impact != c.Impact {
			t.Fatalf("original vehicle tick%d: got%+v (%s), impact%d; want%s, impact%d", i, vehicle.State, got, events.Impact, c.Digest, c.Impact)
		}
	}
}

// Original complete single-tick cases cover all four turns, a higher bank,
// braking, steering and ordinary/damaging pit impacts from independent poses.
func TestOriginalVehicleCornersAndImpacts(t *testing.T) {
	var cases []struct {
		Name   string
		Config Config
		Input  Controls
		Before State
		Digest string
		Impact uint16
	}
	data, err := os.ReadFile("testdata/vehicle-cases.json")
	if err != nil {
		t.Fatal(err)
	}
	if err := json.Unmarshal(data, &cases); err != nil {
		t.Fatal(err)
	}
	world, surface := suspensionGeometry(t)
	data, err = gameassets.Original.ReadFile("original/tire-slip-curve.bin")
	if err != nil {
		t.Fatal(err)
	}
	curve, err := NewContactCurve(data)
	if err != nil {
		t.Fatal(err)
	}
	for _, c := range cases {
		vehicle := Vehicle{Config: c.Config, State: c.Before, World: world, Curve: &curve, Surface: surface}
		events := vehicle.Step(c.Input)
		var state bytes.Buffer
		if err := binary.Write(&state, binary.BigEndian, vehicle.State); err != nil {
			t.Fatal(err)
		}
		if got := fmt.Sprintf("%x", sha256.Sum256(state.Bytes())); got != c.Digest || events.Impact != c.Impact {
			t.Fatalf("%s: state%+v (%s) impact%d, want%s impact%d", c.Name, vehicle.State, got, events.Impact, c.Digest, c.Impact)
		}
	}
}
