package vehicle

import (
	"crypto/sha256"
	"encoding/binary"
	"fmt"
	gameassets "github.com/olivierh59500/indianapolis500/assets"
	"testing"
)

// Every row was recorded from original $5122 calls at each infield/pit
// boundary and fourteen lateral positions. Both support height and carry
// (the off-road flag) are included in each independently captured digest.
func TestOriginalTrackSurfaceSupport(t *testing.T) {
	data, err := gameassets.Original.ReadFile("original/track-surface-wave.bin")
	if err != nil {
		t.Fatal(err)
	}
	surface, err := NewTrackSurface(data)
	if err != nil {
		t.Fatal(err)
	}
	latitudes := []int16{-10, 0, 1, 132, 133, 209, 210, 224, 225, 239, 240, 295, 296, 500}
	cases := []struct {
		progress uint16
		digest   string
	}{
		{0, "f13d2c415114f070305f514c120082573d6c5d635a749d8895ad599c312b42f2"},
		{1, "f13d2c415114f070305f514c120082573d6c5d635a749d8895ad599c312b42f2"},
		{127, "f13d2c415114f070305f514c120082573d6c5d635a749d8895ad599c312b42f2"},
		{128, "f6d4c77a28e305b39b3f0e5f5e7625ff64bb70d7e7f852a1c02404aed94455ac"},
		{255, "f6d4c77a28e305b39b3f0e5f5e7625ff64bb70d7e7f852a1c02404aed94455ac"},
		{256, "f13d2c415114f070305f514c120082573d6c5d635a749d8895ad599c312b42f2"},
		{2819, "f13d2c415114f070305f514c120082573d6c5d635a749d8895ad599c312b42f2"},
		{2820, "f13d2c415114f070305f514c120082573d6c5d635a749d8895ad599c312b42f2"},
		{2821, "f13d2c415114f070305f514c120082573d6c5d635a749d8895ad599c312b42f2"},
		{3119, "df45487e72171567f7e641b1fc17a6c5552455dca032fb6c894afb3df0b83fe1"},
		{3120, "e2ca7e1a9e3639ee0dc04ca82af8fa86d143cb3abe1aedfeaf9328f38e67a173"},
		{3121, "6f1e32c3708ed1bb3e438ef10ac736dc3813487507d5846bef023627e2a9fa42"},
		{8939, "4c5e806f9ac344374d7fcb2ee9becf6a5f92aa194ea43af32e6bc8ea41918f98"},
		{8940, "2809ff570faf0ac844c2b0a3ae6b4f6261812f4a9e0af1fbebc51402d61b39da"},
		{8941, "1a51044fb6ae47af4934693c8d1972016b447868621fca57b61c07476166116d"},
		{9239, "f13d2c415114f070305f514c120082573d6c5d635a749d8895ad599c312b42f2"},
		{9240, "f13d2c415114f070305f514c120082573d6c5d635a749d8895ad599c312b42f2"},
		{9241, "f13d2c415114f070305f514c120082573d6c5d635a749d8895ad599c312b42f2"},
		{9549, "f13d2c415114f070305f514c120082573d6c5d635a749d8895ad599c312b42f2"},
		{9550, "524b9d1f8897c50919d578225e4b19aa518229f31b92629b14c13debda452200"},
		{9551, "86cd02aee3670f30f4663b558f5be488b7969b3ecf645a66c25f1f1e2c1a8286"},
		{9899, "4ae6395dee1d24d101d7f40bdc74b118c3a9d983f01754afe2b5a23720eb5a1f"},
		{9900, "1037c1ad44ef7c9fd5d1bc622aafd58eb858ece78ec32d141513f11f9d1a5b9b"},
		{9901, "c0f64382282f5b45cc5855da177c3767c44abeb799b6bf0827d9f97c0f3ec282"},
		{10394, "1eac1330f6ecca5632bdc95101c8b500ba6c8864facf338a26410bb0c6ed4e05"},
		{10395, "f1dfc7ae815debd28b3fa13a116db3b9424e7f06217408fcd38ff76ffeae44ff"},
		{10396, "c9ba2cdec592acc01b666ce1629c109c4140d08ffb505b21fcacfa5284ff2c7b"},
		{19799, "83455fdda898144c5f308b279ef27f1d39e64991fe29fe7fe8309e75e40851a3"},
		{19800, "ef9a153861521897d2bedec15d4e6807c043d5f7d5ee4e920b3217321027024a"},
		{19801, "842b3c0b48f51769ae4c86da2a9835215589aae2d228d8b9ad6fb4d0834a0130"},
		{21239, "a16bc9c4b5d365b145d18daab12d9b45fc3d2233cc9f28485f17610f0f4b05ec"},
		{21240, "261e351a0c21eabda62f445d70d7bd06ac9bce32934db4814d5576e631bbe57f"},
		{21241, "76ea0e50f7e959e8fee2ac7c9d8564706d208933e6624aac40efb1b1b0bfc9bc"},
		{28259, "536d540114300c904b21f2e05aec78aaf90703073cf962dbbf083ba71523b9ae"},
		{28260, "a41310eb5f276dc0c09143e535e0d53a3e9a34ae970a31d80daedb1d6fe1ea11"},
		{28261, "ffa4b990381316ea897d0f78c34937765aed0614c89cbd9b48adccbdb36cb176"},
		{29699, "a7f8c52d7a0c2441ad3918489c209291c7e0d172022b4c2ce98ad43a81b4f094"},
		{29700, "c633469fdfdc02070fb29d105caac1e405d904548448256a05649f45fc3cf67a"},
		{29701, "97a62a2ae8d37a2fd8207898c5748b9e7ada68222b7911d0d9fe56189754ec41"},
		{36959, "7ddb8eb4f7518ac5f977e0676fecb1bc4fbcd57f8311c4376693acb25e30e2c5"},
		{36960, "762ea80f13aa1fdbb2f9ba650e1f294f33d2982839cbdbe71877d13a3d807623"},
		{36961, "f285377d2362bd9b7323d5ca34029d6644ccb42c51a9a2b9a25bd3ac9cc31142"},
		{37619, "e43224a40668aa862eea74f4343e75066dad0532116aedc6039cd9ba5228df76"},
		{37620, "f6d4c77a28e305b39b3f0e5f5e7625ff64bb70d7e7f852a1c02404aed94455ac"},
		{37621, "f6d4c77a28e305b39b3f0e5f5e7625ff64bb70d7e7f852a1c02404aed94455ac"},
		{65535, "f6d4c77a28e305b39b3f0e5f5e7625ff64bb70d7e7f852a1c02404aed94455ac"},
	}
	for _, c := range cases {
		data := make([]byte, 0, len(latitudes)*3)
		for _, lateral := range latitudes {
			sample := surface.Sample(c.progress, lateral)
			data = binary.BigEndian.AppendUint16(data, uint16(sample.Height))
			flag := byte(0)
			if sample.OffRoad {
				flag = 1
			}
			data = append(data, flag)
		}
		if got := fmt.Sprintf("%x", sha256.Sum256(data)); got != c.digest {
			t.Fatalf("original surface progress%d: got %s, want %s", c.progress, got, c.digest)
		}
	}
}
