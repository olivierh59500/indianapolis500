package sound

import (
	"encoding/json"
	"math"
	"os"
	"reflect"
	"testing"
)

type originalMixerState struct {
	Controller
	Random uint16
}

type originalVoice struct {
	Sample                string
	Length, Period, Level int
}

func TestOriginalAudioController(t *testing.T) {
	data, err := os.ReadFile("testdata/mixer.json")
	if err != nil {
		t.Fatal(err)
	}
	var rows []struct {
		Kind                      string
		Before, After             originalMixerState
		VoicesBefore, VoicesAfter [4]originalVoice
	}
	if err = json.Unmarshal(data, &rows); err != nil {
		t.Fatal(err)
	}
	samples := map[string]Sample{"engine": EngineSample, "noise": NoiseSample, "collision": CollisionSample, "failure": FailureSample}
	for _, row := range rows {
		t.Run(row.Kind, func(t *testing.T) {
			c := row.Before.Controller
			for i, v := range row.VoicesBefore {
				c.Voices[i] = Voice{Sample: samples[v.Sample], Rate: originalPlaybackClock / float64(v.Period), Level: uint8(v.Level)}
			}
			random := row.Before.Random
			c.Step(func() uint16 { random = random*0x6255 + 0x3619; return random })
			for i, want := range row.VoicesAfter {
				got := c.Voices[i]
				if got.Sample != samples[want.Sample] || got.Level != uint8(want.Level) || math.Abs(got.Rate-originalPlaybackClock/float64(want.Period)) > 1e-9 {
					t.Errorf("voice%d differs: got%+v original%+v", i, got, want)
				}
			}
			c.Voices = [4]Voice{}
			if !reflect.DeepEqual(c, row.After.Controller) || random != row.After.Random {
				t.Fatalf("controller differs: got%+v rng%04x original%+v rng%04x", c, random, row.After.Controller, row.After.Random)
			}
		})
	}
}
