package traffic

import (
	"encoding/json"
	"os"
	"testing"
)

type playerEpochState struct {
	Cars                           [33]Car
	Movement                       int16
	Random                         uint16
	Cursor, Tail, Position, Leader int
	RankAhead, RankBehind          [33]int
}
type playerEpochFrame struct {
	Kind                    string
	Mode                    int
	Pose                    PlayerPose
	Before, Prepared, After playerEpochState
}

func TestOriginalPlayerCarPreparationAndCompleteField(t *testing.T) {
	var rows []playerEpochFrame
	for _, file := range []string{"player-epochs.json", "finish-field.json"} {
		data, err := os.ReadFile("testdata/" + file)
		if err != nil {
			t.Fatal(err)
		}
		var cases []playerEpochFrame
		if err = json.Unmarshal(data, &cases); err != nil {
			t.Fatal(err)
		}
		rows = append(rows, cases...)
	}
	m, _ := originalRacing(t)
	for _, source := range rows {
		t.Run(source.Kind, func(t *testing.T) {
			before := State{Cars: source.Before.Cars, Random: source.Before.Random, NextDriver: source.Before.Cursor, Tail: source.Before.Tail, Rules: source.Mode, Leader: source.Before.Leader, Position: uint16(source.Before.Position), RankAhead: source.Before.RankAhead, RankBehind: source.Before.RankBehind}
			m.State = before
			m.PreparePlayer(source.Pose)
			source.Prepared.compareRanking(t, m)
			for i := range m.Cars {
				if m.Cars[i] != source.Prepared.Cars[i] {
					t.Fatalf("prepared car%d native%+v original%+v", i, m.Cars[i], source.Prepared.Cars[i])
				}
			}
			if m.PlayerMovement != source.Prepared.Movement || m.Tail != source.Prepared.Tail {
				t.Fatalf("prepared movement/tail native%d/%d original%d/%d", m.PlayerMovement, m.Tail, source.Prepared.Movement, source.Prepared.Tail)
			}
			m.State = before
			m.StepRacingField(source.Pose, nil)
			source.After.compareRanking(t, m)
			for i := range m.Cars {
				if m.Cars[i] != source.After.Cars[i] {
					t.Fatalf("complete car%d native%+v original%+v", i, m.Cars[i], source.After.Cars[i])
				}
			}
			if m.PlayerMovement != source.After.Movement || m.Tail != source.After.Tail || m.Random != source.After.Random || m.NextDriver != source.After.Cursor {
				t.Fatalf("complete movement/tail/random/cursor native%d/%d/%x/%d original%d/%d/%x/%d", m.PlayerMovement, m.Tail, m.Random, m.NextDriver, source.After.Movement, source.After.Tail, source.After.Random, source.After.Cursor)
			}
		})
	}
}

func (s playerEpochState) compareRanking(t *testing.T, m *Model) {
	t.Helper()
	originalRanking{Leader: s.Leader, Position: uint16(s.Position), Tail: s.Tail, RankAhead: s.RankAhead, RankBehind: s.RankBehind}.compare(t, m)
}

type originalRanking struct {
	Leader, Tail          int
	Position              uint16
	RankAhead, RankBehind [33]int
}

func (s originalRanking) supply(m *Model) {
	m.Leader, m.Tail, m.Position = s.Leader, s.Tail, s.Position
	m.RankAhead, m.RankBehind = s.RankAhead, s.RankBehind
}
func (s originalRanking) compare(t *testing.T, m *Model) {
	t.Helper()
	if m.Leader != s.Leader || m.Position != s.Position || m.Tail != s.Tail {
		t.Fatalf("ranking head/position/tail native%d/%d/%d original%d/%d/%d", m.Leader, m.Position, m.Tail, s.Leader, s.Position, s.Tail)
	}
	for i := range m.Cars {
		if m.RankAhead[i] != s.RankAhead[i] || m.RankBehind[i] != s.RankBehind[i] {
			t.Fatalf("car%d rank links native%d/%d original%d/%d", i, m.RankAhead[i], m.RankBehind[i], s.RankAhead[i], s.RankBehind[i])
		}
	}
}
