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

// These live original-game observations exercise direct pedal changes and
// simultaneous keys, including cruise cancellation and residual pedal ramps.
func TestOriginalPedalChanges(t *testing.T) {
	var cases []struct {
		Input         DriverIntent
		Context       DriverContext
		Before, After DriverState
		Controls      Controls
	}
	data, err := os.ReadFile("testdata/pedal-changes.json")
	if err != nil {
		t.Fatal(err)
	}
	if err := json.Unmarshal(data, &cases); err != nil {
		t.Fatal(err)
	}
	if len(cases) != 96 {
		t.Fatalf("got %d original pedal observations, want 96", len(cases))
	}
	for tick, c := range cases {
		driver := Driver{State: c.Before}
		controls := driver.Step(c.Context, c.Input)
		if driver.State != c.After || controls != c.Controls {
			t.Fatalf("original pedal tick %d: state %+v controls %+v, want %+v %+v", tick, driver.State, controls, c.After, c.Controls)
		}
	}
}

// Original keyboard/control-stage captures include steering rate/centering,
// pedal ramps, reverse engagement, cruise increase and manual cancellation.
func TestOriginalDriverSequence(t *testing.T) {
	cases := []struct {
		input   DriverIntent
		context DriverContext
		digest  string
	}{
		{DriverIntent{Steering: 0, Accelerate: true, Brake: false, Reverse: false, CruiseIncrease: false}, DriverContext{Steering: 0, Damping: 0, Increment: 200, Speed: 0, Failures: 0}, "0bcccc8967bf4a81bafeb6aa380abf7d6c90efeccf1293795d211d30a3fb1346"},
		{DriverIntent{Steering: 0, Accelerate: true, Brake: false, Reverse: false, CruiseIncrease: false}, DriverContext{Steering: 0, Damping: 0, Increment: 200, Speed: 19, Failures: 0}, "3606aeb847f22f52a92a01fa50e24ce426b451af70274e6e13ed1f5819fa198c"},
		{DriverIntent{Steering: 0, Accelerate: true, Brake: false, Reverse: false, CruiseIncrease: false}, DriverContext{Steering: 0, Damping: 9, Increment: 208, Speed: 52, Failures: 0}, "56260386f21bb472158e2e51c02ace25f747e31e9b9606068759f4fd7924938b"},
		{DriverIntent{Steering: 0, Accelerate: true, Brake: false, Reverse: false, CruiseIncrease: false}, DriverContext{Steering: 0, Damping: 25, Increment: 223, Speed: 104, Failures: 0}, "f504e9de086668c890095e9334088d622a616f7777ac66f082244b1fc2c3456b"},
		{DriverIntent{Steering: 0, Accelerate: true, Brake: false, Reverse: false, CruiseIncrease: false}, DriverContext{Steering: 0, Damping: 51, Increment: 248, Speed: 175, Failures: 0}, "93491834750b5992b422b6db8802d33bffb6acb45951ea9f09288bcb85aa42c9"},
		{DriverIntent{Steering: 0, Accelerate: true, Brake: false, Reverse: false, CruiseIncrease: false}, DriverContext{Steering: 0, Damping: 87, Increment: 282, Speed: 264, Failures: 0}, "77e3d36ba5468b75ccf328f5889ab779e4fba676b4747fbe3f8cca3e4003ecdd"},
		{DriverIntent{Steering: -1, Accelerate: true, Brake: false, Reverse: false, CruiseIncrease: false}, DriverContext{Steering: 0, Damping: 131, Increment: 324, Speed: 356, Failures: 0}, "935e16ed682d3fd905b04d4c7416ea5d71b35b01fb1f3065c5caf8b62261e21e"},
		{DriverIntent{Steering: -1, Accelerate: true, Brake: false, Reverse: false, CruiseIncrease: false}, DriverContext{Steering: 193, Damping: 177, Increment: 368, Speed: 448, Failures: 0}, "b50022bd8a4651511774684110dde2e6defe8050422568523ba2db266128473d"},
		{DriverIntent{Steering: -1, Accelerate: true, Brake: false, Reverse: false, CruiseIncrease: false}, DriverContext{Steering: 384, Damping: 223, Increment: 412, Speed: 541, Failures: 0}, "2289624395f72e5d1484093b3654e6d28a3e0aff1f98facc978aabe8c047cdef"},
		{DriverIntent{Steering: -1, Accelerate: true, Brake: false, Reverse: false, CruiseIncrease: false}, DriverContext{Steering: 573, Damping: 269, Increment: 456, Speed: 634, Failures: 0}, "ab389e0a3123a37ca888a65313538a3c25bda8f913d6fa059c9bb4b9497bc022"},
		{DriverIntent{Steering: -1, Accelerate: true, Brake: false, Reverse: false, CruiseIncrease: false}, DriverContext{Steering: 760, Damping: 316, Increment: 501, Speed: 727, Failures: 0}, "7065fd86ef2b9f9b3998e552b1349e80bdc0da8cd4c1cac69bb1adce85c0df97"},
		{DriverIntent{Steering: -1, Accelerate: true, Brake: false, Reverse: false, CruiseIncrease: false}, DriverContext{Steering: 945, Damping: 363, Increment: 546, Speed: 819, Failures: 0}, "ed7de54eb7e0c6e642072575bfda3b95e7c5315aba4e7279e03687cb3ddf0db1"},
		{DriverIntent{Steering: -1, Accelerate: true, Brake: false, Reverse: false, CruiseIncrease: false}, DriverContext{Steering: 1128, Damping: 409, Increment: 590, Speed: 912, Failures: 0}, "8e1070746e6077691d733730f521211492fb8567059aabb55070442a869c29f9"},
		{DriverIntent{Steering: 1, Accelerate: true, Brake: false, Reverse: false, CruiseIncrease: false}, DriverContext{Steering: 1309, Damping: 456, Increment: 635, Speed: 1003, Failures: 0}, "c1a19ab2a01537e7a8aae41e4d657181e0553371d1d6b94cb6950495c556c740"},
		{DriverIntent{Steering: 1, Accelerate: true, Brake: false, Reverse: false, CruiseIncrease: false}, DriverContext{Steering: 218, Damping: 502, Increment: 679, Speed: 1095, Failures: 0}, "77e3d36ba5468b75ccf328f5889ab779e4fba676b4747fbe3f8cca3e4003ecdd"},
		{DriverIntent{Steering: 1, Accelerate: true, Brake: false, Reverse: false, CruiseIncrease: false}, DriverContext{Steering: 0, Damping: 547, Increment: 723, Speed: 1183, Failures: 0}, "e80e49148b8db3475d809703421c04da13c44864c9ef21d295894d70d6e47211"},
		{DriverIntent{Steering: 1, Accelerate: true, Brake: false, Reverse: false, CruiseIncrease: false}, DriverContext{Steering: -176, Damping: 590, Increment: 764, Speed: 1276, Failures: 0}, "29f1f2a6173faa32d7c893a35bc4b7a17a3c9cca5a8f6de8343caa9c796fc1a0"},
		{DriverIntent{Steering: 1, Accelerate: true, Brake: false, Reverse: false, CruiseIncrease: false}, DriverContext{Steering: -350, Damping: 637, Increment: 809, Speed: 1370, Failures: 0}, "687fa2cdac883025fe42ca72139d2ff9aa308e706e18bdfc9794231a20f7698a"},
		{DriverIntent{Steering: 1, Accelerate: true, Brake: false, Reverse: false, CruiseIncrease: false}, DriverContext{Steering: -522, Damping: 684, Increment: 855, Speed: 1463, Failures: 0}, "2920af9e1a283d316d75fd909011e9111ebbcca08c9b7041ecd2211f31ec0725"},
		{DriverIntent{Steering: 1, Accelerate: true, Brake: false, Reverse: false, CruiseIncrease: false}, DriverContext{Steering: -693, Damping: 731, Increment: 900, Speed: 1556, Failures: 0}, "d861588efed1a72865330a511a177ea9180e4b398ff3140925752daa7a158ed7"},
		{DriverIntent{Steering: 0, Accelerate: false, Brake: false, Reverse: false, CruiseIncrease: false}, DriverContext{Steering: -862, Damping: 778, Increment: 945, Speed: 1631, Failures: 0}, "c89cca8874c0a5c83275badef900e11c1d03c65f08496893a721244ce92643dc"},
		{DriverIntent{Steering: 0, Accelerate: false, Brake: false, Reverse: false, CruiseIncrease: false}, DriverContext{Steering: -84, Damping: 815, Increment: 981, Speed: 1719, Failures: 0}, "f504e9de086668c890095e9334088d622a616f7777ac66f082244b1fc2c3456b"},
		{DriverIntent{Steering: 0, Accelerate: false, Brake: false, Reverse: false, CruiseIncrease: false}, DriverContext{Steering: 0, Damping: 858, Increment: 1022, Speed: 1791, Failures: 0}, "56260386f21bb472158e2e51c02ace25f747e31e9b9606068759f4fd7924938b"},
		{DriverIntent{Steering: 0, Accelerate: false, Brake: false, Reverse: false, CruiseIncrease: false}, DriverContext{Steering: 0, Damping: 894, Increment: 1057, Speed: 1843, Failures: 0}, "3606aeb847f22f52a92a01fa50e24ce426b451af70274e6e13ed1f5819fa198c"},
		{DriverIntent{Steering: 0, Accelerate: false, Brake: false, Reverse: false, CruiseIncrease: false}, DriverContext{Steering: 0, Damping: 920, Increment: 1082, Speed: 1874, Failures: 0}, "0bcccc8967bf4a81bafeb6aa380abf7d6c90efeccf1293795d211d30a3fb1346"},
		{DriverIntent{Steering: 0, Accelerate: false, Brake: false, Reverse: false, CruiseIncrease: false}, DriverContext{Steering: 0, Damping: 936, Increment: 1098, Speed: 1886, Failures: 0}, "e7ecebbc590bc88b3761fa6cd03d749f87463dabb67021a5c6768c25ec68b3f2"},
		{DriverIntent{Steering: 0, Accelerate: false, Brake: true, Reverse: false, CruiseIncrease: false}, DriverContext{Steering: 0, Damping: 942, Increment: 1104, Speed: 1879, Failures: 0}, "a83a3a6b2f08470b8f5b44f8482fc0f28057ee19c33e7e3ea79a8a17d8fb56f9"},
		{DriverIntent{Steering: 0, Accelerate: false, Brake: true, Reverse: false, CruiseIncrease: false}, DriverContext{Steering: 0, Damping: 938, Increment: 1100, Speed: 1852, Failures: 0}, "443f9f159920e2f4747538bc41ac2fc7b15c697f27439680065e584933d82f65"},
		{DriverIntent{Steering: 0, Accelerate: false, Brake: true, Reverse: false, CruiseIncrease: false}, DriverContext{Steering: 0, Damping: 924, Increment: 1086, Speed: 1805, Failures: 0}, "474e4e2e80ebfe4abd666d33643814c3d547a1c7b5c0c2f84f5dd5ba71def1f7"},
		{DriverIntent{Steering: 0, Accelerate: false, Brake: true, Reverse: false, CruiseIncrease: false}, DriverContext{Steering: 0, Damping: 901, Increment: 1064, Speed: 1739, Failures: 0}, "494e423c219c49d34f061efbfe7e49edae2d17ace4a9791d0af817a2e02025fb"},
		{DriverIntent{Steering: 0, Accelerate: false, Brake: true, Reverse: false, CruiseIncrease: false}, DriverContext{Steering: 0, Damping: 869, Increment: 1033, Speed: 1653, Failures: 0}, "7803f4c869a0c778d176f4f80fafb901e080fa8ec59a5a86554374ce9df96c33"},
		{DriverIntent{Steering: 0, Accelerate: false, Brake: false, Reverse: true, CruiseIncrease: false}, DriverContext{Steering: 0, Damping: 826, Increment: 992, Speed: 1547, Failures: 0}, "494e423c219c49d34f061efbfe7e49edae2d17ace4a9791d0af817a2e02025fb"},
		{DriverIntent{Steering: 0, Accelerate: false, Brake: false, Reverse: true, CruiseIncrease: false}, DriverContext{Steering: 0, Damping: 772, Increment: 939, Speed: 1461, Failures: 0}, "474e4e2e80ebfe4abd666d33643814c3d547a1c7b5c0c2f84f5dd5ba71def1f7"},
		{DriverIntent{Steering: 0, Accelerate: false, Brake: false, Reverse: true, CruiseIncrease: false}, DriverContext{Steering: 0, Damping: 729, Increment: 898, Speed: 1395, Failures: 0}, "443f9f159920e2f4747538bc41ac2fc7b15c697f27439680065e584933d82f65"},
		{DriverIntent{Steering: 0, Accelerate: false, Brake: false, Reverse: true, CruiseIncrease: false}, DriverContext{Steering: 0, Damping: 696, Increment: 866, Speed: 1342, Failures: 0}, "a83a3a6b2f08470b8f5b44f8482fc0f28057ee19c33e7e3ea79a8a17d8fb56f9"},
		{DriverIntent{Steering: 0, Accelerate: false, Brake: false, Reverse: true, CruiseIncrease: false}, DriverContext{Steering: 0, Damping: 670, Increment: 841, Speed: 1316, Failures: 0}, "e8f58311f768ce499b3319a0498e07c9aa7869545421a53485e964a2df5a2605"},
		{DriverIntent{Steering: 0, Accelerate: false, Brake: false, Reverse: true, CruiseIncrease: false}, DriverContext{Steering: 0, Damping: 657, Increment: 829, Speed: 1281, Failures: 0}, "e8f58311f768ce499b3319a0498e07c9aa7869545421a53485e964a2df5a2605"},
		{DriverIntent{Steering: 0, Accelerate: false, Brake: false, Reverse: false, CruiseIncrease: true}, DriverContext{Steering: 0, Damping: 639, Increment: 811, Speed: 1246, Failures: 0}, "c2ab891ab5c9a55aa5df3d1d299a9bf9b983067829a83692467590773498fa8d"},
		{DriverIntent{Steering: 0, Accelerate: false, Brake: false, Reverse: false, CruiseIncrease: true}, DriverContext{Steering: 0, Damping: 622, Increment: 795, Speed: 1242, Failures: 0}, "1a281538d6ad1aeb04ddfcf5ab866e77b341642f3835f02b22b36b92c975e72f"},
		{DriverIntent{Steering: 0, Accelerate: false, Brake: false, Reverse: false, CruiseIncrease: true}, DriverContext{Steering: 0, Damping: 620, Increment: 793, Speed: 1240, Failures: 0}, "13c6eac5d21da12a6ee7649ab598aea6601e8bc4ab621c37859f0ef3068c2143"},
		{DriverIntent{Steering: 0, Accelerate: false, Brake: false, Reverse: false, CruiseIncrease: true}, DriverContext{Steering: 0, Damping: 619, Increment: 792, Speed: 1239, Failures: 0}, "97d939dc10306ce80bdd9395704f66a732f0dc2fc29d380c93cdc8a8b5249f1b"},
		{DriverIntent{Steering: 0, Accelerate: false, Brake: false, Reverse: false, CruiseIncrease: true}, DriverContext{Steering: 0, Damping: 619, Increment: 792, Speed: 1240, Failures: 0}, "278e60de74df091d3b63139aa447577627740af9b2cf3d159ff3d5ee426a5acb"},
		{DriverIntent{Steering: 0, Accelerate: false, Brake: false, Reverse: false, CruiseIncrease: true}, DriverContext{Steering: 0, Damping: 619, Increment: 792, Speed: 1244, Failures: 0}, "3c4444e86a767fa459fb209e8b3468b025bd608d196e271b37060cf3b3092d04"},
		{DriverIntent{Steering: 0, Accelerate: false, Brake: false, Reverse: false, CruiseIncrease: false}, DriverContext{Steering: 0, Damping: 621, Increment: 794, Speed: 1250, Failures: 0}, "e646b4b36ce3e82ea4c345250159d4020583a2d1fbc37e8b330f8abff84cac5e"},
		{DriverIntent{Steering: 0, Accelerate: false, Brake: false, Reverse: false, CruiseIncrease: false}, DriverContext{Steering: 0, Damping: 624, Increment: 797, Speed: 1258, Failures: 0}, "345771b933ea19a46e0113e9a8d6c0f08944527aa048457af4513f5bedb94f7c"},
		{DriverIntent{Steering: 0, Accelerate: false, Brake: false, Reverse: false, CruiseIncrease: false}, DriverContext{Steering: 0, Damping: 628, Increment: 801, Speed: 1269, Failures: 0}, "a0cd5e2dc585bfe22ac1ddd490a4bb7194b27c0ea0a1c13cabf3ce7e0c179578"},
		{DriverIntent{Steering: 0, Accelerate: false, Brake: false, Reverse: false, CruiseIncrease: false}, DriverContext{Steering: 0, Damping: 634, Increment: 806, Speed: 1282, Failures: 0}, "cddbd607b8a0000442aff054b8081dd6dbb3b3981c57db15fe17858c392244fc"},
		{DriverIntent{Steering: 0, Accelerate: false, Brake: false, Reverse: false, CruiseIncrease: false}, DriverContext{Steering: 0, Damping: 640, Increment: 812, Speed: 1298, Failures: 0}, "52ca0b80f2cbe613a68df89ba5b37292881340f2a71d95970a890dd4108d431a"},
		{DriverIntent{Steering: 0, Accelerate: false, Brake: false, Reverse: false, CruiseIncrease: false}, DriverContext{Steering: 0, Damping: 648, Increment: 820, Speed: 1316, Failures: 0}, "cb602ea64be94df6e8e885cc47b9634beb8f98027402f7a34c5440fe7dcca2a8"},
		{DriverIntent{Steering: 0, Accelerate: false, Brake: false, Reverse: false, CruiseIncrease: false}, DriverContext{Steering: 0, Damping: 657, Increment: 829, Speed: 1337, Failures: 0}, "176a13a90dbea5400dc4f468112e455035bea09242c56fd93ad5a273966176d5"},
		{DriverIntent{Steering: 0, Accelerate: true, Brake: false, Reverse: false, CruiseIncrease: false}, DriverContext{Steering: 0, Damping: 668, Increment: 839, Speed: 1361, Failures: 0}, "360dca420f12ad1821519d7f8d57ff0d7f2b4de03a340d936b362ab3c598773e"},
		{DriverIntent{Steering: 0, Accelerate: true, Brake: false, Reverse: false, CruiseIncrease: false}, DriverContext{Steering: 0, Damping: 680, Increment: 851, Speed: 1407, Failures: 0}, "08912d8e0055f2061d5fa6badf49896d64c6397a1c2b957cf30d6a1cb024cf99"},
		{DriverIntent{Steering: 0, Accelerate: true, Brake: false, Reverse: false, CruiseIncrease: false}, DriverContext{Steering: 0, Damping: 703, Increment: 873, Speed: 1474, Failures: 0}, "26e95fc10b28d7d796caa7b6436b98a5be1f744a17319e2aef904fd2c50147d4"},
		{DriverIntent{Steering: 0, Accelerate: true, Brake: false, Reverse: false, CruiseIncrease: false}, DriverContext{Steering: 0, Damping: 736, Increment: 905, Speed: 1562, Failures: 0}, "54aae8510d41cacdabb0484aa6df3213c5428b928ca2e7b5a6b948a520430d7b"},
	}
	driver := Driver{}
	for i, c := range cases {
		controls := driver.Step(c.context, c.input)
		var data bytes.Buffer
		for _, value := range []any{driver.State, controls} {
			if err := binary.Write(&data, binary.BigEndian, value); err != nil {
				t.Fatal(err)
			}
		}
		if got := fmt.Sprintf("%x", sha256.Sum256(data.Bytes())); got != c.digest {
			t.Fatalf("original driver tick%d: state%+v controls%+v (%s), want%s", i, driver.State, controls, got, c.digest)
		}
	}
}
