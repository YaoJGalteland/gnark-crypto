// Copyright 2020-2025 Consensys Software Inc.
// Licensed under the Apache License, Version 2.0. See the LICENSE file for details.

package bw6633

import (
	"errors"

	"github.com/consensys/gnark-crypto/ecc/bw6-633/fp"
	"github.com/consensys/gnark-crypto/ecc/bw6-633/internal/fptower"
)

// GT target group of the pairing
type GT = fptower.E6

type lineEvaluation struct {
	r0 fp.Element
	r1 fp.Element
	r2 fp.Element
}

// Pair calculates the reduced pairing for a set of points
// ∏ᵢ e(Pᵢ, Qᵢ).
//
// This function doesn't check that the inputs are in the correct subgroup. See IsInSubGroup.
func Pair(P []G1Affine, Q []G2Affine) (GT, error) {
	f, err := MillerLoop(P, Q)
	if err != nil {
		return GT{}, err
	}
	return FinalExponentiation(&f), nil
}

// PairingCheck calculates the reduced pairing for a set of points and returns True if the result is One
// ∏ᵢ e(Pᵢ, Qᵢ) =? 1
//
// This function doesn't check that the inputs are in the correct subgroup. See IsInSubGroup.
func PairingCheck(P []G1Affine, Q []G2Affine) (bool, error) {
	f, err := Pair(P, Q)
	if err != nil {
		return false, err
	}
	var one GT
	one.SetOne()
	return f.Equal(&one), nil
}

// FinalExponentiation computes the exponentiation (∏ᵢ zᵢ)ᵈ
// where d = (p^6-1)/r = (p^6-1)/Φ_6(p) ⋅ Φ_6(p)/r = (p^3-1)(p+1)(p^2 - p +1)/r
// we use instead d=s ⋅ (p^3-1)(p+1)(p^2 - p +1)/r
// where s is the cofactor (x^5-x^4-x) (El Housni and Guillevic)
func FinalExponentiation(z *GT, _z ...*GT) GT {

	var result GT
	result.Set(z)

	for _, e := range _z {
		result.Mul(&result, e)
	}

	var buf GT

	// Easy part
	// (p^3-1)(p+1)
	buf.Conjugate(&result)
	result.Inverse(&result)
	buf.Mul(&buf, &result)
	result.Frobenius(&buf).
		Mul(&result, &buf)

	var one GT
	one.SetOne()
	if result.Equal(&one) {
		return result
	}

	// Hard part (up to permutation)
	// (x₀^5-x₀^4-x₀)(p²-p+1)/r
	// Algorithm 4.5 from https://yelhousni.github.io/phd.pdf
	var a, b, c, d, e, f, g, h, i, t, mp GT
	mp.Frobenius(&result)
	a.ExptMinus1Squared(&mp)
	a.ExptSquarePlus1(&a)
	a.Mul(&result, &a)
	t.Conjugate(&mp)
	b.ExptPlus1(&a).
		Mul(&b, &t)
	t.CyclotomicSquare(&a).
		Mul(&t, &a)
	a.Conjugate(&t)
	c.ExptMinus1Div3(&b)
	d.ExptMinus1(&c)
	d.ExptSquarePlus1(&d)
	e.ExptMinus1Squared(&d)
	e.ExptSquarePlus1(&e)
	e.Mul(&e, &d)
	f.ExptPlus1(&e).
		Mul(&f, &c).
		Conjugate(&f).
		Mul(&f, &d)
	g.Mul(&f, &d).
		Conjugate(&g)
	h.ExptPlus1(&g).
		Mul(&h, &c).
		Mul(&h, &b)
	// ht = −7, hy = −1
	// c1 = (ht-hy)/2 = -3
	i.Expc1(&f).
		Mul(&i, &e)
	// c2 = (ht^2+3*hy^2)/4 = 13
	t.CyclotomicSquare(&i).
		Mul(&t, &i).
		Mul(&t, &b)
	i.Expc2(&h).
		Mul(&i, &t)
	result.Mul(&a, &i)

	return result
}

// MillerLoop computes the multi-Miller loop
// computes the multi-Miller loop ∏ᵢ MillerLoop(Pᵢ, Qᵢ)
// Alg.2 in https://eprint.iacr.org/2021/1359.pdf
func MillerLoop(P []G1Affine, Q []G2Affine) (GT, error) {
	// check input size match
	n := len(P)
	if n == 0 || n != len(Q) {
		return GT{}, errors.New("invalid inputs sizes")
	}

	// filter infinity points
	p := make([]G1Affine, 0, n)
	q0 := make([]G2Affine, 0, n)

	for k := 0; k < n; k++ {
		if P[k].IsInfinity() || Q[k].IsInfinity() {
			continue
		}
		p = append(p, P[k])
		q0 = append(q0, Q[k])
	}

	n = len(p)

	// precomputations
	qProj0 := make([]g2Proj, n)
	q1 := make([]G2Affine, n)
	q1Neg := make([]G2Affine, n)
	q0Neg := make([]G2Affine, n)
	for k := 0; k < n; k++ {
		q1[k].Y.Neg(&q0[k].Y)
		q0Neg[k].X.Set(&q0[k].X)
		q0Neg[k].Y.Set(&q1[k].Y)
		q1[k].X.Mul(&q0[k].X, &thirdRootOneG2)
		qProj0[k].FromAffine(&q0[k])
		q1Neg[k].Neg(&q1[k])
	}

	// f_{a0+λ*a1,Q}(P)
	var result GT
	result.SetOne()
	var l, l0 lineEvaluation
	var prodLines [5]fp.Element

	if n >= 1 {
		// i = 157, separately to avoid an E12 Square
		// (Square(res) = 1² = 1)
		// j = 0
		// k = 0, separately to avoid MulBy014 (res × ℓ)
		// (assign line to res)
		// qProj0[0] ← 2qProj0[0] and l0 the tangent ℓ passing 2qProj0[0]
		qProj0[0].doubleStep(&l0)
		// line evaluation at Q[0] (assign)
		result.B0.A0.Set(&l0.r0)
		result.B0.A1.Mul(&l0.r1, &p[0].X)
		result.B1.A1.Mul(&l0.r2, &p[0].Y)
	}

	// k = 1
	if n >= 2 {
		// qProj0[1] ← 2qProj0[1] and l0 the tangent ℓ passing 2qProj0[1]
		qProj0[1].doubleStep(&l0)
		// line evaluation at Q[1]
		l0.r1.Mul(&l0.r1, &p[1].X)
		l0.r2.Mul(&l0.r2, &p[1].Y)
		prodLines = fptower.Mul014By014(&l0.r0, &l0.r1, &l0.r2, &result.B0.A0, &result.B0.A1, &result.B1.A1)
		result.B0.A0 = prodLines[0]
		result.B0.A1 = prodLines[1]
		result.B0.A2 = prodLines[2]
		result.B1.A1 = prodLines[3]
		result.B1.A2 = prodLines[4]
	}

	// k >= 2
	for k := 2; k < n; k++ {
		// qProj0[k] ← 2qProj0[k] and l0 the tangent ℓ passing 2qProj0[k]
		qProj0[k].doubleStep(&l0)
		// line evaluation at Q[k]
		l0.r1.Mul(&l0.r1, &p[k].X)
		l0.r2.Mul(&l0.r2, &p[k].Y)
		// ℓ × res
		result.MulBy014(&l0.r0, &l0.r1, &l0.r2)
	}

	for i := len(LoopCounter) - 3; i >= 1; i-- {
		// (∏ᵢfᵢ)²
		// mutualize the square among n Miller loops
		result.Square(&result)

		j := LoopCounter[i]*3 + LoopCounter1[i]

		for k := 0; k < n; k++ {
			// qProj0[1] ← 2pProj0[1] and l0 the tangent ℓ qassing 2pProj0[1]
			qProj0[k].doubleStep(&l0)
			// line evaluation at Q[k]
			l0.r1.Mul(&l0.r1, &p[k].X)
			l0.r2.Mul(&l0.r2, &p[k].Y)

			switch j {
			// cases -4, -2, 2, 4 do not occur given the static LoopCounters
			case -3:
				// qProj0[k] ← qProj0[k]-q1[k] and
				// l the line ℓ qassing qProj0[k] and -q1[k]
				qProj0[k].addMixedStep(&l, &q1Neg[k])
				// line evaluation at Q[k]
				l.r1.Mul(&l.r1, &p[k].X)
				l.r2.Mul(&l.r2, &p[k].Y)
				// ℓ × ℓ
				prodLines = fptower.Mul014By014(&l.r0, &l.r1, &l.r2, &l0.r0, &l0.r1, &l0.r2)
				// (ℓ × ℓ) × res
				result.MulBy01245(&prodLines)
			case -1:
				// qProj0[k] ← qProj0[k]-q0[k] and
				// l the line ℓ qassing qProj0[k] and -q0[k]
				qProj0[k].addMixedStep(&l, &q0Neg[k])
				// line evaluation at Q[k]
				l.r1.Mul(&l.r1, &p[k].X)
				l.r2.Mul(&l.r2, &p[k].Y)
				// ℓ × ℓ
				prodLines = fptower.Mul014By014(&l.r0, &l.r1, &l.r2, &l0.r0, &l0.r1, &l0.r2)
				// (ℓ × ℓ) × res
				result.MulBy01245(&prodLines)
			case 0:
				// ℓ × res
				result.MulBy014(&l0.r0, &l0.r1, &l0.r2)
			case 1:
				// qProj0[k] ← qProj0[k]+q0[k] and
				// l the line ℓ qassing qProj0[k] and q0[k]
				qProj0[k].addMixedStep(&l, &q0[k])
				// line evaluation at Q[k]
				l.r1.Mul(&l.r1, &p[k].X)
				l.r2.Mul(&l.r2, &p[k].Y)
				// ℓ × ℓ
				prodLines = fptower.Mul014By014(&l.r0, &l.r1, &l.r2, &l0.r0, &l0.r1, &l0.r2)
				// (ℓ × ℓ) × res
				result.MulBy01245(&prodLines)
			case 3:
				// qProj0[k] ← qProj0[k]+q1[k] and
				// l the line ℓ qassing qProj0[k] and q1[k]
				qProj0[k].addMixedStep(&l, &q1[k])
				// line evaluation at Q[k]
				l.r1.Mul(&l.r1, &p[k].X)
				l.r2.Mul(&l.r2, &p[k].Y)
				// (ℓ × ℓ) × res
				prodLines = fptower.Mul014By014(&l.r0, &l.r1, &l.r2, &l0.r0, &l0.r1, &l0.r2)
				// (ℓ × ℓ) × res
				result.MulBy01245(&prodLines)
			default:
				return GT{}, errors.New("invalid LoopCounter")
			}
		}
	}

	// i = 0, j = 1
	result.Square(&result)
	for k := 0; k < n; k++ {
		// qProj0[1] ← 2pProj0[1] and l0 the tangent ℓ qassing 2pProj0[1]
		qProj0[k].doubleStep(&l0)
		// line evaluation at Q[k]
		l0.r1.Mul(&l0.r1, &p[k].X)
		l0.r2.Mul(&l0.r2, &p[k].Y)
		// qProj0[k] ← qProj0[k]+q0[k] and
		// l the line ℓ qassing qProj0[k] and q0[k]
		qProj0[k].lineCompute(&l, &q0[k])
		// line evaluation at Q[k]
		l.r1.Mul(&l.r1, &p[k].X)
		l.r2.Mul(&l.r2, &p[k].Y)
		// ℓ × ℓ
		prodLines = fptower.Mul014By014(&l.r0, &l.r1, &l.r2, &l0.r0, &l0.r1, &l0.r2)
		// (ℓ × ℓ) × res
		result.MulBy01245(&prodLines)
	}

	// negative x₀
	result.Conjugate(&result)

	return result, nil
}

// doubleStep doubles a point in Homogenous projective coordinates, and evaluates the line in Miller loop
// https://eprint.iacr.org/2013/722.pdf (Section 4.3)
func (p *g2Proj) doubleStep(evaluations *lineEvaluation) {

	// get some Element from our pool
	var t1, A, B, C, D, E, EE, F, G, H, I, J, K fp.Element
	A.Mul(&p.x, &p.y)
	A.Halve()
	B.Square(&p.y)
	C.Square(&p.z)
	D.Double(&C).
		Add(&D, &C)
	E.Double(&D).Double(&E).Double(&E)
	F.Double(&E).
		Add(&F, &E)
	G.Add(&B, &F)
	G.Halve()
	H.Add(&p.y, &p.z).
		Square(&H)
	t1.Add(&B, &C)
	H.Sub(&H, &t1)
	I.Sub(&E, &B)
	J.Square(&p.x)
	EE.Square(&E)
	K.Double(&EE).
		Add(&K, &EE)

	// X, Y, Z
	p.x.Sub(&B, &F).
		Mul(&p.x, &A)
	p.y.Square(&G).
		Sub(&p.y, &K)
	p.z.Mul(&B, &H)

	// Line evaluation
	evaluations.r0.Set(&I)
	evaluations.r1.Double(&J).
		Add(&evaluations.r1, &J)
	evaluations.r2.Neg(&H)
}

// addMixedStep point addition in Mixed Homogenous projective and Affine coordinates
// https://eprint.iacr.org/2013/722.pdf (Section 4.3)
func (p *g2Proj) addMixedStep(evaluations *lineEvaluation, a *G2Affine) {

	// get some Element from our pool
	var Y2Z1, X2Z1, O, L, C, D, E, F, G, H, t0, t1, t2, J fp.Element
	Y2Z1.Mul(&a.Y, &p.z)
	O.Sub(&p.y, &Y2Z1)
	X2Z1.Mul(&a.X, &p.z)
	L.Sub(&p.x, &X2Z1)
	C.Square(&O)
	D.Square(&L)
	E.Mul(&L, &D)
	F.Mul(&p.z, &C)
	G.Mul(&p.x, &D)
	t0.Double(&G)
	H.Add(&E, &F).
		Sub(&H, &t0)
	t1.Mul(&p.y, &E)

	// X, Y, Z
	p.x.Mul(&L, &H)
	p.y.Sub(&G, &H).
		Mul(&p.y, &O).
		Sub(&p.y, &t1)
	p.z.Mul(&E, &p.z)

	t2.Mul(&L, &a.Y)
	J.Mul(&a.X, &O).
		Sub(&J, &t2)

	// Line evaluation
	evaluations.r0.Set(&J)
	evaluations.r1.Neg(&O)
	evaluations.r2.Set(&L)
}

// lineCompute computes the line through p in Homogenous projective coordinates
// and a in affine coordinates. It does not compute the resulting point p+a.
func (p *g2Proj) lineCompute(evaluations *lineEvaluation, a *G2Affine) {

	// get some Element from our pool
	var Y2Z1, X2Z1, O, L, t2, J fp.Element
	Y2Z1.Mul(&a.Y, &p.z)
	O.Sub(&p.y, &Y2Z1)
	X2Z1.Mul(&a.X, &p.z)
	L.Sub(&p.x, &X2Z1)
	t2.Mul(&L, &a.Y)
	J.Mul(&a.X, &O).
		Sub(&J, &t2)

	// Line evaluation
	evaluations.r0.Set(&J)
	evaluations.r1.Neg(&O)
	evaluations.r2.Set(&L)
}

// ----------------------
// Fixed-argument pairing
// ----------------------

type LineEvaluationAff struct {
	R0 fp.Element
	R1 fp.Element
}

// PairFixedQ calculates the reduced pairing for a set of points
// ∏ᵢ e(Pᵢ, Qᵢ) where Q are fixed points in G2.
//
// This function doesn't check that the inputs are in the correct subgroup. See IsInSubGroup.
func PairFixedQ(P []G1Affine, lines [][2][len(LoopCounter) - 1]LineEvaluationAff) (GT, error) {
	f, err := MillerLoopFixedQ(P, lines)
	if err != nil {
		return GT{}, err
	}
	return FinalExponentiation(&f), nil
}

// PairingCheckFixedQ calculates the reduced pairing for a set of points and returns True if the result is One
// ∏ᵢ e(Pᵢ, Qᵢ) =? 1 where Q are fixed points in G2.
//
// This function doesn't check that the inputs are in the correct subgroup. See IsInSubGroup.
func PairingCheckFixedQ(P []G1Affine, lines [][2][len(LoopCounter) - 1]LineEvaluationAff) (bool, error) {
	f, err := PairFixedQ(P, lines)
	if err != nil {
		return false, err
	}
	var one GT
	one.SetOne()
	return f.Equal(&one), nil
}

// PrecomputeLines precomputes the lines for the fixed-argument Miller loop
func PrecomputeLines(Q G2Affine) (PrecomputedLines [2][len(LoopCounter) - 1]LineEvaluationAff) {

	// precomputations
	var accQ, imQ, imQneg, negQ G2Affine
	imQ.Y.Neg(&Q.Y)
	negQ.X.Set(&Q.X)
	negQ.Y.Set(&imQ.Y)
	imQ.X.Mul(&Q.X, &thirdRootOneG2)
	accQ.Set(&Q)
	imQneg.Neg(&imQ)

	for i := len(LoopCounter) - 2; i >= 0; i-- {

		accQ.doubleStep(&PrecomputedLines[0][i])

		j := LoopCounter[i]*3 + LoopCounter1[i]
		switch j {
		// cases -4, -2, 2, 4 do not occur given the static LoopCounters
		case -3:
			accQ.addStep(&PrecomputedLines[1][i], &imQneg)
		case -1:
			accQ.addStep(&PrecomputedLines[1][i], &negQ)
		case 0:
			continue
		case 1:
			accQ.addStep(&PrecomputedLines[1][i], &Q)
		case 3:
			accQ.addStep(&PrecomputedLines[1][i], &imQ)
		default:
			return [2][len(LoopCounter) - 1]LineEvaluationAff{}
		}
	}

	return PrecomputedLines
}

// MillerLoopFixedQ computes the multi-Miller loop as in MillerLoop
// but Qᵢ are fixed points in G2 known in advance.
func MillerLoopFixedQ(P []G1Affine, lines [][2][len(LoopCounter) - 1]LineEvaluationAff) (GT, error) {
	// check input size match
	n := len(P)
	if n == 0 || n != len(lines) {
		return GT{}, errors.New("invalid inputs sizes")
	}

	// no need to filter infinity points:
	// 		1. if Pᵢ=(0,0) then -x/y=1/y=0 by gnark-crypto convention and so
	// 		lines R0 and R1 are 0. It happens that result will stay, through
	// 		the Miller loop, in 𝔽p⁶ because MulBy01(0,0,1),
	// 		Mul01By01(0,0,1,0,0,1) and MulBy01245 set result.C0 to 0. At the
	// 		end result will be in a proper subgroup of Fp¹² so it be reduced to
	// 		1 in FinalExponentiation.
	//
	//      and/or
	//
	// 		2. if Qᵢ=(0,0) then PrecomputeLines(Qᵢ) will return lines R0 and R1
	// 		that are 0 because of gnark-convention (*/0==0) in doubleStep and
	// 		addStep. Similarly to Pᵢ=(0,0) it happens that result be 1
	// 		after the FinalExponentiation.

	// precomputations
	yInv := make([]fp.Element, n)
	xNegOverY := make([]fp.Element, n)
	for k := 0; k < n; k++ {
		yInv[k].Set(&P[k].Y)
	}
	yInv = fp.BatchInvert(yInv)
	for k := 0; k < n; k++ {
		xNegOverY[k].Mul(&P[k].X, &yInv[k]).
			Neg(&xNegOverY[k])
	}

	// f_{a0+λ*a1,Q}(P)
	var result GT
	result.SetOne()
	var prodLines [5]fp.Element

	for i := len(LoopCounter) - 2; i >= 0; i-- {
		result.Square(&result)

		j := LoopCounter[i]*3 + LoopCounter1[i]
		for k := 0; k < n; k++ {
			lines[k][0][i].R1.
				Mul(
					&lines[k][0][i].R1,
					&yInv[k],
				)
			lines[k][0][i].R0.
				Mul(&lines[k][0][i].R0,
					&xNegOverY[k],
				)
			if j == 0 {
				result.MulBy01(
					&lines[k][0][i].R1,
					&lines[k][0][i].R0,
				)

			} else {
				lines[k][1][i].R1.
					Mul(
						&lines[k][1][i].R1,
						&yInv[k],
					)
				lines[k][1][i].R0.
					Mul(
						&lines[k][1][i].R0,
						&xNegOverY[k],
					)
				prodLines = fptower.Mul01By01(
					&lines[k][0][i].R1, &lines[k][0][i].R0,
					&lines[k][1][i].R1, &lines[k][1][i].R0,
				)
				result.MulBy01245(&prodLines)
			}
		}
	}

	// negative x₀
	result.Conjugate(&result)

	return result, nil

}

func (p *G2Affine) doubleStep(evaluations *LineEvaluationAff) {

	var n, d, λ, xr, yr fp.Element
	// λ = 3x²/2y
	n.Square(&p.X)
	λ.Double(&n).
		Add(&λ, &n)
	d.Double(&p.Y)
	λ.Div(&λ, &d)

	// xr = λ²-2x
	xr.Square(&λ).
		Sub(&xr, &p.X).
		Sub(&xr, &p.X)

	// yr = λ(x-xr)-y
	yr.Sub(&p.X, &xr).
		Mul(&yr, &λ).
		Sub(&yr, &p.Y)

	evaluations.R0.Set(&λ)
	evaluations.R1.Mul(&λ, &p.X).
		Sub(&evaluations.R1, &p.Y)

	p.X.Set(&xr)
	p.Y.Set(&yr)
}

func (p *G2Affine) addStep(evaluations *LineEvaluationAff, a *G2Affine) {
	var n, d, λ, λλ, xr, yr fp.Element

	// compute λ = (y2-y1)/(x2-x1)
	n.Sub(&a.Y, &p.Y)
	d.Sub(&a.X, &p.X)
	λ.Div(&n, &d)

	// xr = λ²-x1-x2
	λλ.Square(&λ)
	n.Add(&p.X, &a.X)
	xr.Sub(&λλ, &n)

	// yr = λ(x1-xr) - y1
	yr.Sub(&p.X, &xr).
		Mul(&yr, &λ).
		Sub(&yr, &p.Y)

	evaluations.R0.Set(&λ)
	evaluations.R1.Mul(&λ, &p.X).
		Sub(&evaluations.R1, &p.Y)

	p.X.Set(&xr)
	p.Y.Set(&yr)
}
