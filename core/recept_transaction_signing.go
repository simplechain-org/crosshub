// Copyright 2016 The go-simplechain Authors
// This file is part of the go-simplechain library.
//
// The go-simplechain library is free software: you can redistribute it and/or modify
// it under the terms of the GNU Lesser General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// The go-simplechain library is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
// GNU Lesser General Public License for more details.
//
// You should have received a copy of the GNU Lesser General Public License
// along with the go-simplechain library. If not, see <http://www.gnu.org/licenses/>.

package core

import (
	"fmt"
	"math/big"

	"github.com/simplechain-org/go-simplechain/common"
	"github.com/simplechain-org/go-simplechain/core/types"
	"github.com/simplechain-org/go-simplechain/crypto/sha3"
)

// sigCache is used to cache the derived sender and contains
// the signer used to derive it.
type rtxSigCache struct {
	signer RtxSigner
	from   common.Address
}

// MakeSigner returns a Signer based on the given chain config and block number.
func MakeRtxSigner(chainID *big.Int) RtxSigner {
	return NewEIP155RtxSigner(chainID)
}

// SignTx signs the transaction using the given signer and private key
func SignRtx(tx *ReceptTransaction, s RtxSigner, signHash SignHash) (*ReceptTransaction, error) {
	h := s.Hash(tx)
	sig, err := signHash(h[:])
	if err != nil {
		return nil, err
	}
	return tx.WithSignature(s, sig)
}

// Sender returns the address derived from the signature (V, R, S) using secp256k1
// elliptic curve and an error if it failed deriving or upon an incorrect
// signature.
//
// Sender may cache the address, allowing it to be used regardless of
// signing method. The cache is invalidated if the cached signer does
// not match the signer used in the current call.
func RtxSender(signer RtxSigner, tx *ReceptTransaction) (common.Address, error) {
	if sc := tx.from.Load(); sc != nil {
		sigCache := sc.(rtxSigCache)
		// If the signer used to derive from in a previous
		// call is not the same as used current, invalidate
		// the cache.
		if sigCache.signer.Equal(signer) {
			return sigCache.from, nil
		}
	}

	addr, err := signer.Sender(tx)
	if err != nil {
		return common.Address{}, err
	}
	tx.from.Store(rtxSigCache{signer: signer, from: addr})
	return addr, nil
}

// Signer encapsulates transaction signature handling. Note that this interface is not a
// stable API and may change at any time to accommodate new protocol rules.
type RtxSigner interface {
	// Sender returns the sender address of the transaction.
	Sender(tx *ReceptTransaction) (common.Address, error)
	// SignatureValues returns the raw R, S, V values corresponding to the
	// given signature.
	SignatureValues(tx *ReceptTransaction, sig []byte) (r, s, v *big.Int, err error)
	// Hash returns the hash to be signed.
	Hash(tx *ReceptTransaction) common.Hash
	// Equal returns true if the given signer is the same as the receiver.
	Equal(RtxSigner) bool
}

// EIP155Transaction implements Signer using the EIP155 rules.
type EIP155RtxSigner struct {
	chainId, chainIdMul *big.Int
}

func NewEIP155RtxSigner(chainId *big.Int) EIP155RtxSigner {
	if chainId == nil {
		chainId = new(big.Int)
	}
	return EIP155RtxSigner{
		chainId:    chainId,
		chainIdMul: new(big.Int).Mul(chainId, big.NewInt(2)),
	}
}

func (s EIP155RtxSigner) Equal(s2 RtxSigner) bool {
	eip155, ok := s2.(EIP155RtxSigner)
	return ok && eip155.chainId.Cmp(s.chainId) == 0
}

func (s EIP155RtxSigner) Sender(tx *ReceptTransaction) (common.Address, error) {
	if tx.ChainId().Cmp(s.chainId) != 0 {
		return common.Address{}, types.ErrInvalidChainId
	}
	V := new(big.Int).Sub(tx.Data.V, s.chainIdMul)
	V.Sub(V, big8)
	return types.RecoverPlain(s.Hash(tx), tx.Data.R, tx.Data.S, V, true)
}

// WithSignature returns a new transaction with the given signature. This signature
// needs to be in the [R || S || V] format where V is 0 or 1.
func (s EIP155RtxSigner) SignatureValues(tx *ReceptTransaction, sig []byte) (R, S, V *big.Int, err error) {
	if len(sig) != 65 {
		panic(fmt.Sprintf("wrong size for signature: got %d, want 65", len(sig)))
	}
	R = new(big.Int).SetBytes(sig[:32])
	S = new(big.Int).SetBytes(sig[32:64])
	V = new(big.Int).SetBytes([]byte{sig[64] + 27})

	if s.chainId.Sign() != 0 {
		V = big.NewInt(int64(sig[64] + 35))
		V.Add(V, s.chainIdMul)
	}
	return R, S, V, nil
}

func (s EIP155RtxSigner) Hash(tx *ReceptTransaction) (h common.Hash) {
	hash := sha3.NewKeccak256()
	var b []byte
	b = append(b, tx.Data.CTxId.Bytes()...)
	b = append(b, tx.Data.TxHash.Bytes()...)
	b = append(b, tx.Data.From...)
	b = append(b, tx.Data.To...)
	b = append(b, tx.Data.Purpose)
	b = append(b, tx.Data.Payload...)
	hash.Write(b)
	hash.Sum(h[:0])
	return h
}

//TODO Fabric Sign Func
