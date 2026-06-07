package bitcoin

import (
	"github.com/btcsuite/btcd/btcec/v2"
	"github.com/btcsuite/btcd/btcec/v2/schnorr"
	"github.com/btcsuite/btcd/btcutil"
	"github.com/btcsuite/btcd/chaincfg"
	"github.com/btcsuite/btcd/txscript"
)

type Wallet struct {
	Privkey   string
	Addresses []string
}

func GenKeypair() Wallet {
	privKey, err := btcec.NewPrivateKey()
	if err != nil {
		panic(err)
	}

	wif, err := btcutil.NewWIF(privKey, &chaincfg.MainNetParams, true)
	if err != nil {
		panic(err)
	}

	pubKey := privKey.PubKey()
	compressed := pubKey.SerializeCompressed()
	pubKeyHash := btcutil.Hash160(compressed)

	addresses := make([]string, 0, 5)

	add := func(addr btcutil.Address) {
		addresses = append(addresses, addr.EncodeAddress())
	}

	if addr, err := btcutil.NewAddressPubKeyHash(pubKeyHash, &chaincfg.MainNetParams); err == nil {
		add(addr)
	}

	if addr, err := btcutil.NewAddressPubKeyHash(btcutil.Hash160(pubKey.SerializeUncompressed()), &chaincfg.MainNetParams); err == nil {
		add(addr)
	}

	if addr, err := btcutil.NewAddressWitnessPubKeyHash(pubKeyHash, &chaincfg.MainNetParams); err == nil {
		add(addr)
	}

	redeemScript := append([]byte{0x00, 0x14}, pubKeyHash...)
	if addr, err := btcutil.NewAddressScriptHash(redeemScript, &chaincfg.MainNetParams); err == nil {
		add(addr)
	}

	tapKey := txscript.ComputeTaprootKeyNoScript(pubKey)
	if addr, err := btcutil.NewAddressTaproot(schnorr.SerializePubKey(tapKey), &chaincfg.MainNetParams); err == nil {
		add(addr)
	}

	return Wallet{
		Privkey:   wif.String(),
		Addresses: addresses,
	}
}