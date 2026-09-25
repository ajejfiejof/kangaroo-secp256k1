"""
Unit Tests for Pollard's Kangaroo Solver on secp256k1
=====================================================
Validates point addition, scalar multiplication, and key recovery across
multiple bounded bit ranges.
"""

import unittest
from kangaroo_solver import G, affine_add, affine_mul, KangarooSolver


class TestKangarooSolver(unittest.TestCase):

    def test_curve_base_point_validity(self):
        from kangaroo_solver import P
        Gx, Gy = G
        # Check y^2 == x^3 + 7 (mod P)
        lhs = (Gy * Gy) % P
        rhs = (pow(Gx, 3, P) + 7) % P
        self.assertEqual(lhs, rhs)

    def test_point_addition_identity(self):
        # P + None == P
        self.assertEqual(affine_add(G, None), G)
        self.assertEqual(affine_add(None, G), G)

    def test_point_multiplication_small_scalars(self):
        # 2*G == G + G
        two_g = affine_mul(2, G)
        g_plus_g = affine_add(G, G)
        self.assertEqual(two_g, g_plus_g)

        # 3*G == 2*G + G
        three_g = affine_mul(3, G)
        self.assertEqual(three_g, affine_add(two_g, G))

    def test_kangaroo_solve_16_bit_range(self):
        solver = KangarooSolver(num_jumps=16)
        secret = 45678
        range_min = 40000
        range_max = 65536
        target_pub = affine_mul(secret, G)

        res = solver.solve(target_pub, range_min, range_max)
        self.assertTrue(res["success"])
        self.assertEqual(res["recovered_key"], secret)

    def test_kangaroo_solve_20_bit_range(self):
        solver = KangarooSolver(num_jumps=16)
        secret = 789123
        range_min = 500000
        range_max = 1048576
        target_pub = affine_mul(secret, G)

        res = solver.solve(target_pub, range_min, range_max)
        self.assertTrue(res["success"])
        self.assertEqual(res["recovered_key"], secret)

    def test_kangaroo_solve_24_bit_range(self):
        solver = KangarooSolver(num_jumps=16)
        secret = 12345678
        range_min = 10000000
        range_max = 16777216
        target_pub = affine_mul(secret, G)

        res = solver.solve(target_pub, range_min, range_max)
        self.assertTrue(res["success"])
        self.assertEqual(res["recovered_key"], secret)

    def test_montgomery_batch_inversion(self):
        from stacked_kangaroo import montgomery_batch_invert, P
        import random
        vals = [random.randint(2, P - 1) for _ in range(16)]
        invs = montgomery_batch_invert(vals)
        for v, inv in zip(vals, invs):
            self.assertEqual((v * inv) % P, 1)

    def test_stacked_kangaroo_solve(self):
        from stacked_kangaroo import StackedKangarooSolver
        solver = StackedKangarooSolver(batch_size=16)
        secret = 543210
        range_min = 500000
        range_max = 1048576
        target_pub = affine_mul(secret, G)

        res = solver.solve(target_pub, range_min, range_max)
        self.assertTrue(res["success"])
        self.assertEqual(res["recovered_key"], secret)


if __name__ == "__main__":
    unittest.main()

