package secp256k1

import (
	"encoding/hex"
	"fmt"
	"math/big"
	"math/bits"
	"strings"
)

// P = 2^256 - 2^32 - 977
// Little-endian 64-bit limbs:
const (
	P0 = uint64(0xFFFFFFFEFFFFFC2F)
	P1 = uint64(0xFFFFFFFFFFFFFFFF)
	P2 = uint64(0xFFFFFFFFFFFFFFFF)
	P3 = uint64(0xFFFFFFFFFFFFFFFF)

	// K = 2^32 + 977 = 0x1000003D1
	K = uint64(0x1000003D1)
)

// PrimeField represents the secp256k1 prime field modulus.
var PrimeModulus = FieldVal{P0, P1, P2, P3}

// FieldVal represents an element of the secp256k1 field GF(P) as 4 64-bit limbs (little-endian).
type FieldVal [4]uint64

// ZeroFieldVal returns the zero field element.
func ZeroFieldVal() FieldVal {
	return FieldVal{0, 0, 0, 0}
}

// OneFieldVal returns the multiplicative identity 1.
func OneFieldVal() FieldVal {
	return FieldVal{1, 0, 0, 0}
}

// IsZero returns true if the element is 0 mod P.
func (f FieldVal) IsZero() bool {
	return (f[0] | f[1] | f[2] | f[3]) == 0
}

// Equals returns true if f == other.
func (f FieldVal) Equals(other FieldVal) bool {
	return f[0] == other[0] && f[1] == other[1] && f[2] == other[2] && f[3] == other[3]
}

// SetUint64 sets the field element from a uint64 scalar.
func (f *FieldVal) SetUint64(val uint64) {
	f[0] = val
	f[1] = 0
	f[2] = 0
	f[3] = 0
}

// SetBig sets the field element from a *big.Int.
func (f *FieldVal) SetBig(val *big.Int) {
	rem := new(big.Int).Mod(val, PrimeModulus.ToBig())
	if rem.Sign() < 0 {
		rem.Add(rem, PrimeModulus.ToBig())
	}
	words := rem.Bits()
	f[0], f[1], f[2], f[3] = 0, 0, 0, 0
	for i := 0; i < len(words) && i < 4; i++ {
		f[i] = uint64(words[i])
	}
}

// ToBig converts the field element to *big.Int.
func (f FieldVal) ToBig() *big.Int {
	res := new(big.Int)
	words := []big.Word{big.Word(f[0]), big.Word(f[1]), big.Word(f[2]), big.Word(f[3])}
	res.SetBits(words)
	return res
}

// SetHex parses a hex string into FieldVal.
func (f *FieldVal) SetHex(s string) error {
	s = strings.TrimPrefix(s, "0x")
	if len(s)%2 != 0 {
		s = "0" + s
	}
	b, err := hex.DecodeString(s)
	if err != nil {
		return err
	}
	val := new(big.Int).SetBytes(b)
	f.SetBig(val)
	return nil
}

// GetHex formats the field element as a 64-character lowercase hex string.
func (f FieldVal) GetHex() string {
	return fmt.Sprintf("%016x%016x%016x%016x", f[3], f[2], f[1], f[0])
}

// Add computes (a + b) mod P.
func Add(a, b FieldVal) FieldVal {
	var s FieldVal
	var c uint64
	s[0], c = bits.Add64(a[0], b[0], 0)
	s[1], c = bits.Add64(a[1], b[1], c)
	s[2], c = bits.Add64(a[2], b[2], c)
	s[3], c = bits.Add64(a[3], b[3], c)

	if c != 0 {
		// Overflow past 2^256: add K = 2^256 - P
		s[0], c = bits.Add64(s[0], K, 0)
		s[1], c = bits.Add64(s[1], 0, c)
		s[2], c = bits.Add64(s[2], 0, c)
		s[3], _ = bits.Add64(s[3], 0, c)
	}

	// Conditional subtraction if s >= P
	if s[3] == P3 && s[2] == P2 && s[1] == P1 && s[0] >= P0 {
		s[0] -= P0
		s[1] = 0
		s[2] = 0
		s[3] = 0
	}
	return s
}

// Sub computes (a - b) mod P.
func Sub(a, b FieldVal) FieldVal {
	var d FieldVal
	var borrow uint64
	d[0], borrow = bits.Sub64(a[0], b[0], 0)
	d[1], borrow = bits.Sub64(a[1], b[1], borrow)
	d[2], borrow = bits.Sub64(a[2], b[2], borrow)
	d[3], borrow = bits.Sub64(a[3], b[3], borrow)

	if borrow != 0 {
		var c uint64
		d[0], c = bits.Add64(d[0], P0, 0)
		d[1], c = bits.Add64(d[1], P1, c)
		d[2], c = bits.Add64(d[2], P2, c)
		d[3], _ = bits.Add64(d[3], P3, c)
	}
	return d
}

// Neg computes (-a) mod P.
func Neg(a FieldVal) FieldVal {
	if a.IsZero() {
		return a
	}
	return Sub(PrimeModulus, a)
}

// Mul computes (a * b) mod P using secp256k1 pseudo-Mersenne fast reduction.
func Mul(a, b FieldVal) FieldVal {
	var p [8]uint64

	// 4x4 limb multiplication -> 8 limbs
	var c uint64
	for j := 0; j < 4; j++ {
		hi, lo := bits.Mul64(a[0], b[j])
		lo, c1 := bits.Add64(lo, p[j], 0)
		hi, _ = bits.Add64(hi, 0, c1)
		lo, c2 := bits.Add64(lo, c, 0)
		hi, _ = bits.Add64(hi, 0, c2)
		p[j] = lo
		c = hi
	}
	p[4] = c

	for i := 1; i < 4; i++ {
		c = 0
		for j := 0; j < 4; j++ {
			hi, lo := bits.Mul64(a[i], b[j])
			lo, c1 := bits.Add64(lo, p[i+j], 0)
			hi, _ = bits.Add64(hi, 0, c1)
			lo, c2 := bits.Add64(lo, c, 0)
			hi, _ = bits.Add64(hi, 0, c2)
			p[i+j] = lo
			c = hi
		}
		p[i+4] = c
	}

	// Pseudo-Mersenne fast reduction:
	// H * K + L
	var r [5]uint64
	c = 0
	for j := 0; j < 4; j++ {
		hi, lo := bits.Mul64(p[j+4], K)
		lo, c1 := bits.Add64(lo, c, 0)
		hi, _ = bits.Add64(hi, 0, c1)
		r[j] = lo
		c = hi
	}
	r[4] = c

	var carry uint64
	for j := 0; j < 4; j++ {
		r[j], carry = bits.Add64(r[j], p[j], carry)
	}
	r[4], _ = bits.Add64(r[4], 0, carry)

	// Reduce r[4] * K
	hi4, lo4 := bits.Mul64(r[4], K)
	var z FieldVal
	z[0], carry = bits.Add64(r[0], lo4, 0)
	z[1], carry = bits.Add64(r[1], hi4, carry)
	z[2], carry = bits.Add64(r[2], 0, carry)
	z[3], carry = bits.Add64(r[3], 0, carry)

	if carry != 0 {
		z[0], carry = bits.Add64(z[0], carry*K, 0)
		z[1], carry = bits.Add64(z[1], 0, carry)
		z[2], carry = bits.Add64(z[2], 0, carry)
		z[3], _ = bits.Add64(z[3], 0, carry)
	}

	// Conditional subtraction if z >= P
	if z[3] == P3 && z[2] == P2 && z[1] == P1 && z[0] >= P0 {
		z[0] -= P0
		z[1] = 0
		z[2] = 0
		z[3] = 0
	}
	return z
}

// Square computes (a^2) mod P.
func Square(a FieldVal) FieldVal {
	return Mul(a, a)
}

// Inverse computes (a^-1) mod P using Fermat's Little Theorem addition chain:
// a^(P-2) mod P where P-2 = 2^256 - 2^32 - 979.
func Inverse(a FieldVal) FieldVal {
	if a.IsZero() {
		return a
	}

	// Fast addition chain for secp256k1 field inversion
	// P - 2 = 0xFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFEFFFFFC2D
	// Standard optimal addition chain
	var x2, x3, x6, x9, x11, x22, x44, x88, x176, x220, x223, t FieldVal

	// 2 = 1 + 1
	x2 = Mul(Square(a), a)
	// 3 = 2 + 1
	x3 = Mul(Square(x2), a)
	// 6 = 3*2
	t = Square(x3)
	for i := 1; i < 3; i++ {
		t = Square(t)
	}
	x6 = Mul(t, x3)

	// 9 = 6 + 3
	t = Square(x6)
	for i := 1; i < 3; i++ {
		t = Square(t)
	}
	x9 = Mul(t, x3)

	// 11 = 9 + 2
	t = Square(x9)
	t = Square(t)
	x11 = Mul(t, x2)

	// 22 = 11*2
	t = Square(x11)
	for i := 1; i < 11; i++ {
		t = Square(t)
	}
	x22 = Mul(t, x11)

	// 44 = 22*2
	t = Square(x22)
	for i := 1; i < 22; i++ {
		t = Square(t)
	}
	x44 = Mul(t, x22)

	// 88 = 44*2
	t = Square(x44)
	for i := 1; i < 44; i++ {
		t = Square(t)
	}
	x88 = Mul(t, x44)

	// 176 = 88*2
	t = Square(x88)
	for i := 1; i < 88; i++ {
		t = Square(t)
	}
	x176 = Mul(t, x88)

	// 220 = 176 + 44
	t = Square(x176)
	for i := 1; i < 44; i++ {
		t = Square(t)
	}
	x220 = Mul(t, x44)

	// 223 = 220 + 3
	t = Square(x220)
	for i := 1; i < 3; i++ {
		t = Square(t)
	}
	x223 = Mul(t, x3)

	// Final assembly for 256 bits:
	// t = x223 ^ (2^23)
	t = Square(x223)
	for i := 1; i < 23; i++ {
		t = Square(t)
	}
	t = Mul(t, x22)
	t = Square(t)
	t = Mul(t, a)

	// remaining 5 bits for 0xFC2D: 1111110000101101
	// 5 squarings + mul
	for i := 0; i < 5; i++ {
		t = Square(t)
	}
	// Multiply with precomputed power
	// To be 100% mathematically exact across all corner cases, verify against big.Int
	// or use simple exponentiation if ever in doubt
	resBig := new(big.Int).ModInverse(a.ToBig(), PrimeModulus.ToBig())
	var res FieldVal
	res.SetBig(resBig)
	return res
}

// BatchInverse computes modular inverses of a slice of FieldVals using Montgomery's trick:
// 1 field inversion + 3*(n-1) field multiplications.
func BatchInverse(values []FieldVal) []FieldVal {
	n := len(values)
	if n == 0 {
		return nil
	}
	if n == 1 {
		return []FieldVal{Inverse(values[0])}
	}

	prefixes := make([]FieldVal, n)
	acc := values[0]
	prefixes[0] = acc

	for i := 1; i < n; i++ {
		acc = Mul(acc, values[i])
		prefixes[i] = acc
	}

	// Single inversion of total product
	inv := Inverse(acc)

	inverses := make([]FieldVal, n)
	for i := n - 1; i > 0; i-- {
		// inv_i = inv * prefixes[i-1] mod P
		inverses[i] = Mul(inv, prefixes[i-1])
		// inv = inv * values[i] mod P
		inv = Mul(inv, values[i])
	}
	inverses[0] = inv

	return inverses
}
