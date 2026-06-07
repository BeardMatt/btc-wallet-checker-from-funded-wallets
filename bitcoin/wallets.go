package bitcoin

import (
	"github.com/btcsuite/btcd/btcec/v2"
)

type Wallet struct {
	PrivKey []byte
	Keys    LookupKeys
}

func GenKeypair() Wallet {
	privKey, err := btcec.NewPrivateKey()
	if err != nil {
		panic(err)
	}

	return Wallet{
		PrivKey: privKey.Serialize(),
		Keys:    DeriveLookupKeys(privKey),
	}
}