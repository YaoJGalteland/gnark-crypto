//go:build !purego

// Copyright 2020-2025 Consensys Software Inc.
// Licensed under the Apache License, Version 2.0. See the LICENSE file for details.

// Code generated by consensys/gnark-crypto DO NOT EDIT

package fp

import (
	_ "github.com/consensys/gnark-crypto/field/asm/element_4w"
)

//go:noescape
func MulBy3(x *Element)

//go:noescape
func MulBy5(x *Element)

//go:noescape
func MulBy13(x *Element)

//go:noescape
func mul(res, x, y *Element)

//go:noescape
func fromMont(res *Element)

//go:noescape
func reduce(res *Element)

// Butterfly sets
//
//	a = a + b (mod q)
//	b = a - b (mod q)
//
//go:noescape
func Butterfly(a, b *Element)

// Mul z = x * y (mod q)
//
// x and y must be less than q
func (z *Element) Mul(x, y *Element) *Element {

	// Algorithm 2 of "Faster Montgomery Multiplication and Multi-Scalar-Multiplication for SNARKS"
	// by Y. El Housni and G. Botrel https://doi.org/10.46586/tches.v2023.i3.504-521

	mul(z, x, y)
	return z
}

// Square z = x * x (mod q)
//
// x must be less than q
func (z *Element) Square(x *Element) *Element {
	// see Mul for doc.
	mul(z, x, x)
	return z
}
