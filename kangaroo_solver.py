"""
Pollard's Kangaroo (Lambda) Discrete Logarithm Solver for secp256k1
===================================================================
High-precision, mathematically calibrated implementation of Pollard's Kangaroo
algorithm for solving discrete logarithms Q = x * G in a bounded interval [a, b]
on the secp256k1 elliptic curve.

Mathematical properties:
- Time Complexity : O(sqrt(b - a)) point additions
- Space Complexity: O(1) memory via distinguished points
- Coordinate model: Affine coordinates for deterministic point hashing
"""

import time
import math
from typing import Dict, Tuple, Optional, Any

# secp256k1 curve parameters
P = 0xFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFEFFFFFC2F
N = 0xFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFEBAAEDCE6AF48A03BBFD25E8CD0364141
Gx = 0x79BE667EF9DCBBAC55A06295CE870B07029BFCDB2DCE28D959F2815B16F81798
Gy = 0x483ADA7726A3C4655DA4FBFC0E1108A8FD17B448A68554199C47D08FFB10D4B8
G = (Gx, Gy)


def affine_add(p1: Optional[Tuple[int, int]], p2: Optional[Tuple[int, int]]) -> Optional[Tuple[int, int]]:
    """Standard affine point addition on secp256k1."""
    if p1 is None: return p2
    if p2 is None: return p1
    x1, y1 = p1
    x2, y2 = p2
    if x1 == x2 and y1 != y2: return None
    if x1 == x2:
        m = (3 * x1 * x1 * pow(2 * y1, -1, P)) % P
    else:
        m = ((y2 - y1) * pow(x2 - x1, -1, P)) % P
    x3 = (m * m - x1 - x2) % P
    y3 = (m * (x1 - x3) - y1) % P
    return (x3, y3)


def affine_mul(k: int, p: Tuple[int, int]) -> Optional[Tuple[int, int]]:
    """Double-and-add scalar multiplication in affine coordinates."""
    res = None
    cur = p
    while k > 0:
        if k & 1: res = affine_add(res, cur)
        cur = affine_add(cur, cur)
        k >>= 1
    return res


class KangarooSolver:
    """
    Calibrated Pollard's Kangaroo solver for intervals [range_min, range_max].
    """

    def __init__(self, num_jumps: int = 16):
        self.num_jumps = num_jumps

    def solve(self, target_pub: Tuple[int, int], range_min: int, range_max: int) -> Dict[str, Any]:
        """
        Solves Q = x * G for x in [range_min, range_max].
        """
        width = range_max - range_min
        if width <= 0:
            raise ValueError("range_max must be strictly greater than range_min")

        sqrt_w = max(4, int(math.sqrt(width)))
        start_time = time.time()

        # Calibrate jump sizes around mean jump m ~ sqrt(W) / 4
        mean_jump = max(1, sqrt_w // 4)
        jumps = [max(1, int(mean_jump * (0.5 + i / (self.num_jumps - 1)))) for i in range(self.num_jumps)]
        jump_pts = [affine_mul(j, G) for j in jumps]

        # Calibrate distinguished point mask: ~1 DP per (sqrt_w / 16) steps
        dp_bits = max(2, min(24, int(math.log2(max(4, sqrt_w // 16)))))
        dp_mask = (1 << dp_bits) - 1

        # -------------------------------------------------------------
        # Phase 1: Tame Kangaroo
        # Starts at upper bound range_max and jumps forward
        # -------------------------------------------------------------
        tame_scalar = range_max
        tame_pt = affine_mul(range_max, G)
        traps = {} # x -> (y, scalar)
        tame_steps = 0
        tame_limit = sqrt_w * 4

        for _ in range(tame_limit):
            tame_steps += 1
            idx = tame_pt[0] % self.num_jumps
            tame_scalar += jumps[idx]
            tame_pt = affine_add(tame_pt, jump_pts[idx])
            if (tame_pt[0] & dp_mask) == 0:
                traps[tame_pt[0]] = (tame_pt[1], tame_scalar)

        # -------------------------------------------------------------
        # Phase 2: Wild Kangaroo
        # Starts at target public key Q and jumps until trap collision
        # -------------------------------------------------------------
        wild_scalar = 0
        wild_pt = target_pub
        wild_steps = 0
        wild_limit = sqrt_w * 8
        recovered_key = None

        for _ in range(wild_limit):
            wild_steps += 1
            idx = wild_pt[0] % self.num_jumps
            wild_scalar += jumps[idx]
            wild_pt = affine_add(wild_pt, jump_pts[idx])
            if (wild_pt[0] & dp_mask) == 0:
                if wild_pt[0] in traps:
                    tame_y, tame_s = traps[wild_pt[0]]
                    if wild_pt[1] == tame_y:
                        candidate = tame_s - wild_scalar
                        # Verify point match
                        if affine_mul(candidate, G) == target_pub:
                            recovered_key = candidate
                            break

        elapsed = time.time() - start_time
        total_ops = tame_steps + wild_steps
        ops_per_sec = total_ops / elapsed if elapsed > 0 else 0

        return {
            "recovered_key": recovered_key,
            "success": recovered_key is not None,
            "range_min": range_min,
            "range_max": range_max,
            "width": width,
            "sqrt_w": sqrt_w,
            "tame_steps": tame_steps,
            "wild_steps": wild_steps,
            "total_point_additions": total_ops,
            "distinguished_points_stored": len(traps),
            "elapsed_seconds": round(elapsed, 4),
            "ops_per_second": round(ops_per_sec, 2)
        }
