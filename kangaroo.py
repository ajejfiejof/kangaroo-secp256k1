"""
Pollard's Kangaroo Solver for secp256k1 Bounded Discrete Logarithms
===================================================================
High-efficiency implementation of Pollard's Kangaroo (Lambda) algorithm
using Jacobian coordinates, distinguished points, and deterministic hops.

Mathematical complexity: O(sqrt(b - a)) operations with O(1) storage.
"""

import time
import math
from typing import Dict, Tuple, Optional, Any
from secp256k1_jacobian import (
    P, N, Gx, Gy, G,
    affine_mul, jacobian_add_mixed, to_affine
)


class KangarooSolver:
    """
    Solves Q = x * G for x in [range_min, range_max] on secp256k1.
    """

    def __init__(self, num_jumps: int = 16, dp_bits: int = 12):
        self.num_jumps = num_jumps
        self.dp_bits = dp_bits
        self.dp_mask = (1 << dp_bits) - 1

        # Precompute jump scalars and affine points: 2^0, 2^1, ...
        self.jumps = [2**i for i in range(num_jumps)]
        self.jump_pts = [affine_mul(j, G) for j in self.jumps]
        self.avg_jump = sum(self.jumps) // len(self.jumps)

    def solve(self, target_pub_affine: Tuple[int, int], range_min: int, range_max: int, max_steps: int = 500000) -> Dict[str, Any]:
        """
        Executes Pollard's Kangaroo to find the scalar x such that target_pub_affine = x * G.
        """
        start_time = time.time()
        width = range_max - range_min
        expected_steps = int(math.sqrt(width))

        # Dynamically scale distinguished points mask based on search width
        dp_bits = max(4, min(24, int(math.log2(max(16, expected_steps // 8)))))
        dp_mask = (1 << dp_bits) - 1

        # -------------------------------------------------------------
        # Phase 1: Tame Kangaroo
        # Starts at range_max and jumps forward, leaving traps at DPs.
        # -------------------------------------------------------------
        tame_scalar = range_max
        tame_pt_aff = affine_mul(range_max, G)
        tame_X, tame_Y, tame_Z = tame_pt_aff[0], tame_pt_aff[1], 1

        traps = {} # trap_affine_x -> scalar_distance
        tame_steps = 0
        tame_limit = max(1000, expected_steps * 3)

        for _ in range(tame_limit):
            tame_steps += 1
            # Deterministic jump selection based on current X coordinate
            idx = tame_X % self.num_jumps
            jump_val = self.jumps[idx]
            jx, jy = self.jump_pts[idx]

            tame_scalar += jump_val
            tame_X, tame_Y, tame_Z = jacobian_add_mixed(tame_X, tame_Y, tame_Z, jx, jy)

            # Check for distinguished point
            if (tame_X & dp_mask) == 0:
                aff = to_affine(tame_X, tame_Y, tame_Z)
                if aff is not None:
                    traps[aff[0]] = tame_scalar

        # -------------------------------------------------------------
        # Phase 2: Wild Kangaroo
        # Starts at Q = x * G (with scalar 0) and jumps until trap hit.
        # -------------------------------------------------------------
        wild_scalar = 0
        wild_X, wild_Y, wild_Z = target_pub_affine[0], target_pub_affine[1], 1
        wild_steps = 0
        recovered_key = None

        for _ in range(max_steps):
            wild_steps += 1
            idx = wild_X % self.num_jumps
            jump_val = self.jumps[idx]
            jx, jy = self.jump_pts[idx]

            wild_scalar += jump_val
            wild_X, wild_Y, wild_Z = jacobian_add_mixed(wild_X, wild_Y, wild_Z, jx, jy)

            # Check for distinguished point
            if (wild_X & dp_mask) == 0:
                aff = to_affine(wild_X, wild_Y, wild_Z)
                if aff is not None and aff[0] in traps:
                    # Collision detected!
                    tame_end = traps[aff[0]]
                    candidate = tame_end - wild_scalar
                    if range_min <= candidate <= range_max:
                        # Verify candidate point
                        check_pt = affine_mul(candidate, G)
                        if check_pt == target_pub_affine:
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
            "tame_steps": tame_steps,
            "wild_steps": wild_steps,
            "total_point_additions": total_ops,
            "elapsed_seconds": round(elapsed, 4),
            "ops_per_second": round(ops_per_sec, 2),
            "distinguished_points_stored": len(traps)
        }
