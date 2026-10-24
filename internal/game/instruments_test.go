package game

import (
	"crypto/sha256"
	"fmt"
	"testing"
)

// The source instrument routines rendered these independent combinations
// into a cleared original frame buffer; compare every screen pixel.
func TestOriginalLolaInstrumentRaster(t *testing.T) {
	cases := []struct {
		rpm, temperature, pressure uint16
		digest                     string
	}{
		{1000, 21376, 20000, "b52af737e3dd2041ed21174e378c8bcdc3de35569589d6f9ba3ef09d30735348"},
		{1963, 21376, 25000, "72f8c8b01bbc7e133d90396a697aa3090c83e09207d11332116517244acec9c1"},
		{4000, 22048, 32000, "5bac771f16b83e211b5303196d41117b22e5ab03792d5c6093a1e39cdb62fe4d"},
		{5750, 23008, 36000, "19a321cb1e65692cbe878fc6057419c38a2aa44c559de0ce55f52904617e6b54"},
		{6500, 23488, 40000, "df5942706034dbe9e803b6451a0ff904b846be3f0f772c8555304ecdea6e153e"},
		{0, 18848, 17929, "f41c339b1d85895cbe557f7fce7a66a874210ded9e7e70c32156ba6f69d319c8"},
		{10000, 24000, 40000, "df5942706034dbe9e803b6451a0ff904b846be3f0f772c8555304ecdea6e153e"},
	}
	var d dashboard
	if err := d.load(); err != nil {
		t.Fatal(err)
	}
	for _, c := range cases {
		d.lola(c.rpm, c.temperature, c.pressure)
		if digest := fmt.Sprintf("%x", sha256.Sum256(d.pixels[:])); digest != c.digest {
			t.Fatalf("RPM%d temperature%d pressure%d: hash%s, want%s", c.rpm, c.temperature, c.pressure, digest, c.digest)
		}
	}
}
