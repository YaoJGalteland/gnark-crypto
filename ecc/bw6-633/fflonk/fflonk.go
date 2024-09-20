// Copyright 2020 Consensys Software Inc.
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

package fflonk

import (
	"errors"
	"hash"
	"math/big"

	"github.com/consensys/gnark-crypto/ecc/bw6-633/fr"
	"github.com/consensys/gnark-crypto/ecc/bw6-633/fr/fft"
	"github.com/consensys/gnark-crypto/ecc/bw6-633/kzg"
	"github.com/consensys/gnark-crypto/ecc/bw6-633/shplonk"
)

var (
	ErrRootsOne                       = errors.New("fr does not contain all the t-th roots of 1")
	ErrNbPolynomialsNbPoints          = errors.New("the number of packs of polynomials should be the same as the number of pack of points")
	ErrInonsistentFolding             = errors.New("the outer claimed values are not consistent with the shplonk proof")
	ErrInconsistentNumberFoldedPoints = errors.New("the number of outer claimed values is inconsistent with the number of claimed values in the shplonk proof")
)

// Opening fflonk proof for opening a list of list of polynomials ((fʲᵢ)ᵢ)ⱼ where each
// pack of polynomials (fʲᵢ)ᵢ (the pack is indexed by j) is opened on a powers of elements in
// the set Sʲ (indexed by j), where the power is |(fʲᵢ)ᵢ|.
//
// implements io.ReaderFrom and io.WriterTo
type OpeningProof struct {

	// shplonk opening proof of the folded polynomials
	SOpeningProof shplonk.OpeningProof

	// ClaimedValues ClaimedValues[i][j] contains the values
	// of fʲᵢ on Sⱼ^{|(fʲᵢ)ᵢ|}
	ClaimedValues [][][]fr.Element
}

// FoldAndCommit commits to a list of polynomial by intertwinning them like in the FFT, that is
// returns ∑_{i<t}Pᵢ(Xᵗ)Xⁱ for t polynomials
func FoldAndCommit(p [][]fr.Element, pk kzg.ProvingKey, nbTasks ...int) (kzg.Digest, error) {
	buf := Fold(p)
	com, err := kzg.Commit(buf, pk, nbTasks...)
	return com, err
}

// Fold returns p folded as in the fft, that is ∑_{i<t}Pᵢ(Xᵗ)Xⁱ.
// Say max{degree(P_{i})}=n-1. The total degree of the folded polynomial
// is t(n-1)+(t-1). The total size is therefore t(n-1)+(t-1)+1 = tn.
func Fold(p [][]fr.Element) []fr.Element {

	// we first pick the smallest divisor of r-1 bounding above len(p)
	t := getNextDivisorRMinusOne(len(p))

	sizeResult := 0
	for i := range p {
		if sizeResult < len(p[i]) {
			sizeResult = len(p[i])
		}
	}
	sizeResult = sizeResult * t
	buf := make([]fr.Element, sizeResult)
	for i := range p {
		for j := range p[i] {
			buf[j*t+i].Set(&p[i][j])
		}
	}
	return buf
}

// BatchOpen computes a batch opening proof of p (the (fʲᵢ)ᵢ ) on powers of points (the ((Sʲᵢ)ᵢ)ⱼ).
// The j-th pack of polynomials is opened on the power |(fʲᵢ)ᵢ| of (Sʲᵢ)ᵢ.
// digests is the list (FoldAndCommit(p[i]))ᵢ. It is assumed that the list has been computed beforehand
// and provided as an input to not duplicate computations.
func BatchOpen(p [][][]fr.Element, digests []kzg.Digest, points [][]fr.Element, hf hash.Hash, pk kzg.ProvingKey, dataTranscript ...[]byte) (OpeningProof, error) {

	var res OpeningProof

	if len(p) != len(points) {
		return res, ErrNbPolynomialsNbPoints
	}

	// step 0: compute the relevant powers of the ((Sʲᵢ)ᵢ)ⱼ)
	nbPolysPerPack := make([]int, len(p))
	nextDivisorRminusOnePerPack := make([]int, len(p))
	for i := 0; i < len(p); i++ {
		nbPolysPerPack[i] = len(p[i])
		nextDivisorRminusOnePerPack[i] = getNextDivisorRMinusOne(len(p[i]))
	}
	pointsPowerM := make([][]fr.Element, len(points))
	var tmpBigInt big.Int
	for i := 0; i < len(p); i++ {
		tmpBigInt.SetUint64(uint64(nextDivisorRminusOnePerPack[i]))
		pointsPowerM[i] = make([]fr.Element, len(points[i]))
		for j := 0; j < len(points[i]); j++ {
			pointsPowerM[i][j].Exp(points[i][j], &tmpBigInt)
		}
	}

	// step 1: compute the claimed values, that is the evaluations of the polynomials
	// on the relevant powers of the sets
	res.ClaimedValues = make([][][]fr.Element, len(p))
	for i := 0; i < len(p); i++ {
		res.ClaimedValues[i] = make([][]fr.Element, nextDivisorRminusOnePerPack[i])
		for j := 0; j < len(p[i]); j++ {
			res.ClaimedValues[i][j] = make([]fr.Element, len(points[i]))
			for k := 0; k < len(points[i]); k++ {
				res.ClaimedValues[i][j][k] = eval(p[i][j], pointsPowerM[i][k])
			}
		}
		for j := len(p[i]); j < nextDivisorRminusOnePerPack[i]; j++ { // -> the remaining polynomials are zero
			res.ClaimedValues[i][j] = make([]fr.Element, len(points[i]))
		}
	}

	// step 2: fold polynomials
	foldedPolynomials := make([][]fr.Element, len(p))
	for i := 0; i < len(p); i++ {
		foldedPolynomials[i] = Fold(p[i])
	}

	// step 4: compute the associated roots, that is for each point p corresponding
	// to a pack i of polynomials, we extend to <p, ω p, .., ωᵗ⁻¹p> if
	// the i-th pack contains t polynomials where ω is a t-th root of 1
	newPoints := make([][]fr.Element, len(points))
	var err error
	for i := 0; i < len(p); i++ {
		newPoints[i], err = extendSet(points[i], nextDivisorRminusOnePerPack[i])
		if err != nil {
			return res, err
		}
	}

	// step 5: shplonk open the list of single polynomials on the new sets
	res.SOpeningProof, err = shplonk.BatchOpen(foldedPolynomials, digests, newPoints, hf, pk, dataTranscript...)

	return res, err

}

// BatchVerify uses a proof to check that each digest digests[i] is correctly opened on the set points[i].
// The digests are the commitments to the folded underlying polynomials. The shplonk proof is
// verified directly using the embedded shplonk proof. This function only computes the consistency
// between the claimed values of the underlying shplonk proof and the outer claimed values, using the fft-like
// folding. Namely, the outer claimed values are the evaluation of the original polynomials (so before they
// were folded) at the relevant powers of the points.
func BatchVerify(proof OpeningProof, digests []kzg.Digest, points [][]fr.Element, hf hash.Hash, vk kzg.VerifyingKey, dataTranscript ...[]byte) error {

	// step 0: consistency checks between the folded claimed values of shplonk and the claimed
	// values at the powers of the Sᵢ
	for i := 0; i < len(proof.ClaimedValues); i++ {
		sizeSi := len(proof.ClaimedValues[i][0])
		for j := 1; j < len(proof.ClaimedValues[i]); j++ {
			// each set of opening must be of the same size (opeings on powers of Si)
			if sizeSi != len(proof.ClaimedValues[i][j]) {
				return ErrNbPolynomialsNbPoints
			}
		}
		currNbPolynomials := len(proof.ClaimedValues[i])
		sizeSi = sizeSi * currNbPolynomials
		// |originalPolynomials_{i}|x|Sᵢ| == |foldedPolynomials|x|folded Sᵢ|
		if sizeSi != len(proof.SOpeningProof.ClaimedValues[i]) {
			return ErrInconsistentNumberFoldedPoints
		}
	}

	// step 1: fold the outer claimed values and check that they correspond to the
	// shplonk claimed values
	var curFoldedClaimedValue, omgeaiPoint fr.Element
	for i := 0; i < len(proof.ClaimedValues); i++ {
		t := len(proof.ClaimedValues[i])
		omega, err := getIthRootOne(t)
		if err != nil {
			return err
		}
		sizeSi := len(proof.ClaimedValues[i][0])
		polyClaimedValues := make([]fr.Element, t)
		for j := 0; j < sizeSi; j++ {
			for k := 0; k < t; k++ {
				polyClaimedValues[k].Set(&proof.ClaimedValues[i][k][j])
			}
			omgeaiPoint.Set(&points[i][j])
			for l := 0; l < t; l++ {
				curFoldedClaimedValue = eval(polyClaimedValues, omgeaiPoint)
				if !curFoldedClaimedValue.Equal(&proof.SOpeningProof.ClaimedValues[i][j*t+l]) {
					return ErrInonsistentFolding
				}
				omgeaiPoint.Mul(&omgeaiPoint, &omega)
			}
		}
	}

	// step 2: verify the embedded shplonk proof
	extendedPoints := make([][]fr.Element, len(points))
	var err error
	for i := 0; i < len(points); i++ {
		t := len(proof.ClaimedValues[i])
		extendedPoints[i], err = extendSet(points[i], t)
		if err != nil {
			return err
		}
	}
	err = shplonk.BatchVerify(proof.SOpeningProof, digests, extendedPoints, hf, vk, dataTranscript...)

	return err
}

// utils

// getIthRootOne returns a generator of Z/iZ
func getIthRootOne(i int) (fr.Element, error) {
	var omega fr.Element
	var tmpBigInt, zeroBigInt big.Int
	oneBigInt := big.NewInt(1)
	zeroBigInt.SetUint64(0)
	rMinusOneBigInt := fr.Modulus()
	rMinusOneBigInt.Sub(rMinusOneBigInt, oneBigInt)
	tmpBigInt.SetUint64(uint64(i))
	tmpBigInt.Mod(rMinusOneBigInt, &tmpBigInt)
	if tmpBigInt.Cmp(&zeroBigInt) != 0 {
		return omega, ErrRootsOne
	}
	genFrStar := fft.GeneratorFullMultiplicativeGroup()
	tmpBigInt.SetUint64(uint64(i))
	tmpBigInt.Div(rMinusOneBigInt, &tmpBigInt)
	omega.Exp(genFrStar, &tmpBigInt)
	return omega, nil
}

// computes the smallest i bounding above number_polynomials
// and dividing r-1.
func getNextDivisorRMinusOne(i int) int {
	var zero, tmp, one big.Int
	r := fr.Modulus()
	one.SetUint64(1)
	r.Sub(r, &one)
	tmp.SetUint64(uint64(i))
	tmp.Mod(r, &tmp)
	nbTrials := 100 // prevent DOS attack if the prime is not smooth
	for tmp.Cmp(&zero) != 0 && nbTrials > 0 {
		i += 1
		tmp.SetUint64(uint64(i))
		tmp.Mod(r, &tmp)
		nbTrials--
	}
	if nbTrials == 0 {
		panic("did not find any divisor of r-1")
	}
	return i
}

// extendSet returns [p[0], ω p[0], .. ,ωᵗ⁻¹p[0],p[1],..,ωᵗ⁻¹p[1],..]
func extendSet(p []fr.Element, t int) ([]fr.Element, error) {

	omega, err := getIthRootOne(t)
	if err != nil {
		return nil, err
	}
	nbPoints := len(p)
	newPoints := make([]fr.Element, t*nbPoints)
	for i := 0; i < nbPoints; i++ {
		newPoints[i*t].Set(&p[i])
		for k := 1; k < t; k++ {
			newPoints[i*t+k].Mul(&newPoints[i*t+k-1], &omega)
		}
	}

	return newPoints, nil
}

func eval(f []fr.Element, x fr.Element) fr.Element {
	var y fr.Element
	for i := len(f) - 1; i >= 0; i-- {
		y.Mul(&y, &x).Add(&y, &f[i])
	}
	return y
}
