package vehicle

import "testing"

// Inputs and both returned coefficients were captured from the original
// $5242/$52a8 routines in FS-UAE, including zero vectors and signed limits.
func TestOriginalDrivenWheelBlends(t *testing.T) {
	cases := []struct {
		slip, speed     int16
		brake, throttle uint16
		skid            byte
		blend           WheelBlend
		afterSkid       byte
	}{
		{0, 0, 0, 0, 17, WheelBlend{65535, 65535}, 17},
		{1, 0, 0, 0, 17, WheelBlend{65535, 0}, 17},
		{-1, 0, 0, 0, 17, WheelBlend{65535, 0}, 17},
		{0, 1000, 0, 40000, 17, WheelBlend{0, 65535}, 92},
		{100, 1000, 0, 40000, 17, WheelBlend{20041, 62529}, 92},
		{-100, -1000, 0, 40000, 17, WheelBlend{20041, 62529}, 92},
		{100, 300, 16385, 0, 17, WheelBlend{23918, 61230}, 0},
		{100, 299, 16385, 0, 17, WheelBlend{23918, 61230}, 17},
		{100, 3999, 60000, 0, 17, WheelBlend{6957, 65188}, 170},
		{100, 4000, 60000, 0, 17, WheelBlend{6957, 65188}, 17},
		{-32768, -32768, 65535, 65535, 17, WheelBlend{65506, 2045}, 17},
		{27735, 2629, 61411, 33506, 192, WheelBlend{65498, 2264}, 175},
		{-4871, -4231, 44476, 11970, 9, WheelBlend{64738, 10193}, 9},
		{-6221, -588, 30389, 52191, 150, WheelBlend{65346, 5031}, 191},
		{-26363, -4867, 42609, 58744, 157, WheelBlend{65491, 2451}, 157},
		{16289, -3809, 63912, 15291, 97, WheelBlend{65411, 4075}, 187},
		{15742, 2764, 21263, 48973, 187, WheelBlend{65519, 1589}, 191},
		{11230, 1279, 23931, 10496, 3, WheelBlend{65501, 2187}, 61},
		{-13638, 1660, 65135, 23901, 244, WheelBlend{65353, 4883}, 191},
		{26289, 3051, 51640, 23102, 231, WheelBlend{65493, 2366}, 155},
		{3323, 2626, 36318, 7953, 242, WheelBlend{64583, 11175}, 95},
		{12556, -1637, 44312, 46858, 28, WheelBlend{65322, 5316}, 127},
		{-18522, -295, 33937, 60833, 93, WheelBlend{65444, 3487}, 93},
		{10402, 5445, 54981, 5942, 185, WheelBlend{65297, 5618}, 185},
		{2529, -5631, 4813, 3940, 15, WheelBlend{65510, 2460}, 15},
		{-29478, -3143, 51630, 9560, 243, WheelBlend{65507, 1951}, 173},
		{-1801, -2051, 35336, 50797, 146, WheelBlend{58114, 30331}, 142},
		{29447, 1554, 22944, 64561, 1, WheelBlend{65527, 1134}, 189},
		{-32205, -3620, 10553, 62740, 57, WheelBlend{65529, 1005}, 189},
		{11575, -414, 54409, 19316, 148, WheelBlend{65304, 5540}, 159},
		{15658, -5810, 52454, 62553, 98, WheelBlend{65398, 4256}, 98},
		{-26022, 3202, 4578, 232, 170, WheelBlend{65535, 178}, 170},
		{14611, -222, 38405, 50576, 2, WheelBlend{65388, 4417}, 2},
		{-25400, 5752, 64196, 55796, 253, WheelBlend{65484, 2627}, 253},
		{-8705, 4999, 30654, 45675, 11, WheelBlend{65430, 3833}, 11},
		{3000, -4700, 11683, 7147, 184, WheelBlend{65426, 3991}, 184},
		{30862, 5826, 8066, 44945, 110, WheelBlend{65531, 813}, 110},
		{-4304, 4423, 23513, 50722, 22, WheelBlend{65112, 7488}, 22},
		{3286, -287, 44506, 377, 125, WheelBlend{64130, 13563}, 125},
		{2018, -3189, 13635, 65088, 251, WheelBlend{63551, 16029}, 191},
		{20870, 1266, 6509, 48387, 72, WheelBlend{65526, 1202}, 125},
		{8936, 1634, 12063, 49344, 133, WheelBlend{65455, 3259}, 175},
		{-6158, 2824, 5624, 54324, 189, WheelBlend{65313, 5419}, 149},
		{30855, -504, 47641, 47535, 178, WheelBlend{65502, 2163}, 123},
		{-10411, -143, 9671, 51781, 66, WheelBlend{65491, 2560}, 66},
		{-30818, 3107, 63085, 16108, 253, WheelBlend{65501, 2170}, 190},
		{-13611, 736, 19920, 29774, 47, WheelBlend{65492, 2458}, 61},
	}
	for i, c := range cases {
		blend, skid := DrivenBlend(c.slip, c.speed, c.brake, c.throttle, c.skid)
		if blend != c.blend || skid != c.afterSkid {
			t.Fatalf("original driven blend %d: got %+v/%d, want %+v/%d", i, blend, skid, c.blend, c.afterSkid)
		}
	}
}

func TestOriginalFrontWheelBlends(t *testing.T) {
	cases := []struct {
		slip  int16
		brake uint16
		blend WheelBlend
	}{
		{0, 0, WheelBlend{65535, 65535}},
		{1, 0, WheelBlend{65535, 0}},
		{-1, 0, WheelBlend{65535, 0}},
		{0, 65535, WheelBlend{0, 65535}},
		{-32768, 65535, WheelBlend{65506, 2045}},
		{16384, 16384, WheelBlend{65532, 1023}},
		{-1466, 4455, WheelBlend{65491, 3082}},
		{-7707, 2203, WheelBlend{65535, 289}},
		{-2943, 40142, WheelBlend{64098, 13656}},
		{31316, 54353, WheelBlend{65512, 1776}},
		{-1537, 9164, WheelBlend{65281, 6073}},
		{3817, 6850, WheelBlend{65518, 1836}},
		{13554, 24810, WheelBlend{65511, 1870}},
		{21625, 12046, WheelBlend{65535, 569}},
		{-27256, 16415, WheelBlend{65533, 615}},
		{-28287, 43961, WheelBlend{65517, 1588}},
		{-1489, 7213, WheelBlend{65360, 4916}},
		{-29975, 10796, WheelBlend{65535, 367}},
		{10577, 65083, WheelBlend{65239, 6266}},
		{114, 42740, WheelBlend{11051, 64663}},
		{-27083, 44060, WheelBlend{65516, 1664}},
		{8506, 43895, WheelBlend{65328, 5261}},
		{-8641, 44717, WheelBlend{65324, 5276}},
		{-30710, 45115, WheelBlend{65518, 1501}},
		{5846, 62580, WheelBlend{64640, 10802}},
		{-1495, 3585, WheelBlend{65492, 2453}},
		{-14667, 47456, WheelBlend{65455, 3306}},
		{-3761, 46942, WheelBlend{64338, 12539}},
		{25704, 47687, WheelBlend{65510, 1898}},
		{29497, 10793, WheelBlend{65535, 373}},
	}
	for i, c := range cases {
		if blend := FrontBlend(c.slip, c.brake); blend != c.blend {
			t.Fatalf("original front blend %d: got %+v, want %+v", i, blend, c.blend)
		}
	}
}

// These are original $52e6 force-circle results with independent loads,
// available grip, blend coefficients and all four sign combinations.
func TestOriginalWheelForces(t *testing.T) {
	cases := []struct {
		lateral, longitudinal, load, grip     uint16
		blend                                 WheelBlend
		negativeLateral, negativeLongitudinal bool
		forceLateral, forceLongitudinal       int16
	}{
		{0, 0, 0, 0, WheelBlend{46340, 46340}, false, false, 0, 0},
		{0, 0, 0, 0, WheelBlend{46340, 46340}, true, false, 0, 0},
		{0, 0, 0, 0, WheelBlend{46340, 46340}, false, true, 0, 0},
		{0, 0, 0, 0, WheelBlend{46340, 46340}, true, true, 0, 0},
		{65535, 65535, 1000, 10000, WheelBlend{46340, 46340}, false, false, 431, 431},
		{65535, 65535, 1000, 10000, WheelBlend{46340, 46340}, true, false, -431, 431},
		{65535, 65535, 1000, 10000, WheelBlend{46340, 46340}, false, true, 431, -431},
		{65535, 65535, 1000, 10000, WheelBlend{46340, 46340}, true, true, -431, -431},
		{30000, 40000, 1131, 28000, WheelBlend{46340, 46340}, false, false, 1035, 1380},
		{30000, 40000, 1131, 28000, WheelBlend{46340, 46340}, true, false, -1035, 1380},
		{30000, 40000, 1131, 28000, WheelBlend{46340, 46340}, false, true, 1035, -1380},
		{30000, 40000, 1131, 28000, WheelBlend{46340, 46340}, true, true, -1035, -1380},
		{40000, 40000, 1131, 10000, WheelBlend{46340, 46340}, false, false, 488, 488},
		{40000, 40000, 1131, 10000, WheelBlend{46340, 46340}, true, false, -488, 488},
		{40000, 40000, 1131, 10000, WheelBlend{46340, 46340}, false, true, 488, -488},
		{40000, 40000, 1131, 10000, WheelBlend{46340, 46340}, true, true, -488, -488},
		{0, 50000, 1400, 25000, WheelBlend{46340, 46340}, false, false, 0, 2136},
		{0, 50000, 1400, 25000, WheelBlend{46340, 46340}, true, false, 0, 2136},
		{0, 50000, 1400, 25000, WheelBlend{46340, 46340}, false, true, 0, -2136},
		{0, 50000, 1400, 25000, WheelBlend{46340, 46340}, true, true, 0, -2136},
		{50000, 0, 1400, 25000, WheelBlend{46340, 46340}, false, false, 2136, 0},
		{50000, 0, 1400, 25000, WheelBlend{46340, 46340}, true, false, -2136, 0},
		{50000, 0, 1400, 25000, WheelBlend{46340, 46340}, false, true, 2136, 0},
		{50000, 0, 1400, 25000, WheelBlend{46340, 46340}, true, true, -2136, 0},
		{16011, 19434, 8744, 17810, WheelBlend{15210, 5208}, false, false, 4272, 5185},
		{16619, 14717, 10440, 22170, WheelBlend{27000, 34590}, false, true, 5294, -4688},
		{47538, 29137, 15834, 16960, WheelBlend{39511, 43226}, false, true, 9881, -10810},
		{1592, 40433, 13015, 19850, WheelBlend{32714, 54061}, true, true, -632, -13007},
		{51321, 36900, 19251, 31457, WheelBlend{19283, 20442}, true, true, -10875, -11528},
		{59720, 58511, 5413, 17998, WheelBlend{59383, 53837}, true, false, -5387, 4884},
		{16694, 46461, 5921, 14450, WheelBlend{34114, 13811}, true, false, -2718, 1100},
		{28759, 57288, 18728, 11005, WheelBlend{29876, 33444}, false, true, 5734, -6419},
		{54988, 53600, 11445, 23824, WheelBlend{4890, 31333}, false, false, 1241, 7956},
		{42492, 11782, 9144, 10879, WheelBlend{3360, 12894}, false, true, 311, -1194},
		{71, 48081, 12565, 3023, WheelBlend{38930, 57557}, false, false, 27, 2035},
		{49411, 6454, 1370, 31381, WheelBlend{17033, 4935}, false, false, 2065, 269},
		{52066, 44615, 18971, 8630, WheelBlend{22563, 34315}, false, false, 3440, 5231},
		{14859, 63395, 5585, 6592, WheelBlend{30267, 6008}, true, true, -1037, -205},
		{17467, 23546, 17878, 31735, WheelBlend{23647, 34507}, true, false, -9529, 12846},
		{3665, 21586, 9558, 17534, WheelBlend{27628, 41984}, false, true, 1069, -6296},
		{29763, 51159, 10896, 8938, WheelBlend{8420, 42654}, false, false, 763, 3868},
		{45511, 51768, 18096, 4227, WheelBlend{5033, 53384}, false, true, 358, -3802},
		{45044, 47072, 12950, 25006, WheelBlend{24875, 60727}, true, false, -7501, 18314},
		{32126, 39649, 4357, 27066, WheelBlend{15941, 48103}, true, false, -4271, 5271},
		{38025, 54530, 7539, 30775, WheelBlend{53145, 39954}, false, true, 8748, -8632},
		{3375, 46321, 16019, 32537, WheelBlend{51542, 20631}, false, true, 1649, -22644},
		{59727, 19526, 19102, 21000, WheelBlend{53727, 13152}, true, true, -20071, -4913},
		{59983, 39799, 17074, 268, WheelBlend{13077, 25306}, true, true, -55, -107},
		{14302, 16396, 11294, 4393, WheelBlend{41582, 736}, false, false, 1921, 33},
		{4825, 64998, 4613, 26580, WheelBlend{25126, 11969}, false, false, 679, 1366},
		{54562, 39036, 5836, 3186, WheelBlend{28130, 11563}, false, true, 487, -200},
		{25045, 32102, 4846, 7155, WheelBlend{18784, 8596}, true, false, -606, 277},
		{52331, 49166, 11947, 18473, WheelBlend{33343, 12799}, true, true, -6853, -2630},
		{30677, 11460, 9071, 4211, WheelBlend{28125, 48911}, false, false, 1000, 1739},
		{53858, 1610, 6695, 32265, WheelBlend{42547, 31000}, true, true, -11004, -328},
		{6966, 43984, 17560, 21166, WheelBlend{51606, 8328}, false, false, 3733, 2882},
		{52600, 661, 8063, 13280, WheelBlend{63508, 18750}, true, true, -6333, -162},
		{31303, 25443, 9495, 14875, WheelBlend{6658, 57711}, true, true, -875, -7372},
		{42204, 58334, 5128, 3672, WheelBlend{20685, 39562}, true, false, -362, 693},
		{59916, 36640, 17231, 31192, WheelBlend{47505, 25737}, true, true, -23778, -12882},
	}
	for i, c := range cases {
		lat, long := ResolveWheelForces(c.lateral, c.longitudinal, c.load, c.grip, c.blend, c.negativeLateral, c.negativeLongitudinal)
		if lat != c.forceLateral || long != c.forceLongitudinal {
			t.Fatalf("original wheel forces %d: got %d/%d, want %d/%d", i, lat, long, c.forceLateral, c.forceLongitudinal)
		}
	}
}

func TestOriginalLoadGrip(t *testing.T) {
	cases := []struct{ load, result uint16 }{
		{0, 65416},
		{1, 65416},
		{300, 65291},
		{1000, 64830},
		{1131, 64717},
		{1400, 64459},
		{5000, 57603},
		{10000, 37584},
		{10884, 32774},
		{10885, 0},
		{11585, 0},
		{32767, 0},
		{64835, 0},
		{64836, 65535},
		{65535, 65416},
	}
	for _, c := range cases {
		if result := LoadGrip(c.load); result != c.result {
			t.Fatalf("original load grip %d: got %d, want %d", c.load, result, c.result)
		}
	}
}
