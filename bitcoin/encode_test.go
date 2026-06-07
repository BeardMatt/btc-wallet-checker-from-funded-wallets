package bitcoin

import (
	"fmt"
	"testing"

	"github.com/btcsuite/btcd/btcec/v2"
	"github.com/btcsuite/btcd/btcutil"
	"github.com/btcsuite/btcd/chaincfg"
)

func btcdEncodePubKeyHash(hash [20]byte) string {
	addr, err := btcutil.NewAddressPubKeyHash(hash[:], &chaincfg.MainNetParams)
	if err != nil {
		return ""
	}
	return addr.EncodeAddress()
}

func btcdEncodeWitnessPubKeyHash(hash [20]byte) string {
	addr, err := btcutil.NewAddressWitnessPubKeyHash(hash[:], &chaincfg.MainNetParams)
	if err != nil {
		return ""
	}
	return addr.EncodeAddress()
}

func btcdEncodeScriptHash(hash [20]byte) string {
	addr, err := btcutil.NewAddressScriptHashFromHash(hash[:], &chaincfg.MainNetParams)
	if err != nil {
		return ""
	}
	return addr.EncodeAddress()
}

func btcdEncodeTaproot(key [32]byte) string {
	addr, err := btcutil.NewAddressTaproot(key[:], &chaincfg.MainNetParams)
	if err != nil {
		return ""
	}
	return addr.EncodeAddress()
}

func TestEncodeCrossValidateRandomKeys(t *testing.T) {
	for i := 0; i < 10_000; i++ {
		privKey, err := btcec.NewPrivateKey()
		if err != nil {
			t.Fatal(err)
		}

		keys := DeriveLookupKeys(privKey)

		assertMatch(t, "legacy compressed", encodePubKeyHash(keys.CompressedHash), btcdEncodePubKeyHash(keys.CompressedHash))
		assertMatch(t, "legacy uncompressed", encodePubKeyHash(keys.UncompressedHash), btcdEncodePubKeyHash(keys.UncompressedHash))
		assertMatch(t, "segwit v0", encodeWitnessPubKeyHash(keys.CompressedHash), btcdEncodeWitnessPubKeyHash(keys.CompressedHash))
		assertMatch(t, "p2sh", encodeScriptHash(keys.P2SHHash), btcdEncodeScriptHash(keys.P2SHHash))
		assertMatch(t, "taproot", encodeTaproot(keys.TaprootKey), btcdEncodeTaproot(keys.TaprootKey))
	}
}

func TestEncodeMatchAddressKinds(t *testing.T) {
	privKey, err := btcec.NewPrivateKey()
	if err != nil {
		t.Fatal(err)
	}
	keys := DeriveLookupKeys(privKey)

	cases := []struct {
		kind MatchKind
		want func() string
	}{
		{MatchLegacyCompressed, func() string { return btcdEncodePubKeyHash(keys.CompressedHash) }},
		{MatchLegacyUncompressed, func() string { return btcdEncodePubKeyHash(keys.UncompressedHash) }},
		{MatchSegwitV0, func() string { return btcdEncodeWitnessPubKeyHash(keys.CompressedHash) }},
		{MatchP2SH, func() string { return btcdEncodeScriptHash(keys.P2SHHash) }},
		{MatchTaproot, func() string { return btcdEncodeTaproot(keys.TaprootKey) }},
	}

	for _, tc := range cases {
		got := EncodeMatchAddress(tc.kind, keys)
		want := tc.want()
		assertMatch(t, fmt.Sprint(tc.kind), got, want)
	}
}

func assertMatch(t *testing.T, label string, got, want string) {
	t.Helper()
	if got != want {
		t.Fatalf("%s mismatch:\n  got:  %s\n  want: %s", label, got, want)
	}
}

func BenchmarkHandRolledEncode(b *testing.B) {
	var hash [20]byte
	var key [32]byte
	for i := range hash {
		hash[i] = byte(i)
	}
	for i := range key {
		key[i] = byte(i + 1)
	}

	b.Run("legacy", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			encodePubKeyHash(hash)
		}
	})
	b.Run("segwit", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			encodeWitnessPubKeyHash(hash)
		}
	})
	b.Run("p2sh", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			encodeScriptHash(hash)
		}
	})
	b.Run("taproot", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			encodeTaproot(key)
		}
	})
}

func BenchmarkBtcdEncode(b *testing.B) {
	var hash [20]byte
	var key [32]byte
	for i := range hash {
		hash[i] = byte(i)
	}
	for i := range key {
		key[i] = byte(i + 1)
	}

	b.Run("legacy", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			btcdEncodePubKeyHash(hash)
		}
	})
	b.Run("segwit", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			btcdEncodeWitnessPubKeyHash(hash)
		}
	})
	b.Run("p2sh", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			btcdEncodeScriptHash(hash)
		}
	})
	b.Run("taproot", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			btcdEncodeTaproot(key)
		}
	})
}