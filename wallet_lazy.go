package main

import (
	"btcfind/bitcoin"

	"github.com/bits-and-blooms/bloom/v3"
	"github.com/btcsuite/btcd/btcec/v2"
	"github.com/btcsuite/btcd/btcec/v2/schnorr"
	"github.com/btcsuite/btcd/btcutil"
	"github.com/btcsuite/btcd/txscript"
)

func genWalletStaged(sets FundedSets, mask bitcoin.FormatMask) bitcoin.Wallet {
	privKey, err := btcec.NewPrivateKey()
	if err != nil {
		panic(err)
	}

	return bitcoin.Wallet{
		PrivKey: privKey.Serialize(),
		Keys:    deriveKeysStaged(privKey.PubKey(), sets, mask),
	}
}

func deriveKeysStaged(pubKey *btcec.PublicKey, sets FundedSets, mask bitcoin.FormatMask) bitcoin.LookupKeys {
	var keys bitcoin.LookupKeys

	if mask.NeedsCompressedPubkeyHash() {
		compressed := pubKey.SerializeCompressed()
		copy(keys.CompressedHash[:], btcutil.Hash160(compressed))
	}

	if mask.P2SH && mask.NeedsCompressedPubkeyHash() && len(sets.P2SH) > 0 {
		var redeemScript [22]byte
		redeemScript[0] = 0x00
		redeemScript[1] = 0x14
		copy(redeemScript[2:], keys.CompressedHash[:])
		copy(keys.P2SHHash[:], btcutil.Hash160(redeemScript[:]))
	}

	if mask.LegacyUncompressed && len(sets.Legacy) > 0 {
		copy(keys.UncompressedHash[:], btcutil.Hash160(pubKey.SerializeUncompressed()))
	}

	if shouldDeriveTaproot(sets, keys, mask) {
		tapKey := txscript.ComputeTaprootKeyNoScript(pubKey)
		copy(keys.TaprootKey[:], schnorr.SerializePubKey(tapKey))
	}

	return keys
}

func shouldDeriveTaproot(sets FundedSets, keys bitcoin.LookupKeys, mask bitcoin.FormatMask) bool {
	if !mask.Taproot || len(sets.TaprootV1) == 0 {
		return false
	}
	if !mask.LegacyCompressed && !mask.LegacyUncompressed && !mask.Segwit && !mask.P2SH {
		return true
	}
	return hash160BloomMaybe(sets, keys, mask)
}

func hash160BloomMaybe(sets FundedSets, keys bitcoin.LookupKeys, mask bitcoin.FormatMask) bool {
	if mask.LegacyCompressed && bloomTest20(sets.LegacyBloom, keys.CompressedHash) {
		return true
	}
	if mask.Segwit && bloomTest20(sets.SegwitV0Bloom, keys.CompressedHash) {
		return true
	}
	if mask.LegacyUncompressed && bloomTest20(sets.LegacyBloom, keys.UncompressedHash) {
		return true
	}
	if mask.P2SH && bloomTest20(sets.P2SHBloom, keys.P2SHHash) {
		return true
	}
	return false
}

func bloomTest20(bf *bloom.BloomFilter, key [20]byte) bool {
	if bf == nil {
		return false
	}
	return bf.Test(key[:])
}