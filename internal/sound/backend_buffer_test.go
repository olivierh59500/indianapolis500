//go:build audiocheck

package sound

import (
	"encoding/json"
	"fmt"
	"os"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/audio"
)

// This optional integration check exercises the actual desktop audio backend.
// Both readers return only zero PCM, so no sound is played during the test.
type silentRead struct {
	Bytes       int
	Seconds     float64
	ControlTick int64
}

type silentReadProbe struct {
	mu    sync.Mutex
	reads []silentRead
	start time.Time
	tick  *atomic.Int64
}

func (p *silentReadProbe) Read(b []byte) (int, error) {
	clear(b)
	p.mu.Lock()
	p.reads = append(p.reads, silentRead{len(b), time.Since(p.start).Seconds(), p.tick.Load()})
	p.mu.Unlock()
	return len(b), nil
}

func (p *silentReadProbe) result() []silentRead {
	p.mu.Lock()
	defer p.mu.Unlock()
	return append([]silentRead(nil), p.reads...)
}

var bufferProbeRequests = make(chan bool)
var bufferProbeResults = make(chan map[string][]silentRead)
var bufferProbeFinished = make(chan int, 1)

type bufferProbeRunner struct {
	players  []*audio.Player
	readers  []*silentReadProbe
	started  time.Time
	tick     atomic.Int64
	exitCode int
}

func (r *bufferProbeRunner) Update() error {
	r.tick.Add(1)
	select {
	case r.exitCode = <-bufferProbeFinished:
		return ebiten.Termination
	default:
	}
	select {
	case <-bufferProbeRequests:
		context := audio.NewContext(SampleRate)
		r.started = time.Now()
		for i := 0; i < 2; i++ {
			source := &silentReadProbe{start: r.started, tick: &r.tick}
			var player *audio.Player
			var err error
			if i == 0 {
				player, err = context.NewPlayer(source)
			} else {
				player, err = newPCMPlayer(context, source)
			}
			if err != nil {
				return err
			}
			player.SetVolume(0)
			if i == 0 {
				player.Play()
			}
			r.players = append(r.players, player)
			r.readers = append(r.readers, source)
		}
	default:
	}
	if len(r.players) != 0 && time.Since(r.started) > 2500*time.Millisecond {
		for _, player := range r.players {
			if err := player.Close(); err != nil {
				return err
			}
		}
		bufferProbeResults <- map[string][]silentRead{"backendDefault": r.readers[0].result(), "gamePlayer": r.readers[1].result()}
		r.players = nil
	}
	return nil
}

func (*bufferProbeRunner) Draw(*ebiten.Image)         {}
func (*bufferProbeRunner) Layout(int, int) (int, int) { return 32, 32 }

func TestMain(m *testing.M) {
	ebiten.SetWindowVisible(false)
	ebiten.SetWindowSize(32, 32)
	runner := &bufferProbeRunner{}
	go func() { bufferProbeFinished <- m.Run() }()
	if err := ebiten.RunGame(runner); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	os.Exit(runner.exitCode)
}

func TestNativeAudioControlBuffer(t *testing.T) {
	bufferProbeRequests <- true
	results := <-bufferProbeResults
	if path := os.Getenv("INDY_AUDIO_BUFFER_REPORT"); path != "" {
		data, err := json.MarshalIndent(results, "", "  ")
		if err != nil {
			t.Fatal(err)
		}
		if err = os.WriteFile(path, data, 0600); err != nil {
			t.Fatal(err)
		}
	}
	largest := map[string]int{}
	for name, reads := range results {
		if len(reads) == 0 {
			t.Fatalf("%s source was never read", name)
		}
		for _, read := range reads {
			largest[name] = max(largest[name], read.Bytes)
		}
		t.Logf("%s: %d reads, maximum%dbytes / %.3fms source PCM; control ticks%d..%d", name, len(reads), largest[name], 1000*float64(largest[name])/(SampleRate*4), reads[0].ControlTick, reads[len(reads)-1].ControlTick)
	}
	// A half-second read freezes one snapshot of all pitch/gain/cue parameters
	// for that whole block. The real game factory must avoid that backend default.
	if largest["backendDefault"] < SampleRate*4/4 {
		t.Fatal("backend default no longer exposes the expected large queue; update the measured regression")
	}
	if largest["gamePlayer"] > SampleRate*4/20 {
		t.Fatal("game player buffers more than fifty milliseconds of one control snapshot")
	}
	if len(results["gamePlayer"]) < len(results["backendDefault"])*2 {
		t.Fatal("game control snapshots were not requested substantially more often")
	}
}
