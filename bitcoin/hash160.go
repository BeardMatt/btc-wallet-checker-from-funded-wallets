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