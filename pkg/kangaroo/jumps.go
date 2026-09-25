package kangaroo

import (
	"math"
	"math/big"

	"github.com/ajejfiejof/kangaroo-secp256k1/pkg/secp256k1"
)

// JumpSet represents Teske's geometrically distributed jump parameters.
type JumpSet struct {
	Scalars []*big.Int
	Points  []secp256k1.Point
	Count   int
}

// GenerateTeskeJumps constructs geometrically spaced jumps to maximize walk mixing
// and prevent short cycles: j_k ~ base * alpha^(k / 2).
func GenerateTeskeJumps(meanJump *big.Int, count int) *JumpSet {
	scalars := make([]*big.Int, count)
	points := make([]secp256k1.Point, count)

	meanF, _ := new(big.Float).SetInt(meanJump).Float64()
	if meanF < 4.0 {
		meanF = 4.0
	}

	base := meanF / 8.0
	if base < 1.0 {
		base = 1.0
	}

	alpha := math.Pow(2.0, 1.0/float64(count/4))

	for i := 0; i < count; i++ {
		val := base * math.Pow(alpha, float64(i)/2.0)
		if val < 1.0 {
			val = 1.0
		}
		s := big.NewInt(int64(val))
		scalars[i] = s
		points[i] = secp256k1.ScalarMul(s, secp256k1.G)
	}

	return &JumpSet{
		Scalars: scalars,
		Points:  points,
		Count:   count,
	}
}

// JumpIndex maps a point's X coordinate to a jump index in [0, Count-1].
func (js *JumpSet) JumpIndex(px secp256k1.FieldVal) int {
	if (js.Count & (js.Count - 1)) == 0 {
		mask := uint64(js.Count - 1)
		return int(px[0] & mask)
	}
	return int(px[0] % uint64(js.Count))
}
