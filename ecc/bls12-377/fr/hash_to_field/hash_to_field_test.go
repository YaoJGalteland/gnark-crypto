// Copyright 2020-2025 Consensys Software Inc.
// Licensed under the Apache License, Version 2.0. See the LICENSE file for details.

// Code generated by consensys/gnark-crypto DO NOT EDIT

package hash_to_field

import (
	"testing"

	"github.com/consensys/gnark-crypto/ecc/bls12-377/fr"
)

func TestHashInterface(t *testing.T) {
	msg := []byte("test")
	sep := []byte("separator")
	res, err := fr.Hash(msg, sep, 1)
	if err != nil {
		t.Fatal("hash to field", err)
	}

	htfFn := New(sep)
	htfFn.Write(msg)
	bts := htfFn.Sum(nil)
	var res2 fr.Element
	res2.SetBytes(bts[:fr.Bytes])
	if !res[0].Equal(&res2) {
		t.Error("not equal")
	}
}
