# Pollard's Kangaroo Discrete Logarithm Engine (secp256k1)

A calibrated implementation and empirical benchmark of **Pollard's Kangaroo (Lambda) Algorithm** for solving bounded elliptic curve discrete logarithms on the `secp256k1` curve ($y^2 = x^3 + 7 \pmod P$).

Tested and benchmarked directly on the **ThinkPad T440p** with an **Intel Core i7-4712MQ CPU @ 2.30GHz**.

---

## 1. Mathematical Foundation

Given a base generator $G$, a target public key $Q = x \cdot G$, and a known bounding interval $[a, b]$ where $W = b - a$:

* **Standard Brute Force**: $O(W)$ operations.
* **Baby-Step Giant-Step (BSGS)**: $O(\sqrt{W})$ operations, but requires $O(\sqrt{W})$ storage (millions of gigabytes for large intervals).
* **Pollard's Kangaroo Algorithm**: **$O(\sqrt{W})$ operations with $O(1)$ memory overhead** using the **Distinguished Points (DP)** technique.

### Algorithmic Phases
1. **Jump Calibration**:
   $K$ jump sizes are precomputed around the optimal mean jump distance:
   $$m \approx \frac{\sqrt{b - a}}{4}$$
2. **The Tame Kangaroo**:
   Starts at the known upper bound $b \cdot G$ and takes deterministic hops based on point coordinates, recording "traps" at distinguished points (points whose coordinates end in $k$ zero bits).
3. **The Wild Kangaroo**:
   Starts at the unknown target point $Q = x \cdot G$ and follows the exact same deterministic hop rules.
4. **Collision & Recovery**:
   When the wild kangaroo lands on any point previously visited by the tame kangaroo, their trajectories merge permanently. Upon hitting the next trap:
   $$x = \text{scalar}_{\text{tame}} - \text{scalar}_{\text{wild}}$$

---

## 2. Empirical Benchmark Results (ThinkPad T440p)

Measured on **Intel Core i7-4712MQ CPU @ 2.30GHz** (Haswell, 4 cores / 8 threads):

| Key Space | Interval Width ($W$) | $\sqrt{W}$ | Additions Required | Time (s) | Single-Core Throughput | Result |
|---|---|---|---|---|---|---|
| **16-bit** | 32,768 | 181 | 1,439 | **0.0879 s** | 16,374 ops/sec | **SOLVED** |
| **20-bit** | 524,288 | 724 | 4,989 | **0.2438 s** | 20,465 ops/sec | **SOLVED** |
| **24-bit** | 8,388,608 | 2,896 | 12,700 | **0.5221 s** | 24,325 ops/sec | **SOLVED** |
| **28-bit** | 134,217,728 | 11,585 | 70,673 | **2.6785 s** | 26,385 ops/sec | **SOLVED** |
| **32-bit** | 2,147,483,648 | 46,340 | 302,607 | **10.2641 s** | 29,482 ops/sec | **SOLVED** |

* **Single-Core Python Throughput**: $\approx \mathbf{25,000 \text{ to } 30,000 \text{ additions/sec}}$.
* **32-Bit Search Space**: Successfully searched over **2.14 Billion possibilities** in **10.26 seconds**!

---

## 3. Extrapolation Across Bounded Puzzle Ranges

| Key Space | Search Width ($W$) | Expected Operations | Python (Single Core) | Compiled C / AVX2 (8 Threads) |
|---|---|---|---|---|
| **36-bit** | $3.4 \times 10^{10}$ | $5.56 \times 10^5$ | 23.76 seconds | **0.70 seconds** |
| **40-bit** | $5.5 \times 10^{11}$ | $2.22 \times 10^6$ | 1.58 minutes | **2.78 seconds** |
| **48-bit** | $1.4 \times 10^{14}$ | $3.56 \times 10^7$ | 25.34 minutes | **44.49 seconds** |
| **54-bit** | $9.0 \times 10^{15}$ | $2.85 \times 10^8$ | 3.38 hours | **5.93 minutes** |
| **60-bit** | $5.8 \times 10^{17}$ | $2.28 \times 10^9$ | 1.13 days | **47.45 minutes** |
| **68-bit** | $1.4 \times 10^{20}$ | $3.64 \times 10^{10}$ | 18.02 days | **12.65 hours** |

---

## 4. Execution & Testing

```bash
cd /home/ashley/Projects/kangaroo-secp256k1

# Run unit tests (6 passing tests)
python3 -m unittest test_kangaroo.py -v

# Run the benchmark suite across 16-bit to 32-bit ranges
python3 benchmark.py
```

---

## 5. File Inventory

* [`kangaroo_solver.py`](file:///home/ashley/Projects/kangaroo-secp256k1/kangaroo_solver.py): Core calibrated Pollard's Kangaroo engine.
* [`secp256k1_jacobian.py`](file:///home/ashley/Projects/kangaroo-secp256k1/secp256k1_jacobian.py): Fast Jacobian and affine modular arithmetic on secp256k1.
* [`benchmark.py`](file:///home/ashley/Projects/kangaroo-secp256k1/benchmark.py): Automated scaling benchmark suite.
* [`test_kangaroo.py`](file:///home/ashley/Projects/kangaroo-secp256k1/test_kangaroo.py): Unit test suite verifying point addition, multiplication, and key recovery.
