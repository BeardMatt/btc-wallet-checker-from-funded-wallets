package main

import (
	"fmt"

	"btcfind/bitcoin"
)

func applyInjectIndexHit(sets FundedSets, keys *bitcoin.LookupKeys, bucket string) (fundedAddress string, err error) {
	switch bucket {
	case "legacy":
		if len(sets.Legacy) == 0 {
			return "", fmt.Errorf("legacy bucket is empty")
		}
		keys.CompressedHash = sets.Legacy[0]
		return bitcoin.EncodePubKeyHash(sets.Legacy[0]), nil
	case "legacy-uncompressed":
		if len(sets.Legacy) == 0 {
			return "", fmt.Errorf("legacy bucket is empty")
		}
		keys.UncompressedHash = sets.Legacy[0]
		return bitcoin.EncodePubKeyHash(sets.Legacy[0]), nil
	case "p2sh":
		if len(sets.P2SH) == 0 {
			return "", fmt.Errorf("p2sh bucket is empty")
		}
		keys.P2SHHash = sets.P2SH[0]
		return bitcoin.EncodeScriptHash(sets.P2SH[0]), nil
	case "segwit":
		if len(sets.SegwitV0) == 0 {
			return "", fmt.Errorf("segwit bucket is empty")
		}
		keys.CompressedHash = sets.SegwitV0[0]
		return bitcoin.EncodeWitnessPubKeyHash(sets.SegwitV0[0]), nil
	case "taproot":
		if len(sets.TaprootV1) == 0 {
			return "", fmt.Errorf("taproot bucket is empty")
		}
		keys.TaprootKey = sets.TaprootV1[0]
		return bitcoin.EncodeTaprootKey(sets.TaprootV1[0]), nil
	default:
		return "", fmt.Errorf("unknown inject bucket %q", bucket)
	}
}

func printLookupVerify(sets FundedSets, keys bitcoin.LookupKeys, realKind bitcoin.MatchKind, realHit bool) {
	fmt.Printf(
		"lookup verify: legacy_compressed=%t legacy_uncompressed=%t segwit_v0=%t p2sh=%t taproot=%t real_match=%t",
		lookupHit20(sets.LegacyBloom, sets.Legacy, keys.CompressedHash),
		lookupHit20(sets.LegacyBloom, sets.Legacy, keys.UncompressedHash),
		lookupHit20(sets.SegwitV0Bloom, sets.SegwitV0, keys.CompressedHash),
		lookupHit20(sets.P2SHBloom, sets.P2SH, keys.P2SHHash),
		lookupHit32(sets.TaprootV1Bloom, sets.TaprootV1, keys.TaprootKey),
		realHit,
	)
	if realHit {
		fmt.Printf(" kind=%s", matchKindName(realKind))
	}
	fmt.Println()
}

func lookupHit20(bf interface{ Test([]byte) bool }, set [][20]byte, key [20]byte) bool {
	if bf != nil && !bf.Test(key[:]) {
		return false
	}
	return inFunded20(set, key)
}

func lookupHit32(bf interface{ Test([]byte) bool }, set [][32]byte, key [32]byte) bool {
	if bf != nil && !bf.Test(key[:]) {
		return false
	}
	return inFunded32(set, key)
}

func matchKindName(kind bitcoin.MatchKind) string {
	switch kind {
	case bitcoin.MatchLegacyCompressed:
		return "legacy_compressed"
	case bitcoin.MatchLegacyUncompressed:
		return "legacy_uncompressed"
	case bitcoin.MatchSegwitV0:
		return "segwit_v0"
	case bitcoin.MatchP2SH:
		return "p2sh"
	case bitcoin.MatchTaproot:
		return "taproot"
	default:
		return "unknown"
	}
}

func printHit(wallet bitcoin.Wallet, kind bitcoin.MatchKind, simulated bool) {
	addr := bitcoin.EncodeMatchAddress(kind, wallet.Keys)
	if addr == "" {
		panic("failed to encode hit address")
	}

	prefix := ""
	if simulated {
		prefix = "(simulated hit) "
	}
	fmt.Println(prefix+formatWIF(wallet.PrivKey), " : ", addr)
}