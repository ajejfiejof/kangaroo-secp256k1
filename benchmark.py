"""
Pollard's Kangaroo Performance Benchmark Suite
===============================================
Measures empirical discrete logarithm solving performance across increasing
bit-width key intervals on secp256k1 on this host machine (ThinkPad T440p).
"""

import time
import random
from kangaroo_solver import G, affine_mul, KangarooSolver


def run_benchmark():
    print("=" * 72)
    print("  POLLARD'S KANGAROO (SECP256K1) EMPIRICAL BENCHMARK")
    print("=" * 72)

    benchmarks = [
        {"bits": 16, "min": 2**15, "max": 2**16},
        {"bits": 20, "min": 2**19, "max": 2**20},
        {"bits": 24, "min": 2**23, "max": 2**24},
        {"bits": 28, "min": 2**27, "max": 2**28},
        {"bits": 32, "min": 2**31, "max": 2**32}
    ]

    solver = KangarooSolver(num_jumps=16)
    results = []

    print(f"\n{'Bits':<6} | {'Interval Width (W)':<20} | {'Sqrt(W)':<10} | {'Additions':<10} | {'Time (s)':<10} | {'Ops/sec':<10} | {'Success'}")
    print("-" * 84)

    for bench in benchmarks:
        bits = bench["bits"]
        rmin = bench["min"]
        rmax = bench["max"]
        secret = random.randint(rmin + 50, rmax - 50)
        target_pub = affine_mul(secret, G)

        res = solver.solve(target_pub, rmin, rmax)
        assert res["success"] and res["recovered_key"] == secret, f"Failed at {bits} bits!"

        results.append(res)
        print(f"{bits:<6} | {res['width']:<20,d} | {res['sqrt_w']:<10,d} | {res['total_point_additions']:<10,d} | {res['elapsed_seconds']:<10.4f} | {res['ops_per_second']:<10,.0f} | {res['success']}")

    print("-" * 84)
    avg_ops_sec = sum(r["ops_per_second"] for r in results) / len(results)
    print(f"\nAverage Throughput: {avg_ops_sec:,.0f} point additions / sec (single-core Python)")

    print("\nTheoretical Extrapolation for Bounded Bitcoin Puzzle Intervals:")
    print("------------------------------------------------------------------------")
    for bits in [36, 40, 48, 54, 60, 68]:
        w = 2**(bits - 1)
        sqrt_w = int(w**0.5)
        # Expected point additions is approx 3 * sqrt(w) for tame + wild
        exp_ops = 3 * sqrt_w
        # Single-core Python time
        py_sec = exp_ops / avg_ops_sec
        # 8-thread C / AVX2 estimate (approx 200,000 ops/sec per core * 4 cores = 800,000 ops/sec)
        c_sec = exp_ops / 800000.0

        def format_time(s):
            if s < 60: return f"{s:.2f} s"
            if s < 3600: return f"{s/60:.2f} min"
            if s < 86400: return f"{s/3600:.2f} hours"
            if s < 31536000: return f"{s/86400:.2f} days"
            return f"{s/31536000:.2f} years"

        print(f"  • {bits}-bit range: {exp_ops:>14,d} expected ops | Python: {format_time(py_sec):<12} | Compiled C (8-thread): {format_time(c_sec)}")
    print("=" * 72)


if __name__ == "__main__":
    run_benchmark()
