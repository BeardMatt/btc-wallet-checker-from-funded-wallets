package bitcoin

import (
	"testing"

	"github.com/btcsuite/btcd/btcutil"
	"github.com/btcsuite/btcd/btcec/v2"
)

func TestHash160MatchesBtcd(t *testing.T) {
	for i := 0; i < 5000; i++ {
		priv, err := btcec.NewPrivateKey()
		if err != nil {
			t.Fatal(err)
		}
		compressed := priv.PubKey().SerializeCompressed()
		want := btcutil.Hash160(compressed)
		got := Hash160(compressed)
		if len(got) != 20 || string(got) != string(want) {
			t.Fatalf("hash160 mismatch at %d", i)
		}
	}
}