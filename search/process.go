package search

import (
	"fmt"

	"btcfind/bitcoin"
	"btcfind/funded"
)

// ProcessWallet checks one generated wallet against the funded index and invokes hooks.
func ProcessWallet(cfg Config, sets funded.Sets, wallet bitcoin.Wallet, keyIndex int, hooks Hooks) (bool, error) {
	keys := wallet.Keys
	if cfg.InjectIndexHit != "" && keyIndex == cfg.SimulateHitAt {
		fundedAddr, err := applyInjectIndexHit(sets, &keys, cfg.InjectIndexHit)
		if err != nil {
			return false, err
		}
		if hooks.OnInjectLog != nil {
			hooks.OnInjectLog(cfg.InjectIndexHit, fundedAddr)
		}
	}

	kind, ok := funded.Match(sets, keys, cfg.Formats)
	if ok {
		wallet.Keys = keys
		if hooks.OnHit != nil {
			if err := hooks.OnHit(HitEvent{
				Wallet:      wallet,
				Kind:        kind,
				KeyIndex:    keyIndex,
				BalanceSats: sets.BalanceForMatch(kind, keys),
				Simulated:   false,
			}); err != nil {
				return true, err
			}
		}
	}

	if cfg.VerifyLookup && keyIndex == cfg.SimulateHitAt {
		emitLookupVerify(hooks, sets, keys, kind, ok)
	}

	if cfg.SimulateHit && keyIndex == cfg.SimulateHitAt && !ok {
		wallet.Keys = keys
		if hooks.OnHit != nil {
			if err := hooks.OnHit(HitEvent{
				Wallet:      wallet,
				Kind:        simulateDisplayKind(keys, kind, ok),
				KeyIndex:    keyIndex,
				BalanceSats: 0,
				Simulated:   true,
			}); err != nil {
				return false, err
			}
		}
	}

	return ok, nil
}

func applyInjectIndexHit(sets funded.Sets, keys *bitcoin.LookupKeys, bucket string) (fundedAddress string, err error) {
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

func emitLookupVerify(hooks Hooks, sets funded.Sets, keys bitcoin.LookupKeys, realKind bitcoin.MatchKind, realHit bool) {
	if hooks.OnLookupVerify == nil {
		return
	}
	event := LookupVerifyEvent{
		LegacyCompressed:   funded.LookupHit20(sets.LegacyBloom, sets.Legacy, keys.CompressedHash),
		LegacyUncompressed: funded.LookupHit20(sets.LegacyBloom, sets.Legacy, keys.UncompressedHash),
		SegwitV0:           funded.LookupHit20(sets.SegwitV0Bloom, sets.SegwitV0, keys.CompressedHash),
		P2SH:               funded.LookupHit20(sets.P2SHBloom, sets.P2SH, keys.P2SHHash),
		Taproot:            funded.LookupHit32(sets.TaprootV1Bloom, sets.TaprootV1, keys.TaprootKey),
		RealMatch:          realHit,
	}
	if realHit {
		event.Kind = realKind
		event.KindName = funded.MatchKindName(realKind)
	}
	hooks.OnLookupVerify(event)
}

func simulateDisplayKind(keys bitcoin.LookupKeys, matchedKind bitcoin.MatchKind, matched bool) bitcoin.MatchKind {
	if matched {
		return matchedKind
	}
	return bitcoin.MatchLegacyCompressed
}