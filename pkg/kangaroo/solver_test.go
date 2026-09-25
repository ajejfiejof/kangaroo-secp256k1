package kangaroo

import (
	"crypto/rand"
	"math/big"
	"testing"

	"github.com/ajejfiejof/kangaroo-secp256k1/pkg/secp256k1"
)

func TestSolverKeyRecovery(t *testing.T) {
	testBits := []int{16, 20, 24, 28}
	solver := NewSolver(Config{
		NumGoroutines: 4,
		BatchSize:     32,
		NumJumps:      32,
	})

	for _, bits := range testBits {
		rmin := new(big.Int).Lsh(big.NewInt(1), uint(bits-1))
		rmax := new(big.Int).Lsh(big.NewInt(1), uint(bits))
		diff := new(big.Int).Sub(rmax, rmin)

		offset, _ := rand.Int(rand.Reader, diff)
		secret := new(big.Int).Add(rmin, offset)
		targetPub := secp256k1.ScalarMul(secret, secp256k1.G)

		res, err := solver.Solve(targetPub, rmin, rmax)
		if err != nil {
			t.Fatalf("%d-bit solve error: %v", bits, err)
		}
		if !res.Success || res.RecoveredKey == nil {
			t.Fatalf("%d-bit solve failed to recover key", bits)
		}
		if res.RecoveredKey.Cmp(secret) != 0 {
			t.Fatalf("%d-bit recovered key %x != expected %x", bits, res.RecoveredKey, secret)
		}
		t.Logf("%d-bit solve SUCCESS! Elapsed: %v, TotalOps: %d (%.0f ops/sec)",
			bits, res.Elapsed, res.TotalOperations, res.OperationsPerSec)
	}
}
