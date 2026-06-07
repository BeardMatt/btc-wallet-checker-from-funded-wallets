package bitcoin

import (
	"fmt"
	"strings"

	"github.com/btcsuite/btcd/btcutil"
	"github.com/btcsuite/btcd/btcutil/base58"
	"github.com/btcsuite/btcd/btcutil/bech32"
	"github.com/btcsuite/btcd/chaincfg"
)

type AddressKind int

const (
	KindLegacy AddressKind = iota
	KindP2SH
	KindSegwitV0
	KindTaprootV1
)

func DecodeFundedAddress(addr string) (AddressKind, []byte, error) {
	decoded, err := btcutil.DecodeAddress(addr, &chaincfg.MainNetParams)
	if err != nil {
		return 0, nil, err
	}

	switch a := decoded.(type) {
	case *btcutil.AddressPubKeyHash:
		hash := a.ScriptAddress()
		if len(hash) != 20 {
			return 0, nil, fmt.Errorf("unexpected legacy hash length %d", len(hash))
		}
		return KindLegacy, hash, nil
	case *btcutil.AddressScriptHash:
		hash := a.ScriptAddress()
		if len(hash) != 20 {
			return 0, nil, fmt.Errorf("unexpected p2sh hash length %d", len(hash))
		}
		return KindP2SH, hash, nil
	case *btcutil.AddressWitnessPubKeyHash:
		hash := a.ScriptAddress()
		if len(hash) != 20 {
			return 0, nil, fmt.Errorf("unexpected segwit v0 hash length %d", len(hash))
		}
		return KindSegwitV0, hash, nil
	case *btcutil.AddressTaproot:
		hash := a.ScriptAddress()
		if len(hash) != 32 {
			return 0, nil, fmt.Errorf("unexpected taproot hash length %d", len(hash))
		}
		return KindTaprootV1, hash, nil
	default:
		return 0, nil, fmt.Errorf("unsupported address type %T", decoded)
	}
}

// DecodeFundedAddressForLoad uses lightweight parsing for bulk TSV loading.
func DecodeFundedAddressForLoad(addr string) (AddressKind, []byte, error) {
	if len(addr) == 0 {
		return 0, nil, fmt.Errorf("empty address")
	}

	switch addr[0] {
	case '1':
		payload, version, err := base58.CheckDecode(addr)
		if err != nil {
			return 0, nil, err
		}
		if version != chaincfg.MainNetParams.PubKeyHashAddrID || len(payload) != 20 {
			return 0, nil, fmt.Errorf("invalid legacy address")
		}
		return KindLegacy, payload, nil
	case '3':
		payload, version, err := base58.CheckDecode(addr)
		if err != nil {
			return 0, nil, err
		}
		if version != chaincfg.MainNetParams.ScriptHashAddrID || len(payload) != 20 {
			return 0, nil, fmt.Errorf("invalid p2sh address")
		}
		return KindP2SH, payload, nil
	default:
		if strings.HasPrefix(addr, "bc1") {
			return decodeBech32ForLoad(addr)
		}
		return 0, nil, fmt.Errorf("unsupported prefix")
	}
}

func decodeBech32ForLoad(addr string) (AddressKind, []byte, error) {
	hrp, data, bech32version, err := bech32.DecodeGeneric(addr)
	if err != nil {
		return 0, nil, err
	}
	if hrp != chaincfg.MainNetParams.Bech32HRPSegwit {
		return 0, nil, fmt.Errorf("unexpected hrp %q", hrp)
	}
	if len(data) < 1 {
		return 0, nil, fmt.Errorf("no witness version")
	}

	version := data[0]
	regrouped, err := bech32.ConvertBits(data[1:], 5, 8, false)
	if err != nil {
		return 0, nil, err
	}

	switch {
	case version == 0 && len(regrouped) == 20 && bech32version == bech32.Version0:
		return KindSegwitV0, regrouped, nil
	case version == 1 && len(regrouped) == 32 && bech32version == bech32.VersionM:
		return KindTaprootV1, regrouped, nil
	default:
		return 0, nil, fmt.Errorf("unsupported witness program")
	}
}