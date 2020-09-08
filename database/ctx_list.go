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

package db

import (
	lru "github.com/hashicorp/golang-lru"
	"github.com/simplechain-org/crosshub/core"
	"github.com/simplechain-org/go-simplechain/common"
	"math/big"
)

//type CtxSortedByBlockNum struct {
//	items map[common.Hash]*cc.CrossTransactionWithSignatures
//	index *redblacktree.Tree
//	lock  sync.RWMutex
//}
//
//func NewCtxSortedMap() *CtxSortedByBlockNum {
//	return &CtxSortedByBlockNum{
//		items: make(map[common.Hash]*cc.CrossTransactionWithSignatures),
//		index: redblacktree.NewWith(container.UInt64Comparator, true),
//	}
//}
//
//func (m *CtxSortedByBlockNum) Get(txId common.Hash) *cc.CrossTransactionWithSignatures {
//	m.lock.RLock()
//	defer m.lock.RUnlock()
//	return m.items[txId]
//}
//
//func (m *CtxSortedByBlockNum) Put(ctx *cc.CrossTransactionWithSignatures) {
//	m.lock.Lock()
//	defer m.lock.Unlock()
//	id := ctx.ID()
//	if m.items[id] != nil {
//		return
//	}
//
//	m.items[id] = ctx
//	m.index.Put(ctx.BlockNum, ctx.ID())
//}
//
//func (m *CtxSortedByBlockNum) Len() int {
//	m.lock.RLock()
//	defer m.lock.RUnlock()
//	return len(m.items)
//}
//
//func (m *CtxSortedByBlockNum) RemoveByID(id common.Hash) {
//	m.lock.Lock()
//	defer m.lock.Unlock()
//	if tx, ok := m.items[id]; ok {
//		for itr := m.index.LowerBound(tx.BlockNum); itr != m.index.UpperBound(tx.BlockNum); itr.Next() {
//			if itr.Value().(common.Hash) == id {
//				m.index.RemoveOne(itr)
//				delete(m.items, id)
//				break
//			}
//		}
//	}
//}
//
//func (m *CtxSortedByBlockNum) RemoveUnderNum(num uint64) (removed cc.CtxIDs) {
//	m.lock.Lock()
//	defer m.lock.Unlock()
//	for itr := m.index.Begin(); itr.HasNext(); {
//		this := itr
//		itr.Next()
//		number, id := this.Key().(uint64), this.Value().(common.Hash)
//		if number > num {
//			break
//		}
//		m.index.RemoveOne(this)
//		delete(m.items, id)
//	}
//	return
//}
//
//// Do calls function f on each element of the map, in forward order.
//func (m *CtxSortedByBlockNum) Do(do func(*cc.CrossTransactionWithSignatures) bool) {
//	m.lock.RLock()
//	defer m.lock.RUnlock()
//	for itr := m.index.Begin(); !itr.IsEnd(); itr.Next() {
//		if ctx, ok := m.items[itr.Value().(common.Hash)]; ok && do(ctx) {
//			break
//		}
//	}
//}

type CrossTransactionIndexed struct {
	PK       	uint64         `storm:"id,increment"`
	CtxId    	common.Hash    `storm:"unique"`
	TxHash   	common.Hash    `storm:"index"`
	BlockHash 	common.Hash
	Value    	*big.Int
	Charge   	*big.Int		`storm:"index"`
	From     	string 			`storm:"index"`
	To       	string 			`storm:"index"`
	Origin      uint8
	Purpose     uint8
	Payload     []byte

	Price    *big.Float     `storm:"index"`
	//BlockNum uint64         `storm:"index"`
	// normal field
	//Status uint8 			`storm:"index"`

	V *big.Int
	R *big.Int
	S *big.Int
}

func NewCrossTransactionIndexed(ctx *core.CrossTransaction) *CrossTransactionIndexed {
	return &CrossTransactionIndexed{
		CtxId:            ctx.ID(),
		TxHash:           ctx.Data.TxHash,
		BlockHash:        ctx.Data.BlockHash,
		Value:            ctx.Data.Value,
		Charge:           ctx.Data.Charge,
		//Status:           uint8(ctx.Status),
		//BlockNum:         ctx.BlockNum,
		From:             ctx.Data.From,
		To:               ctx.Data.To,
		Origin:           ctx.Data.Origin,
		Purpose:          ctx.Data.Purpose,
		Payload:          ctx.Data.Payload,
		Price:            new(big.Float).SetRat(ctx.Price()),
		V:                ctx.Data.V,
		R:                ctx.Data.R,
		S:                ctx.Data.S,
	}

}

func (c CrossTransactionIndexed) ToCrossTransaction() *core.CrossTransaction {
	var cts core.CrossTransaction
	cts.Data.CTxId = c.CtxId
	cts.Data.TxHash = c.TxHash
	cts.Data.BlockHash = c.BlockHash
	cts.Data.Value = c.Value
	cts.Data.Charge = c.Charge
	cts.Data.From = c.From
	cts.Data.To = c.To
	cts.Data.Origin = c.Origin
	cts.Data.Purpose = c.Purpose
	cts.Data.Payload = c.Payload
	cts.Data.V = c.V
	cts.Data.R = c.R
	cts.Data.S = c.S

	return &cts
}

type IndexDbCache lru.ARCCache

func newIndexDbCache(cap int) *IndexDbCache {
	cache, err := lru.NewARC(cap)
	if cache == nil || err != nil {
		return nil
	}
	return (*IndexDbCache)(cache)
}

func indexCacheKey(index FieldName, value interface{}) interface{} {
	return struct {
		Index FieldName
		Value interface{}
	}{index, value}
}

func (m *IndexDbCache) Get(index FieldName, key interface{}) *CrossTransactionIndexed {
	item, ok := (*lru.ARCCache)(m).Get(indexCacheKey(index, key))
	if !ok {
		return nil
	}
	return item.(*CrossTransactionIndexed)
}

func (m *IndexDbCache) Put(index FieldName, key interface{}, ctx *CrossTransactionIndexed) {
	(*lru.ARCCache)(m).Add(indexCacheKey(index, key), ctx)
}

func (m *IndexDbCache) Remove(index FieldName, key interface{}) {
	(*lru.ARCCache)(m).Remove(indexCacheKey(index, key))
}
