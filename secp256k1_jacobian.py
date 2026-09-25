"""
secp256k1 Fast Jacobian Curve Arithmetic Engine
================================================
Implements modular arithmetic and fast point addition on secp256k1:
    y^2 = x^3 + 7 (mod P)

Uses Jacobian coordinates (X : Y : Z) where:
    x = X / Z^2 (mod P)
    y = Y / Z^3 (mod P)

Mixed Jacobian addition (Jacobian + Affine) requires 0 modular inversions,
enabling maximum throughput on CPU loops.
"""

from typing import Tuple, Optional

# Curve parameters (Standards for Efficient Cryptography: secp256k1)
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


def jacobian_add_mixed(X1: int, Y1: int, Z1: int, x2: int, y2: int) -> Tuple[int, int, int]:
    """
    Fast mixed addition: (X1 : Y1 : Z1) + (x2, y2, 1).
    Takes a point in Jacobian coordinates and adds an affine point (Z2 = 1).
    COST: 8 modular multiplications + 3 modular squarings. ZERO modular inversions.
    """
    if Z1 == 0:
        return (x2, y2, 1)

    Z1_sq = (Z1 * Z1) % P
    U2 = (x2 * Z1_sq) % P
    S2 = (y2 * Z1 * Z1_sq) % P
    H = (U2 - X1) % P
    R = (S2 - Y1) % P

    if H == 0:
        if R == 0:
            # Point doubling
            S = (4 * X1 * Y1 * Y1) % P
            M = (3 * X1 * X1) % P
            X3 = (M * M - 2 * S) % P
            Y3 = (M * (S - X3) - 8 * pow(Y1, 4, P)) % P
            Z3 = (2 * Y1 * Z1) % P
            return (X3, Y3, Z3)
        return (0, 0, 0) # Point at infinity

    H2 = (H * H) % P
    H3 = (H * H2) % P
    V = (X1 * H2) % P
    X3 = (R * R - H3 - 2 * V) % P
    Y3 = (R * (V - X3) - Y1 * H3) % P
    Z3 = (Z1 * H) % P
    return (X3, Y3, Z3)


def to_affine(X: int, Y: int, Z: int) -> Optional[Tuple[int, int]]:
    """Converts a Jacobian point (X : Y : Z) to affine (x, y)."""
    if Z == 0:
        return None
    Z_inv = pow(Z, -1, P)
    Z_inv2 = (Z_inv * Z_inv) % P
    x = (X * Z_inv2) % P
    y = (Y * Z_inv * Z_inv2) % P
    return (x, y)
