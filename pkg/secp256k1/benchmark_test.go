package secp256k1

import (
	"math/big"
	"testing"
)

func BenchmarkSingleAffineAdd(b *testing.B) {
	p1 := ScalarMul(big.NewInt(1234567), G)
	p2 := ScalarMul(big.NewInt(7654321), G)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		p1 = AffineAdd(p1, p2)
	}
}

func BenchmarkBatchAffineAdd32(b *testing.B) {
	batchSize := 32
	points := make([]Point, batchSize)
	jumps := make([]Point, batchSize)

	for i := 0; i < batchSize; i++ {
		points[i] = ScalarMul(big.NewInt(int64(1000+i*3)), G)
		jumps[i] = ScalarMul(big.NewInt(int64(5000+i*7)), G)
	}

	ctx := NewBatchContext(batchSize)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		ctx.Step(points, jumps)
	}
	b.ReportMetric(float64(b.N*batchSize)/b.Elapsed().Seconds(), "points/s")
}
