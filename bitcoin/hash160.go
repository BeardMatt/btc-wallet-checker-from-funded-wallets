package bitcoin

import (
	sha256simd "github.com/minio/sha256-simd"
	"golang.org/x/crypto/ripemd160"
)

func Hash160(data []byte) []byte {
	shaSum := sha256simd.Sum256(data)
	r := ripemd160.New()
	_, _ = r.Write(shaSum[:])
	out := make([]byte, 20)
	copy(out, r.Sum(nil))
	return out
}

func Hash160To20(data []byte) [20]byte {
	var out [20]byte
	copy(out[:], Hash160(data))
	return out
}