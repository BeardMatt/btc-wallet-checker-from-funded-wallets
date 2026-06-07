package bitcoin

import "testing"

func BenchmarkGenKeypair(b *testing.B) {
	for i := 0; i < b.N; i++ {
		GenKeypair()
	}
}