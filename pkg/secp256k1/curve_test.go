package secp256k1

import (
	"math/big"
	"testing"
)

func TestGeneratorPoint(t *testing.T) {
	if !G.IsOnCurve() {
		t.Fatalf("Base generator point G is not on the secp256k1 curve!")
	}
}

func TestAffineArithmetic(t *testing.T) {
	// G + G == 2G
	g2Add := AffineAdd(G, G)
	g2Double := AffineDouble(G)
	g2Mul := ScalarMul(big.NewInt(2), G)

	if !g2Add.Equals(g2Double) {
		t.Fatalf("AffineAdd(G, G) != AffineDouble(G)")
	}
	if !g2Mul.Equals(g2Double) {
		t.Fatalf("ScalarMul(2, G) != AffineDouble(G)")
	}
	if !g2Double.IsOnCurve() {
		t.Fatalf("2*G is not on curve!")
	}

	// 3*G == 2*G + G
	g3Add := AffineAdd(g2Double, G)
	g3Mul := ScalarMul(big.NewInt(3), G)
	if !g3Add.Equals(g3Mul) {
		t.Fatalf("AffineAdd(2G, G) != ScalarMul(3, G)")
	}
	if !g3Add.IsOnCurve() {
		t.Fatalf("3*G is not on curve!")
	}
}

func TestBatchAffineAddMatchesSingle(t *testing.T) {
	b := 16
	points := make([]Point, b)
	jumps := make([]Point, b)

	for i := 0; i < b; i++ {
		points[i] = ScalarMul(big.NewInt(int64(100+i*7)), G)
		jumps[i] = ScalarMul(big.NewInt(int64(500+i*13)), G)
	}

	batchRes := BatchAffineAdd(points, jumps)

	for i := 0; i < b; i++ {
		singleRes := AffineAdd(points[i], jumps[i])
		if !batchRes[i].Equals(singleRes) {
			t.Fatalf("BatchAffineAdd mismatch at index %d! Batch=(%x, %x), Single=(%x, %x)",
				i, batchRes[i].X.ToBig(), batchRes[i].Y.ToBig(), singleRes.X.ToBig(), singleRes.Y.ToBig())
		}
	}
}
