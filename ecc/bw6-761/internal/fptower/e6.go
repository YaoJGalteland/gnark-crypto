// Copyright 2020 ConsenSys Software Inc.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

// Code generated by consensys/gnark-crypto DO NOT EDIT

package fptower

import (
	"errors"
	"math/big"

	"github.com/consensys/gnark-crypto/ecc/bw6-761/fp"
)

// E6 is a degree two finite field extension of fp3
type E6 struct {
	B0, B1 E3
}

// Equal returns true if z equals x, fasle otherwise
func (z *E6) Equal(x *E6) bool {
	return z.B0.Equal(&x.B0) && z.B1.Equal(&x.B1)
}

// String puts E6 in string form
func (z *E6) String() string {
	return (z.B0.String() + "+(" + z.B1.String() + ")*v")
}

// SetString sets a E6 from string
func (z *E6) SetString(s0, s1, s2, s3, s4, s5 string) *E6 {
	z.B0.SetString(s0, s1, s2)
	z.B1.SetString(s3, s4, s5)
	return z
}

// Set copies x into z and returns z
func (z *E6) Set(x *E6) *E6 {
	*z = *x
	return z
}

// SetOne sets z to 1 in Montgomery form and returns z
func (z *E6) SetOne() *E6 {
	*z = E6{}
	z.B0.A0.SetOne()
	return z
}

// ToMont converts to Mont form
func (z *E6) ToMont() *E6 {
	z.B0.ToMont()
	z.B1.ToMont()
	return z
}

// FromMont converts from Mont form
func (z *E6) FromMont() *E6 {
	z.B0.FromMont()
	z.B1.FromMont()
	return z
}

// Add set z=x+y in E6 and return z
func (z *E6) Add(x, y *E6) *E6 {
	z.B0.Add(&x.B0, &y.B0)
	z.B1.Add(&x.B1, &y.B1)
	return z
}

// Sub sets z to x sub y and return z
func (z *E6) Sub(x, y *E6) *E6 {
	z.B0.Sub(&x.B0, &y.B0)
	z.B1.Sub(&x.B1, &y.B1)
	return z
}

// Double sets z=2*x and returns z
func (z *E6) Double(x *E6) *E6 {
	z.B0.Double(&x.B0)
	z.B1.Double(&x.B1)
	return z
}

// SetRandom used only in tests
func (z *E6) SetRandom() (*E6, error) {
	if _, err := z.B0.SetRandom(); err != nil {
		return nil, err
	}
	if _, err := z.B1.SetRandom(); err != nil {
		return nil, err
	}
	return z, nil
}

// Mul set z=x*y in E6 and return z
func (z *E6) Mul(x, y *E6) *E6 {
	var a, b, c E3
	a.Add(&x.B0, &x.B1)
	b.Add(&y.B0, &y.B1)
	a.Mul(&a, &b)
	b.Mul(&x.B0, &y.B0)
	c.Mul(&x.B1, &y.B1)
	z.B1.Sub(&a, &b).Sub(&z.B1, &c)
	z.B0.MulByNonResidue(&c).Add(&z.B0, &b)
	return z
}

// Square set z=x*x in E6 and return z
func (z *E6) Square(x *E6) *E6 {

	//Algorithm 22 from https://eprint.iacr.org/2010/354.pdf
	var c0, c2, c3 E3
	c0.Sub(&x.B0, &x.B1)
	c3.MulByNonResidue(&x.B1).Neg(&c3).Add(&x.B0, &c3)
	c2.Mul(&x.B0, &x.B1)
	c0.Mul(&c0, &c3).Add(&c0, &c2)
	z.B1.Double(&c2)
	c2.MulByNonResidue(&c2)
	z.B0.Add(&c0, &c2)

	return z
}

// Karabina's compressed cyclotomic square
// https://eprint.iacr.org/2010/542.pdf
// Th. 3.2 with minor modifications to fit our tower
func (z *E6) CyclotomicSquareCompressed(x *E6) *E6 {

	var t [7]fp.Element

	// t0 = g1^2
	t[0].Square(&x.B0.A1)
	// t1 = g5^2
	t[1].Square(&x.B1.A2)
	// t5 = g1 + g5
	t[5].Add(&x.B0.A1, &x.B1.A2)
	// t2 = (g1 + g5)^2
	t[2].Square(&t[5])

	// t3 = g1^2 + g5^2
	t[3].Add(&t[0], &t[1])
	// t5 = 2 * g1 * g5
	t[5].Sub(&t[2], &t[3])

	// t6 = g3 + g2
	t[6].Add(&x.B1.A0, &x.B0.A2)
	// t3 = (g3 + g2)^2
	t[3].Square(&t[6])
	// t2 = g3^2
	t[2].Square(&x.B1.A0)

	// t6 = 2 * nr * g1 * g5
	t[6].MulByNonResidue(&t[5])
	// t5 = 4 * nr * g1 * g5 + 2 * g3
	t[5].Add(&t[6], &x.B1.A0).
		Double(&t[5])
	// z3 = 6 * nr * g1 * g5 + 2 * g3
	z.B1.A0.Add(&t[5], &t[6])

	// t4 = nr * g5^2
	t[4].MulByNonResidue(&t[1])
	// t5 = nr * g5^2 + g1^2
	t[5].Add(&t[0], &t[4])
	// t6 = nr * g5^2 + g1^2 - g2
	t[6].Sub(&t[5], &x.B0.A2)

	// t1 = g2^2
	t[1].Square(&x.B0.A2)

	// t6 = 2 * nr * g5^2 + 2 * g1^2 - 2*g2
	t[6].Double(&t[6])
	// z2 = 3 * nr * g5^2 + 3 * g1^2 - 2*g2
	z.B0.A2.Add(&t[6], &t[5])

	// t4 = nr * g2^2
	t[4].MulByNonResidue(&t[1])
	// t5 = g3^2 + nr * g2^2
	t[5].Add(&t[2], &t[4])
	// t6 = g3^2 + nr * g2^2 - g1
	t[6].Sub(&t[5], &x.B0.A1)
	// t6 = 2 * g3^2 + 2 * nr * g2^2 - 2 * g1
	t[6].Double(&t[6])
	// z1 = 3 * g3^2 + 3 * nr * g2^2 - 2 * g1
	z.B0.A1.Add(&t[6], &t[5])

	// t0 = g2^2 + g3^2
	t[0].Add(&t[2], &t[1])
	// t5 = 2 * g3 * g2
	t[5].Sub(&t[3], &t[0])
	// t6 = 2 * g3 * g2 + g5
	t[6].Add(&t[5], &x.B1.A2)
	// t6 = 4 * g3 * g2 + 2 * g5
	t[6].Double(&t[6])
	// z5 = 6 * g3 * g2 + 2 * g5
	z.B1.A2.Add(&t[5], &t[6])

	return z
}

// Decompress Karabina's cyclotomic square result
func (z *E6) Decompress(x *E6) *E6 {

	var t [3]fp.Element
	var one fp.Element
	one.SetOne()

	// t0 = g1^2
	t[0].Square(&x.B0.A1)
	// t1 = 3 * g1^2 - 2 * g2
	t[1].Sub(&t[0], &x.B0.A2).
		Double(&t[1]).
		Add(&t[1], &t[0])
		// t0 = E * g5^2 + t1
	t[2].Square(&x.B1.A2)
	t[0].MulByNonResidue(&t[2]).
		Add(&t[0], &t[1])
	// t1 = 1/(4 * g3)
	t[1].Double(&x.B1.A0).
		Double(&t[1]).
		Inverse(&t[1]) // costly
	// z4 = g4
	z.B1.A1.Mul(&t[0], &t[1])

	// t1 = g2 * g1
	t[1].Mul(&x.B0.A2, &x.B0.A1)
	// t2 = 2 * g4^2 - 3 * g2 * g1
	t[2].Square(&x.B1.A1).
		Sub(&t[2], &t[1]).
		Double(&t[2]).
		Sub(&t[2], &t[1])
	// t1 = g3 * g5
	t[1].Mul(&x.B1.A0, &x.B1.A2)
	// c_0 = E * (2 * g4^2 + g3 * g5 - 3 * g2 * g1) + 1
	t[2].Add(&t[2], &t[1])
	z.B0.A0.MulByNonResidue(&t[2]).
		Add(&z.B0.A0, &one)

	z.B0.A1.Set(&x.B0.A1)
	z.B0.A2.Set(&x.B0.A2)
	z.B1.A0.Set(&x.B1.A0)
	z.B1.A2.Set(&x.B1.A2)

	return z
}

// Granger-Scott's cyclotomic square
// https://eprint.iacr.org/2009/565.pdf, 3.2
func (z *E6) CyclotomicSquare(x *E6) *E6 {
	// x=(x0,x1,x2,x3,x4,x5,x6,x7) in E3^6
	// cyclosquare(x)=(3*x4^2*u + 3*x0^2 - 2*x0,
	//					3*x2^2*u + 3*x3^2 - 2*x1,
	//					3*x5^2*u + 3*x1^2 - 2*x2,
	//					6*x1*x5*u + 2*x3,
	//					6*x0*x4 + 2*x4,
	//					6*x2*x3 + 2*x5)

	var t [9]fp.Element

	t[0].Square(&x.B1.A1)
	t[1].Square(&x.B0.A0)
	t[6].Add(&x.B1.A1, &x.B0.A0).Square(&t[6]).Sub(&t[6], &t[0]).Sub(&t[6], &t[1]) // 2*x4*x0
	t[2].Square(&x.B0.A2)
	t[3].Square(&x.B1.A0)
	t[7].Add(&x.B0.A2, &x.B1.A0).Square(&t[7]).Sub(&t[7], &t[2]).Sub(&t[7], &t[3]) // 2*x2*x3
	t[4].Square(&x.B1.A2)
	t[5].Square(&x.B0.A1)
	t[8].Add(&x.B1.A2, &x.B0.A1).Square(&t[8]).Sub(&t[8], &t[4]).Sub(&t[8], &t[5]).MulByNonResidue(&t[8]) // 2*x5*x1*u

	t[0].MulByNonResidue(&t[0]).Add(&t[0], &t[1]) // x4^2*u + x0^2
	t[2].MulByNonResidue(&t[2]).Add(&t[2], &t[3]) // x2^2*u + x3^2
	t[4].MulByNonResidue(&t[4]).Add(&t[4], &t[5]) // x5^2*u + x1^2

	z.B0.A0.Sub(&t[0], &x.B0.A0).Double(&z.B0.A0).Add(&z.B0.A0, &t[0])
	z.B0.A1.Sub(&t[2], &x.B0.A1).Double(&z.B0.A1).Add(&z.B0.A1, &t[2])
	z.B0.A2.Sub(&t[4], &x.B0.A2).Double(&z.B0.A2).Add(&z.B0.A2, &t[4])

	z.B1.A0.Add(&t[8], &x.B1.A0).Double(&z.B1.A0).Add(&z.B1.A0, &t[8])
	z.B1.A1.Add(&t[6], &x.B1.A1).Double(&z.B1.A1).Add(&z.B1.A1, &t[6])
	z.B1.A2.Add(&t[7], &x.B1.A2).Double(&z.B1.A2).Add(&z.B1.A2, &t[7])

	return z
}

// Inverse set z to the inverse of x in E6 and return z
func (z *E6) Inverse(x *E6) *E6 {
	// Algorithm 23 from https://eprint.iacr.org/2010/354.pdf

	var t0, t1, tmp E3
	t0.Square(&x.B0)
	t1.Square(&x.B1)
	tmp.MulByNonResidue(&t1)
	t0.Sub(&t0, &tmp)
	t1.Inverse(&t0)
	z.B0.Mul(&x.B0, &t1)
	z.B1.Mul(&x.B1, &t1).Neg(&z.B1)

	return z
}

// Exp sets z=x**e and returns it
func (z *E6) Exp(x *E6, e big.Int) *E6 {
	var res E6
	res.SetOne()
	b := e.Bytes()
	for i := range b {
		w := b[i]
		mask := byte(0x80)
		for j := 7; j >= 0; j-- {
			res.Square(&res)
			if (w&mask)>>j != 0 {
				res.Mul(&res, x)
			}
			mask = mask >> 1
		}
	}
	z.Set(&res)
	return z
}

// InverseUnitary inverse a unitary element
func (z *E6) InverseUnitary(x *E6) *E6 {
	return z.Conjugate(x)
}

// Conjugate set z to x conjugated and return z
func (z *E6) Conjugate(x *E6) *E6 {
	*z = *x
	z.B1.Neg(&z.B1)
	return z
}

// SizeOfGT represents the size in bytes that a GT element need in binary form
const SizeOfGT = fp.Bytes * 6

// Bytes returns the regular (non montgomery) value
// of z as a big-endian byte array.
// z.C1.B2.A1 | z.C1.B2.A0 | z.C1.B1.A1 | ...
func (z *E6) Bytes() (r [SizeOfGT]byte) {

	offset := 0
	var buf [fp.Bytes]byte

	buf = z.B1.A2.Bytes()
	copy(r[offset:offset+fp.Bytes], buf[:])
	offset += fp.Bytes

	buf = z.B1.A1.Bytes()
	copy(r[offset:offset+fp.Bytes], buf[:])
	offset += fp.Bytes

	buf = z.B1.A0.Bytes()
	copy(r[offset:offset+fp.Bytes], buf[:])
	offset += fp.Bytes

	buf = z.B0.A2.Bytes()
	copy(r[offset:offset+fp.Bytes], buf[:])
	offset += fp.Bytes

	buf = z.B0.A1.Bytes()
	copy(r[offset:offset+fp.Bytes], buf[:])
	offset += fp.Bytes

	buf = z.B0.A0.Bytes()
	copy(r[offset:offset+fp.Bytes], buf[:])

	return
}

// SetBytes interprets e as the bytes of a big-endian GT
// sets z to that value (in Montgomery form), and returns z.
// z.C1.B2.A1 | z.C1.B2.A0 | z.C1.B1.A1 | ...
func (z *E6) SetBytes(e []byte) error {
	if len(e) != SizeOfGT {
		return errors.New("invalid buffer size")
	}
	offset := 0
	z.B1.A2.SetBytes(e[offset : offset+fp.Bytes])
	offset += fp.Bytes
	z.B1.A1.SetBytes(e[offset : offset+fp.Bytes])
	offset += fp.Bytes
	z.B1.A0.SetBytes(e[offset : offset+fp.Bytes])
	offset += fp.Bytes
	z.B0.A2.SetBytes(e[offset : offset+fp.Bytes])
	offset += fp.Bytes
	z.B0.A1.SetBytes(e[offset : offset+fp.Bytes])
	offset += fp.Bytes
	z.B0.A0.SetBytes(e[offset : offset+fp.Bytes])
	offset += fp.Bytes

	return nil
}

// IsInSubGroup ensures GT/E6 is in correct sugroup
func (z *E6) IsInSubGroup() bool {
	var tmp, a, _a, b E6
	var t [6]E6

	// check z^(Phi_k(p)) == 1
	a.Frobenius(z)
	b.Frobenius(&a).Mul(&b, z)

	if !a.Equal(&b) {
		return false
	}

	// check z^(p+1-t) == 1
	_a.Frobenius(z)
	a.CyclotomicSquare(&_a).Mul(&a, &_a) // z^(3p)

	// t(x)-1 = (13x^6 − 23x^5 − 9x^4 + 35x^3 + 10x + 19)/3
	t[0].CyclotomicSquare(z) // z^2
	t[1].CyclotomicSquare(&t[0]).
		CyclotomicSquare(&t[1]) // z^8
	t[2].CyclotomicSquare(&t[1]).
		Mul(&t[2], &t[0]).
		Mul(&t[2], z) // z^19*
	t[3].Mul(&t[0], &t[1]).
		Expt(&t[3]) // z^(10u)*
	t[4].CyclotomicSquare(&t[3]).
		Mul(&t[4], &t[3]) // z^(30u)
	t[0].CyclotomicSquare(&t[0]).
		Mul(&t[0], z).
		Expt(&t[0]) // z^(5u)
	t[4].Mul(&t[4], &t[0]).
		Expt(&t[4]).
		Expt(&t[4]) // z^(35u^3)*
	t[1].Mul(&t[1], z).
		Expt(&t[1]).
		Expt(&t[1]).
		Expt(&t[1]).
		Expt(&t[1]).
		Conjugate(&t[1]) // z^(-9u^4)*
	t[0].Expt(&t[0]).
		Expt(&t[0]).
		Expt(&t[0]).
		Conjugate(&t[0]) // z^(-5u^4)
	t[5].CyclotomicSquare(&t[1]).
		Mul(&t[5], &t[0]).
		Expt(&t[5]) // z^(-23u^5)*
	tmp.CyclotomicSquare(&t[1]).
		Conjugate(&tmp) // z^(18u^4)
	t[0].Mul(&t[0], &tmp).
		Expt(&t[0]).
		Expt(&t[0]) // z^(13u^6)*

	b.Mul(&t[2], &t[3]).
		Mul(&b, &t[4]).
		Mul(&b, &t[1]).
		Mul(&b, &t[5]).
		Mul(&b, &t[0]) // z^(3(t-1))

	return a.Equal(&b)
}
