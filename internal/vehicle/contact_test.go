package vehicle

import (
	gameassets "github.com/olivierh59500/indianapolis500/assets"
	"github.com/olivierh59500/indianapolis500/internal/scene"
	"testing"
)

// Each fixture is a complete original $4fe8 call. It checks the preceding
// polar velocity calculation as well as the curve response and skid level.
func TestOriginalWheelContactCurve(t *testing.T) {
	read := func(name string) []byte {
		data, err := gameassets.Original.ReadFile("original/" + name)
		if err != nil {
			t.Fatal(err)
		}
		return data
	}
	world, err := scene.New(read("world-vertices.bin"), read("world-data.bin"), read("world-trigonometry.bin"), read("world-horizon.bin"), read("world-lod.bin"))
	if err != nil {
		t.Fatal(err)
	}
	curve, err := NewContactCurve(read("tire-slip-curve.bin"))
	if err != nil {
		t.Fatal(err)
	}
	cases := []struct {
		speed, angle        int16
		grip, normalization uint16
		motion              WheelMotion
		demand              ContactDemand
	}{
		{6000, 0, 57400, 38000, WheelMotion{6000, 0}, ContactDemand{57400, 0, 0}},
		{6000, 1, 57400, 38000, WheelMotion{6000, 0}, ContactDemand{57400, 0, 0}},
		{6000, 499, 57400, 38000, WheelMotion{5992, 294}, ContactDemand{57400, 25652, 0}},
		{6000, 500, 57400, 38000, WheelMotion{5992, 294}, ContactDemand{57400, 25700, 0}},
		{6000, 501, 57400, 38000, WheelMotion{5992, 294}, ContactDemand{57400, 25748, 0}},
		{6000, 750, 57400, 38000, WheelMotion{5983, 441}, ContactDemand{57400, 36046, 25}},
		{6000, 1769, 57400, 38000, WheelMotion{5911, 1025}, ContactDemand{57400, 57278, 126}},
		{6000, 1770, 57400, 38000, WheelMotion{5911, 1025}, ContactDemand{57400, 57283, 127}},
		{6000, 10000, 57400, 38000, WheelMotion{3454, 4905}, ContactDemand{57400, 35034, 127}},
		{6000, 16383, 57400, 38000, WheelMotion{0, 6000}, ContactDemand{0, 35034, 127}},
		{6000, 16384, 57400, 38000, WheelMotion{0, 6000}, ContactDemand{0, 35034, 127}},
		{6000, 20000, 57400, 38000, WheelMotion{-2022, 5649}, ContactDemand{57400, 35034, 127}},
		{6000, 32767, 57400, 38000, WheelMotion{-6000, 0}, ContactDemand{57400, 0, 0}},
		{6000, -32768, 57400, 38000, WheelMotion{-6000, 0}, ContactDemand{57400, 0, 0}},
		{6000, -1, 57400, 38000, WheelMotion{6000, 0}, ContactDemand{57400, 0, 0}},
		{6000, -500, 57400, 38000, WheelMotion{5992, -295}, ContactDemand{57400, 25700, 0}},
		{6000, -1770, 57400, 38000, WheelMotion{5911, -1026}, ContactDemand{57400, 57283, 127}},
		{6000, -16384, 57400, 38000, WheelMotion{0, -6000}, ContactDemand{0, 35034, 127}},
		{0, 1024, 57400, 38000, WheelMotion{0, 0}, ContactDemand{0, 0, 52}},
		{1, 1024, 57400, 38000, WheelMotion{0, 0}, ContactDemand{0, 0, 52}},
		{10, 1024, 57400, 38000, WheelMotion{9, 0}, ContactDemand{2671, 0, 52}},
		{300, 1024, 57400, 38000, WheelMotion{298, 29}, ContactDemand{57400, 8609, 52}},
		{1000, 1024, 57400, 38000, WheelMotion{995, 98}, ContactDemand{57400, 29093, 52}},
		{5000, 1024, 57400, 38000, WheelMotion{4975, 490}, ContactDemand{57400, 45544, 52}},
		{10000, 1024, 57400, 38000, WheelMotion{9951, 980}, ContactDemand{57400, 45544, 52}},
		{32767, 1024, 57400, 38000, WheelMotion{32609, 3211}, ContactDemand{57400, 45544, 52}},
		{-32768, 1024, 57400, 38000, WheelMotion{-32610, -3212}, ContactDemand{57400, 45544, 52}},
		{15125, 28643, 33506, 49278, WheelMotion{-13974, 5788}, ContactDemand{33506, 21414, 127}},
		{6974, -18612, 44476, 11970, WheelMotion{-1486, -6814}, ContactDemand{44476, 27145, 127}},
		{593, -6221, 43300, 30389, WheelMotion{491, -333}, ContactDemand{43300, 26428, 127}},
		{13047, 5880, 6405, 9071, WheelMotion{11022, 6980}, ContactDemand{6405, 3909, 127}},
		{10652, 25976, 40315, 49057, WheelMotion{-8478, 6450}, ContactDemand{40315, 24606, 127}},
		{4383, 31144, 15291, 25030, WheelMotion{-4332, 669}, ContactDemand{15291, 15055, 112}},
		{12127, -11505, 48973, 47977, WheelMotion{5452, -10833}, ContactDemand{48973, 29890, 127}},
		{10999, 25465, 23931, 10496, WheelMotion{-8417, 7081}, ContactDemand{23931, 14606, 127}},
		{19862, -31773, 19130, 61286, WheelMotion{-19767, -1947}, ContactDemand{19130, 14848, 49}},
		{16283, -8867, 62577, 59057, WheelMotion{10711, -12264}, ContactDemand{62577, 38193, 127}},
		{18102, 18872, 23102, 59299, WheelMotion{-4291, 17586}, ContactDemand{23102, 14100, 127}},
		{24710, 3323, 36318, 7953, WheelMotion{23462, 7752}, ContactDemand{36318, 30533, 127}},
		{15548, 12556, 34906, 44312, WheelMotion{5595, 14506}, ContactDemand{34906, 21304, 127}},
		{11714, -25402, 14246, 45640, WheelMotion{-8917, -7597}, ContactDemand{14246, 8695, 127}},
		{23217, 1169, 60833, 24028, WheelMotion{23075, 2559}, ContactDemand{60833, 52474, 66}},
		{17566, 10402, 54981, 5942, WheelMotion{9488, 14782}, ContactDemand{54981, 33557, 127}},
		{18952, 14768, 35297, 2957, WheelMotion{2895, 18729}, ContactDemand{35297, 21543, 127}},
		{1203, -28828, 3843, 3290, WheelMotion{-1118, -447}, ContactDemand{3843, 2570, 127}},
		{5715, 18862, 9560, 62461, WheelMotion{-1355, 5552}, ContactDemand{9560, 5834, 127}},
		{7741, -1173, 35336, 50797, WheelMotion{7693, -854}, ContactDemand{35336, 30548, 67}},
		{9365, 29447, 60435, 22944, WheelMotion{-8893, 2937}, ContactDemand{60435, 50844, 127}},
		{18984, 31793, 309, 563, WheelMotion{-18905, 1744}, ContactDemand{309, 236, 47}},
		{4760, -22215, 62740, 14678, WheelMotion{-2522, -4038}, ContactDemand{62740, 38293, 127}},
		{11085, 11926, 54409, 19316, WheelMotion{4616, 10078}, ContactDemand{54409, 33208, 127}},
		{9534, 15658, 1524, 52454, WheelMotion{643, 9512}, ContactDemand{1524, 930, 127}},
		{15638, -7649, 6746, 4578, WheelMotion{11587, -10503}, ContactDemand{6746, 4117, 127}},
		{16792, -32536, 43690, 47379, WheelMotion{-16787, -413}, ContactDemand{43690, 9666, 0}},
		{11556, 5637, 50576, 589, WheelMotion{9911, 5940}, ContactDemand{45606, 27333, 127}},
		{1842, 31428, 55796, 64871, WheelMotion{-1827, 236}, ContactDemand{55796, 51880, 84}},
		{18348, -8705, 30654, 45675, WheelMotion{12321, -13596}, ContactDemand{30654, 18709, 127}},
		{754, 3000, 10401, 11683, WheelMotion{722, 214}, ContactDemand{10401, 9700, 127}},
		{23635, -25621, 47273, 63630, WheelMotion{-18271, -14995}, ContactDemand{47273, 28853, 127}},
		{23652, -24702, 44945, 28260, WheelMotion{-16928, -16518}, ContactDemand{44945, 27432, 127}},
		{7116, -9255, 50722, 5833, WheelMotion{4480, -5529}, ContactDemand{50722, 30958, 127}},
		{9013, 12938, 44506, 377, WheelMotion{2932, 8522}, ContactDemand{8635, 25099, 127}},
		{21400, -649, 34786, 22495, WheelMotion{21359, -1312}, ContactDemand{34786, 19331, 14}},
		{3408, 32320, 64482, 53638, WheelMotion{-3405, 146}, ContactDemand{64482, 26073, 0}},
		{14532, -26259, 48387, 18506, WheelMotion{-11778, -8514}, ContactDemand{48387, 29533, 127}},
		{10426, 28305, 12063, 49344, WheelMotion{-9480, 4341}, ContactDemand{12063, 7466, 127}},
		{8535, -6158, 5624, 54324, WheelMotion{7096, -4742}, ContactDemand{5624, 3432, 127}},
		{12141, 30855, 43974, 47641, WheelMotion{-11936, 2222}, ContactDemand{43974, 43973, 127}},
		{11883, 12818, 22357, 46859, WheelMotion{4003, 11188}, ContactDemand{22357, 13645, 127}},
		{2417, 19013, 16917, 1950, WheelMotion{-602, 2340}, ContactDemand{9171, 10325, 127}},
		{18214, 30317, 16108, 64947, WheelMotion{-17722, 4208}, ContactDemand{16108, 16107, 127}},
		{19215, -13611, 53893, 19920, WheelMotion{5011, -18551}, ContactDemand{53893, 32893, 127}},
		{7443, -20656, 31302, 4455, WheelMotion{-2975, -6823}, ContactDemand{31302, 19105, 127}},
		{6265, -30565, 29825, 40142, WheelMotion{-6130, -1298}, ContactDemand{29825, 29824, 127}},
		{16021, 21585, 31231, 9164, WheelMotion{-7639, 14082}, ContactDemand{31231, 19061, 127}},
		{9146, -25918, 46322, 24810, WheelMotion{-7245, -5583}, ContactDemand{46322, 28272, 127}},
		{13598, -20722, 5512, 16415, WheelMotion{-5511, -12432}, ContactDemand{5512, 3364, 127}},
		{1120, 11193, 31279, 7213, WheelMotion{534, 984}, ContactDemand{30091, 19091, 127}},
		{24668, -29975, 10796, 43345, WheelMotion{-23774, -6580}, ContactDemand{10796, 10565, 127}},
		{16270, 114, 42740, 5685, WheelMotion{16269, 199}, ContactDemand{42740, 4646, 0}},
		{11015, 8506, 43895, 24127, WheelMotion{7545, 8023}, ContactDemand{43895, 26791, 127}},
		{16747, 11949, 2058, 45115, WheelMotion{6880, 15267}, ContactDemand{2058, 1256, 127}},
		{17401, 5846, 62580, 31273, WheelMotion{14758, 9218}, ContactDemand{62580, 38195, 127}},
		{896, -14667, 47456, 29007, WheelMotion{147, -884}, ContactDemand{33312, 28964, 127}},
		{11735, 25704, 47687, 62265, WheelMotion{-9163, 7332}, ContactDemand{47687, 29105, 127}},
		{24313, -21975, 16011, 19434, WheelMotion{-12372, -20930}, ContactDemand{16011, 9772, 127}},
		{24018, 2211, 35620, 15210, WheelMotion{23466, 5119}, ContactDemand{35620, 35619, 127}},
		{20486, -27560, 13642, 21605, WheelMotion{-18008, -9768}, ContactDemand{13642, 8326, 127}},
		{21858, -16149, 14717, 41761, WheelMotion{536, -21852}, ContactDemand{14717, 8982, 127}},
		{11085, -5768, 34590, 27649, WheelMotion{9437, -5816}, ContactDemand{34590, 21112, 127}},
		{14999, 14770, 29137, 63337, WheelMotion{2291, 14823}, ContactDemand{29137, 17783, 127}},
	}
	for i, c := range cases {
		velocity := world.VelocityToWorld(c.speed, 0, c.angle)
		motion := WheelMotion{velocity.X, -velocity.Y}
		if motion != c.motion {
			t.Fatalf("original contact velocity%d: got %+v, want %+v", i, motion, c.motion)
		}
		if demand := curve.Demand(motion, c.angle, c.grip, c.normalization); demand != c.demand {
			t.Fatalf("original contact demand%d: got %+v, want %+v", i, demand, c.demand)
		}
	}
}
