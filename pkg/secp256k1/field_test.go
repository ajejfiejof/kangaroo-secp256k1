package secp256k1

import (
	"crypto/rand"
	"math/big"
	"testing"
)

func TestFieldArithmeticAgainstBig(t *testing.T) {
	pBig := PrimeModulus.ToBig()

	for iter := 0; iter < 1000; iter++ {
		aBig, _ := rand.Int(rand.Reader, pBig)
		bBig, _ := rand.Int(rand.Reader, pBig)

		var a, b FieldVal
		a.SetBig(aBig)
		b.SetBig(bBig)

		// 1. Addition
		expAdd := new(big.Int).Mod(new(big.Int).Add(aBig, bBig), pBig)
		gotAdd := Add(a, b)
		if gotAdd.ToBig().Cmp(expAdd) != 0 {
			t.Fatalf("Add mismatch! a=%x, b=%x, got=%x, exp=%x", aBig, bBig, gotAdd.ToBig(), expAdd)
		}

		// 2. Subtraction
		expSub := new(big.Int).Mod(new(big.Int).Sub(aBig, bBig), pBig)
		if expSub.Sign() < 0 {
			expSub.Add(expSub, pBig)
		}
		gotSub := Sub(a, b)
		if gotSub.ToBig().Cmp(expSub) != 0 {
			t.Fatalf("Sub mismatch! a=%x, b=%x, got=%x, exp=%x", aBig, bBig, gotSub.ToBig(), expSub)
		}

		// 3. Multiplication
		expMul := new(big.Int).Mod(new(big.Int).Mul(aBig, bBig), pBig)
		gotMul := Mul(a, b)
		if gotMul.ToBig().Cmp(expMul) != 0 {
			t.Fatalf("Mul mismatch! a=%x, b=%x, got=%x, exp=%x", aBig, bBig, gotMul.ToBig(), expMul)
		}

		// 4. Squaring
		expSq := new(big.Int).Mod(new(big.Int).Mul(aBig, aBig), pBig)
		gotSq := Square(a)
		if gotSq.ToBig().Cmp(expSq) != 0 {
			t.Fatalf("Square mismatch! a=%x, got=%x, exp=%x", aBig, gotSq.ToBig(), expSq)
		}

		// 5. Negation
		expNeg := new(big.Int).Mod(new(big.Int).Neg(aBig), pBig)
		if expNeg.Sign() < 0 {
			expNeg.Add(expNeg, pBig)
		}
		gotNeg := Neg(a)
		if gotNeg.ToBig().Cmp(expNeg) != 0 {
			t.Fatalf("Neg mismatch! a=%x, got=%x, exp=%x", aBig, gotNeg.ToBig(), expNeg)
		}
	}
}

func TestFieldInverses(t *testing.T) {
	pBig := PrimeModulus.ToBig()

	for iter := 0; iter < 100; iter++ {
		aBig, _ := rand.Int(rand.Reader, pBig)
		if aBig.Sign() == 0 {
			continue
		}
		var a FieldVal
		a.SetBig(aBig)

		inv := Inverse(a)
		prod := Mul(a, inv)
		if !prod.Equals(OneFieldVal()) {
			t.Fatalf("Inverse failure! a=%x, inv=%x, prod=%x", aBig, inv.ToBig(), prod.ToBig())
		}
	}
}

func TestBatchInverse(t *testing.T) {
	pBig := PrimeModulus.ToBig()
	b := 32
	vals := make([]FieldVal, b)

	for i := 0; i < b; i++ {
		aBig, _ := rand.Int(rand.Reader, pBig)
		vals[i].SetBig(aBig)
	}

	invs := BatchInverse(vals)
	if len(invs) != b {
		t.Fatalf("BatchInverse length mismatch: %d != %d", len(invs), b)
	}

	for i := 0; i < b; i++ {
		prod := Mul(vals[i], invs[i])
		if !prod.Equals(OneFieldVal()) {
			t.Fatalf("BatchInverse failed at index %d: a=%x, inv=%x, prod=%x",
				i, vals[i].ToBig(), invs[i].ToBig(), prod.ToBig())
		}
	}
}
