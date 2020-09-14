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

//import (
//	"math/big"
//	"sync/atomic"
//
//	"github.com/simplechain-org/go-simplechain/common"
//	"github.com/simplechain-org/go-simplechain/crypto/sha3"
//	"github.com/simplechain-org/go-simplechain/rlp"
//)
//
//type FabricCross struct {
//	Data fcdata
//	// caches
//	hash     atomic.Value
//	signHash atomic.Value
//	size     atomic.Value
//	from     atomic.Value
//}
//
//type fcdata struct {
//	CTxId     common.Hash `json:"ctxId" gencodec:"required"` //cross_transaction ID
//	TxHash    common.Hash `json:"txHash" gencodec:"required"`
//	BlockHash common.Hash `json:"blockHash" gencodec:"required"`        //The Hash of block in which the message resides
//	Value     *big.Int    `json:"value" gencodec:"required"` //Token for sell
//	Charge    *big.Int    `json:"charge" gencodec:"required"`
//	From      string      `json:"from" gencodec:"required"`             //Token owner
//	To        string      `json:"to" gencodec:"required"`               //Token to
//	Origin    uint8       `json:"origin" gencodec:"required"`
//	Purpose   uint8       `json:"purpose" gencodec:"required"` //Message destination networkId
//	Payload   []byte      `json:"payload" gencodec:"required"`
//
//	// Signature values
//	Proof     []byte      `json:"proof" gencodec:"required"`
//}
//
//func NewFabricCross(amount, charge *big.Int, from, to string, origin, purpose uint8, id, txHash, bHash common.Hash,  input []byte) *FabricCross {
//	return &FabricCross{
//		Data: fcdata{
//			Value:     amount,
//			CTxId:     id,
//			TxHash:    txHash,
//			From:      from,
//			To:        to,
//			BlockHash: bHash,
//			Origin:    origin,
//			Purpose:   purpose,
//			Charge:    charge,
//			Payload:   input,
//		}}
//}
//
//func (tx *FabricCross) WithSignature(sig []byte) (*FabricCross, error) {
//	cpy := &FabricCross{Data: tx.Data}
//	cpy.Data.Proof = sig
//	return cpy, nil
//}
//
//func (tx *FabricCross) ID() common.Hash {
//	return tx.Data.CTxId
//}
//
//func (tx *FabricCross) Origin() uint8 {
//	return tx.Data.Origin
//}
//
//func (tx *FabricCross) Destination() uint8 {
//	return tx.Data.Purpose
//}
//
//func (tx *FabricCross) Hash() (h common.Hash) {
//	if hash := tx.hash.Load(); hash != nil {
//		return hash.(common.Hash)
//	}
//	hash := sha3.NewKeccak256()
//	var b []byte
//
//	b = append(b, tx.Data.CTxId.Bytes()...)
//	b = append(b, tx.Data.TxHash.Bytes()...)
//	b = append(b, tx.Data.BlockHash.Bytes()...)
//	b = append(b, common.LeftPadBytes(tx.Data.Value.Bytes(), 32)...)
//	b = append(b, common.LeftPadBytes(tx.Data.Charge.Bytes(), 32)...)
//	b = append(b, tx.Data.From...)
//	b = append(b, tx.Data.To...)
//	b = append(b, tx.Data.Origin)
//	b = append(b, tx.Data.Purpose)
//	b = append(b, tx.Data.Payload...)
//	hash.Write(b)
//	hash.Sum(h[:0])
//	tx.hash.RemoteStore(h)
//	return h
//}
//
//func (tx *FabricCross) BlockHash() common.Hash {
//	return tx.Data.BlockHash
//}
//
//func (tx *FabricCross) From() string {
//	return tx.Data.From
//}
//
//// Transactions is a Transaction slice type for basic sorting.
//type FabricCrosses []*FabricCross
//
//// Len returns the length of s.
//func (s FabricCrosses) Len() int { return len(s) }
//
//// Swap swaps the i'th and the j'th element in s.
//func (s FabricCrosses) Swap(i, j int) { s[i], s[j] = s[j], s[i] }
//
//// GetRlp implements Rlpable and returns the i'th element of s in rlp.
//func (s FabricCrosses) GetRlp(i int) []byte {
//	enc, _ := rlp.EncodeToBytes(s[i])
//	return enc
//}
//
//// TxByPrice implements both the sort and the heap interface, making it useful
//// for all at once sorting as well as individually adding and removing elements.
//type FcsByPrice FabricCrosses
//
//func (s FcsByPrice) Len() int { return len(s) }
//func (s FcsByPrice) Less(i, j int) bool {
//	return s[i].Data.Charge.Cmp(s[j].Data.Charge) > 0
//}
//func (s FcsByPrice) Swap(i, j int) { s[i], s[j] = s[j], s[i] }
//
//func (s *FcsByPrice) Push(x interface{}) {
//	*s = append(*s, x.(*FabricCross))
//}
//
//func (s *FcsByPrice) Pop() interface{} {
//	old := *s
//	n := len(old)
//	x := old[n-1]
//	*s = old[0 : n-1]
//	return x
//}

