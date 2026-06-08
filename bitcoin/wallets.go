package bitcoin

import (
	"github.com/btcsuite/btcd/btcec/v2"
)

type Wallet struct {
	PrivKey []byte
	Keys    LookupKeys
}

func GenKeypair() Wallet {
	return GenKeypairMasked(AllFormats())
}

func GenKeypairMasked(mask FormatMask) Wallet {
	privKey, err := btcec.NewPrivateKey()
	if err != nil {
		panic(err)
	}

	return Wallet{
		PrivKey: privKey.Serialize(),
		Keys:    DeriveLookupKeysMasked(privKey, mask),
	}
}