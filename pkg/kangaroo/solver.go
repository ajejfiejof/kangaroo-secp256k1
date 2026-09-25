package kangaroo

import (
	"context"
	"fmt"
	"math"
	"math/big"
	"sync"
	"sync/atomic"
	"time"

	"github.com/ajejfiejof/kangaroo-secp256k1/pkg/secp256k1"
)

// Config defines execution and tuning options for the solver.
type Config struct {
	NumGoroutines int
	BatchSize     int
	NumJumps      int
}

// SolveResult contains performance telemetry and the recovered secret key.
type SolveResult struct {
	RecoveredKey     *big.Int
	Success          bool
	RangeMin         *big.Int
	RangeMax         *big.Int
	Width            *big.Int
	SqrtW            *big.Int
	TameSteps        uint64
	WildSteps        uint64
	TotalOperations  uint64
	DistinguishedPts uint64
	Elapsed          time.Duration
	OperationsPerSec float64
	GoroutinesUsed   int
}

// Solver implements the high-throughput Galbraith-Pollard-Ruprai multi-kangaroo algorithm.
type Solver struct {
	Config Config
}

// NewSolver initializes a Solver with given configuration defaults.
func NewSolver(cfg Config) *Solver {
	if cfg.NumGoroutines <= 0 {
		cfg.NumGoroutines = 8
	}
	if cfg.BatchSize <= 0 {
		cfg.BatchSize = 32
	}
	if cfg.NumJumps <= 0 {
		cfg.NumJumps = 32
	}
	return &Solver{Config: cfg}
}

// Solve searches for x in [rangeMin, rangeMax] such that targetPub = x * G.
func (s *Solver) Solve(targetPub secp256k1.Point, rangeMin, rangeMax *big.Int) (*SolveResult, error) {
	width := new(big.Int).Sub(rangeMax, rangeMin)
	if width.Sign() <= 0 {
		return nil, fmt.Errorf("rangeMax must be strictly greater than rangeMin")
	}

	startTime := time.Now()
	sqrtW := new(big.Int).Sqrt(width)
	if sqrtW.Cmp(big.NewInt(4)) < 0 {
		sqrtW = big.NewInt(4)
	}

	// Mean jump size m ~ sqrt(W) / 4
	meanJump := new(big.Int).Div(sqrtW, big.NewInt(4))
	if meanJump.Sign() == 0 {
		meanJump = big.NewInt(1)
	}

	jumpSet := GenerateTeskeJumps(meanJump, s.Config.NumJumps)

	numTameTotal := s.Config.NumGoroutines * s.Config.BatchSize
	tameSpacing := new(big.Int).Div(new(big.Int).Mul(width, big.NewInt(2)), big.NewInt(int64(numTameTotal)))
	if tameSpacing.Sign() == 0 {
		tameSpacing = big.NewInt(1)
	}

	tameStepsPerBatch := int(new(big.Int).Div(new(big.Int).Mul(sqrtW, big.NewInt(6)), big.NewInt(int64(numTameTotal))).Int64()) + 80

	// Calibrate distinguished point mask: each kangaroo hits ~4-8 DPs
	dpBits := int(math.Log2(math.Max(4.0, float64(tameStepsPerBatch)/6.0)))
	if dpBits < 2 {
		dpBits = 2
	}
	if dpBits > 24 {
		dpBits = 24
	}
	dpMask := uint64((1 << dpBits) - 1)

	trapTable := NewTrapTable()

	// -------------------------------------------------------------
	// Phase 1: Batched Tame Herd (Parallel Goroutines)
	// -------------------------------------------------------------
	var tameOps uint64
	var tameWg sync.WaitGroup

	for g := 0; g < s.Config.NumGoroutines; g++ {
		tameWg.Add(1)
		go func(gID int) {
			defer tameWg.Done()
			b := s.Config.BatchSize
			batchCtx := secp256k1.NewBatchContext(b)
			points := make([]secp256k1.Point, b)
			scalars := make([]*big.Int, b)
			activeJumps := make([]secp256k1.Point, b)

			for i := 0; i < b; i++ {
				k := gID*b + i
				sOffset := new(big.Int).Mul(big.NewInt(int64(k)), tameSpacing)
				sVal := new(big.Int).Add(rangeMax, sOffset)
				scalars[i] = sVal
				pt := secp256k1.ScalarMul(sVal, secp256k1.G)
				points[i] = pt
				if (pt.X[0] & dpMask) == 0 {
					trapTable.Insert(pt.X, pt.Y, sVal)
				}
			}

			var localOps uint64
			for step := 0; step < tameStepsPerBatch; step++ {
				for i := 0; i < b; i++ {
					idx := jumpSet.JumpIndex(points[i].X)
					activeJumps[i] = jumpSet.Points[idx]
					scalars[i].Add(scalars[i], jumpSet.Scalars[idx])
				}

				batchCtx.Step(points, activeJumps)
				localOps += uint64(b)

				for i := 0; i < b; i++ {
					if (points[i].X[0] & dpMask) == 0 {
						trapTable.Insert(points[i].X, points[i].Y, scalars[i])
					}
				}
			}
			atomic.AddUint64(&tameOps, localOps)
		}(g)
	}
	tameWg.Wait()

	// -------------------------------------------------------------
	// Phase 2: Batched Wild Herd (Parallel Goroutines with Early Termination)
	// -------------------------------------------------------------
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	numWildTotal := s.Config.NumGoroutines * s.Config.BatchSize
	wildSpacing := new(big.Int).Div(width, big.NewInt(int64(numWildTotal)))
	if wildSpacing.Sign() == 0 {
		wildSpacing = big.NewInt(1)
	}

	var wildOps uint64
	var recoveredKey *big.Int
	var keyFound atomic.Bool
	var wildWg sync.WaitGroup

	maxWildStepsPerBatch := int(new(big.Int).Div(new(big.Int).Mul(sqrtW, big.NewInt(64)), big.NewInt(int64(numWildTotal))).Int64()) + 1500

	for g := 0; g < s.Config.NumGoroutines; g++ {
		wildWg.Add(1)
		go func(gID int) {
			defer wildWg.Done()
			b := s.Config.BatchSize
			batchCtx := secp256k1.NewBatchContext(b)
			points := make([]secp256k1.Point, b)
			scalars := make([]*big.Int, b)
			activeJumps := make([]secp256k1.Point, b)

			for i := 0; i < b; i++ {
				k := gID*b + i
				offset := new(big.Int).Mul(big.NewInt(int64(k)), wildSpacing)
				scalars[i] = new(big.Int).Set(offset)
				if offset.Sign() == 0 {
					points[i] = targetPub
				} else {
					offsetPt := secp256k1.ScalarMul(offset, secp256k1.G)
					points[i] = secp256k1.AffineAdd(targetPub, offsetPt)
				}
			}

			var localOps uint64
			for step := 0; step < maxWildStepsPerBatch; step++ {
				select {
				case <-ctx.Done():
					atomic.AddUint64(&wildOps, localOps)
					return
				default:
				}

				for i := 0; i < b; i++ {
					idx := jumpSet.JumpIndex(points[i].X)
					activeJumps[i] = jumpSet.Points[idx]
					scalars[i].Add(scalars[i], jumpSet.Scalars[idx])
				}

				batchCtx.Step(points, activeJumps)
				localOps += uint64(b)

				for i := 0; i < b; i++ {
					if (points[i].X[0] & dpMask) == 0 {
						if trap, found := trapTable.Lookup(points[i].X); found {
							// Exact point match
							if points[i].Y.Equals(trap.Y) {
								cand := new(big.Int).Sub(trap.Scalar, scalars[i])
								checkPt := secp256k1.ScalarMul(cand, secp256k1.G)
								if checkPt.Equals(targetPub) {
									if keyFound.CompareAndSwap(false, true) {
										recoveredKey = cand
										cancel()
									}
									atomic.AddUint64(&wildOps, localOps)
									return
								}
							}
						}
					}
				}
			}
			atomic.AddUint64(&wildOps, localOps)
		}(g)
	}
	wildWg.Wait()

	elapsed := time.Since(startTime)
	totalOps := tameOps + wildOps
	opsPerSec := float64(totalOps) / elapsed.Seconds()

	return &SolveResult{
		RecoveredKey:     recoveredKey,
		Success:          recoveredKey != nil,
		RangeMin:         rangeMin,
		RangeMax:         rangeMax,
		Width:            width,
		SqrtW:            sqrtW,
		TameSteps:        tameOps,
		WildSteps:        wildOps,
		TotalOperations:  totalOps,
		DistinguishedPts: trapTable.Size(),
		Elapsed:          elapsed,
		OperationsPerSec: opsPerSec,
		GoroutinesUsed:   s.Config.NumGoroutines,
	}, nil
}
