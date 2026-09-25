package main

import (
	"crypto/rand"
	"flag"
	"fmt"
	"math/big"
	"os"
	"strings"
	"time"

	"github.com/ajejfiejof/kangaroo-secp256k1/pkg/kangaroo"
	"github.com/ajejfiejof/kangaroo-secp256k1/pkg/secp256k1"
)

func formatTime(d time.Duration) string {
	s := d.Seconds()
	if s < 0.001 {
		return fmt.Sprintf("%.2f ms", s*1000)
	}
	if s < 60 {
		return fmt.Sprintf("%.3f s", s)
	}
	if s < 3600 {
		return fmt.Sprintf("%.2f min", s/60)
	}
	if s < 86400 {
		return fmt.Sprintf("%.2f hours", s/3600)
	}
	if s < 31536000 {
		return fmt.Sprintf("%.2f days", s/86400)
	}
	return fmt.Sprintf("%.2f years", s/31536000)
}

func runConcurrencyScaling() {
	fmt.Println("\n" + strings.Repeat("=", 82))
	fmt.Println("  1. CONCURRENCY SCALING ON THINKPAD T440P (i7-4712MQ: 4 Cores / 8 Threads)")
	fmt.Println(strings.Repeat("=", 82))
	fmt.Printf("%-12s | %-12s | %-12s | %-18s | %-10s\n",
		"Goroutines", "Elapsed", "Total Ops", "Throughput (ops/s)", "Speedup")
	fmt.Println(strings.Repeat("-", 82))

	bits := 24
	rmin := new(big.Int).Lsh(big.NewInt(1), uint(bits-1))
	rmax := new(big.Int).Lsh(big.NewInt(1), uint(bits))
	diff := new(big.Int).Sub(rmax, rmin)

	// Fix target secret for fair comparison across thread counts
	secret := new(big.Int).Add(rmin, new(big.Int).Div(diff, big.NewInt(3)))
	targetPub := secp256k1.ScalarMul(secret, secp256k1.G)

	threadCounts := []int{1, 2, 4, 8}
	var baselineRate float64

	for _, tc := range threadCounts {
		solver := kangaroo.NewSolver(kangaroo.Config{
			NumGoroutines: tc,
			BatchSize:     32,
			NumJumps:      32,
		})

		res, err := solver.Solve(targetPub, rmin, rmax)
		if err != nil || !res.Success {
			fmt.Printf("%-12d | FAILED: %v\n", tc, err)
			continue
		}

		if tc == 1 {
			baselineRate = res.OperationsPerSec
		}
		speedup := res.OperationsPerSec / baselineRate

		fmt.Printf("%-12d | %-12s | %-12d | %-18.0f | %-10.2fx\n",
			tc, formatTime(res.Elapsed), res.TotalOperations, res.OperationsPerSec, speedup)
	}
}

func runIntervalScaling() {
	fmt.Println("\n" + strings.Repeat("=", 88))
	fmt.Println("  2. INTERVAL WIDTH SCALING (16 to 36 BITS) - 8 GOROUTINES")
	fmt.Println(strings.Repeat("=", 88))
	fmt.Printf("%-6s | %-18s | %-10s | %-10s | %-12s | %-16s\n",
		"Bits", "Interval Width", "Sqrt(W)", "Total Ops", "Elapsed", "Ops/Sec")
	fmt.Println(strings.Repeat("-", 88))

	testBits := []int{16, 20, 24, 28, 30, 32, 34, 36}
	solver := kangaroo.NewSolver(kangaroo.Config{
		NumGoroutines: 8,
		BatchSize:     32,
		NumJumps:      32,
	})

	var totalOps uint64
	var totalDuration time.Duration

	for _, bits := range testBits {
		rmin := new(big.Int).Lsh(big.NewInt(1), uint(bits-1))
		rmax := new(big.Int).Lsh(big.NewInt(1), uint(bits))
		diff := new(big.Int).Sub(rmax, rmin)

		offset, _ := rand.Int(rand.Reader, diff)
		secret := new(big.Int).Add(rmin, offset)
		targetPub := secp256k1.ScalarMul(secret, secp256k1.G)

		res, err := solver.Solve(targetPub, rmin, rmax)
		if err != nil || !res.Success {
			fmt.Printf("%-6d | FAILED\n", bits)
			continue
		}

		totalOps += res.TotalOperations
		totalDuration += res.Elapsed

		fmt.Printf("%-6d | %-18s | %-10s | %-10d | %-12s | %-16.0f\n",
			bits, res.Width.String(), res.SqrtW.String(), res.TotalOperations,
			formatTime(res.Elapsed), res.OperationsPerSec)
	}

	avgOpsSec := float64(totalOps) / totalDuration.Seconds()
	fmt.Println(strings.Repeat("-", 88))
	fmt.Printf("Average 8-Goroutine Throughput: %.0f point additions / sec\n", avgOpsSec)

	// Theoretical projections
	fmt.Println("\n" + strings.Repeat("=", 88))
	fmt.Println("  3. THEORETICAL EXTRAPOLATION FOR HIGHER PUZZLE INTERVALS")
	fmt.Println(strings.Repeat("=", 88))
	fmt.Printf("%-6s | %-18s | %-14s | %-16s | %-16s\n",
		"Bits", "Expected Ops", "Go (8-threads)", "Python Baseline", "Theoretical Speedup")
	fmt.Println(strings.Repeat("-", 88))

	higherBits := []int{40, 48, 54, 60, 68, 71}
	pythonRate := 28000.0 // Single-thread Python baseline

	for _, bits := range higherBits {
		w := new(big.Int).Lsh(big.NewInt(1), uint(bits-1))
		sqrtW := new(big.Int).Sqrt(w)
		expOpsF, _ := new(big.Float).SetInt(sqrtW).Float64()
		expOps := 1.714 * expOpsF

		goSec := expOps / avgOpsSec
		pySec := (3.0 * expOpsF) / pythonRate
		speedup := pySec / goSec

		fmt.Printf("%-6d | %-18.0f | %-14s | %-16s | %-16.1fx\n",
			bits, expOps, formatTime(time.Duration(goSec*1e9)),
			formatTime(time.Duration(pySec*1e9)), speedup)
	}
	fmt.Println(strings.Repeat("=", 88))
}

func main() {
	mode := flag.String("mode", "all", "Execution mode: 'all', 'threads', 'intervals', or 'solve'")
	flag.Parse()

	fmt.Println("================================================================================")
	fmt.Println("  POLLARD'S KANGAROO (SECP256K1) - HIGH-THROUGHPUT NATIVE GO SOLVER")
	fmt.Println("================================================================================")
	fmt.Println("• Pure Go 4-limb (256-bit) secp256k1 Pseudo-Mersenne Fast Reduction")
	fmt.Println("• Montgomery Batch Modular Inversion (O(1) Amortized Inversion)")
	fmt.Println("• Teske's Geometrically Distributed Jump Sets (r=32)")
	fmt.Println("• Galbraith-Pollard-Ruprai Asymmetric Multi-Kangaroo Topology")
	fmt.Println("• Sharded Zero-Allocation Trap Table with Goroutine Worker Pool")

	switch *mode {
	case "threads":
		runConcurrencyScaling()
	case "intervals":
		runIntervalScaling()
	case "all":
		runConcurrencyScaling()
		runIntervalScaling()
	default:
		fmt.Printf("Unknown mode: %s\n", *mode)
		os.Exit(1)
	}
}
