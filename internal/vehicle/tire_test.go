package vehicle

import (
	"bytes"
	"crypto/sha256"
	"encoding/binary"
	"fmt"
	"testing"
)

// Initial states and every expected digest come from the four original tire
// blocks inside complete physics interrupts. The digest includes all state
// fields; grip and puncture events are verified independently.
var originalTireConfigs = [4]TireConfig{
	TireConfig{HeatGain: 11034, Cooling: 785, OptimumTemperature: 22368, ColdPressure: 10240, CamberDivisor: 3604, PressureDivisor: 32000, ReferencePressure: 20000, GripReference: 10000},
	TireConfig{HeatGain: 8220, Cooling: 785, OptimumTemperature: 22688, ColdPressure: 10240, CamberDivisor: 3604, PressureDivisor: 32000, ReferencePressure: 20000, GripReference: 10000},
	TireConfig{HeatGain: 15998, Cooling: 785, OptimumTemperature: 22368, ColdPressure: 10240, CamberDivisor: 3604, PressureDivisor: 32000, ReferencePressure: 20000, GripReference: 10000},
	TireConfig{HeatGain: 11918, Cooling: 785, OptimumTemperature: 22688, ColdPressure: 10240, CamberDivisor: 3604, PressureDivisor: 32000, ReferencePressure: 20000, GripReference: 10000},
}
var originalTireStates = [4]TireState{
	TireState{Life: 45875200, HeatTick: 3, PressureTick: 33, TemperatureGrip: 65535, RollingResistance: 1310, Pressure: 11612, PressureError: 0, Bands: [3]uint16{19488, 19488, 19488}, Failed: false},
	TireState{Life: 45875200, HeatTick: 1, PressureTick: 23, TemperatureGrip: 65535, RollingResistance: 1310, Pressure: 11612, PressureError: 0, Bands: [3]uint16{19488, 19488, 19488}, Failed: false},
	TireState{Life: 55705600, HeatTick: 8, PressureTick: 13, TemperatureGrip: 59773, RollingResistance: 1310, Pressure: 11612, PressureError: -8388, Bands: [3]uint16{19484, 19483, 19482}, Failed: false},
	TireState{Life: 55705600, HeatTick: 6, PressureTick: 3, TemperatureGrip: 59133, RollingResistance: 1310, Pressure: 11612, PressureError: -8388, Bands: [3]uint16{19484, 19483, 19482}, Failed: false},
}

func tireStateDigest(t *testing.T, state TireState) string {
	t.Helper()
	var data bytes.Buffer
	if err := binary.Write(&data, binary.BigEndian, state); err != nil {
		t.Fatal(err)
	}
	return fmt.Sprintf("%x", sha256.Sum256(data.Bytes()))
}

func TestOriginalTiresDuringAcceleration(t *testing.T) {
	cases := [][4]struct {
		input     TireInput
		digest    string
		grip      uint16
		punctured bool
	}{
		{
			{TireInput{Energy: 0, Camber: 213, Ambient: 17184, Speed: 0, Load: 1131}, "b77c66c7b9ebc51c0579dc6e959f8dad4c2f9b275ac753647819533e7a542299", 62066, false},
			{TireInput{Energy: 0, Camber: 50, Ambient: 17184, Speed: 0, Load: 1131}, "1353050a45aa25d8a2179753aec9d4a88e60759da6b0a7f4950d93ed0108ea4f", 50358, false},
			{TireInput{Energy: 0, Camber: 123, Ambient: 17184, Speed: 0, Load: 754}, "3bd5880b2d91abd5847c30faac9705138ebdca3a1cde9f893953ab91433c8832", 50308, false},
			{TireInput{Energy: 0, Camber: -37, Ambient: 17184, Speed: 0, Load: 754}, "cf9bc53dea0d26180fff6d86a581b25a35c6bb347a744a3fbb3e4280401e1f59", 50740, false},
		},
		{
			{TireInput{Energy: 0, Camber: 213, Ambient: 17184, Speed: 19, Load: 1131}, "ed7d959f3c1a2cb1ac95fb4ccdae1d9c85ec57afe09d33ce6f82a7ac41d3b51b", 62066, false},
			{TireInput{Energy: 0, Camber: 50, Ambient: 17184, Speed: 19, Load: 1131}, "2981edb9b3487ea903bfaf7959506a9ea5da8e5799714869750dc71a5e50dda1", 50358, false},
			{TireInput{Energy: 0, Camber: 123, Ambient: 17184, Speed: 19, Load: 754}, "339fb51fe912622cdc01a8407a29211a9494bb0b44a21a3510ae1192a9902b62", 50308, false},
			{TireInput{Energy: 0, Camber: -37, Ambient: 17184, Speed: 19, Load: 754}, "ed8a5e0b7b5e616f26abad0d7a977fc8c220504438152d8344caf0beba129b36", 50740, false},
		},
		{
			{TireInput{Energy: 0, Camber: 228, Ambient: 17184, Speed: 52, Load: 1158}, "bba6200fff294adc6a0a520819f677365709a43d73f9a6b45a634da8f8f74cb1", 48866, false},
			{TireInput{Energy: 0, Camber: 65, Ambient: 17184, Speed: 52, Load: 1157}, "c818fa3cbaf7dea228b607f8d4ea289f9acb8f418ea8b108907742e426067714", 50172, false},
			{TireInput{Energy: 0, Camber: 138, Ambient: 17184, Speed: 52, Load: 727}, "f6f23888bb5dfade471e09a87958aea1a067677c276526c7fa88f4d1c476b29c", 50152, false},
			{TireInput{Energy: 0, Camber: -22, Ambient: 17184, Speed: 52, Load: 728}, "3ec3af84e8d90c170a141b0c89751b9e1bd9561e36eb26bb33b3091ac96c7394", 50923, false},
		},
		{
			{TireInput{Energy: 0, Camber: 245, Ambient: 17184, Speed: 104, Load: 1189}, "cb8f3a1bd877c6834cc9586453821e1346d841e3b4459a21a076ebf67bd8dba6", 48651, false},
			{TireInput{Energy: 0, Camber: 82, Ambient: 17184, Speed: 104, Load: 1186}, "fe7d1d7cf98ed287b1dd8273a3c1a5981d06278bbee493d52353b4d5b53223b7", 49960, false},
			{TireInput{Energy: 0, Camber: 155, Ambient: 17184, Speed: 104, Load: 696}, "8902af78338f0e92eed86662209c9d8648477e445ba54efde5bd2f8a1860ab5c", 49974, false},
			{TireInput{Energy: 0, Camber: -5, Ambient: 17184, Speed: 104, Load: 699}, "4ef5c0fbd888855fb53878d1c6a13b8b9b670faedb9ccfd34a6b1615f8e27148", 51131, false},
		},
		{
			{TireInput{Energy: 0, Camber: 267, Ambient: 17184, Speed: 175, Load: 1228}, "f803e389d1e404ac6d3a95d977d21eb2386f8e522f7c1fc6c6f9c4f300d9c2d9", 48374, false},
			{TireInput{Energy: 0, Camber: 103, Ambient: 17184, Speed: 175, Load: 1222}, "70914aa31a68aca840072f4fddfba646ff9822b614c6c85a8afdac7163e426de", 49699, false},
			{TireInput{Energy: 0, Camber: 177, Ambient: 17184, Speed: 175, Load: 657}, "862d23c4d982399cf5e7faa9453ad6b2de6744871ee832bb91405bc3679a6b22", 49744, false},
			{TireInput{Energy: 0, Camber: 16, Ambient: 17184, Speed: 175, Load: 663}, "d2da9d173466eb57ba65e26c6f773c20fd2dfd6d94e2bfe6e0b4e51612fcf2ff", 51026, false},
		},
		{
			{TireInput{Energy: 1, Camber: 290, Ambient: 17184, Speed: 264, Load: 1270}, "16978627500792a4655110cc5fd7c75c199d7d3f624548ba59964ecd302b4d5b", 48084, false},
			{TireInput{Energy: 1, Camber: 126, Ambient: 17184, Speed: 264, Load: 1260}, "747a17322ec95a760d16b4deaa9688497939e1c6a65fc52f7f025fd86848dc11", 49414, false},
			{TireInput{Energy: 0, Camber: 200, Ambient: 17184, Speed: 264, Load: 615}, "f83d7281f3a6e2e6cf9dc5b46cad6a7539ee29f30aff6ea61abaa32053cf47e7", 49503, false},
			{TireInput{Energy: 0, Camber: 39, Ambient: 17184, Speed: 264, Load: 625}, "3077a772360416599b3da8411826633a43e813c0e40d1ca87e6d5fce7eeb6436", 50776, false},
		},
		{
			{TireInput{Energy: 1, Camber: 312, Ambient: 17184, Speed: 356, Load: 1311}, "6b6069c14817775309e4eca126963dfcaf2a2ba46a08ebcf0e423afed5d0946f", 47805, false},
			{TireInput{Energy: 1, Camber: 149, Ambient: 17184, Speed: 356, Load: 1298}, "c2ce7dc390bd629f43f34cdee3ab5c4057fcc31ae0da4e794aa6db0798ce8a59", 49128, false},
			{TireInput{Energy: 0, Camber: 222, Ambient: 17184, Speed: 356, Load: 574}, "bdc008e4effb9bc8971717d63d58eb039bce0cbde5872ef6efc996b54c5a66eb", 49271, false},
			{TireInput{Energy: 0, Camber: 62, Ambient: 17184, Speed: 356, Load: 587}, "b08566ceb3b3c0d703cf3bc3ef24307ed7dbb786745e4d6f0c7e766dda788d97", 50535, false},
		},
		{
			{TireInput{Energy: 2, Camber: 322, Ambient: 17184, Speed: 448, Load: 1328}, "e9bf2c9e30804e2636f11805bcb09427272525e31d62c923fc7dae8e72d0238d", 47680, false},
			{TireInput{Energy: 2, Camber: 158, Ambient: 17184, Speed: 448, Load: 1314}, "4db01949a0f3172842910206f7e850ba04cf5958991056e809a5f41f80a738fb", 49015, false},
			{TireInput{Energy: 0, Camber: 232, Ambient: 17184, Speed: 448, Load: 557}, "d47337e0d2675a76dbf5eef32cf194cc9a9751f59e33de3170fdc363aaccf0f5", 49158, false},
			{TireInput{Energy: 1, Camber: 71, Ambient: 17184, Speed: 448, Load: 571}, "30984b7f2383a8c5f7ebc9a93ea05f0c16bbe646d638b92779a62adf45f1b095", 50441, false},
		},
		{
			{TireInput{Energy: 2, Camber: 324, Ambient: 17184, Speed: 541, Load: 1336}, "bc7cbf1ee9cac132484b8618f39ad3f584664db49ce2afe1ddad1b055f4513fd", 47652, false},
			{TireInput{Energy: 2, Camber: 162, Ambient: 17184, Speed: 541, Load: 1322}, "aca6e5c41ee2413b409966b5a04e99edb2fec100a2cb749e98a2005060a9778d", 48964, false},
			{TireInput{Energy: 1, Camber: 235, Ambient: 17184, Speed: 541, Load: 551}, "25f97bb9eeb3c278b9e31c90325ec297df25623bcb75399a0e705e757c5b0182", 49125, false},
			{TireInput{Energy: 1, Camber: 75, Ambient: 17184, Speed: 541, Load: 565}, "b93cc620bee9760fd887b09192d34ca4df362e4b9efdd6f422e50bb5c2dd99ae", 50400, false},
		},
		{
			{TireInput{Energy: 3, Camber: 326, Ambient: 17184, Speed: 634, Load: 1339}, "43c08b5e8814258b0bac2f1bd723ecb9faf9da371d4550d0a175a7842a702a93", 47627, false},
			{TireInput{Energy: 3, Camber: 163, Ambient: 17184, Speed: 634, Load: 1325}, "233a9f93d3e6a132a3cf31c466d3eb7ee84adbc6a60b8a6dfb10e5bde0bf8b9d", 48951, false},
			{TireInput{Energy: 1, Camber: 237, Ambient: 17184, Speed: 634, Load: 548}, "e97eeb5e7335d06be815838609c013fef73039fd90fce36fe1dcc38830342563", 49104, false},
			{TireInput{Energy: 1, Camber: 76, Ambient: 17184, Speed: 634, Load: 562}, "5b0b0321e8cb9fc50f4bbf53e8765c7a247be66d37cfdaba9568e0118d5ec924", 50390, false},
		},
		{
			{TireInput{Energy: 4, Camber: 327, Ambient: 17184, Speed: 727, Load: 1342}, "2113c0cabbd45c9d76c7e17b1b0974a0af4c3e1e783ea1e9ef278177b8d2c777", 47613, false},
			{TireInput{Energy: 4, Camber: 163, Ambient: 17184, Speed: 727, Load: 1329}, "9ede986e2e67b063a35ce2a2e2d3bffd57b0b1d65d1bf2e8ee167e0eebb0674a", 48940, false},
			{TireInput{Energy: 1, Camber: 238, Ambient: 17184, Speed: 727, Load: 549}, "f161c768f50574ffa730f031d19543010248b18e07fe03ca31730235c0a28854", 49092, false},
			{TireInput{Energy: 2, Camber: 77, Ambient: 17184, Speed: 727, Load: 562}, "a4747cc62d84e256b117afaff3652950ed48306290d96478c0c43c65e35f2857", 50378, false},
		},
		{
			{TireInput{Energy: 4, Camber: 327, Ambient: 17184, Speed: 820, Load: 1342}, "f8a28d7437de40791b9033d1e86973174ae18cf58d4bba6aeb9cae14872074dd", 47613, false},
			{TireInput{Energy: 4, Camber: 163, Ambient: 17184, Speed: 820, Load: 1330}, "06f566148cd9360469bbca5733e0fdfa1bbf4f0f45cd1a84bd00b92ba7a15f2a", 48939, false},
			{TireInput{Energy: 1, Camber: 238, Ambient: 17184, Speed: 820, Load: 549}, "bc149ccb86e49f058ddc4a10fe169bc8df5285c33461e64ede22171e5c754a10", 49092, false},
			{TireInput{Energy: 2, Camber: 77, Ambient: 17184, Speed: 820, Load: 561}, "4ca9805b7a9f5e24ce4b21adc52a901916bfb8bcfcf6a7cb0f6ea2d19b9092a8", 50378, false},
		},
		{
			{TireInput{Energy: 5, Camber: 327, Ambient: 17184, Speed: 913, Load: 1344}, "d65338b34bf7cb45fb8dbae7e5383db74b3905b33ae5897fee4ba572334e5567", 47603, false},
			{TireInput{Energy: 5, Camber: 163, Ambient: 17184, Speed: 913, Load: 1332}, "c777ffbeede13dbe7fc212aa74930c026fa51d207ed5d5a65ed39b68dbb88604", 48937, false},
			{TireInput{Energy: 2, Camber: 238, Ambient: 17184, Speed: 913, Load: 549}, "9a537285a6c0c2a18ba40dac8d726139c519ed9b0de3da73a6159044bcc6331f", 49092, false},
			{TireInput{Energy: 2, Camber: 77, Ambient: 17184, Speed: 913, Load: 561}, "da1d94381c28747a69bf27216c6aee09ea5b6327757a8b911f0c989b53755f35", 50378, false},
		},
		{
			{TireInput{Energy: 5, Camber: 327, Ambient: 17184, Speed: 1007, Load: 1347}, "87bd53715c4bab3e6e3c26c91f0fb01b03cc39fb1bd8c0fb22e17ccf24234ad3", 47601, false},
			{TireInput{Energy: 5, Camber: 163, Ambient: 17184, Speed: 1007, Load: 1335}, "0b67a9cb1874c77718a4a8de1e2a3554950e932a00899455fc29842554279112", 48936, false},
			{TireInput{Energy: 1, Camber: 266, Ambient: 17184, Speed: 1007, Load: 295}, "74390c92ec6c8c5faba107a01186c80cdced11eb150cd3a249a9cf79beabcf26", 48879, false},
			{TireInput{Energy: 1, Camber: 105, Ambient: 17184, Speed: 1007, Load: 307}, "037a15c80ebd30d99663277a90ca909eced0a86591d5c53a4e4d78cf7d98ef14", 50171, false},
		},
		{
			{TireInput{Energy: 6, Camber: 328, Ambient: 17184, Speed: 1100, Load: 1351}, "4a4815f1ddb55c05ff3f7f81aae33dd2de2f0d742b73a8b71951e5fc157cec88", 47587, false},
			{TireInput{Energy: 6, Camber: 165, Ambient: 17184, Speed: 1100, Load: 1339}, "59a35052127994182fd869d903266d34b26cb63d09f528b78d1b174bb33723ed", 48910, false},
			{TireInput{Energy: 3, Camber: 238, Ambient: 17184, Speed: 1100, Load: 549}, "4bfcae373eb7200048839ff68a3f3773468ec39bf55a69fd42a282b284d0f2d7", 49094, false},
			{TireInput{Energy: 3, Camber: 78, Ambient: 17184, Speed: 1100, Load: 561}, "c813d8f78c920a45d0f32736f4b9d0a715cd40a6114fe1af0274ab368ecea422", 50368, false},
		},
		{
			{TireInput{Energy: 7, Camber: 326, Ambient: 17184, Speed: 1188, Load: 1352}, "6a7712603326354e25b7b67dc5124fa2d9192e7ba6d1427c1f719f9d47a92743", 47609, false},
			{TireInput{Energy: 7, Camber: 163, Ambient: 17184, Speed: 1188, Load: 1340}, "2b9002da21decb4a861f5d62cae67447137baa2d9e7ff791083f158a4346db54", 48931, false},
			{TireInput{Energy: 3, Camber: 237, Ambient: 17184, Speed: 1188, Load: 552}, "db2bcf46ff6f0b96898f4759fbda28ec51245d167648700cc6dd0228437ee650", 49104, false},
			{TireInput{Energy: 3, Camber: 77, Ambient: 17184, Speed: 1188, Load: 564}, "731f897163d644796873277213c33db337c96a145f0f9f4259e9f30e05a41b5f", 50368, false},
		},
		{
			{TireInput{Energy: 7, Camber: 322, Ambient: 17184, Speed: 1281, Load: 1346}, "5f9ab7bcd2d529b330c80ebac4948b4ccf9ae17d6ee313b9ae9d384b76e26d3c", 47659, false},
			{TireInput{Energy: 7, Camber: 159, Ambient: 17184, Speed: 1281, Load: 1335}, "342014b3ed8f424dae28c08bbb29d6cbef50443084e50e0da33c7c7655005795", 48980, false},
			{TireInput{Energy: 4, Camber: 233, Ambient: 17184, Speed: 1281, Load: 563}, "add3bf06207256466312d2b975d1f4796a310d540f2f41bdc4babc13cb0a8906", 49144, false},
			{TireInput{Energy: 4, Camber: 73, Ambient: 17184, Speed: 1281, Load: 574}, "2258ffed72659186667d434d488a40478c5acae6ad08959d5c733da1731a3d3d", 50409, false},
		},
		{
			{TireInput{Energy: 8, Camber: 325, Ambient: 17184, Speed: 1375, Load: 1353}, "8622b905288290ad71b749e878fae78531bbb74a0371d66046f7b734f2fc87da", 47620, false},
			{TireInput{Energy: 8, Camber: 161, Ambient: 17184, Speed: 1375, Load: 1343}, "5277d19ba3c460d4136349325f90bf691ab8f563b6b0825841011b584f269074", 48951, false},
			{TireInput{Energy: 4, Camber: 236, Ambient: 17184, Speed: 1375, Load: 559}, "0b0d2e848403a55e67804717edd600de71866b582ac112f2e882dcbae073303a", 49103, false},
			{TireInput{Energy: 4, Camber: 75, Ambient: 17184, Speed: 1375, Load: 569}, "cd771957d35b0472484e0be172393d22d27d61af7f77b81933d45313e6e0e99f", 50389, false},
		},
		{
			{TireInput{Energy: 9, Camber: 326, Ambient: 17184, Speed: 1469, Load: 1359}, "d0cd1e4ff34ef6a1afe84fd06e0229da19588124a3d34ed381d2f8a7a0d7064a", 47604, false},
			{TireInput{Energy: 8, Camber: 162, Ambient: 17184, Speed: 1469, Load: 1350}, "c0e5065702cc04b2e63d7d2abf16a9c6b64a203ad1b8fab18d6b20d844bbe061", 48935, false},
			{TireInput{Energy: 4, Camber: 237, Ambient: 17184, Speed: 1469, Load: 559}, "88597c1c8bade4e8ef78432679f0a419a94941dcd270077543b6e39fae78cf9e", 49092, false},
			{TireInput{Energy: 4, Camber: 77, Ambient: 17184, Speed: 1469, Load: 568}, "a253f4cc0dfa25a2d0a103c197d5d38ea2e075f57abd7b446c59250e73fdd996", 50366, false},
		},
		{
			{TireInput{Energy: 9, Camber: 326, Ambient: 17184, Speed: 1564, Load: 1363}, "0b4e40819522baca3e4538b0add93eba53b44499e01820afec98e51a8c2438b3", 47601, false},
			{TireInput{Energy: 9, Camber: 163, Ambient: 17184, Speed: 1564, Load: 1354}, "97c5ac410752781034d823824bdad98a610f9c786dd5d87a38476780b8115d3f", 48921, false},
			{TireInput{Energy: 5, Camber: 237, Ambient: 17184, Speed: 1564, Load: 559}, "a70a1f0034900c5977de08b5e5758faa2b2f157e18f1e04c880fbde88b5c01ee", 49092, false},
			{TireInput{Energy: 5, Camber: 77, Ambient: 17184, Speed: 1564, Load: 568}, "c7f3ff8373ea87f35e9c8e7aa1210070abce560e35bd40a9e480ab8b9c0bf876", 50366, false},
		},
		{
			{TireInput{Energy: 8, Camber: 349, Ambient: 17184, Speed: 1641, Load: 1112}, "41ad1bb99e16c4abfa6ed819293eff66b748897ac0ce1b9206dc92125c1b1a60", 47515, false},
			{TireInput{Energy: 8, Camber: 190, Ambient: 17184, Speed: 1641, Load: 1104}, "95213feb1c9426bbfd09223e4aa38352da391b26e91ffdd9754b9dabeb072952", 48788, false},
			{TireInput{Energy: 5, Camber: 238, Ambient: 17184, Speed: 1641, Load: 561}, "e5b7df73fdc6efda7025fe6a91a02ef56d37cdbef8da64b479747953d6d78558", 49079, false},
			{TireInput{Energy: 5, Camber: 77, Ambient: 17184, Speed: 1641, Load: 569}, "a490ca1e5ad547daa17ea489eada558083c54846fa5e5264418fda2f84647287", 50366, false},
		},
		{
			{TireInput{Energy: 10, Camber: 312, Ambient: 17184, Speed: 1734, Load: 1343}, "6d35af5e159ebc2dfd07bc1c807bd025cbd3cacf98989e8116cee3dbec04ea38", 47774, false},
			{TireInput{Energy: 10, Camber: 148, Ambient: 17184, Speed: 1734, Load: 1338}, "16859b1e091775a3a7db66cc13b94d8124326c0b8a94f80a930351a2eb43715a", 49093, false},
			{TireInput{Energy: 6, Camber: 223, Ambient: 17184, Speed: 1734, Load: 591}, "34a07440cfd72d08601713a1de2addf87c2fc49266387890db9815656b2679c4", 49236, false},
			{TireInput{Energy: 6, Camber: 63, Ambient: 17184, Speed: 1734, Load: 596}, "5def4c988203118fe4eb344d6925e99dc0b15d710b3c49b8db572174189df117", 50511, false},
		},
		{
			{TireInput{Energy: 11, Camber: 320, Ambient: 17184, Speed: 1829, Load: 1364}, "d858061e89511f9d43022c5c0c7f6f5eb9b213a7772a4fcd67a070dcf985d633", 47659, false},
			{TireInput{Energy: 11, Camber: 156, Ambient: 17184, Speed: 1829, Load: 1357}, "b3cd7d486ba9f83d6f8276339ac11ff2a160c391ddb8ac3e38629f4f8c0b3ca9", 48989, false},
			{TireInput{Energy: 6, Camber: 232, Ambient: 17184, Speed: 1829, Load: 575}, "8b0ca1b22df1a244085fef4eb4fcc9e1c47e4cd0aa7452c35b344e77d7877fcc", 49142, false},
			{TireInput{Energy: 6, Camber: 71, Ambient: 17184, Speed: 1829, Load: 582}, "03172bd380da501da6ed460382a7a31389c6ceb8a956700b77fecc8b8dbd3fa2", 50428, false},
		},
		{
			{TireInput{Energy: 12, Camber: 324, Ambient: 17184, Speed: 1923, Load: 1376}, "02b045a8419ef622b3c08e6a89ce0f629b9e74f1faa0f0acbd771fc5c6dfd96c", 47606, false},
			{TireInput{Energy: 15, Camber: 160, Ambient: 17184, Speed: 1923, Load: 1369}, "4637f3b9a906d503fa91150415472f0acde4b5c4ee707a6b2a72df3f6c1ab091", 48936, false},
			{TireInput{Energy: 6, Camber: 235, Ambient: 17184, Speed: 1923, Load: 571}, "a1b3a8742cd587873eff6936ea12a64641e4f7cee5d1628c3d2659ebaae900b1", 49108, false},
			{TireInput{Energy: 6, Camber: 75, Ambient: 17184, Speed: 1923, Load: 578}, "07fd27bd1faee52abc1323ce5662bfa8f0503723460aab2b104ef7adaf6e26ac", 50385, false},
		},
		{
			{TireInput{Energy: 12, Camber: 325, Ambient: 17184, Speed: 2018, Load: 1383}, "57d6422eac34dd57dd3d609feaed30f0492e4fee1345d9758943425d54b7b0e1", 47590, false},
			{TireInput{Energy: 16, Camber: 161, Ambient: 17184, Speed: 2018, Load: 1377}, "4791f778b54dab6fab7456a863097dda3ede2f6ff89ddd644565058f06d3b421", 48918, false},
			{TireInput{Energy: 6, Camber: 237, Ambient: 17184, Speed: 2018, Load: 571}, "e709e28bdfe8b8fce9afddc56c8ec78c489016670dc02c0504e39bea92a7602d", 49086, false},
			{TireInput{Energy: 6, Camber: 76, Ambient: 17184, Speed: 2018, Load: 577}, "4daca9523c05745bcfbcb2e4667af7722fb60c9a00d84301f723d46b72f88744", 50374, false},
		},
		{
			{TireInput{Energy: 13, Camber: 326, Ambient: 17184, Speed: 2113, Load: 1389}, "b2dd0b60d1d91d3981a4e791fc702aa3c41b6bf8a91904d84a34b324df2fc631", 47574, false},
			{TireInput{Energy: 17, Camber: 161, Ambient: 17184, Speed: 2113, Load: 1383}, "ea4d52cb114231bcf4736f63acaf93aadf0af0e4ef7802ac63867781750af34c", 48913, false},
			{TireInput{Energy: 7, Camber: 238, Ambient: 17184, Speed: 2113, Load: 572}, "c6c896ffa5966a46931a355aec8d984488b67126e8df94862cc242af707cc998", 49073, false},
			{TireInput{Energy: 7, Camber: 77, Ambient: 17184, Speed: 2113, Load: 578}, "254bf2067dd2859405745e5efbe804a5b0f44315e98095251c2a5866a833be92", 50350, false},
		},
		{
			{TireInput{Energy: 14, Camber: 326, Ambient: 17184, Speed: 2208, Load: 1395}, "abe066f83c9acdd3dac8e040344e0acaa35a5cbe300a20b4b2a5505a2100fdf3", 47569, false},
			{TireInput{Energy: 18, Camber: 162, Ambient: 17184, Speed: 2208, Load: 1389}, "6b06bb6e358439a92fb94622f4131a105bee1886802a05e86f4cfd042d639407", 48898, false},
			{TireInput{Energy: 7, Camber: 238, Ambient: 17184, Speed: 2208, Load: 573}, "93a67d07bca7c1236e05a50d2e68c412c6b1b96850945257811ffed5b65f2170", 49073, false},
			{TireInput{Energy: 7, Camber: 78, Ambient: 17184, Speed: 2208, Load: 579}, "60f74653bb8a15b56912ec3428bca5da7048c630bcd29bcefd0953d95f9234ae", 50339, false},
		},
		{
			{TireInput{Energy: 14, Camber: 326, Ambient: 17184, Speed: 2304, Load: 1400}, "04b295b78c530e002045e0840376e0ba6987b649bff3042e7dca46c81641b182", 47566, false},
			{TireInput{Energy: 19, Camber: 161, Ambient: 17184, Speed: 2304, Load: 1394}, "96c9dc5fb3e818ec0566fa6d0dec1767368b13de0d4983aaf8b9b6fcee4f83e9", 48905, false},
			{TireInput{Energy: 7, Camber: 238, Ambient: 17184, Speed: 2304, Load: 574}, "fe3cd0b928192ff50cfdf8c39da717ed86b0eaa3ef19244f62501544f1338368", 49061, false},
			{TireInput{Energy: 7, Camber: 77, Ambient: 17184, Speed: 2304, Load: 580}, "d209621160215b252f3542c11f0fbf8769d3f2de3a1311b6df85bd196a907e9d", 50348, false},
		},
		{
			{TireInput{Energy: 15, Camber: 326, Ambient: 17184, Speed: 2400, Load: 1405}, "5870d45abcfce61f7b407b690f0041c9ddbf4aff900a7b1c127612f8ee25bc00", 47562, false},
			{TireInput{Energy: 20, Camber: 162, Ambient: 17184, Speed: 2400, Load: 1400}, "11df70873cdaa76cb874db218110dbab8d15a99a410062135da215e0bbb1eb59", 48890, false},
			{TireInput{Energy: 8, Camber: 238, Ambient: 17184, Speed: 2400, Load: 579}, "7e5714fa231389bbee50079500b001a1f01eb7aab022d6ba04321a1c76e1301b", 49059, false},
			{TireInput{Energy: 8, Camber: 78, Ambient: 17184, Speed: 2400, Load: 584}, "7c5fadbb27817c888356e784b7dcb135b737cd6986794075f739ff7baa3063a8", 50336, false},
		},
		{
			{TireInput{Energy: 16, Camber: 326, Ambient: 17184, Speed: 2497, Load: 1412}, "39d2c0c7472c4fb9eff47867ae00f8a7750591549f2f45c1e6e02515ce288ba1", 47556, false},
			{TireInput{Energy: 21, Camber: 161, Ambient: 17184, Speed: 2497, Load: 1407}, "28f5e1cda2c328fb97b235f6d244b285b5ac08a53061129d71867d3ffd9e3963", 48896, false},
			{TireInput{Energy: 8, Camber: 239, Ambient: 17184, Speed: 2497, Load: 580}, "f5d0fa214a01d2f7abf11ab188fe7281b18a7f2e9486bc886afbc59e2902e273", 49046, false},
			{TireInput{Energy: 8, Camber: 78, Ambient: 17184, Speed: 2497, Load: 585}, "8ef7099ab79b98d3ee3d7c45c2d6430164f54a7d49f4de8e71772fbc6f2a7004", 50335, false},
		},
		{
			{TireInput{Energy: 17, Camber: 326, Ambient: 17184, Speed: 2594, Load: 1418}, "445cbd1e6d0336662d4802e5ac50fe8cbc33b1f29ba6de6329201786b147d0ac", 47552, false},
			{TireInput{Energy: 22, Camber: 162, Ambient: 17184, Speed: 2594, Load: 1413}, "99ee8617a0b183fce69ff603f273816b0556e37e4fcdfba8f47970f363ec35eb", 48867, false},
			{TireInput{Energy: 9, Camber: 238, Ambient: 17184, Speed: 2594, Load: 581}, "f93bdeeacc6f27674fb891892f8ae4695ab756126697e4f4e565b31a5f6daf03", 49058, false},
			{TireInput{Energy: 9, Camber: 78, Ambient: 17184, Speed: 2594, Load: 586}, "6a41d4d2edc7343bd0d795504617f9d72a2f3dc7c913be6b5d3ba5c700791407", 50335, false},
		},
		{
			{TireInput{Energy: 17, Camber: 326, Ambient: 17184, Speed: 2691, Load: 1425}, "179135d2bf0cd14e56b8922286026f660025dcc7b1191140742002ec969ce084", 47547, false},
			{TireInput{Energy: 23, Camber: 161, Ambient: 17184, Speed: 2691, Load: 1421}, "f34d0b7815a430ca7dbef8f7873c503048c201db9141db0c353156f931607580", 48871, false},
			{TireInput{Energy: 9, Camber: 239, Ambient: 17184, Speed: 2691, Load: 586}, "0bffa55e4a316c7f327a05e1803c2ed9c60d9667b554bed9e3261b3d90e86f2c", 49044, false},
			{TireInput{Energy: 9, Camber: 78, Ambient: 17184, Speed: 2691, Load: 590}, "315e37f5393042629d93cc31ccbb3d88647fd5246edacf43661296a404a98293", 50333, false},
		},
		{
			{TireInput{Energy: 18, Camber: 326, Ambient: 17184, Speed: 2787, Load: 1432}, "c95f706d3f19bf268f57daf8e7123d18bc9da898cc73f099b37db6d1b5ca01e0", 47534, false},
			{TireInput{Energy: 23, Camber: 161, Ambient: 17184, Speed: 2787, Load: 1429}, "9f4cad84fc0acba61013c48dabdbae03c8168cf43ba3012cecabfa1a854f803c", 48865, false},
			{TireInput{Energy: 14, Camber: 211, Ambient: 17184, Speed: 2787, Load: 843}, "b55e4282b2aab9ecd5a02d5b0f1f2e2591477a226cd78f38613c77250dc03fad", 49229, false},
			{TireInput{Energy: 14, Camber: 51, Ambient: 17184, Speed: 2787, Load: 846}, "db020f51448235a73c579dc3d1539c15b037983bc98b4d96a33e1264c1a926b0", 50500, false},
		},
		{
			{TireInput{Energy: 25, Camber: 325, Ambient: 17184, Speed: 2884, Load: 1439}, "20b9abcdabb9db728b33fc67623eb0da148fd3ab8b7d5fede3379e7ec14629dc", 47541, false},
			{TireInput{Energy: 25, Camber: 160, Ambient: 17184, Speed: 2884, Load: 1436}, "696be0f95b3624d4d4f3a657a7d07835f7281ba163edb42b1a81c011101044f6", 48872, false},
			{TireInput{Energy: 10, Camber: 238, Ambient: 17184, Speed: 2884, Load: 591}, "b8de4823e1abddbcb4b75e500137e14b1ccb2ccc44ef2be770553bb285959bae", 49054, false},
			{TireInput{Energy: 10, Camber: 78, Ambient: 17184, Speed: 2884, Load: 594}, "4c390ab5e56fe23151832c724f9a673893279890e365635e59369e8fe2bf4bcb", 50332, false},
		},
		{
			{TireInput{Energy: 26, Camber: 325, Ambient: 17184, Speed: 2982, Load: 1447}, "d1615e5db26f3a9072dd7ec25e09ce0aefd0c817795666a8625c5558d9e0161a", 47535, false},
			{TireInput{Energy: 26, Camber: 161, Ambient: 17184, Speed: 2982, Load: 1444}, "acad78c1f989926ea2b1b21ad4d24cf05370c673646e920f2553ee9c6917c251", 48854, false},
			{TireInput{Energy: 10, Camber: 239, Ambient: 17184, Speed: 2982, Load: 595}, "4117fd7f42753d8e4a53fdf77384882708a10bc756140e6e6da9029614f84177", 49040, false},
			{TireInput{Energy: 10, Camber: 79, Ambient: 17184, Speed: 2982, Load: 598}, "02761ec8da46ffe627ceb1ea06d72f6dc2656bf87e1f52611cb47089593bb86c", 50317, false},
		},
		{
			{TireInput{Energy: 32, Camber: 302, Ambient: 17184, Speed: 3090, Load: 1711}, "afa9308e6cc61dd3f4aa8ef6b28e7c2d536bd3b331fac4a0c7c75934fc69d786", 47577, false},
			{TireInput{Energy: 32, Camber: 131, Ambient: 17184, Speed: 3090, Load: 1708}, "66ded972ede50c0557f885d38c96ef33a4bf7e92dee1690fc44dc23f1c324849", 48965, false},
			{TireInput{Energy: 10, Camber: 239, Ambient: 17184, Speed: 3090, Load: 596}, "2a158ccf9066814273188d9d7f2965c015f19612d02e246042cda9f0803a4497", 49039, false},
			{TireInput{Energy: 10, Camber: 78, Ambient: 17184, Speed: 3090, Load: 599}, "8eaf0edac6b354b730fb3d261457c36f4cb5cfa82a78dce40e3222cb59dbd887", 50317, false},
		},
		{
			{TireInput{Energy: 28, Camber: 333, Ambient: 17184, Speed: 3190, Load: 1477}, "46d964ae6a0cd0c01330603e20e26349317664e5db3dfca07a30be8e395dad7f", 47421, false},
			{TireInput{Energy: 28, Camber: 168, Ambient: 17184, Speed: 3190, Load: 1473}, "b6590f9bf52a0cead5a09583dac18045a099e29ab0ad66c22e08a0b1b62453ca", 48753, false},
			{TireInput{Energy: 11, Camber: 247, Ambient: 17184, Speed: 3190, Load: 585}, "7b6b4755a6d8eb627e912b8086b49201e8e8f1c35e8cbfc3a37ef7e5b67f7da1", 48953, false},
			{TireInput{Energy: 11, Camber: 87, Ambient: 17184, Speed: 3190, Load: 589}, "307734e0cc9aa935dbbb0de8c0880f7585396d86ac6a4b099c38159a399f9d22", 50220, false},
		},
		{
			{TireInput{Energy: 29, Camber: 329, Ambient: 17184, Speed: 3289, Load: 1479}, "87d3b19d9bd92be86f0691fda2e1827b47466d2db0a8ce10725e491985ab109a", 47465, false},
			{TireInput{Energy: 29, Camber: 163, Ambient: 17184, Speed: 3289, Load: 1476}, "1a27cdec7d913a5382b09632589c0a86b9cb501a865d00e9fc00939acfe36bb5", 48805, false},
			{TireInput{Energy: 11, Camber: 243, Ambient: 17184, Speed: 3289, Load: 596}, "402392aff2926895c33cc5bb61ca664ae20bd8506914f0027f452decd64c1974", 48982, false},
			{TireInput{Energy: 11, Camber: 82, Ambient: 17184, Speed: 3289, Load: 599}, "5db7eb3d3c35df82a4b366fc14973922765c1edc2c30d8ee011e7d129be64756", 50271, false},
		},
		{
			{TireInput{Energy: 30, Camber: 327, Ambient: 17184, Speed: 3388, Load: 1483}, "9e9afecfcc847f31a5c7efcb9fd95f25f59ae7e75624c7ff8e4583837f8a86fd", 47484, false},
			{TireInput{Energy: 30, Camber: 161, Ambient: 17184, Speed: 3388, Load: 1481}, "054a90c693e5936c06355d4f956cbdcb154bec82f2e505c255083d65dde049c3", 48824, false},
			{TireInput{Energy: 12, Camber: 241, Ambient: 17184, Speed: 3388, Load: 604}, "2d9c83ad2e0b8c25da71108f97370c59d9465756ba7671a6f60a7b4c003ffe38", 49001, false},
			{TireInput{Energy: 12, Camber: 81, Ambient: 17184, Speed: 3388, Load: 606}, "989d9bf5c84d63c502af709c5a36e91d13de4caa4f5150b3d10960d864975dd8", 50280, false},
		},
		{
			{TireInput{Energy: 31, Camber: 326, Ambient: 17184, Speed: 3488, Load: 1491}, "8e8b62d16f2319474d751dd2e134b931024fc8e1b430e1abc186b3055b83e8ee", 47489, false},
			{TireInput{Energy: 31, Camber: 160, Ambient: 17184, Speed: 3488, Load: 1490}, "028e9a6c917470b75aa39138a650ae284fcdc6ab8e913a03f232b2c6e052271c", 48829, false},
			{TireInput{Energy: 12, Camber: 240, Ambient: 17184, Speed: 3488, Load: 609}, "dc8b74c6840f2bf872389c164c2866bd287bb88e1be27c9602c29d20a196d963", 49011, false},
			{TireInput{Energy: 12, Camber: 80, Ambient: 17184, Speed: 3488, Load: 610}, "58a80e6e5c9cd5153b44d5062cfc8dc539c1833e233b448852f9d2befe4328e1", 50290, false},
		},
		{
			{TireInput{Energy: 32, Camber: 325, Ambient: 17184, Speed: 3588, Load: 1500}, "fba78192c1e6fe82e43fcef06af1368d63af5125b8ee08b0832226c695f47288", 47494, false},
			{TireInput{Energy: 32, Camber: 159, Ambient: 17184, Speed: 3588, Load: 1500}, "d5358e3c761347f0e147f4287c389697d4aea3cd50ce3db446c7bc7569bc3297", 48823, false},
			{TireInput{Energy: 13, Camber: 240, Ambient: 17184, Speed: 3588, Load: 614}, "25c7196341e2c4145d8e0b164b337b03431350e6f51ee86d6269dbe6b4d081c4", 49008, false},
			{TireInput{Energy: 13, Camber: 80, Ambient: 17184, Speed: 3588, Load: 614}, "f73c9833cfb1e93107262ef276cac63ad2c14cb93ea47a4b4d991c04e061f80f", 50287, false},
		},
		{
			{TireInput{Energy: 34, Camber: 325, Ambient: 17184, Speed: 3689, Load: 1510}, "82563adcc283347ca778969c2b3929b9cecf66ff6e2970352c7667ee7820d2ea", 47486, false},
			{TireInput{Energy: 34, Camber: 159, Ambient: 17184, Speed: 3689, Load: 1511}, "ea48a7e7ec7f19e3a0ab98f330e42ef2ede98a29da1bf7c6df8e493dfde4bac6", 48813, false},
			{TireInput{Energy: 13, Camber: 240, Ambient: 17184, Speed: 3689, Load: 617}, "5204f258ca3dc3f4ba6e3d0b088a789b205fcfdf94698857bc83e7db568721ea", 49007, false},
			{TireInput{Energy: 13, Camber: 80, Ambient: 17184, Speed: 3689, Load: 616}, "5051c06ee949a7bc4374a3b4af4ccb6eba31b038241f3b27cfb4d91f8dd00b24", 50286, false},
		},
		{
			{TireInput{Energy: 35, Camber: 326, Ambient: 17184, Speed: 3790, Load: 1518}, "4b09054f8db1c17f1e4bfce3787bac71e5441dd796c8f7d6b7c5ce57768f5f39", 47459, false},
			{TireInput{Energy: 35, Camber: 160, Ambient: 17184, Speed: 3790, Load: 1520}, "e90fa779287d89d8f937789b298b6303f452a4fa72c94762134ed6b7f67eda53", 48795, false},
			{TireInput{Energy: 8, Camber: 269, Ambient: 17184, Speed: 3790, Load: 367}, "85c5b7663aca05a504697afae7da72032da28fa038d2274a61235a9f0072127f", 48785, false},
			{TireInput{Energy: 8, Camber: 109, Ambient: 17184, Speed: 3790, Load: 365}, "94e8f9862da2d14d71bd79faf73354e9259cffabbb208e826913d7961d080b18", 50072, false},
		},
		{
			{TireInput{Energy: 36, Camber: 326, Ambient: 17184, Speed: 3892, Load: 1531}, "181bc59788a0c83a6c84969aa598c05dd453e5adbe8986b43f93a5ca131c26fc", 47448, false},
			{TireInput{Energy: 36, Camber: 159, Ambient: 17184, Speed: 3892, Load: 1533}, "e7c63286bbd221d6dba8450790048f5a64bd1cae08e359148a56af50dbee7629", 48795, false},
			{TireInput{Energy: 14, Camber: 241, Ambient: 17184, Speed: 3892, Load: 625}, "258848a993e1fbb3d633e861dc80f7175dd1e7a358fe069392752df5d9854b5c", 48992, false},
			{TireInput{Energy: 14, Camber: 81, Ambient: 17184, Speed: 3892, Load: 623}, "8bda358d75bf23464709efe0b2eda15341619c66ea18e6241f3eafe5a25c9d56", 50272, false},
		},
		{
			{TireInput{Energy: 31, Camber: 349, Ambient: 17184, Speed: 3976, Load: 1285}, "707f1b1a7959bb7b24622d329e60479abbb2dbf263f4873eebbec01babb51a73", 47376, false},
			{TireInput{Energy: 31, Camber: 187, Ambient: 17184, Speed: 3976, Load: 1287}, "cb0d5053dc3d541605b566ae86111862d7dc82ba8e9b3ed67a7c0817ee1e72c0", 48675, false},
			{TireInput{Energy: 15, Camber: 242, Ambient: 17184, Speed: 3976, Load: 629}, "2d7065b931548eed9cafb93f854ea9036b37fb7bfe0cfaa416561cfd17e27cc3", 48978, false},
			{TireInput{Energy: 15, Camber: 81, Ambient: 17184, Speed: 3976, Load: 627}, "619a83726ec2ffbdeff4b0ed1892ea71fcd7aec86b8c85fc02208be23074c3a9", 50270, false},
		},
		{
			{TireInput{Energy: 37, Camber: 311, Ambient: 17184, Speed: 4077, Load: 1524}, "05b3ebb76076fa7a61f48f3e47dfe1994dd8889a36154cc65a6960b1cb17522a", 47623, false},
			{TireInput{Energy: 37, Camber: 143, Ambient: 17184, Speed: 4077, Load: 1529}, "7565d825edee830cd5dbb3f7b2352ada91ef0f75999c232744be8156e366e16d", 48978, false},
			{TireInput{Energy: 16, Camber: 227, Ambient: 17184, Speed: 4077, Load: 659}, "b4fbd2e206466250b5056287e8c81c553f2f0f7adb4a8a47b9801a90f63acbf5", 49133, false},
			{TireInput{Energy: 15, Camber: 66, Ambient: 17184, Speed: 4077, Load: 654}, "decc6bf915809da11d534207dc557ba6cc8df62192d7873774310a2279fc7b24", 50409, false},
		},
		{
			{TireInput{Energy: 39, Camber: 319, Ambient: 17184, Speed: 4179, Load: 1549}, "9482a1acef829371d803275b8b0dbe9954abd858b6889be812510a6b6c4f4696", 47514, false},
			{TireInput{Energy: 39, Camber: 151, Ambient: 17184, Speed: 4179, Load: 1553}, "a9a53d850edfa32c2e025c91ba3e4f96efad50cf30505119cb00ddf340550aeb", 48867, false},
			{TireInput{Energy: 16, Camber: 235, Ambient: 17184, Speed: 4179, Load: 650}, "d379431344b6bd78b2ae74235790e44d2821de1a581e291de07b2c2b6de3f2c6", 49048, false},
			{TireInput{Energy: 16, Camber: 74, Ambient: 17184, Speed: 4179, Load: 646}, "39124951ba548c712517216ec5924fa1dd8563ed338fad00882c38157da9ab3f", 50322, false},
		},
		{
			{TireInput{Energy: 41, Camber: 323, Ambient: 17184, Speed: 4282, Load: 1569}, "39de0b30ac8c1fe36bd957510332950ae122fb26820443971d584a6d3d9bf6fd", 47452, false},
			{TireInput{Energy: 41, Camber: 155, Ambient: 17184, Speed: 4282, Load: 1573}, "4645e94601426b3604a198eacd6b3226e2ab5302152ac75ab84f50bb7fdc8b71", 48807, false},
			{TireInput{Energy: 16, Camber: 240, Ambient: 17184, Speed: 4282, Load: 646}, "d25d65b9fb8db74234e9900a3678986a2565653c9ff217e92f9f2fa9c735828c", 48978, false},
			{TireInput{Energy: 16, Camber: 79, Ambient: 17184, Speed: 4282, Load: 642}, "4c36e5e63d40c4651a9d4496a684343dc4084c5fefe97e2014730d31eeb22d8c", 50269, false},
		},
		{
			{TireInput{Energy: 42, Camber: 324, Ambient: 17184, Speed: 4386, Load: 1582}, "688d1176f73cebc4a2150db8274caa845e97c0ef1bcb996aeee2fcf7dab56bf1", 47430, false},
			{TireInput{Energy: 42, Camber: 157, Ambient: 17184, Speed: 4386, Load: 1587}, "655aad8ef67b35eb23544e105441b3fc63a084b0264692be7708e6798f268c09", 48773, false},
			{TireInput{Energy: 17, Camber: 241, Ambient: 17184, Speed: 4386, Load: 648}, "5bd71e42d1365917d18d974f91b01aadaad033f9aacf600d163927c2549f85ec", 48965, false},
			{TireInput{Energy: 17, Camber: 81, Ambient: 17184, Speed: 4386, Load: 643}, "b9917d5068ef31c991ae02073b75027673cdfc307aa2a3a73edd8ca09908931b", 50245, false},
		},
		{
			{TireInput{Energy: 43, Camber: 325, Ambient: 17184, Speed: 4491, Load: 1596}, "4d7b89438ef18d9c292536d31ec011573dc9172aca8143493a85f1816741ccd1", 47407, false},
			{TireInput{Energy: 43, Camber: 157, Ambient: 17184, Speed: 4491, Load: 1602}, "a2e24b2f2b15d8983608f096f656c5e8e99e70e8ee4e1d5dbb6dbc6c5b2ca421", 48761, false},
			{TireInput{Energy: 17, Camber: 242, Ambient: 17184, Speed: 4491, Load: 653}, "117a894b06d95354c36ef1d88200a097c556a4eca3f8f54abbeb8261d059f7f1", 48952, false},
			{TireInput{Energy: 17, Camber: 81, Ambient: 17184, Speed: 4491, Load: 647}, "a6ee56b80c72f6cc054d01857fc5396aae1b7e5cd57907162f057900475740f2", 50244, false},
		},
	}
	var tires [4]Tire
	for i := range tires {
		tires[i] = Tire{Config: originalTireConfigs[i], State: originalTireStates[i]}
	}
	for frame, c := range cases {
		for i, w := range c {
			result := tires[i].Step(w.input)
			if actual := tireStateDigest(t, tires[i].State); actual != w.digest || result.Grip != w.grip || result.Punctured != w.punctured {
				t.Fatalf("original tire tick%d wheel%d: got %+v (%s), result%+v; want %s grip%d punctured%v", frame, i, tires[i].State, actual, result, w.digest, w.grip, w.punctured)
			}
		}
	}
}

// Isolated original blocks verify diffusion, the pressure-update phase,
// its equality tie, load limits and the first/repeated failure behavior.
func TestOriginalTireBoundaryCases(t *testing.T) {
	cases := []struct {
		name   string
		setup  func(*TireState)
		input  TireInput
		digest string
		result TireResult
	}{
		{"heat-positive-camber", func(s *TireState) { s.HeatTick = 1 }, TireInput{Energy: 10000, Camber: 1000, Ambient: 17184, Speed: 0, Load: 1131}, "83bdfad0fdd11342c79e210c5784fc11d687d9350420998b14eb2b3ae065d7f6", TireResult{Grip: 42315, Punctured: false}},
		{"heat-negative-camber", func(s *TireState) { s.HeatTick = 1 }, TireInput{Energy: 10000, Camber: -1000, Ambient: 17184, Speed: 0, Load: 1131}, "5444862b62bbf99b915f665fff5982b77147a76256e5272e0e95d6ab0f467532", TireResult{Grip: 42315, Punctured: false}},
		{"pressure-high", func(s *TireState) { s.HeatTick = 1; s.Pressure = 25000 }, TireInput{Energy: 10000, Camber: 213, Ambient: 17184, Speed: 0, Load: 1131}, "36db7e02835ad896e34a78a600e6142f686581b3caa4a66051d4b05fc8aa5f67", TireResult{Grip: 33879, Punctured: false}},
		{"pressure-low", func(s *TireState) { s.HeatTick = 1; s.Pressure = 8000 }, TireInput{Energy: 10000, Camber: 213, Ambient: 17184, Speed: 0, Load: 1131}, "9d3583b7dd2a618ccc142f9922bcfd5664afe29ea8a0db625b024e10033cf959", TireResult{Grip: 53966, Punctured: false}},
		{"cooling-floor", func(s *TireState) { s.HeatTick = 1; s.Bands = [3]uint16{18000, 18000, 18000} }, TireInput{Energy: 0, Camber: 213, Ambient: 17184, Speed: 10000, Load: 1131}, "4dd77fdb9fa70178cbea06945093948ff1db596bdf8b016fb823045fb0b7d7c3", TireResult{Grip: 46612, Punctured: false}},
		{"uneven-temperatures", func(s *TireState) { s.HeatTick = 1; s.Bands = [3]uint16{24000, 26000, 30000} }, TireInput{Energy: 10000, Camber: 213, Ambient: 17184, Speed: 0, Load: 1131}, "bf8de63cb167e35b4ae86a56a6de1955c0eb6f9de876ea1dec928a5bda8cc473", TireResult{Grip: 44322, Punctured: false}},
		{"puncture", func(s *TireState) { s.Life = 1; s.HeatTick = 1; s.Bands = [3]uint16{24000, 26000, 30000} }, TireInput{Energy: 10000, Camber: 213, Ambient: 17184, Speed: 0, Load: 1131}, "5c9d8bc9096fa047c4748326fb908aabaef03f698b90c94cb3520df0e59f1fb5", TireResult{Grip: 44322, Punctured: true}},
		{"already-punctured", func(s *TireState) {
			s.Life = 0
			s.HeatTick = 1
			s.Bands = [3]uint16{24000, 26000, 30000}
			s.Failed = true
		}, TireInput{Energy: 10000, Camber: 213, Ambient: 17184, Speed: 0, Load: 1131}, "b1f4d1562cdf7f80a4821e7d818a8c75e09a5cdaa72977317d93067ff0328627", TireResult{Grip: 44322, Punctured: false}},
		{"pressure-update", func(s *TireState) { s.HeatTick = 2; s.PressureTick = 1 }, TireInput{Energy: 0, Camber: 213, Ambient: 17184, Speed: 0, Load: 1131}, "f635928e1a121cd4c86a782547c850c11adcc409b79f58cf004d3dcaf08a20c7", TireResult{Grip: 62066, Punctured: false}},
		{"heat-and-pressure", func(s *TireState) { s.HeatTick = 1; s.PressureTick = 1 }, TireInput{Energy: 8000, Camber: 213, Ambient: 17184, Speed: 0, Load: 1131}, "ee15689b83b07d799610d55cf1944046bc6e2337a767effc6f292460f066c05e", TireResult{Grip: 51193, Punctured: false}},
		{"pressure-equality", func(s *TireState) {
			s.HeatTick = 2
			s.PressureTick = 1
			s.Pressure = 10240
			s.Bands = [3]uint16{17184, 17184, 17184}
		}, TireInput{Energy: 0, Camber: 213, Ambient: 17184, Speed: 0, Load: 1131}, "8e992421a003b5a0f5ed931ddad55e6406b86c2a2dc0c4cf96c4cfa7a7fd6a38", TireResult{Grip: 64321, Punctured: false}},
		{"pressure-one-above", func(s *TireState) {
			s.HeatTick = 2
			s.PressureTick = 1
			s.Pressure = 10241
			s.Bands = [3]uint16{17184, 17184, 17184}
		}, TireInput{Energy: 0, Camber: 213, Ambient: 17184, Speed: 0, Load: 1131}, "57f7781838841d4fb88b41ab5e0a3cca0450b6cf67df034eb01e7ab3815285bc", TireResult{Grip: 64319, Punctured: false}},
		{"warm-air", func(s *TireState) { s.HeatTick = 1; s.PressureTick = 1 }, TireInput{Energy: 10000, Camber: 213, Ambient: 19000, Speed: 0, Load: 1131}, "a5ea2cd7eed8f79e109a5da9a04dc26d5d4971601d42c302038c71805f8a1a90", TireResult{Grip: 51728, Punctured: false}},
		{"high-speed-cooling", func(s *TireState) { s.HeatTick = 1 }, TireInput{Energy: 10000, Camber: 213, Ambient: 17184, Speed: 30000, Load: 1131}, "14dd5e49ba7999f35d3e564818dbd7bdc45379dcfd02634c469173162e4d2217", TireResult{Grip: 51728, Punctured: false}},
		{"signed-camber-limit", func(s *TireState) { s.HeatTick = 1 }, TireInput{Energy: 1000, Camber: -32768, Ambient: 17184, Speed: 0, Load: 1131}, "7eef4bc759bfeec0439034f871600c1f843c5b8e93ee4085a9079f003b539f46", TireResult{Grip: 51750, Punctured: false}},
		{"zero-load", nil, TireInput{Energy: 0, Camber: 213, Ambient: 17184, Speed: 0, Load: 0}, "b77c66c7b9ebc51c0579dc6e959f8dad4c2f9b275ac753647819533e7a542299", TireResult{Grip: 62736, Punctured: false}},
		{"overload", nil, TireInput{Energy: 0, Camber: 213, Ambient: 17184, Speed: 0, Load: 10885}, "b77c66c7b9ebc51c0579dc6e959f8dad4c2f9b275ac753647819533e7a542299", TireResult{Grip: 0, Punctured: false}},
	}
	for _, c := range cases {
		tire := Tire{Config: originalTireConfigs[0], State: originalTireStates[0]}
		if c.setup != nil {
			c.setup(&tire.State)
		}
		result := tire.Step(c.input)
		if actual := tireStateDigest(t, tire.State); actual != c.digest || result != c.result {
			t.Fatalf("%s: got %+v (%s), result%+v; want %s, result%+v", c.name, tire.State, actual, result, c.digest, c.result)
		}
	}
}
