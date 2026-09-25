package secp256k1

import (
	"math/big"
)

// secp256k1 Curve Constants
var (
	// Gx = 0x79BE667EF9DCBBAC55A06295CE870B07029BFCDB2DCE28D959F2815B16F81798
	Gx = FieldVal{
		0x59F2815B16F81798,
		0x029BFCDB2DCE28D9,
		0x55A06295CE870B07,
		0x79BE667EF9DCBBAC,
	}
	// Gy = 0x483ADA7726A3C4655DA4FBFC0E1108A8FD17B448A68554199C47D08FFB10D4B8
	Gy = FieldVal{
		0x9C47D08FFB10D4B8,
		0xFD17B448A6855419,
		0x5DA4FBFC0E1108A8,
		0x483ADA7726A3C465,
	}

	// G is the secp256k1 base point generator.
	G = Point{X: Gx, Y: Gy}

	// Curve coefficient b = 7
	CurveB = FieldVal{7, 0, 0, 0}

	// N is the curve order
	CurveOrder, _ = new(big.Int).SetString("FFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFEBAAEDCE6AF48A03BBFD25E8CD0364141", 16)
)

// Point represents an affine elliptic curve point (X, Y).
type Point struct {
	X FieldVal
	Y FieldVal
}

// IsInfinity checks if point is at infinity (represented by zero coordinates).
func (p Point) IsInfinity() bool {
	return p.X.IsZero() && p.Y.IsZero()
}

// Equals checks if two points are identical.
func (p Point) Equals(other Point) bool {
	return p.X.Equals(other.X) && p.Y.Equals(other.Y)
}

// IsOnCurve checks if y^2 == x^3 + 7 mod P.
func (p Point) IsOnCurve() bool {
	if p.IsInfinity() {
		return true
	}
	y2 := Square(p.Y)
	x3 := Mul(Square(p.X), p.X)
	rhs := Add(x3, CurveB)
	return y2.Equals(rhs)
}

// AffineDouble computes 2 * p in affine coordinates.
func AffineDouble(p Point) Point {
	if p.IsInfinity() || p.Y.IsZero() {
		return Point{}
	}

	// lambda = (3 * x^2) / (2 * y) mod P
	x2 := Square(p.X)
	num := Add(Add(x2, x2), x2) // 3 * x^2
	den := Add(p.Y, p.Y)        // 2 * y
	lambda := Mul(num, Inverse(den))

	// x3 = lambda^2 - 2*x mod P
	x3 := Sub(Sub(Square(lambda), p.X), p.X)

	// y3 = lambda * (x - x3) - y mod P
	y3 := Sub(Mul(lambda, Sub(p.X, x3)), p.Y)

	return Point{X: x3, Y: y3}
}

// AffineAdd computes p1 + p2 in affine coordinates.
func AffineAdd(p1, p2 Point) Point {
	if p1.IsInfinity() {
		return p2
	}
	if p2.IsInfinity() {
		return p1
	}

	if p1.X.Equals(p2.X) {
		if !p1.Y.Equals(p2.Y) {
			// p1 + (-p1) = Infinity
			return Point{}
		}
		// Point doubling
		return AffineDouble(p1)
	}

	// lambda = (y2 - y1) / (x2 - x1) mod P
	dx := Sub(p2.X, p1.X)
	dy := Sub(p2.Y, p1.Y)
	lambda := Mul(dy, Inverse(dx))

	// x3 = lambda^2 - x1 - x2 mod P
	x3 := Sub(Sub(Square(lambda), p1.X), p2.X)

	// y3 = lambda * (x1 - x3) - y1 mod P
	y3 := Sub(Mul(lambda, Sub(p1.X, x3)), p1.Y)

	return Point{X: x3, Y: y3}
}

// ScalarMul computes k * p using double-and-add.
func ScalarMul(k *big.Int, p Point) Point {
	if k == nil || k.Sign() == 0 || p.IsInfinity() {
		return Point{}
	}

	res := Point{}
	cur := p

	bitLen := k.BitLen()
	for i := 0; i < bitLen; i++ {
		if k.Bit(i) == 1 {
			res = AffineAdd(res, cur)
		}
		cur = AffineDouble(cur)
	}
	return res
}

// BatchContext manages preallocated FieldVal buffers for zero-allocation Montgomery batch inversion.
type BatchContext struct {
	B        int
	Dxs      []FieldVal
	Dys      []FieldVal
	Prefixes []FieldVal
	Inverses []FieldVal
}

// NewBatchContext preallocates buffers for batch size B.
func NewBatchContext(b int) *BatchContext {
	return &BatchContext{
		B:        b,
		Dxs:      make([]FieldVal, b),
		Dys:      make([]FieldVal, b),
		Prefixes: make([]FieldVal, b),
		Inverses: make([]FieldVal, b),
	}
}

// Step advances B points in-place by adding jumps[i] with Montgomery batch inversion.
// COST: ~6 FieldVal multiplications per kangaroo step.
func (ctx *BatchContext) Step(points []Point, jumps []Point) {
	b := ctx.B

	// 1. Differences
	for i := 0; i < b; i++ {
		ctx.Dxs[i] = Sub(jumps[i].X, points[i].X)
		ctx.Dys[i] = Sub(jumps[i].Y, points[i].Y)
	}

	// 2. Montgomery forward pass
	ctx.Prefixes[0] = ctx.Dxs[0]
	for i := 1; i < b; i++ {
		ctx.Prefixes[i] = Mul(ctx.Prefixes[i-1], ctx.Dxs[i])
	}

	// 3. Single modular inversion
	inv := Inverse(ctx.Prefixes[b-1])

	// 4. Montgomery backward pass
	for i := b - 1; i > 0; i-- {
		ctx.Inverses[i] = Mul(inv, ctx.Prefixes[i-1])
		inv = Mul(inv, ctx.Dxs[i])
	}
	ctx.Inverses[0] = inv

	// 5. In-place point updates
	for i := 0; i < b; i++ {
		lambda := Mul(ctx.Dys[i], ctx.Inverses[i])
		x3 := Sub(Sub(Square(lambda), points[i].X), jumps[i].X)
		y3 := Sub(Mul(lambda, Sub(points[i].X, x3)), points[i].Y)
		points[i].X = x3
		points[i].Y = y3
	}
}

// BatchAffineAdd computes points[i] + jumps[i] using Montgomery batch inversion.
func BatchAffineAdd(points []Point, jumps []Point) []Point {
	b := len(points)
	res := make([]Point, b)
	copy(res, points)
	ctx := NewBatchContext(b)
	ctx.Step(res, jumps)
	return res
}
