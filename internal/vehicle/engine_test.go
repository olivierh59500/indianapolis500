package vehicle

import (
	"bytes"
	"crypto/sha256"
	"encoding/binary"
	"fmt"
	"testing"
)

// These snapshots come from the original engine block inside complete
// physics interrupts. The digest covers every engine state quantity, using
// the same explicit integer field order as the independent original capture.
var originalEngineConfig = EngineConfig{Consumption: 850, ShiftUp: 5750, ShiftDown: 2500, PowerLinear: 1472, PowerCubic: 10719, DamageRPM: 6000, DamageScale: 18, Gears: [4]uint16{60999, 39135, 26728, 26236}}
var originalEngineState = EngineState{RPM: 1000, Gear: 0, Ratio: 60999, ThrottleResponse: 0, Pressure: 20000, PressureTarget: 20000, Temperature: 21376, FuelUsed: 305, Mass: 8780, Boost: 57770, LeftDrive: 0, RightDrive: 0, LeftDemand: 0, RightDemand: 0, Fuel: 83886080, Life: 9830400, FuelTick: 8, Warning: false, Failures: 0}

func engineStateDigest(t *testing.T, state EngineState) string {
	t.Helper()
	var data bytes.Buffer
	if err := binary.Write(&data, binary.BigEndian, state); err != nil {
		t.Fatal(err)
	}
	return fmt.Sprintf("%x", sha256.Sum256(data.Bytes()))
}

func TestOriginalEngineDuringAcceleration(t *testing.T) {
	cases := []struct {
		input  EngineInput
		digest string
	}{
		{EngineInput{Speed: 0, Ambient: 17184, Throttle: 8192, ThrottleLimit: 65535, BoostTarget: 61439, LeftLoad: 1131, RightLoad: 1131, LeftGrip: 57400, RightGrip: 56000, RearSpeed: 0, Differential: 0}, "c0916f605382e8dcf097e742a3f224bce9719ce744cde458864908fae4c37eef"},
		{EngineInput{Speed: 19, Ambient: 17184, Throttle: 16384, ThrottleLimit: 65535, BoostTarget: 61439, LeftLoad: 1158, RightLoad: 1157, LeftGrip: 57400, RightGrip: 56000, RearSpeed: 18, Differential: 0}, "3787292ead2f2659d382a933c1f58fabccbd618d45600c3e126ef89a4c68a74a"},
		{EngineInput{Speed: 52, Ambient: 17184, Throttle: 24576, ThrottleLimit: 65535, BoostTarget: 61439, LeftLoad: 1189, RightLoad: 1186, LeftGrip: 57400, RightGrip: 56000, RearSpeed: 50, Differential: 0}, "f10c9ade89cecfbbc5f19c8aaaea204a88048dc8a7afd11efa1414259ed53fe5"},
		{EngineInput{Speed: 104, Ambient: 17184, Throttle: 32768, ThrottleLimit: 65535, BoostTarget: 61439, LeftLoad: 1228, RightLoad: 1222, LeftGrip: 57400, RightGrip: 56000, RearSpeed: 102, Differential: 0}, "557dd1d8490c828495a067795556a7b90627a6a0892d30a59cacf3aab667d0ba"},
		{EngineInput{Speed: 175, Ambient: 17184, Throttle: 40960, ThrottleLimit: 65535, BoostTarget: 61439, LeftLoad: 1270, RightLoad: 1260, LeftGrip: 57400, RightGrip: 56000, RearSpeed: 174, Differential: 0}, "276b8ab86786a71fc7a6b3d2d26e925137e97d64294133055f728d43d919e286"},
		{EngineInput{Speed: 264, Ambient: 17184, Throttle: 49152, ThrottleLimit: 65535, BoostTarget: 61439, LeftLoad: 1311, RightLoad: 1298, LeftGrip: 57400, RightGrip: 56000, RearSpeed: 262, Differential: 0}, "8439003854a01558c89aa368e9621b945bcdda1724b65fb8e79125efdd325d69"},
		{EngineInput{Speed: 356, Ambient: 17184, Throttle: 49152, ThrottleLimit: 65535, BoostTarget: 61439, LeftLoad: 1328, RightLoad: 1314, LeftGrip: 57400, RightGrip: 56000, RearSpeed: 354, Differential: 0}, "e6631f6d9ed77e819be276ebf258c1fa27d085e33b0340b39eda5ba29977a24e"},
		{EngineInput{Speed: 448, Ambient: 17184, Throttle: 49152, ThrottleLimit: 65535, BoostTarget: 61439, LeftLoad: 1336, RightLoad: 1322, LeftGrip: 57400, RightGrip: 56000, RearSpeed: 446, Differential: 0}, "59f34eb3de1d5b7373df85bdeb8a971b4a8999d18f829f0d76aff0856a7992bb"},
		{EngineInput{Speed: 541, Ambient: 17184, Throttle: 49152, ThrottleLimit: 65535, BoostTarget: 61439, LeftLoad: 1339, RightLoad: 1325, LeftGrip: 57400, RightGrip: 56000, RearSpeed: 540, Differential: 0}, "8f4b63fe5a46011df018c6156d48610fa2aa0c480a260204db72298078d9386c"},
		{EngineInput{Speed: 634, Ambient: 17184, Throttle: 49152, ThrottleLimit: 65535, BoostTarget: 61439, LeftLoad: 1342, RightLoad: 1329, LeftGrip: 57400, RightGrip: 56000, RearSpeed: 632, Differential: 0}, "b8edd996e42cbed4ff6ba619f9bf911ac081cbaac11ac0753136c717ec10b5da"},
		{EngineInput{Speed: 727, Ambient: 17184, Throttle: 49152, ThrottleLimit: 65535, BoostTarget: 61439, LeftLoad: 1342, RightLoad: 1330, LeftGrip: 57400, RightGrip: 56000, RearSpeed: 726, Differential: 0}, "4b4d4de6ee3af106a1f0bf169cddfa9d2db14d6906dc8984079e183eaa52dc8b"},
		{EngineInput{Speed: 820, Ambient: 17184, Throttle: 49152, ThrottleLimit: 65535, BoostTarget: 61439, LeftLoad: 1344, RightLoad: 1332, LeftGrip: 57400, RightGrip: 56000, RearSpeed: 818, Differential: 0}, "29cd4fec4dd3e8de80e6614753806533fc483910c6ce53aa44d60381e81e1db9"},
		{EngineInput{Speed: 913, Ambient: 17184, Throttle: 49152, ThrottleLimit: 65535, BoostTarget: 61439, LeftLoad: 1347, RightLoad: 1335, LeftGrip: 57400, RightGrip: 56000, RearSpeed: 912, Differential: 0}, "49a83ae7a158ba04bc201f7066ed3a33051173c205b4769d139e52fb63994bd7"},
		{EngineInput{Speed: 1007, Ambient: 17184, Throttle: 49152, ThrottleLimit: 65535, BoostTarget: 61439, LeftLoad: 1351, RightLoad: 1339, LeftGrip: 57400, RightGrip: 56000, RearSpeed: 1006, Differential: 0}, "d4dd3e9acd134e01f8d625ae6a37b370543066e63f98270a20275e6da1017688"},
		{EngineInput{Speed: 1100, Ambient: 17184, Throttle: 49152, ThrottleLimit: 65535, BoostTarget: 61439, LeftLoad: 1352, RightLoad: 1340, LeftGrip: 57400, RightGrip: 56000, RearSpeed: 1098, Differential: 0}, "da77d41c8efa5ea65d60a0d8d4338dbce5b40f315eafcbcaf255cd0dcfaf2aa8"},
		{EngineInput{Speed: 1188, Ambient: 17184, Throttle: 49152, ThrottleLimit: 65535, BoostTarget: 61439, LeftLoad: 1346, RightLoad: 1335, LeftGrip: 57400, RightGrip: 56000, RearSpeed: 1186, Differential: 0}, "ebd9f38b2feb5adf409dacee18ad42f32d082608d0e70c72744bbe09ee2aea39"},
		{EngineInput{Speed: 1281, Ambient: 17184, Throttle: 49152, ThrottleLimit: 65535, BoostTarget: 61439, LeftLoad: 1353, RightLoad: 1343, LeftGrip: 57400, RightGrip: 56000, RearSpeed: 1280, Differential: 0}, "828cd0dab9694e354342f041ef6cd44604fc8fe71605e9d04a1013b454062ecf"},
		{EngineInput{Speed: 1375, Ambient: 17184, Throttle: 49152, ThrottleLimit: 65535, BoostTarget: 61439, LeftLoad: 1359, RightLoad: 1350, LeftGrip: 57400, RightGrip: 56000, RearSpeed: 1374, Differential: 0}, "7b5d9d6b745de239769159614af6a4dcfcbc72eb8755f0a45e7661670bedf412"},
		{EngineInput{Speed: 1469, Ambient: 17184, Throttle: 49152, ThrottleLimit: 65535, BoostTarget: 61439, LeftLoad: 1363, RightLoad: 1354, LeftGrip: 57400, RightGrip: 56000, RearSpeed: 1468, Differential: 0}, "ef3a8f6297faff4148f9bcd729a355f9b06973f99f1ffb65837844003ba90108"},
		{EngineInput{Speed: 1564, Ambient: 17184, Throttle: 49152, ThrottleLimit: 65535, BoostTarget: 61439, LeftLoad: 1112, RightLoad: 1104, LeftGrip: 57400, RightGrip: 56000, RearSpeed: 1562, Differential: 0}, "881445255a3f544a0c368cd807e8c6f1599cecc3598b407ff63c32f776c10044"},
		{EngineInput{Speed: 1641, Ambient: 17184, Throttle: 49152, ThrottleLimit: 65535, BoostTarget: 61439, LeftLoad: 1343, RightLoad: 1338, LeftGrip: 57400, RightGrip: 56000, RearSpeed: 1640, Differential: 0}, "3d5c17037a027be7c83941177e5644e2d405a5f806439e995e84102c554e75c4"},
		{EngineInput{Speed: 1734, Ambient: 17184, Throttle: 49152, ThrottleLimit: 65535, BoostTarget: 61439, LeftLoad: 1364, RightLoad: 1357, LeftGrip: 57400, RightGrip: 56000, RearSpeed: 1732, Differential: 0}, "c4f89b0bbda1306d415c2e1a8d33081287a60e9e8923b161b506941260e55fa1"},
		{EngineInput{Speed: 1829, Ambient: 17184, Throttle: 49152, ThrottleLimit: 65535, BoostTarget: 61439, LeftLoad: 1376, RightLoad: 1369, LeftGrip: 57400, RightGrip: 56000, RearSpeed: 1826, Differential: 0}, "d88cd391ca3cdc3b3818ab2c3a72f37f0aeda2683f92476367a78012afe840c1"},
		{EngineInput{Speed: 1923, Ambient: 17184, Throttle: 49152, ThrottleLimit: 65535, BoostTarget: 61439, LeftLoad: 1383, RightLoad: 1377, LeftGrip: 57400, RightGrip: 56000, RearSpeed: 1920, Differential: 0}, "90a63872718396e2c92f04b58dc638376887a6c9bfff083051479721281a63b3"},
		{EngineInput{Speed: 2018, Ambient: 17184, Throttle: 49152, ThrottleLimit: 65535, BoostTarget: 61439, LeftLoad: 1389, RightLoad: 1383, LeftGrip: 57400, RightGrip: 56000, RearSpeed: 2016, Differential: 0}, "c8e34dd812af392a60f0279e205134b9eb89d4a2b8b795a71f7cc52307dc51f3"},
		{EngineInput{Speed: 2113, Ambient: 17184, Throttle: 49152, ThrottleLimit: 65535, BoostTarget: 61439, LeftLoad: 1395, RightLoad: 1389, LeftGrip: 57400, RightGrip: 56000, RearSpeed: 2110, Differential: 0}, "908a5c994536b59c84ef859e43942c56d38aac6c356801efcca11b3b3812cf22"},
		{EngineInput{Speed: 2208, Ambient: 17184, Throttle: 49152, ThrottleLimit: 65535, BoostTarget: 61439, LeftLoad: 1400, RightLoad: 1394, LeftGrip: 57400, RightGrip: 56000, RearSpeed: 2206, Differential: 0}, "ef80a642c016e71b856dac64f7b28ce2fb44b9ed44355e235fafa46c9fcbf682"},
		{EngineInput{Speed: 2304, Ambient: 17184, Throttle: 49152, ThrottleLimit: 65535, BoostTarget: 61439, LeftLoad: 1405, RightLoad: 1400, LeftGrip: 57400, RightGrip: 56000, RearSpeed: 2302, Differential: 0}, "a40755329662b1fc9f35fb554e2288cdeaabcb9d56fe88980634beb03e01742b"},
		{EngineInput{Speed: 2400, Ambient: 17184, Throttle: 49152, ThrottleLimit: 65535, BoostTarget: 61439, LeftLoad: 1412, RightLoad: 1407, LeftGrip: 57400, RightGrip: 56000, RearSpeed: 2398, Differential: 0}, "59f9e296812b9054b48540f4c24779ed8acfa799b9d321fd477984d02e5ec08d"},
		{EngineInput{Speed: 2497, Ambient: 17184, Throttle: 49152, ThrottleLimit: 65535, BoostTarget: 61439, LeftLoad: 1418, RightLoad: 1413, LeftGrip: 57400, RightGrip: 56000, RearSpeed: 2494, Differential: 0}, "3d496d1ffe98cb58cd0aaa2a6ce1d3e79ba7ae2ba91de67f856bc1516fce645e"},
	}
	engine := Engine{Config: originalEngineConfig, State: originalEngineState}
	for i, c := range cases {
		engine.Step(c.input)
		if got := engineStateDigest(t, engine.State); got != c.digest {
			t.Fatalf("original engine tick %d: got %+v, digest %s; want %s", i, engine.State, got, c.digest)
		}
	}
}

// These isolated original calls cover shifts, signed coefficients, grip
// limits, fuel exhaustion and overheating. Original notification IDs also
// verify the events returned to the shared game's presentation layer.
func TestOriginalEngineBoundaryCases(t *testing.T) {
	cases := []struct {
		name   string
		setup  func(*EngineState)
		input  EngineInput
		digest string
		events EngineEvents
	}{
		{"shift-up", func(s *EngineState) { s.RPM = 5740 }, EngineInput{Speed: 0, Ambient: 17184, Throttle: 8192, ThrottleLimit: 65535, BoostTarget: 61439, LeftLoad: 1131, RightLoad: 1131, LeftGrip: 57400, RightGrip: 56000, RearSpeed: 7000, Differential: 0}, "f0adc1564ca70d6592e7d10bbde2cbe860db4b9df2e0d4811bd85f3ee730c5e8", EngineEvents{FuelExhausted: false, EngineFailure: false}},
		{"shift-down", func(s *EngineState) { s.RPM = 2550; s.Gear = 2; s.Ratio = 26728 }, EngineInput{Speed: 0, Ambient: 17184, Throttle: 8192, ThrottleLimit: 65535, BoostTarget: 61439, LeftLoad: 1131, RightLoad: 1131, LeftGrip: 57400, RightGrip: 56000, RearSpeed: 1000, Differential: 0}, "cf16310d7d5c571a9bdc98e4b15e33b7650ddbfe2edb2100886c57f76dc926b5", EngineEvents{FuelExhausted: false, EngineFailure: false}},
		{"idle-downshift", func(s *EngineState) { s.Gear = 1; s.Ratio = 39135 }, EngineInput{Speed: 0, Ambient: 17184, Throttle: 8192, ThrottleLimit: 65535, BoostTarget: 61439, LeftLoad: 1131, RightLoad: 1131, LeftGrip: 57400, RightGrip: 56000, RearSpeed: 0, Differential: 0}, "c0916f605382e8dcf097e742a3f224bce9719ce744cde458864908fae4c37eef", EngineEvents{FuelExhausted: false, EngineFailure: false}},
		{"top-gear", func(s *EngineState) { s.RPM = 5900; s.Gear = 3; s.Ratio = 26236 }, EngineInput{Speed: 0, Ambient: 17184, Throttle: 8192, ThrottleLimit: 65535, BoostTarget: 61439, LeftLoad: 1131, RightLoad: 1131, LeftGrip: 57400, RightGrip: 56000, RearSpeed: 20000, Differential: 0}, "75b7ff8c2d9412d87b3a0bac0ec88b73feb5020346e132183868c06efcf6c879", EngineEvents{FuelExhausted: false, EngineFailure: false}},
		{"fuel-empty", func(s *EngineState) { s.Fuel = 0; s.FuelTick = 1 }, EngineInput{Speed: 0, Ambient: 17184, Throttle: 49152, ThrottleLimit: 65535, BoostTarget: 61439, LeftLoad: 1131, RightLoad: 1131, LeftGrip: 57400, RightGrip: 56000, RearSpeed: 1000, Differential: 0}, "f5f0c3a3ad26c9c128aef35e6f48cf12f1892a477d2ae25eb30fb482fd5ce65a", EngineEvents{FuelExhausted: false, EngineFailure: false}},
		{"fuel-exhaustion", func(s *EngineState) { s.RPM = 5000; s.Fuel = 100; s.FuelTick = 1 }, EngineInput{Speed: 0, Ambient: 17184, Throttle: 49152, ThrottleLimit: 65535, BoostTarget: 61439, LeftLoad: 1131, RightLoad: 1131, LeftGrip: 57400, RightGrip: 56000, RearSpeed: 6000, Differential: 0}, "fa0a17a3d2960f44332b04035df0b296c7f3949715b4a22f516ed82d041079b9", EngineEvents{FuelExhausted: true, EngineFailure: false}},
		{"engine-already-failed", func(s *EngineState) { s.RPM = 5000; s.FuelTick = 1; s.Failures = 16 }, EngineInput{Speed: 0, Ambient: 17184, Throttle: 49152, ThrottleLimit: 65535, BoostTarget: 61439, LeftLoad: 1131, RightLoad: 1131, LeftGrip: 57400, RightGrip: 56000, RearSpeed: 6000, Differential: 0}, "17d54aa80bff4495a16dd4d96e275e795ac15ba0564f29bff565c177d95edb8f", EngineEvents{FuelExhausted: false, EngineFailure: false}},
		{"no-fuel-idle-flag", func(s *EngineState) { s.Failures = 32 }, EngineInput{Speed: 0, Ambient: 17184, Throttle: 0, ThrottleLimit: 65535, BoostTarget: 61439, LeftLoad: 1131, RightLoad: 1131, LeftGrip: 57400, RightGrip: 56000, RearSpeed: 0, Differential: 0}, "7c2c6c072f2c63bf638b3857b170e2b52c2147ebed21b3aa8554550bc83d4efb", EngineEvents{FuelExhausted: false, EngineFailure: false}},
		{"overheat-warning", func(s *EngineState) { s.RPM = 8500; s.Temperature = 30000; s.FuelTick = 1 }, EngineInput{Speed: 9000, Ambient: 17184, Throttle: 49152, ThrottleLimit: 65535, BoostTarget: 61439, LeftLoad: 1131, RightLoad: 1131, LeftGrip: 57400, RightGrip: 56000, RearSpeed: 9000, Differential: 0}, "5d279b3c8c787a1f726238676ba80ce0b0e7fb538c5b89546bf9fd9d11500995", EngineEvents{FuelExhausted: false, EngineFailure: false}},
		{"low-life-warning", func(s *EngineState) { s.RPM = 8500; s.Temperature = 30000; s.Life = 3276800; s.FuelTick = 1 }, EngineInput{Speed: 9000, Ambient: 17184, Throttle: 49152, ThrottleLimit: 65535, BoostTarget: 61439, LeftLoad: 1131, RightLoad: 1131, LeftGrip: 57400, RightGrip: 56000, RearSpeed: 9000, Differential: 0}, "48af08dc72dd9512158ac80f80e3849ea70d38132c811d0995a3c552701403e1", EngineEvents{FuelExhausted: false, EngineFailure: false}},
		{"engine-life-underflow", func(s *EngineState) { s.RPM = 8500; s.Temperature = 30000; s.Life = 10; s.FuelTick = 1 }, EngineInput{Speed: 9000, Ambient: 17184, Throttle: 49152, ThrottleLimit: 65535, BoostTarget: 61439, LeftLoad: 1131, RightLoad: 1131, LeftGrip: 57400, RightGrip: 56000, RearSpeed: 9000, Differential: 0}, "53d538387db03841d9c0701dde676a3fdc2b213db26c221f5f280ffc1de727e0", EngineEvents{FuelExhausted: false, EngineFailure: true}},
		{"reverse-wheel-motion", nil, EngineInput{Speed: 0, Ambient: 17184, Throttle: 49152, ThrottleLimit: 65535, BoostTarget: 61439, LeftLoad: 1131, RightLoad: 1131, LeftGrip: 57400, RightGrip: 56000, RearSpeed: -1000, Differential: 0}, "d179eaea69f3deff4fc85fbb8f86a8b07340ca8ffa2f70bd841112f8b5efdd13", EngineEvents{FuelExhausted: false, EngineFailure: false}},
		{"differential-left", nil, EngineInput{Speed: 0, Ambient: 17184, Throttle: 49152, ThrottleLimit: 65535, BoostTarget: 61439, LeftLoad: 1131, RightLoad: 1131, LeftGrip: 57400, RightGrip: 56000, RearSpeed: 3000, Differential: 20000}, "577f8cf9c182aba95a86bc4993b9282c425c4a2fc80503bf1791964701d38d38", EngineEvents{FuelExhausted: false, EngineFailure: false}},
		{"differential-right", nil, EngineInput{Speed: 0, Ambient: 17184, Throttle: 49152, ThrottleLimit: 65535, BoostTarget: 61439, LeftLoad: 1131, RightLoad: 1131, LeftGrip: 57400, RightGrip: 56000, RearSpeed: 3000, Differential: -20000}, "009b949b7525bbd7a7bb4bea3fde353c9f263881ab94f925bf33721c6c2940a5", EngineEvents{FuelExhausted: false, EngineFailure: false}},
		{"wheel-force-cap", nil, EngineInput{Speed: 0, Ambient: 17184, Throttle: 49152, ThrottleLimit: 65535, BoostTarget: 61439, LeftLoad: 1, RightLoad: 1, LeftGrip: 1000, RightGrip: 1200, RearSpeed: 3000, Differential: 0}, "ce7583457db2b8c61850c849334159528a4138920d649bfe3674d046a73656de", EngineEvents{FuelExhausted: false, EngineFailure: false}},
		{"throttle-limit", nil, EngineInput{Speed: 0, Ambient: 17184, Throttle: 49152, ThrottleLimit: 10000, BoostTarget: 61439, LeftLoad: 1131, RightLoad: 1131, LeftGrip: 57400, RightGrip: 56000, RearSpeed: 3000, Differential: 0}, "b6a5e092bcd93793734df1705328fe9ca50d2e3bdf2a82d410556a6e6bd3e78d", EngineEvents{FuelExhausted: false, EngineFailure: false}},
		{"signed-boost-wrap", func(s *EngineState) { s.Boost = 65000 }, EngineInput{Speed: 0, Ambient: 17184, Throttle: 49152, ThrottleLimit: 65535, BoostTarget: 1000, LeftLoad: 1131, RightLoad: 1131, LeftGrip: 57400, RightGrip: 56000, RearSpeed: 3000, Differential: 0}, "42368f8976293580406c470d09c1e1f5af83c501e0524899f1260b436e5ab326", EngineEvents{FuelExhausted: false, EngineFailure: false}},
		{"cooling-floor", func(s *EngineState) { s.Temperature = 20768; s.FuelTick = 1 }, EngineInput{Speed: 9000, Ambient: 17184, Throttle: 0, ThrottleLimit: 65535, BoostTarget: 61439, LeftLoad: 1131, RightLoad: 1131, LeftGrip: 57400, RightGrip: 56000, RearSpeed: 7000, Differential: 0}, "db6b18cd304253392c25c1d32d3084c4cce18853f8ae21caa7e27725442ab55c", EngineEvents{FuelExhausted: false, EngineFailure: false}},
		{"high-wheel-speed", func(s *EngineState) { s.RPM = 9000; s.Gear = 3; s.Ratio = 26236; s.FuelTick = 1 }, EngineInput{Speed: 10000, Ambient: 17184, Throttle: 49152, ThrottleLimit: 65535, BoostTarget: 61439, LeftLoad: 1131, RightLoad: 1131, LeftGrip: 57400, RightGrip: 56000, RearSpeed: 32767, Differential: 0}, "5b4447d5d16c3665eb4d7f9d3e4196762f76e2c84d36397592136c42f4c24026", EngineEvents{FuelExhausted: false, EngineFailure: false}},
	}
	for _, c := range cases {
		engine := Engine{Config: originalEngineConfig, State: originalEngineState}
		if c.setup != nil {
			c.setup(&engine.State)
		}
		events := engine.Step(c.input)
		if got := engineStateDigest(t, engine.State); got != c.digest || events != c.events {
			t.Fatalf("%s: got %+v (%s), events %+v; want %s, events %+v", c.name, engine.State, got, events, c.digest, c.events)
		}
	}
}
