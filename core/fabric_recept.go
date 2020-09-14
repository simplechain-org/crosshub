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
//	"sync/atomic"
//
//	"github.com/simplechain-org/go-simplechain/common"
//	"github.com/simplechain-org/go-simplechain/crypto/sha3"
//)
//
//
////type ReceptTransaction struct {
////	CTxId       common.Hash `json:"ctxId" gencodec:"required"`         //cross_transaction ID
////	TxHash      common.Hash `json:"txHash" gencodec:"required"`        //taker txHash
////	From        interface{} `json:"from" gencodec:"required"`          //Token seller
////	To          interface{} `json:"to" gencodec:"required"`            //Token buyer
////	Origin     *big.Int    `json:"chainId" gencodec:"required"`
////	Purpose *big.Int    `json:"destinationId" gencodec:"required"` //Message destination networkId
////	Payload     []byte      `json:"Payload" gencodec:"required"`
////	V           *big.Int    `json:"v" gencodec:"required"` //chainId
////	R           *big.Int    `json:"r" gencodec:"required"`
////	S           *big.Int    `json:"s" gencodec:"required"`
////}
//type FabricRecept struct {
//	Data frdata
//	// caches
//	hash     atomic.Value
//	signHash atomic.Value
//	size     atomic.Value
//	from     atomic.Value
//}
//
//type frdata struct {
//	CTxId   common.Hash `json:"ctxId" gencodec:"required"`         //cross_transaction ID
//	TxHash  common.Hash `json:"txHash" gencodec:"required"`        //taker txHash
//	From    string      `json:"from" gencodec:"required"`          //Token seller
//	To      string      `json:"to" gencodec:"required"`            //Token buyer
//	taker   string      `json:"taker" gencodec:"required"`         //Token buyer address
//	Origin  uint8       `json:"origin" gencodec:"required"`
//	Purpose uint8       `json:"purpose" gencodec:"required"` //Message destination networkId
//	Payload []byte      `json:"Payload" gencodec:"required"`
//	// Signature values
//	Proof     []byte      `json:"proof" gencodec:"required"`
//}
//
//func NewFabricRecept(id, txHash common.Hash, from, to string, origin, purpose uint8,input []byte) *ReceptTransaction {
//	return &ReceptTransaction{
//		Data: rtxdata{
//			CTxId:   id,
//			TxHash:  txHash,
//			From:    from,
//			To:      to,
//			Origin:  origin,
//			Purpose: purpose,
//			Payload: input,
//		}}
//}
//
//func (tx *FabricRecept) WithSignature(sig []byte) (*FabricRecept, error) {
//	cpy := &FabricRecept{Data: tx.Data}
//	cpy.Data.Proof = sig
//	return cpy, nil
//}
//
//func (tx *FabricRecept) ID() common.Hash {
//	return tx.Data.CTxId
//}
//
////func (tx *FabricRecept) ChainId() *big.Int {
////	return types.DeriveChainId(tx.Data.V)
////}
//
//func (tx *FabricRecept) Destination() uint8 {
//	return tx.Data.Purpose
//}
//
//func (tx *FabricRecept) Hash() (h common.Hash) {
//	if hash := tx.hash.Load(); hash != nil {
//		return hash.(common.Hash)
//	}
//	hash := sha3.NewKeccak256()
//	var b []byte
//	b = append(b, tx.Data.CTxId.Bytes()...)
//	b = append(b, tx.Data.TxHash.Bytes()...)
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
////func (tx *ReceptTransaction) BlockHash() common.Hash {
////	return tx.Data.BlockHash
////}
//
//func (tx *FabricRecept) From() interface{} {
//	return tx.Data.From
//}

//func (tx *FabricRecept) SignHash() (h common.Hash) {
//	if hash := tx.signHash.Load(); hash != nil {
//		return hash.(common.Hash)
//	}
//	hash := sha3.NewKeccak256()
//	var b []byte
//	b = append(b, tx.Data.CTxId.Bytes()...)
//	b = append(b, tx.Data.TxHash.Bytes()...)
//	b = append(b, tx.Data.From...)
//	b = append(b, tx.Data.To...)
//	b = append(b, tx.Data.Origin)
//	b = append(b, tx.Data.Purpose)
//	b = append(b, tx.Data.Payload...)
//	b = append(b, common.LeftPadBytes(tx.Data.V.Bytes(), 32)...)
//	b = append(b, common.LeftPadBytes(tx.Data.R.Bytes(), 32)...)
//	b = append(b, common.LeftPadBytes(tx.Data.S.Bytes(), 32)...)
//	hash.Write(b)
//	hash.Sum(h[:0])
//	tx.signHash.RemoteStore(h)
//	return h
//}

//func (rtx ReceptTransaction) Check(maker *CrossTransactionWithSignatures) error {
//	if maker == nil {
//		return ErrInvalidRecept
//	}
//	if maker.DestinationId().Cmp(rtx.Origin) != 0 {
//		return ErrChainIdMissMatch
//	}
//	if maker.Data.From != rtx.From {
//		return ErrFromMissMatch
//	}
//	if maker.Data.To != (common.Address{}) && maker.Data.To != rtx.To {
//		return ErrToMissMatch
//	}
//	return nil
//}

//type Recept struct {
//	TxId   common.Hash
//	TxHash common.Hash
//	From   common.Address
//	To     common.Address
//	//Payload  []byte //TODO delete
//}
//
//func (rtx *ReceptTransaction) ConstructData(crossContract abi.ABI) ([]byte, error) {
//	rep := Recept{
//		TxId:   rtx.CTxId,
//		TxHash: rtx.TxHash,
//		From:   rtx.From,
//		To:     rtx.To,
//	}
//	return crossContract.Pack("makerFinish", rep, rtx.Origin)
//}
