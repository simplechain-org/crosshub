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
	"bytes"
	"errors"
	"math/big"
	"sync/atomic"

	"github.com/simplechain-org/go-simplechain/common"
	"github.com/simplechain-org/go-simplechain/common/math"
	"github.com/simplechain-org/go-simplechain/core/types"
	"github.com/simplechain-org/go-simplechain/crypto/sha3"
	"github.com/simplechain-org/go-simplechain/rlp"
)

type SignHash func(hash []byte) ([]byte, error)

type CtxID = common.Hash
type CtxIDs []CtxID

func (ids CtxIDs) String() string {
	var buffer bytes.Buffer
	buffer.WriteString("[")
	for _, id := range ids {
		buffer.WriteString(" ")
		buffer.WriteString(id.String())
	}
	buffer.WriteString(" ]")
	return buffer.String()
}

var ErrDuplicateSign = errors.New("signatures already exist")
var ErrInvalidSign = errors.New("invalid signature, different sign hash")

type CrossTransaction struct {
	Data ctxdata
	// caches
	hash     atomic.Value
	signHash atomic.Value
	size     atomic.Value
	from     atomic.Value
}

type ctxdata struct {
	CTxId     common.Hash `json:"ctxId" gencodec:"required"` //cross_transaction ID
	TxHash    common.Hash `json:"txHash" gencodec:"required"`
	BlockHash common.Hash `json:"blockHash" gencodec:"required"`        //The Hash of block in which the message resides
	Value     *big.Int    `json:"value" gencodec:"required"` //Token for sell
	Charge    *big.Int    `json:"charge" gencodec:"required"`
	From      string      `json:"from" gencodec:"required"`             //Token owner
	To        string      `json:"to" gencodec:"required"`               //Token to
	Origin    uint8       `json:"origin" gencodec:"required"`
	Purpose   uint8       `json:"purpose" gencodec:"required"` //Message destination networkId
	Payload   []byte      `json:"payload"    gencodec:"required"`

	// Signature values
	V *big.Int `json:"v" gencodec:"required"` //chainId
	R *big.Int `json:"r" gencodec:"required"`
	S *big.Int `json:"s" gencodec:"required"`
}

func NewCrossTransaction(amount, charge *big.Int, from, to string, origin, purpose uint8, id, txHash, bHash common.Hash,  input []byte) *CrossTransaction {
	return &CrossTransaction{
		Data: ctxdata{
			Value:     amount,
			CTxId:     id,
			TxHash:    txHash,
			From:      from,
			To:        to,
			BlockHash: bHash,
			Origin:    origin,
			Purpose:   purpose,
			Charge:    charge,
			Payload:   input,
			V:         new(big.Int),
			R:         new(big.Int),
			S:         new(big.Int),
		}}
}

func (tx *CrossTransaction) WithSignature(signer CtxSigner, sig []byte) (*CrossTransaction, error) {
	r, s, v, err := signer.SignatureValues(tx, sig)
	if err != nil {
		return nil, err
	}
	cpy := &CrossTransaction{Data: tx.Data}
	cpy.Data.R, cpy.Data.S, cpy.Data.V = r, s, v
	return cpy, nil
}

func (tx *CrossTransaction) ID() common.Hash {
	return tx.Data.CTxId
}

func (tx *CrossTransaction) ChainId() *big.Int {
	return types.DeriveChainId(tx.Data.V)
}

func (tx *CrossTransaction) Destination() uint8 {
	return tx.Data.Purpose
}

func (tx *CrossTransaction) Hash() (h common.Hash) {
	if hash := tx.hash.Load(); hash != nil {
		return hash.(common.Hash)
	}
	hash := sha3.NewKeccak256()
	var b []byte

	b = append(b, tx.Data.CTxId.Bytes()...)
	b = append(b, tx.Data.TxHash.Bytes()...)
	b = append(b, tx.Data.BlockHash.Bytes()...)
	b = append(b, common.LeftPadBytes(tx.Data.Value.Bytes(), 32)...)
	b = append(b, common.LeftPadBytes(tx.Data.Charge.Bytes(), 32)...)
	b = append(b, tx.Data.From...)
	b = append(b, tx.Data.To...)
	b = append(b, tx.Data.Origin)
	b = append(b, tx.Data.Purpose)
	b = append(b, tx.Data.Payload...)
	hash.Write(b)
	hash.Sum(h[:0])
	tx.hash.Store(h)
	return h
}

func (tx *CrossTransaction) SimpleHash() (h common.Hash) {
	hash := sha3.NewKeccak256()
	var b []byte

	b = append(b, tx.Data.CTxId.Bytes()...)
	b = append(b, tx.Data.TxHash.Bytes()...)
	b = append(b, tx.Data.BlockHash.Bytes()...)
	b = append(b, common.LeftPadBytes(tx.Data.Value.Bytes(), 32)...)
	b = append(b, common.LeftPadBytes(tx.Data.Charge.Bytes(), 32)...)
	b = append(b, common.HexToAddress(tx.Data.From).Bytes()...)
	b = append(b, common.HexToAddress(tx.Data.To).Bytes()...)
	b = append(b, tx.Data.Origin)
	b = append(b, tx.Data.Purpose)
	b = append(b, tx.Data.Payload...)
	hash.Write(b)
	hash.Sum(h[:0])
	return h
}

func (tx *CrossTransaction) BlockHash() common.Hash {
	return tx.Data.BlockHash
}

func (tx *CrossTransaction) From() string {
	return tx.Data.From
}

func (tx *CrossTransaction) SignHash() (h common.Hash) {
	if hash := tx.signHash.Load(); hash != nil {
		return hash.(common.Hash)
	}
	hash := sha3.NewKeccak256()
	var b []byte
	b = append(b, tx.Data.CTxId.Bytes()...)
	b = append(b, tx.Data.TxHash.Bytes()...)
	b = append(b, tx.Data.BlockHash.Bytes()...)
	b = append(b, common.LeftPadBytes(tx.Data.Value.Bytes(), 32)...)
	b = append(b, common.LeftPadBytes(tx.Data.Charge.Bytes(), 32)...)
	b = append(b, tx.Data.From...)
	b = append(b, tx.Data.To...)
	b = append(b, tx.Data.Origin)
	b = append(b, tx.Data.Purpose)
	b = append(b, tx.Data.Payload...)
	b = append(b, common.LeftPadBytes(tx.Data.V.Bytes(), 32)...)
	b = append(b, common.LeftPadBytes(tx.Data.R.Bytes(), 32)...)
	b = append(b, common.LeftPadBytes(tx.Data.S.Bytes(), 32)...)
	hash.Write(b)
	hash.Sum(h[:0])
	tx.signHash.Store(h)
	return h
}

func (cws *CrossTransaction) Price() *big.Rat {
	if cws.Data.Value.Cmp(common.Big0) == 0 {
		return new(big.Rat).SetUint64(math.MaxUint64) // set a max rat
	}
	return new(big.Rat).SetFrac(cws.Data.Charge, cws.Data.Value)
}

// Transactions is a Transaction slice type for basic sorting.
type CrossTransactions []*CrossTransaction

// Len returns the length of s.
func (s CrossTransactions) Len() int { return len(s) }

// Swap swaps the i'th and the j'th element in s.
func (s CrossTransactions) Swap(i, j int) { s[i], s[j] = s[j], s[i] }

// GetRlp implements Rlpable and returns the i'th element of s in rlp.
func (s CrossTransactions) GetRlp(i int) []byte {
	enc, _ := rlp.EncodeToBytes(s[i])
	return enc
}
// TxByPrice implements both the sort and the heap interface, making it useful
// for all at once sorting as well as individually adding and removing elements.
type CTxByPrice CrossTransactions

func (s CTxByPrice) Len() int { return len(s) }
func (s CTxByPrice) Less(i, j int) bool {
	return s[i].Price().Cmp(s[j].Price()) > 0
}
func (s CTxByPrice) Swap(i, j int) { s[i], s[j] = s[j], s[i] }

func (s *CTxByPrice) Push(x interface{}) {
	*s = append(*s, x.(*CrossTransaction))
}

func (s *CTxByPrice) Pop() interface{} {
	old := *s
	n := len(old)
	x := old[n-1]
	*s = old[0 : n-1]
	return x
}

//type CrossTransactionWithSignatures struct {
//	Data     CtxDatas
//	Status   CtxStatus `json:"status" gencodec:"required"` // default = pending
//	BlockNum uint64    `json:"blockNum" gencodec:"required"`
//
//	// caches
//	hash atomic.Value
//	size atomic.Value
//	from atomic.Value
//	lock sync.RWMutex
//}
//
//type CtxDatas struct {
//	CTxId     common.Hash `json:"ctxId" gencodec:"required"` //cross_transaction ID
//	TxHash    common.Hash `json:"txHash" gencodec:"required"`
//	BlockHash common.Hash `json:"blockHash" gencodec:"required"`        //The Hash of block in which the message resides
//	Value     *big.Int    `json:"value" gencodec:"required"` //Token for sell
//	Charge    *big.Int    `json:"charge" gencodec:"required"`
//	From      string      `json:"from" gencodec:"required"`             //Token owner
//	To        string      `json:"to" gencodec:"required"`               //Token to
//	Origin    uint8       `json:"origin" gencodec:"required"`
//	Purpose   uint8       `json:"purpose" gencodec:"required"` //Message destination networkId
//	Payload   []byte      `json:"payload"    gencodec:"required"`
//
//	// Signature values
//	V []*big.Int `json:"v" gencodec:"required"` //chainId
//	R []*big.Int `json:"r" gencodec:"required"`
//	S []*big.Int `json:"s" gencodec:"required"`
//}
//
//func NewCrossTransactionWithSignatures(ctx *CrossTransaction, num uint64) *CrossTransactionWithSignatures {
//	d := CtxDatas{
//		Value:     ctx.Data.Value,
//		CTxId:     ctx.Data.CTxId,
//		TxHash:    ctx.Data.TxHash,
//		From:      ctx.Data.From,
//		To:        ctx.Data.To,
//		BlockHash: ctx.Data.BlockHash,
//		Origin:    ctx.Data.Origin,
//		Purpose:   ctx.Data.Purpose,
//		Charge:    ctx.Data.Charge,
//		Payload:   ctx.Data.Payload,
//	}
//
//	if ctx.Data.V != nil && ctx.Data.R != nil && ctx.Data.S != nil {
//		d.V = append(d.V, ctx.Data.V)
//		d.R = append(d.R, ctx.Data.R)
//		d.S = append(d.S, ctx.Data.S)
//	}
//
//	return &CrossTransactionWithSignatures{Data: d, BlockNum: num}
//}
//
//func (cws *CrossTransactionWithSignatures) ID() common.Hash {
//	return cws.Data.CTxId
//}
//
//func (cws *CrossTransactionWithSignatures) ChainId() *big.Int {
//	cws.lock.RLock()
//	defer cws.lock.RUnlock()
//	if cws.signaturesLength() > 0 {
//		return types.DeriveChainId(cws.Data.V[0])
//	}
//	return big.NewInt(0)
//}
//func (cws *CrossTransactionWithSignatures) Destination() uint8 {
//	return cws.Data.Purpose
//}
//
//func (cws *CrossTransactionWithSignatures) Hash() (h common.Hash) {
//	if hash := cws.hash.Load(); hash != nil {
//		return hash.(common.Hash)
//	}
//	hash := sha3.NewKeccak256()
//	var b []byte
//	b = append(b, cws.Data.CTxId.Bytes()...)
//	b = append(b, cws.Data.TxHash.Bytes()...)
//	b = append(b, cws.Data.BlockHash.Bytes()...)
//	b = append(b, common.LeftPadBytes(cws.Data.Value.Bytes(), 32)...)
//	b = append(b, common.LeftPadBytes(cws.Data.Charge.Bytes(), 32)...)
//	b = append(b, cws.Data.From...)
//	b = append(b, cws.Data.To...)
//	b = append(b, cws.Data.Origin)
//	b = append(b, cws.Data.Purpose)
//	b = append(b, cws.Data.Payload...)
//	hash.Write(b)
//	hash.Sum(h[:0])
//	cws.hash.Store(h)
//	return h
//}
//
//func (cws *CrossTransactionWithSignatures) BlockHash() common.Hash {
//	return cws.Data.BlockHash
//}
//
//func (cws *CrossTransactionWithSignatures) From() string {
//	return cws.Data.From
//}
//
//func (cws *CrossTransactionWithSignatures) SetStatus(status CtxStatus) {
//	cws.Status = status
//}
//
//func (cws *CrossTransactionWithSignatures) AddSignature(ctx *CrossTransaction) error {
//	if cws.Hash() != ctx.Hash() {
//		return ErrInvalidSign
//	}
//	cws.lock.Lock()
//	defer cws.lock.Unlock()
//	for _, r := range cws.Data.R {
//		if r.Cmp(ctx.Data.R) == 0 {
//			return ErrDuplicateSign
//		}
//	}
//	cws.Data.V = append(cws.Data.V, ctx.Data.V)
//	cws.Data.R = append(cws.Data.R, ctx.Data.R)
//	cws.Data.S = append(cws.Data.S, ctx.Data.S)
//	return nil
//}
//func (cws *CrossTransactionWithSignatures) RemoveSignature(index int) {
//	cws.lock.Lock()
//	defer cws.lock.Unlock()
//	if index < cws.signaturesLength() {
//		cws.Data.V = append(cws.Data.V[:index], cws.Data.V[index+1:]...)
//		cws.Data.R = append(cws.Data.R[:index], cws.Data.R[index+1:]...)
//		cws.Data.S = append(cws.Data.S[:index], cws.Data.S[index+1:]...)
//	}
//}
//
//func (cws *CrossTransactionWithSignatures) SignaturesLength() int {
//	cws.lock.RLock()
//	defer cws.lock.RUnlock()
//	return cws.signaturesLength()
//}
//func (cws *CrossTransactionWithSignatures) signaturesLength() int {
//	l := len(cws.Data.V)
//	if l == len(cws.Data.R) && l == len(cws.Data.V) {
//		return l
//	}
//	return 0
//}
//
//func (cws *CrossTransactionWithSignatures) CrossTransaction() *CrossTransaction {
//	return &CrossTransaction{
//		Data: ctxdata{
//			Value:     cws.Data.Value,
//			CTxId:     cws.Data.CTxId,
//			TxHash:    cws.Data.TxHash,
//			From:      cws.Data.From,
//			To:        cws.Data.To,
//			BlockHash: cws.Data.BlockHash,
//			Origin:    cws.Data.Origin,
//			Purpose:   cws.Data.Purpose,
//			Charge:    cws.Data.Charge,
//			Payload:   cws.Data.Payload,
//		},
//	}
//}
//
//func (cws *CrossTransactionWithSignatures) Resolution() []*CrossTransaction {
//	cws.lock.RLock()
//	defer cws.lock.RUnlock()
//	l := cws.signaturesLength()
//	var ctxs []*CrossTransaction
//	for i := 0; i < l; i++ {
//		ctxs = append(ctxs, &CrossTransaction{
//			Data: ctxdata{
//				Value:     cws.Data.Value,
//				CTxId:     cws.Data.CTxId,
//				TxHash:    cws.Data.TxHash,
//				From:      cws.Data.From,
//				To:        cws.Data.To,
//				BlockHash: cws.Data.BlockHash,
//				Origin:    cws.Data.Origin,
//				Purpose:   cws.Data.Purpose,
//				Charge:    cws.Data.Charge,
//				Payload:   cws.Data.Payload,
//				V:         cws.Data.V[i],
//				R:         cws.Data.R[i],
//				S:         cws.Data.S[i],
//			},
//		})
//	}
//	return ctxs
//}
//
//func (cws *CrossTransactionWithSignatures) Price() *big.Rat {
//	if cws.Data.Value.Cmp(common.Big0) == 0 {
//		return new(big.Rat).SetUint64(math.MaxUint64) // set a max rat
//	}
//	return new(big.Rat).SetFrac(cws.Data.Charge, cws.Data.Value)
//}
//
//func (cws *CrossTransactionWithSignatures) Size() common.StorageSize {
//	if size := cws.size.Load(); size != nil {
//		return size.(common.StorageSize)
//	}
//	c := types.WriteCounter(0)
//	rlp.Encode(&c, &cws.Data)
//	cws.size.Store(common.StorageSize(c))
//	return common.StorageSize(c)
//}
//
//type RemoteChainInfo struct {
//	RemoteChainId uint64
//	BlockNumber   uint64
//}
//
//type OwnerCrossTransactionWithSignatures struct {
//	Cws  *CrossTransactionWithSignatures
//	Time uint64
//}
