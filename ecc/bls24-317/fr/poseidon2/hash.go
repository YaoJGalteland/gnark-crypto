// Copyright 2020-2025 Consensys Software Inc.
// Licensed under the Apache License, Version 2.0. See the LICENSE file for details.

// Code generated by consensys/gnark-crypto DO NOT EDIT

package poseidon2

import (
	"hash"
	"sync"

	"github.com/consensys/gnark-crypto/ecc/bls24-317/fr"
	gnarkHash "github.com/consensys/gnark-crypto/hash"
)

// NewMerkleDamgardHasher returns a Poseidon2 hasher using the Merkle-Damgard
// construction with the default parameters.
func NewMerkleDamgardHasher() gnarkHash.StateStorer {
	return gnarkHash.NewMerkleDamgardHasher(
		&Permutation{GetDefaultParameters()}, make([]byte, fr.Bytes))
}

// GetDefaultParameters returns a set of parameters for the Poseidon2 permutation.
// The default parameters are:

// - width: 2
// - nbFullRounds: 6
// - nbPartialRounds: 26
var GetDefaultParameters = sync.OnceValue(func() *Parameters {
	return NewParameters(2, 6, 40)
})

func init() {
	gnarkHash.RegisterHash(gnarkHash.POSEIDON2_BLS24_317, func() hash.Hash {
		return NewMerkleDamgardHasher()
	})
}
