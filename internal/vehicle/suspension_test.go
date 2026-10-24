package vehicle

import (
	"bytes"
	"crypto/sha256"
	"encoding/binary"
	"fmt"
	gameassets "github.com/olivierh59500/indianapolis500/assets"
	"github.com/olivierh59500/indianapolis500/internal/scene"
	"testing"
)

// The original complete $2824-$2dac stage supplied both the state and output
// digests. Wheel supports, transfer, tilt, camber, presentation and road effects
// are checked together, rather than treating the springs as isolated equations.
var originalSuspensionConfig = SuspensionConfig{RollStiffness: 18172, PitchStiffness: 10748, RollDistribution: 19153, PitchDistribution: 17164, Springs: [4]uint16{3071, 2559, 2559, 2559}, Camber: [4]int16{-32, -192, -160, -320}}
var originalSuspensionState = SuspensionState{Surface: [4]int16{128, 128, 128, 128}, Roll: [2]int16{0, 0}, Pitch: [2]int16{0, 0}}

func suspensionGeometry(t *testing.T) (*scene.World, *TrackSurface) {
	t.Helper()
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
	if err := world.LoadCameras(read("camera-arctangent.bin"), read("camera-bank-gradient.bin"), read("camera-trackside.bin")); err != nil {
		t.Fatal(err)
	}
	if err := world.LoadTrack(read("track-cosine.bin"), read("track-bank.bin")); err != nil {
		t.Fatal(err)
	}
	if err := world.LoadTrackLimits(read("track-wall-inner.bin"), read("track-wall-outer.bin")); err != nil {
		t.Fatal(err)
	}
	if err := world.LoadBarriers(read("track-barriers.bin"), read("collision-damage.bin")); err != nil {
		t.Fatal(err)
	}
	surface, err := NewTrackSurface(read("track-surface-wave.bin"))
	if err != nil {
		t.Fatal(err)
	}
	return world, &surface
}
func suspensionDigest(t *testing.T, state SuspensionState, output SuspensionOutput) string {
	t.Helper()
	var data bytes.Buffer
	for _, value := range []any{state, output} {
		if err := binary.Write(&data, binary.BigEndian, value); err != nil {
			t.Fatal(err)
		}
	}
	return fmt.Sprintf("%x", sha256.Sum256(data.Bytes()))
}

func TestOriginalSuspensionDuringAcceleration(t *testing.T) {
	cases := []struct {
		input   SuspensionInput
		heading int16
		digest  string
	}{
		{SuspensionInput{Progress: 7005, Lateral: 39, BaseLoads: [4]uint16{1131, 1131, 754, 754}, Grip: [4]uint16{62066, 50358, 50308, 50740}, Rolling: [4]uint16{1310, 1310, 1310, 1310}, LateralForce: 0, LongitudinalForce: 0}, 0, "8d5ba69ffa082c0c913b57ac3b2491227f2aa4de674dffd8f73120a8c94b9074"},
		{SuspensionInput{Progress: 7005, Lateral: 39, BaseLoads: [4]uint16{1131, 1131, 754, 754}, Grip: [4]uint16{62066, 50358, 50308, 50740}, Rolling: [4]uint16{1310, 1310, 1310, 1310}, LateralForce: 0, LongitudinalForce: 658}, 0, "a24ca19c8b41cdf0250f3f28100b753bffddba49d163e879298bf9b84a954ddf"},
		{SuspensionInput{Progress: 7005, Lateral: 39, BaseLoads: [4]uint16{1131, 1131, 754, 754}, Grip: [4]uint16{48866, 50172, 50152, 50923}, Rolling: [4]uint16{1310, 1310, 1310, 1693}, LateralForce: 0, LongitudinalForce: 1172}, 0, "cf80bfd4b3d196ff453b21f83ce4332ac2600db036713311beb6f2ca58b3bc70"},
		{SuspensionInput{Progress: 7005, Lateral: 39, BaseLoads: [4]uint16{1131, 1131, 754, 754}, Grip: [4]uint16{48651, 49960, 49974, 51131}, Rolling: [4]uint16{1310, 1310, 1310, 1693}, LateralForce: 0, LongitudinalForce: 1829}, 0, "6203a423ea4579ebd82fb80577e32d946b58ae4e051a634bf1f847566b69386e"},
		{SuspensionInput{Progress: 7006, Lateral: 39, BaseLoads: [4]uint16{1131, 1131, 754, 754}, Grip: [4]uint16{48374, 49699, 49744, 51026}, Rolling: [4]uint16{1310, 1310, 1310, 1693}, LateralForce: 0, LongitudinalForce: 2495}, 0, "3204e720dde5d9efe90e7b92664148da11ecbdeb3797461a2c3932087301813a"},
		{SuspensionInput{Progress: 7007, Lateral: 39, BaseLoads: [4]uint16{1131, 1131, 754, 754}, Grip: [4]uint16{48084, 49414, 49503, 50776}, Rolling: [4]uint16{1310, 1310, 1310, 1693}, LateralForce: 0, LongitudinalForce: 3118}, 0, "1b0bdb7f146822dd46f0a121d73b097efd37f0eb368bcbc295b48b5c474d42df"},
		{SuspensionInput{Progress: 7008, Lateral: 39, BaseLoads: [4]uint16{1131, 1131, 754, 754}, Grip: [4]uint16{47805, 49128, 49271, 50535}, Rolling: [4]uint16{1310, 1310, 1310, 1693}, LateralForce: 0, LongitudinalForce: 3201}, 0, "cbbdd6ce0e718b682672cbf6466ffa9b49023236a660399a3117e4626a55e169"},
		{SuspensionInput{Progress: 7010, Lateral: 39, BaseLoads: [4]uint16{1133, 1133, 754, 754}, Grip: [4]uint16{47680, 49015, 49158, 50441}, Rolling: [4]uint16{1310, 1310, 1310, 1693}, LateralForce: 0, LongitudinalForce: 3223}, 0, "fe88cfe722486a96d184139c091e1780950a130f2ea3415cb725a1e6f72a01e1"},
		{SuspensionInput{Progress: 7012, Lateral: 39, BaseLoads: [4]uint16{1133, 1133, 754, 754}, Grip: [4]uint16{47652, 48964, 49125, 50400}, Rolling: [4]uint16{1310, 1310, 1310, 1693}, LateralForce: 0, LongitudinalForce: 3235}, 0, "67c288ed2e2142be9fff4ff853aaa5c170ef455e0eef1af368dfe78c748287bd"},
		{SuspensionInput{Progress: 7015, Lateral: 39, BaseLoads: [4]uint16{1135, 1135, 756, 756}, Grip: [4]uint16{47627, 48951, 49104, 50390}, Rolling: [4]uint16{1310, 1310, 1310, 1693}, LateralForce: 0, LongitudinalForce: 3241}, 0, "946d87522540e73b3030f420f8de331b44868c41938f15e7ff7e4bb937998747"},
		{SuspensionInput{Progress: 7017, Lateral: 39, BaseLoads: [4]uint16{1135, 1135, 756, 756}, Grip: [4]uint16{47613, 48940, 49092, 50378}, Rolling: [4]uint16{1310, 1310, 1310, 1693}, LateralForce: 0, LongitudinalForce: 3248}, 0, "b62341f046134928a254feda9d4c380f77a8bf10a7b5dfa6d1589d467251f6e1"},
		{SuspensionInput{Progress: 7021, Lateral: 39, BaseLoads: [4]uint16{1137, 1137, 756, 756}, Grip: [4]uint16{47613, 48939, 49092, 50378}, Rolling: [4]uint16{1310, 1310, 1310, 1693}, LateralForce: 0, LongitudinalForce: 3249}, 0, "ea3050fdb7661a542dc4e73f56ab50730749f860bb52148401dea22fc7c8b75b"},
		{SuspensionInput{Progress: 7024, Lateral: 39, BaseLoads: [4]uint16{1140, 1140, 758, 758}, Grip: [4]uint16{47603, 48937, 49092, 50378}, Rolling: [4]uint16{1310, 1310, 1693, 1693}, LateralForce: 0, LongitudinalForce: 3254}, 0, "7a57e452d6265a770ad864477127900da38f76542c41a39e5932c43018bb48f4"},
		{SuspensionInput{Progress: 7028, Lateral: 39, BaseLoads: [4]uint16{1142, 1142, 758, 758}, Grip: [4]uint16{47601, 48936, 48879, 50171}, Rolling: [4]uint16{1310, 1310, 1693, 1693}, LateralForce: 0, LongitudinalForce: 3280}, 0, "9906a1a38a904b1db17da260d2412c117774cba91ffbcb78550b6de3ba9b73b8"},
		{SuspensionInput{Progress: 7032, Lateral: 39, BaseLoads: [4]uint16{1144, 1144, 760, 760}, Grip: [4]uint16{47587, 48910, 49094, 50368}, Rolling: [4]uint16{1310, 1310, 1693, 1693}, LateralForce: 0, LongitudinalForce: 3237}, 0, "38df47c8da2243715e7e790c7ece0c9115004440b3ee462036bb94c7c606e42a"},
		{SuspensionInput{Progress: 7037, Lateral: 39, BaseLoads: [4]uint16{1146, 1146, 763, 763}, Grip: [4]uint16{47609, 48931, 49104, 50368}, Rolling: [4]uint16{1310, 1310, 1693, 1693}, LateralForce: 0, LongitudinalForce: 3076}, 0, "b5ce95e3d389e450bb7dbf37a6ab1502cb3e1af68bdcd0559abdca1fb6c72158"},
		{SuspensionInput{Progress: 7042, Lateral: 39, BaseLoads: [4]uint16{1149, 1149, 763, 763}, Grip: [4]uint16{47659, 48980, 49144, 50409}, Rolling: [4]uint16{1310, 1310, 1693, 1693}, LateralForce: 0, LongitudinalForce: 3251}, 0, "4f3c1ba139ac06916ac3d059fc8a50f4cf0e5d180c1fbfa0b3ba67467ed81530"},
		{SuspensionInput{Progress: 7047, Lateral: 39, BaseLoads: [4]uint16{1153, 1153, 765, 765}, Grip: [4]uint16{47620, 48951, 49103, 50389}, Rolling: [4]uint16{1310, 1310, 1693, 1693}, LateralForce: 0, LongitudinalForce: 3274}, 0, "878c8c9b1e7cf6c64579dc362e5fc7dc94243984998a3c493996056b243b3605"},
		{SuspensionInput{Progress: 7053, Lateral: 39, BaseLoads: [4]uint16{1155, 1155, 767, 767}, Grip: [4]uint16{47604, 48935, 49092, 50366}, Rolling: [4]uint16{1310, 1310, 1693, 1693}, LateralForce: 0, LongitudinalForce: 3288}, 0, "133603ddb32aaea9648a157ccbb6d690632f3809f1753e623ed03c56842a0bdc"},
		{SuspensionInput{Progress: 7059, Lateral: 39, BaseLoads: [4]uint16{1160, 1160, 769, 769}, Grip: [4]uint16{47601, 48921, 49092, 50366}, Rolling: [4]uint16{1310, 1310, 1693, 1693}, LateralForce: 0, LongitudinalForce: 3296}, 0, "6633af82b22bb4afbcb1e23400e0e650df7c1378c635f18f1bc7cc14b2490581"},
		{SuspensionInput{Progress: 7066, Lateral: 39, BaseLoads: [4]uint16{1162, 1162, 772, 772}, Grip: [4]uint16{47515, 48788, 49079, 50366}, Rolling: [4]uint16{1310, 1310, 1693, 1693}, LateralForce: 0, LongitudinalForce: 2677}, 0, "e00545f91772e99025b4cb10b9786ae7e3bbb15c4c4515b064deaaa5f485f245"},
		{SuspensionInput{Progress: 7073, Lateral: 39, BaseLoads: [4]uint16{1167, 1167, 772, 772}, Grip: [4]uint16{47774, 49093, 49236, 50511}, Rolling: [4]uint16{1310, 1310, 1693, 1693}, LateralForce: 0, LongitudinalForce: 3241}, 0, "4aab2e8dc316dda121f12cd065d749499e8b33f0c8db78c3d2765166e85c237a"},
		{SuspensionInput{Progress: 7080, Lateral: 39, BaseLoads: [4]uint16{1171, 1171, 776, 776}, Grip: [4]uint16{47659, 48989, 49142, 50428}, Rolling: [4]uint16{1310, 1693, 1693, 1693}, LateralForce: 0, LongitudinalForce: 3311}, 0, "5af89e15cdbb30a166d69e63857dc4c82ee55ce8f04e8cf6aa8992a64c01e38c"},
		{SuspensionInput{Progress: 7087, Lateral: 39, BaseLoads: [4]uint16{1176, 1176, 778, 778}, Grip: [4]uint16{47606, 48936, 49108, 50385}, Rolling: [4]uint16{1310, 1693, 1693, 1693}, LateralForce: 0, LongitudinalForce: 3317}, -1, "fb8a38cb0bcebefb9ae504a56287e267a798c9b4abeb45a5619c77b1f2e9ed7b"},
		{SuspensionInput{Progress: 7095, Lateral: 39, BaseLoads: [4]uint16{1180, 1180, 781, 781}, Grip: [4]uint16{47590, 48918, 49086, 50374}, Rolling: [4]uint16{1310, 1693, 1693, 1693}, LateralForce: 0, LongitudinalForce: 3332}, -2, "4f51b1901d6c73ba088e096f7875a96f6e82c367e313b4b3606acafab3286adc"},
		{SuspensionInput{Progress: 7103, Lateral: 39, BaseLoads: [4]uint16{1185, 1185, 783, 783}, Grip: [4]uint16{47574, 48913, 49073, 50350}, Rolling: [4]uint16{1310, 1693, 1693, 1693}, LateralForce: 0, LongitudinalForce: 3346}, -3, "78133597e9828cd23e9141da184d1a805f4888f9976519ac27dc2517c00162b7"},
		{SuspensionInput{Progress: 7112, Lateral: 39, BaseLoads: [4]uint16{1189, 1189, 785, 785}, Grip: [4]uint16{47569, 48898, 49073, 50339}, Rolling: [4]uint16{1310, 1693, 1693, 1693}, LateralForce: 0, LongitudinalForce: 3360}, -4, "19961ce246f3730fc99acf6068776aba0dca1a151c9f4cc56bf8fc13379f2f86"},
		{SuspensionInput{Progress: 7121, Lateral: 39, BaseLoads: [4]uint16{1194, 1194, 790, 790}, Grip: [4]uint16{47566, 48905, 49061, 50348}, Rolling: [4]uint16{1310, 1693, 1693, 1693}, LateralForce: 0, LongitudinalForce: 3371}, -5, "5a5e1145d03fca47653056253c932ab78ac1d1e0ff6d3d55736bfc68fb695d02"},
		{SuspensionInput{Progress: 7130, Lateral: 39, BaseLoads: [4]uint16{1200, 1200, 792, 792}, Grip: [4]uint16{47562, 48890, 49059, 50336}, Rolling: [4]uint16{1310, 1693, 1693, 1693}, LateralForce: 0, LongitudinalForce: 3384}, -6, "6f17f5de3304e7df5364e74b081e901aa134bb478d602d7a1b8c1373cc336535"},
		{SuspensionInput{Progress: 7140, Lateral: 39, BaseLoads: [4]uint16{1205, 1205, 794, 794}, Grip: [4]uint16{47556, 48896, 49046, 50335}, Rolling: [4]uint16{1310, 1693, 1693, 1693}, LateralForce: 0, LongitudinalForce: 3400}, -7, "5e7138fe7078f84506f1b4c10e00e8c3fae0b045daafea3d820ffff231b32604"},
		{SuspensionInput{Progress: 7150, Lateral: 39, BaseLoads: [4]uint16{1212, 1212, 799, 799}, Grip: [4]uint16{47552, 48867, 49058, 50335}, Rolling: [4]uint16{1310, 1693, 1693, 1693}, LateralForce: 0, LongitudinalForce: 3414}, -8, "cee01d4ff30e22605073f498ef4645dc978c8e13749f3299ae8ebc24799671e5"},
		{SuspensionInput{Progress: 7161, Lateral: 39, BaseLoads: [4]uint16{1218, 1218, 801, 801}, Grip: [4]uint16{47547, 48871, 49044, 50333}, Rolling: [4]uint16{1310, 1693, 1693, 1693}, LateralForce: 0, LongitudinalForce: 3432}, -9, "6644149c4435c1de678f260141d132ad3868b4f7f45a8bda8b5d79046c379c92"},
		{SuspensionInput{Progress: 7172, Lateral: 39, BaseLoads: [4]uint16{1225, 1225, 805, 805}, Grip: [4]uint16{47534, 48865, 49229, 50500}, Rolling: [4]uint16{1693, 1693, 1693, 1693}, LateralForce: 0, LongitudinalForce: 3424}, -10, "37a86bd489e0e1615c4a3f514628e8493c431fe1587abe4ed22bc877fd299c4e"},
		{SuspensionInput{Progress: 7183, Lateral: 39, BaseLoads: [4]uint16{1232, 1232, 810, 810}, Grip: [4]uint16{47541, 48872, 49054, 50332}, Rolling: [4]uint16{1693, 1693, 1693, 1693}, LateralForce: 0, LongitudinalForce: 3449}, -10, "f5882feada0d1e1f693c52d85e014077f7b2fae9e72f7f8a3b9f9fa2672f9270"},
		{SuspensionInput{Progress: 7195, Lateral: 39, BaseLoads: [4]uint16{1239, 1239, 812, 812}, Grip: [4]uint16{47535, 48854, 49040, 50317}, Rolling: [4]uint16{1693, 1693, 1693, 1693}, LateralForce: 0, LongitudinalForce: 3470}, -10, "9b34e4d2c8ba8eccdea307d9afbb281ada564aaa877cdaa15ea33d787826178c"},
		{SuspensionInput{Progress: 7207, Lateral: 39, BaseLoads: [4]uint16{1245, 1245, 817, 817}, Grip: [4]uint16{47577, 48965, 49039, 50317}, Rolling: [4]uint16{1693, 1693, 1693, 1693}, LateralForce: 0, LongitudinalForce: 3834}, -10, "57c9eb854eff71cecf7a466ce111d2288506c536a966369fd08a1ca7e433c830"},
		{SuspensionInput{Progress: 7219, Lateral: 39, BaseLoads: [4]uint16{1254, 1254, 821, 821}, Grip: [4]uint16{47421, 48753, 48953, 50220}, Rolling: [4]uint16{1693, 1693, 1693, 1693}, LateralForce: 0, LongitudinalForce: 3546}, -10, "a4d7dcfa76df77f295ce9cff393c93cf0babf7dc421de2c526e8a373b21fb883"},
		{SuspensionInput{Progress: 7232, Lateral: 39, BaseLoads: [4]uint16{1261, 1261, 826, 826}, Grip: [4]uint16{47465, 48805, 48982, 50271}, Rolling: [4]uint16{1693, 1693, 1693, 1693}, LateralForce: 0, LongitudinalForce: 3538}, -10, "71dde2f62e1740e5fa358693b145b08d986eb579ab9e211c9968df6896b43a74"},
		{SuspensionInput{Progress: 7245, Lateral: 39, BaseLoads: [4]uint16{1270, 1270, 830, 830}, Grip: [4]uint16{47484, 48824, 49001, 50280}, Rolling: [4]uint16{1693, 1693, 1693, 1693}, LateralForce: 0, LongitudinalForce: 3551}, -10, "7c9fbe0f7d746a68621731613aca4d076d87feb0da8fd83f9ad9fada07570ac9"},
		{SuspensionInput{Progress: 7259, Lateral: 39, BaseLoads: [4]uint16{1279, 1279, 835, 835}, Grip: [4]uint16{47489, 48829, 49011, 50290}, Rolling: [4]uint16{1693, 1693, 1693, 1693}, LateralForce: 0, LongitudinalForce: 3573}, -10, "0c251cfb5fbe2e85b2e1db637a3c99f707c563d703ae10ae2f0b362ddddc1d8d"},
		{SuspensionInput{Progress: 7273, Lateral: 39, BaseLoads: [4]uint16{1288, 1288, 839, 839}, Grip: [4]uint16{47494, 48823, 49008, 50287}, Rolling: [4]uint16{1693, 1693, 1693, 1693}, LateralForce: 0, LongitudinalForce: 3596}, -10, "5de1476b01a6a6b31289e736a2da7731d2d565d94b52ccdc2d59302ba2c3d8aa"},
		{SuspensionInput{Progress: 7287, Lateral: 39, BaseLoads: [4]uint16{1295, 1295, 846, 846}, Grip: [4]uint16{47486, 48813, 49007, 50286}, Rolling: [4]uint16{1693, 1693, 1693, 1693}, LateralForce: 0, LongitudinalForce: 3621}, -10, "9fb80df50979c9cef7b5673ffd7f448208fd2765cf01be82554a81b04b63f95b"},
		{SuspensionInput{Progress: 7302, Lateral: 39, BaseLoads: [4]uint16{1306, 1306, 850, 850}, Grip: [4]uint16{47459, 48795, 48785, 50072}, Rolling: [4]uint16{1693, 1693, 1693, 1693}, LateralForce: 0, LongitudinalForce: 3668}, -10, "6223760f7696abb0c6d074e7b5cba742df47e7d215214783d5a748ecce6be202"},
		{SuspensionInput{Progress: 7317, Lateral: 39, BaseLoads: [4]uint16{1315, 1315, 855, 855}, Grip: [4]uint16{47448, 48795, 48992, 50272}, Rolling: [4]uint16{1693, 1693, 1693, 1693}, LateralForce: 0, LongitudinalForce: 3670}, -10, "31362f6b067dbbc8aee1dcd8bc7f75dbcfaf2c2a91cae0afa91b1a33b8ee9664"},
		{SuspensionInput{Progress: 7333, Lateral: 39, BaseLoads: [4]uint16{1324, 1324, 859, 859}, Grip: [4]uint16{47376, 48675, 48978, 50270}, Rolling: [4]uint16{1693, 1693, 1693, 1693}, LateralForce: 0, LongitudinalForce: 3070}, -10, "617ceb6a970a714a964e6d664892e189de0d6267969bca95632c0208221b0059"},
		{SuspensionInput{Progress: 7349, Lateral: 39, BaseLoads: [4]uint16{1333, 1333, 866, 866}, Grip: [4]uint16{47623, 48978, 49133, 50409}, Rolling: [4]uint16{1693, 1693, 1693, 1693}, LateralForce: 0, LongitudinalForce: 3646}, -10, "0b5ad8b9665d7f0eee58a2728cf106eb6ca1cb1c0e141699950ab6ea529d0b0c"},
		{SuspensionInput{Progress: 7365, Lateral: 39, BaseLoads: [4]uint16{1344, 1344, 871, 871}, Grip: [4]uint16{47514, 48867, 49048, 50322}, Rolling: [4]uint16{1693, 1693, 1693, 1693}, LateralForce: 0, LongitudinalForce: 3728}, -10, "443820bc9ea462576748fefd0e619e1f8efce56f36fc6974a161d015a4b2d5ab"},
		{SuspensionInput{Progress: 7382, Lateral: 39, BaseLoads: [4]uint16{1353, 1353, 877, 877}, Grip: [4]uint16{47452, 48807, 48978, 50269}, Rolling: [4]uint16{1693, 1693, 1693, 1693}, LateralForce: 0, LongitudinalForce: 3768}, -10, "8658324c5188502b85910176e8410829b902a7bfe1ffe511a95c5795c3b9263b"},
		{SuspensionInput{Progress: 7399, Lateral: 39, BaseLoads: [4]uint16{1365, 1365, 884, 884}, Grip: [4]uint16{47430, 48773, 48965, 50245}, Rolling: [4]uint16{1693, 1693, 1693, 1693}, LateralForce: 0, LongitudinalForce: 3797}, -9, "26ae2791188d0a67c23cefa22da9cdfd5a6545950bbc13022b8f6baa4cbd9f8e"},
		{SuspensionInput{Progress: 7416, Lateral: 39, BaseLoads: [4]uint16{1376, 1376, 891, 891}, Grip: [4]uint16{47407, 48761, 48952, 50244}, Rolling: [4]uint16{1693, 1693, 1693, 1693}, LateralForce: 0, LongitudinalForce: 3830}, -8, "f9048bb35eedfbedc3c7a638efc736754416e64c65ce41cdaf0d11860187f980"},
	}
	world, surface := suspensionGeometry(t)
	suspension := Suspension{Config: originalSuspensionConfig, State: originalSuspensionState}
	for i, c := range cases {
		trig := world.VelocityToWorld(16384, 0, c.heading)
		output := suspension.Step(c.input, trig.X, -trig.Y, surface)
		if got := suspensionDigest(t, suspension.State, output); got != c.digest {
			t.Fatalf("original suspension tick%d: state%+v output%+v digest%s, want%s", i, suspension.State, output, got, c.digest)
		}
	}
}

func TestOriginalSuspensionBoundaryCases(t *testing.T) {
	cases := []struct {
		name    string
		setup   func(*SuspensionState)
		input   SuspensionInput
		heading int16
		digest  string
	}{
		{"lap-start", nil, SuspensionInput{Progress: 0, Lateral: 300, BaseLoads: [4]uint16{1131, 1131, 754, 754}, Grip: [4]uint16{62066, 50358, 50308, 50740}, Rolling: [4]uint16{1310, 1310, 1310, 1310}, LateralForce: 0, LongitudinalForce: 0}, 0, "c849f228bb73c42a00d1a3c6c2a6b72b606d810c707d4316c71c701d34d1ff0c"},
		{"lap-end", nil, SuspensionInput{Progress: 39599, Lateral: 300, BaseLoads: [4]uint16{1131, 1131, 754, 754}, Grip: [4]uint16{62066, 50358, 50308, 50740}, Rolling: [4]uint16{1310, 1310, 1310, 1310}, LateralForce: 0, LongitudinalForce: 0}, 0, "def2ecffdb92d672feebcc40fbf648306792cab1837ae139334d04c58091a343"},
		{"quarter-turn", nil, SuspensionInput{Progress: 7005, Lateral: 275, BaseLoads: [4]uint16{1131, 1131, 754, 754}, Grip: [4]uint16{62066, 50358, 50308, 50740}, Rolling: [4]uint16{1310, 1310, 1310, 1310}, LateralForce: 0, LongitudinalForce: 0}, 16384, "6731ca500089c94fbc6e29a6b8e1c24f5ae001ea0f28a3e330f3322f2a3feeed"},
		{"diagonal", nil, SuspensionInput{Progress: 7005, Lateral: 275, BaseLoads: [4]uint16{1131, 1131, 754, 754}, Grip: [4]uint16{62066, 50358, 50308, 50740}, Rolling: [4]uint16{1310, 1310, 1310, 1310}, LateralForce: 0, LongitudinalForce: 0}, 8192, "5d78c8177f03a3e0ec58183080d39b33a3a938344e9ca0f7b8a4a001d1198c81"},
		{"reverse-heading", nil, SuspensionInput{Progress: 7005, Lateral: 275, BaseLoads: [4]uint16{1131, 1131, 754, 754}, Grip: [4]uint16{62066, 50358, 50308, 50740}, Rolling: [4]uint16{1310, 1310, 1310, 1310}, LateralForce: 0, LongitudinalForce: 0}, -32768, "ae771ebd3a6e6f1010a275b507f4db5725e9cefbc76f6b4bb471257749738e34"},
		{"opposite-quarter", nil, SuspensionInput{Progress: 7005, Lateral: 275, BaseLoads: [4]uint16{1131, 1131, 754, 754}, Grip: [4]uint16{62066, 50358, 50308, 50740}, Rolling: [4]uint16{1310, 1310, 1310, 1310}, LateralForce: 0, LongitudinalForce: 0}, -16384, "f9e46b3ec98365d26ababcde2be063397b385728ab37c03ac856e7b5c4248178"},
		{"pit-island-edge", nil, SuspensionInput{Progress: 3100, Lateral: 250, BaseLoads: [4]uint16{1131, 1131, 754, 754}, Grip: [4]uint16{62066, 50358, 50308, 50740}, Rolling: [4]uint16{1310, 1310, 1310, 1310}, LateralForce: 0, LongitudinalForce: 0}, 0, "1668d3a85a620989f8290d84c1f65201a62c2c08d8e864ac81d4878c391a5e1b"},
		{"grass", nil, SuspensionInput{Progress: 7005, Lateral: 275, BaseLoads: [4]uint16{1131, 1131, 754, 754}, Grip: [4]uint16{62066, 50358, 50308, 50740}, Rolling: [4]uint16{1310, 1310, 1310, 1310}, LateralForce: 0, LongitudinalForce: 0}, 0, "0b5cc98eb89a5a539840c1c77747ca397c7d280c92055cdafe5848f8b8353dd0"},
		{"negative-lateral", nil, SuspensionInput{Progress: 7005, Lateral: -10, BaseLoads: [4]uint16{1131, 1131, 754, 754}, Grip: [4]uint16{62066, 50358, 50308, 50740}, Rolling: [4]uint16{1310, 1310, 1310, 1310}, LateralForce: 0, LongitudinalForce: 0}, 0, "f19ddb47e323bab3abf516ddfb2b1fbce987a7420292e74d6649e6221000059b"},
		{"roll-transfer", nil, SuspensionInput{Progress: 7005, Lateral: 39, BaseLoads: [4]uint16{1131, 1131, 754, 754}, Grip: [4]uint16{62066, 50358, 50308, 50740}, Rolling: [4]uint16{1310, 1310, 1310, 1310}, LateralForce: 8000, LongitudinalForce: 0}, 0, "1a6122d89e71b3750f30eef221082eafeafb94a7baae01e3eaac42def78e4973"},
		{"pitch-transfer", nil, SuspensionInput{Progress: 7005, Lateral: 39, BaseLoads: [4]uint16{1131, 1131, 754, 754}, Grip: [4]uint16{62066, 50358, 50308, 50740}, Rolling: [4]uint16{1310, 1310, 1310, 1310}, LateralForce: 0, LongitudinalForce: 12000}, 0, "d8afdc6f194b151b87e371d63248a894debedd27ab45db00b7042dc43f4c38c2"},
		{"opposite-forces", nil, SuspensionInput{Progress: 7005, Lateral: 39, BaseLoads: [4]uint16{1131, 1131, 754, 754}, Grip: [4]uint16{62066, 50358, 50308, 50740}, Rolling: [4]uint16{1310, 1310, 1310, 1310}, LateralForce: -16000, LongitudinalForce: -16000}, 0, "c0a277f949a315726af92172b4df72fae51d7048afb39bd41d6af595c06b2eac"},
		{"no-base-load", nil, SuspensionInput{Progress: 7005, Lateral: 39, BaseLoads: [4]uint16{0, 0, 0, 0}, Grip: [4]uint16{62066, 50358, 50308, 50740}, Rolling: [4]uint16{1310, 1310, 1310, 1310}, LateralForce: 0, LongitudinalForce: 0}, 0, "1a90159d381fe9c52df2eb90d364dcfa257014a4bf5503b154c6bd464ce2f36c"},
		{"unequal-base-loads", nil, SuspensionInput{Progress: 7005, Lateral: 39, BaseLoads: [4]uint16{200, 4000, 1200, 800}, Grip: [4]uint16{62066, 50358, 50308, 50740}, Rolling: [4]uint16{1310, 1310, 1310, 1310}, LateralForce: 0, LongitudinalForce: 0}, 0, "30bdabce7d32092d4f7516e31419d396d281c1f3ff7db95899b002f11786aefb"},
		{"support-step", func(s *SuspensionState) { s.Surface = [4]int16{-512, 256, 128, -256} }, SuspensionInput{Progress: 7005, Lateral: 39, BaseLoads: [4]uint16{1131, 1131, 754, 754}, Grip: [4]uint16{62066, 50358, 50308, 50740}, Rolling: [4]uint16{1310, 1310, 1310, 1310}, LateralForce: 0, LongitudinalForce: 0}, 0, "129909d3be9e6b4902b1259111fc221cc3cc1c06d7d5053a593fb6688032f321"},
		{"existing-transfer", func(s *SuspensionState) { s.Roll = [2]int16{100, 200}; s.Pitch = [2]int16{-100, 200} }, SuspensionInput{Progress: 7005, Lateral: 39, BaseLoads: [4]uint16{1131, 1131, 754, 754}, Grip: [4]uint16{62066, 50358, 50308, 50740}, Rolling: [4]uint16{1310, 1310, 1310, 1310}, LateralForce: 0, LongitudinalForce: 0}, 0, "18fb8816523f8678915cb4f77f8ad969ecf3cdede01925cec617c4e811c52ee1"},
		{"grip-and-rolling-wrap", nil, SuspensionInput{Progress: 7005, Lateral: 275, BaseLoads: [4]uint16{1131, 1131, 754, 754}, Grip: [4]uint16{65535, 60000, 50000, 40000}, Rolling: [4]uint16{65500, 65500, 65500, 65500}, LateralForce: 0, LongitudinalForce: 0}, 0, "79d0b0d144dc04707f779037fcc077b5a8a9e7ec6d3336fb6de14414b70fe700"},
	}
	world, surface := suspensionGeometry(t)
	for _, c := range cases {
		suspension := Suspension{Config: originalSuspensionConfig, State: originalSuspensionState}
		if c.setup != nil {
			c.setup(&suspension.State)
		}
		trig := world.VelocityToWorld(16384, 0, c.heading)
		output := suspension.Step(c.input, trig.X, -trig.Y, surface)
		if got := suspensionDigest(t, suspension.State, output); got != c.digest {
			t.Fatalf("%s: state%+v output%+v digest%s, want%s", c.name, suspension.State, output, got, c.digest)
		}
	}
}
