package big

import (
	"github.com/twpayne/go-geom"
	"github.com/twpayne/go-geom/algorithm/orientation"
	"math"
	"math/big"
)

/**
 * A value which is safely greater than the
 * relative round-off error in double-precision numbers
 */
var dp_safe_epsilon = 1e-15

/**
 * Returns the index of the direction of the point <code>point</code> relative to
 * a vector specified by <code>vectorOrigin-vectorEnd</code>.
 *
 * @param vectorOrigin the origin point of the vector
 * @param vectorEnd the final point of the vector
 * @param point the point to compute the direction to
 *
 * @return COUNTER_CLOCKWISE if point is counter-clockwise (left) from vectorOrigin-vectorEnd
 * @return CLOCKWISE if point is clockwise (right) from vectorOrigin-vectorEnd
 * @return COLLINEAR if point is collinear with vectorOrigin-vectorEnd
 */
func OrientationIndex(vectorOrigin, vectorEnd, point geom.Coord) orientation.Orientation {
	// fast filter for orientation index
	// avoids use of slow extended-precision arithmetic in many cases
	index := orientationIndexFilter(vectorOrigin, vectorEnd, point)
	if index <= 1 {
		return index
	}

	var dx1, dy1, dx2, dy2 big.Float

	// normalize coordinates
	dx1.SetFloat64(vectorEnd[0]).Add(&dx1, big.NewFloat(-vectorOrigin[0]))
	dy1.SetFloat64(vectorEnd[1]).Add(&dy1, big.NewFloat(-vectorOrigin[1]))
	dx2.SetFloat64(point[0]).Add(&dx2, big.NewFloat(-vectorEnd[0]))
	dy2.SetFloat64(point[1]).Add(&dy2, big.NewFloat(-vectorEnd[1]))

	// calculate determinant.  Calculation takes place in dx1 for performance
	dx1.Mul(&dx1, &dy2)
	dy1.Mul(&dy1, &dx2)
	dx1.Sub(&dx1, &dy1)

	return orientation.Orientation(orientationBasedOnSignForBig(dx1))
}

// Computes the intersection point of the two lines using math.big.Float arithmetic.
// The lines are considered infinate in length.  For example, (0,0), (1, 0) and (2, 1) (2, 2) will have intersection of (2, 0)
// Currently does not handle case of parallel lines.
func Intersection(line1Start, line1End, line2Start, line2End geom.Coord) geom.Coord {
	var denom1, denom2, denom, tmp1, tmp2 big.Float

	denom1.SetFloat64(line2End[1]).Sub(&denom1, tmp2.SetFloat64(line2Start[1])).Mul(&denom1, tmp1.SetFloat64(line1End[0]).Sub(&tmp1, tmp2.SetFloat64(line1Start[0])))
	denom2.SetFloat64(line2End[0]).Sub(&denom2, tmp2.SetFloat64(line2Start[0])).Mul(&denom2, tmp1.SetFloat64(line1End[1]).Sub(&tmp1, tmp2.SetFloat64(line1Start[1])))
	denom.Sub(&denom1, &denom2)

	// Cases:
	// - denom is 0 if lines are parallel
	// - intersection point lies within line segment p if fracP is between 0 and 1
	// - intersection point lies within line segment q if fracQ is between 0 and 1

	// reusing previous variables for performance
	numx1 := &denom1
	numx2 := &denom2
	var numx big.Float

	numx1.SetFloat64(line2End[0]).Sub(numx1, tmp2.SetFloat64(line2Start[0])).Mul(numx1, tmp1.SetFloat64(line1Start[1]).Sub(&tmp1, tmp2.SetFloat64(line2Start[1])))
	numx2.SetFloat64(line2End[1]).Sub(numx2, tmp2.SetFloat64(line2Start[1])).Mul(numx2, tmp1.SetFloat64(line1Start[0]).Sub(&tmp1, tmp2.SetFloat64(line2Start[0])))
	numx.Sub(numx1, numx2)

	fracP, _ := numx.Quo(&numx, &denom).Float64()

	x, _ := numx1.SetFloat64(line1Start[0]).Add(numx1, tmp2.SetFloat64(line1End[0])).Sub(numx1, tmp2.SetFloat64(line1Start[0])).Mul(numx1, tmp1.SetFloat64(fracP)).Float64()

	// reusing previous variables for performance
	numy1 := &denom1
	numy2 := &denom2
	var numy big.Float

	numy1.SetFloat64(line1End[0]).Sub(numy1, tmp2.SetFloat64(line1Start[0])).Mul(numy1, tmp1.SetFloat64(line1Start[1]).Sub(&tmp1, tmp2.SetFloat64(line2Start[1])))
	numy2.SetFloat64(line1End[1]).Sub(numy2, tmp2.SetFloat64(line1Start[1])).Mul(numy2, tmp1.SetFloat64(line1Start[0]).Sub(&tmp1, tmp2.SetFloat64(line2Start[0])))
	numy.Sub(numy1, numy2)

	fracQ, _ := numy.Quo(&numy, &denom).Float64()

	tmp2.SetFloat64(line1End[1]).Sub(&tmp2, tmp1.SetFloat64(line1Start[1]))

	if tmp2.IsInf() && fracQ == 0 || tmp1.SetFloat64(0).Cmp(&tmp2) == 0 && math.IsInf(fracQ, 0) {
		// can't perform calculation
		return geom.Coord{math.Inf(1), math.Inf(1)}
	}

	y, _ := numx1.SetFloat64(line1Start[1]).Add(numx1, tmp2.Mul(&tmp2, tmp1.SetFloat64(fracQ))).Float64()

	return geom.Coord{x, y}
}

/////////////////  Implementation /////////////////////////////////

// A filter for computing the orientation index of three coordinates.
//
// If the orientation can be computed safely using standard DP
// arithmetic, this routine returns the orientation index.
// Otherwise, a value i > 1 is returned.
// In this case the orientation index must
// be computed using some other more robust method.
// The filter is fast to compute, so can be used to
// avoid the use of slower robust methods except when they are really needed,
// thus providing better average performance.
//
// Uses an approach due to Jonathan Shewchuk, which is in the public domain.
//
// Return the orientation index if it can be computed safely
// Return i > 1 if the orientation index cannot be computed safely
func orientationIndexFilter(vectorOrigin, vectorEnd, point geom.Coord) orientation.Orientation {
	var detsum float64

	detleft := (vectorOrigin[0] - point[0]) * (vectorEnd[1] - point[1])
	detright := (vectorOrigin[1] - point[1]) * (vectorEnd[0] - point[0])
	det := detleft - detright

	if detleft > 0.0 {
		if detright <= 0.0 {
			return orientationBasedOnSign(det)
		} else {
			detsum = detleft + detright
		}
	} else if detleft < 0.0 {
		if detright >= 0.0 {
			return orientationBasedOnSign(det)
		} else {
			detsum = -detleft - detright
		}
	} else {
		return orientationBasedOnSign(det)
	}

	errbound := dp_safe_epsilon * detsum
	if (det >= errbound) || (-det >= errbound) {
		return orientationBasedOnSign(det)
	}

	return 2
}

func orientationBasedOnSign(x float64) orientation.Orientation {
	if x > 0 {
		return orientation.COUNTER_CLOCKWISE
	}
	if x < 0 {
		return orientation.CLOCKWISE
	}
	return orientation.COLLINEAR
}
func orientationBasedOnSignForBig(x big.Float) orientation.Orientation {
	if x.IsInf() {
		return orientation.COLLINEAR
	}
	switch x.Sign() {
	case -1:
		return orientation.CLOCKWISE
	case 0:
		return orientation.COLLINEAR
	default:
		return orientation.COUNTER_CLOCKWISE
	}
}
