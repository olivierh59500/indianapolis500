//go:build rendercheck

package game

import (
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"os"
	"testing"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/olivierh59500/indianapolis500/internal/traffic"
)

var headerRenderRequests = make(chan *headerRenderCheck)
var headerRenderResults = make(chan error)
var headerRenderFinished = make(chan int, 1)

// Cocoa and the real Ebitengine painter run on the process main thread.
// The verification window stays hidden throughout the comparison.
func TestMain(m *testing.M) {
	ebiten.SetWindowVisible(false)
	ebiten.SetWindowSize(640, 400)
	ebiten.SetWindowTitle("Indianapolis 500 header verification")
	go func() { headerRenderFinished <- m.Run() }()
	runner := &headerRenderRunner{}
	if err := ebiten.RunGame(runner); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	os.Exit(runner.exitCode)
}

type headerRenderCheck struct {
	game          *Game
	digest        string
	noticeOnly    bool
	pitBodyDigest string
	worldDigest   string
	renderCheck   func(*ebiten.Image) error
}

type headerRenderRunner struct {
	check    *headerRenderCheck
	exitCode int
}

func (r *headerRenderRunner) Update() error {
	select {
	case r.exitCode = <-headerRenderFinished:
		return ebiten.Termination
	case r.check = <-headerRenderRequests:
	default:
	}
	return nil
}

func (*headerRenderRunner) Layout(int, int) (int, int) { return 320, 200 }

func (r *headerRenderRunner) Draw(screen *ebiten.Image) {
	if r.check == nil {
		return
	}
	screen.Fill(r.check.game.palette[0])
	if r.check.renderCheck != nil {
		headerRenderResults <- r.check.renderCheck(screen)
		r.check = nil
		return
	}
	if r.check.worldDigest != "" {
		r.check.game.drawOriginalWorld(r.check.game.cockpitView, inCarView)
		op := &ebiten.DrawImageOptions{}
		op.GeoM.Translate(0, 24)
		screen.DrawImage(r.check.game.cockpitView, op)
		pixels := make([]byte, 320*200*4)
		screen.ReadPixels(pixels)
		got := fmt.Sprintf("%x", sha256.Sum256(pixels[320*24*4:320*148*4]))
		var err error
		if got != r.check.worldDigest {
			err = fmt.Errorf("native game world RGBA hash%s, original%s", got, r.check.worldDigest)
		}
		if r.check.game.err != nil {
			err = r.check.game.err
		}
		headerRenderResults <- err
		r.check = nil
		return
	}
	if r.check.pitBodyDigest != "" {
		r.check.game.drawPitBoard(screen)
		r.check.game.drawCockpit(screen)
	} else if !r.check.noticeOnly {
		r.check.game.drawInformation(screen)
	}
	r.check.game.drawNotification(screen)
	pixels := make([]byte, 320*200*4)
	screen.ReadPixels(pixels)
	got := fmt.Sprintf("%x", sha256.Sum256(pixels[:320*24*4]))
	var err error
	if got != r.check.digest {
		err = fmt.Errorf("playable standings RGBA hash%s, original%s", got, r.check.digest)
	}
	if r.check.pitBodyDigest != "" {
		body := make([]byte, 0, 184*96*4)
		for y := 30; y < 126; y++ {
			body = append(body, pixels[(y*320+24)*4:(y*320+208)*4]...)
		}
		if got := fmt.Sprintf("%x", sha256.Sum256(body)); got != r.check.pitBodyDigest {
			err = fmt.Errorf("composited native pit body %s, original%s", got, r.check.pitBodyDigest)
		}
	}
	headerRenderResults <- err
	r.check = nil
}

func TestOriginalNotificationCompositing(t *testing.T) {
	data, err := os.ReadFile("testdata/notification-renders.json")
	if err != nil {
		t.Fatal(err)
	}
	var rows []struct {
		Visible    notification
		SHA256RGBA string
	}
	if err = json.Unmarshal(data, &rows); err != nil {
		t.Fatal(err)
	}
	g, err := New(Options{Mute: true})
	if err != nil {
		t.Fatal(err)
	}
	for _, row := range rows {
		g.notifications.Visible = row.Visible
		headerRenderRequests <- &headerRenderCheck{game: g, digest: row.SHA256RGBA, noticeOnly: true}
		if err = <-headerRenderResults; err != nil {
			t.Fatalf("%s: %v", notificationLabels[row.Visible], err)
		}
	}
}

func TestNormallyPlayedPauseCompositing(t *testing.T) {
	data, err := os.ReadFile("testdata/notification-paused-played.json")
	if err != nil {
		t.Fatal(err)
	}
	var rows []struct {
		playedLapInfo
		Visible notification
	}
	if err = json.Unmarshal(data, &rows); err != nil {
		t.Fatal(err)
	}
	for _, row := range rows {
		g, err := New(Options{Mute: true})
		if err != nil {
			t.Fatal(err)
		}
		g.supplyLapInfo(row.playedLapInfo)
		g.paused, g.noticeWasPaused = true, true
		g.notifications.Visible = row.Visible
		g.flags = flagRuntime{Index: 0}
		headerRenderRequests <- &headerRenderCheck{game: g, digest: row.SHA256RGBA}
		if err = <-headerRenderResults; err != nil {
			t.Fatalf("%s: %v", row.Kind, err)
		}
	}
}

func TestNormallyPlayedStandingsCompositing(t *testing.T) {
	data, err := os.ReadFile("testdata/played-standings.json")
	if err != nil {
		t.Fatal(err)
	}
	var source struct {
		Mode       driveMode
		TotalLaps  int
		Leader     int
		Position   uint16
		Cars       [33]traffic.Car
		RankBehind [33]int
		Timing     [33]traffic.LapTiming
		SHA256RGBA string
	}
	if err = json.Unmarshal(data, &source); err != nil {
		t.Fatal(err)
	}
	g, err := New(Options{Mute: true})
	if err != nil {
		t.Fatal(err)
	}
	g.mode, g.raceLaps = source.Mode, source.TotalLaps-1
	g.trafficModel.State = traffic.State{Cars: source.Cars, Leader: source.Leader, Position: source.Position, RankBehind: source.RankBehind}
	g.trafficModel.Timing.Cars = source.Timing
	g.flags = flagRuntime{Index: 0}
	g.selectInformationPage(standingsInformationPage)
	headerRenderRequests <- &headerRenderCheck{game: g, digest: source.SHA256RGBA}
	if err = <-headerRenderResults; err != nil {
		t.Fatal(err)
	}
}

func TestNormallyPlayedLapInfoCompositing(t *testing.T) {
	data, err := os.ReadFile("testdata/played-lap-info.json")
	if err != nil {
		t.Fatal(err)
	}
	var rows []playedLapInfo
	if err = json.Unmarshal(data, &rows); err != nil {
		t.Fatal(err)
	}
	for _, source := range rows {
		t.Run(source.Kind, func(t *testing.T) {
			g, err := New(Options{Mute: true})
			if err != nil {
				t.Fatal(err)
			}
			g.supplyLapInfo(source)
			g.flags = flagRuntime{Index: 0}
			headerRenderRequests <- &headerRenderCheck{game: g, digest: source.SHA256RGBA}
			if err = <-headerRenderResults; err != nil {
				t.Fatal(err)
			}
		})
	}
}
