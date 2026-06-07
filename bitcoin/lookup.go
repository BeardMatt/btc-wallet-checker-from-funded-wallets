package bitcoin

import (
	"github.com/btcsuite/btcd/btcec/v2"
	"github.com/btcsuite/btcd/btcec/v2/schnorr"
	"github.com/btcsuite/btcd/btcutil"
	"github.com/btcsuite/btcd/txscript"
)

type LookupKeys struct {
	CompressedHash   [20]byte
	UncompressedHash [20]byte
	P2SHHash         [20]byte
	TaprootKey       [32]byte
}

func DeriveLookupKeys(privKey *btcec.PrivateKey) LookupKeys {
	pubKey := privKey.PubKey()
	compressed := pubKey.SerializeCompressed()

	var keys LookupKeys
	copy(keys.CompressedHash[:], btcutil.Hash160(compressed))
	copy(keys.UncompressedHash[:], btcutil.Hash160(pubKey.SerializeUncompressed()))

	var redeemScript [22]byte
	redeemScript[0] = 0x00
	redeemScript[1] = 0x14
	copy(redeemScript[2:], keys.CompressedHash[:])
	copy(keys.P2SHHash[:], btcutil.Hash160(redeemScript[:]))

	tapKey := txscript.ComputeTaprootKeyNoScript(pubKey)
	copy(keys.TaprootKey[:], schnorr.SerializePubKey(tapKey))

	return keys
}