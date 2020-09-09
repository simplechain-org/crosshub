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
	"errors"
	"fmt"
	"github.com/simplechain-org/crosshub/core"
	"math/big"

	"github.com/simplechain-org/go-simplechain/common"
	"github.com/simplechain-org/go-simplechain/log"

	"github.com/asdine/storm/v3"
	"github.com/asdine/storm/v3/q"
)

type IndexDB struct {
	chainID *big.Int
	root    *storm.DB // root db of stormDB
	db      storm.Node
	cache   *IndexDbCache
	logger  log.Logger
}

type FieldName = string

const (
	PK               FieldName = "PK"
	CtxIdIndex       FieldName = "CtxId"
	TxHashIndex      FieldName = "TxHash"
	PriceIndex       FieldName = "Price"
	StatusField      FieldName = "Status"
	FromField        FieldName = "From"
	ToField          FieldName = "To"
	DestinationValue FieldName = "Charge"
	//BlockNumField    FieldName = "BlockNum"
)

func NewIndexDB(chainID *big.Int, path string, cacheSize uint64) *IndexDB {
	dbName := "chain" + chainID.String()
	log.Info("Open IndexDB", "dbName", dbName, "cacheSize", cacheSize)
	rootDB,err := storm.Open(path)
	if err != nil {
		log.Error("NewIndexDB","err",err)
	}
	return &IndexDB{
		chainID: chainID,
		db:      rootDB.From(dbName).WithBatch(true),
		cache:   newIndexDbCache(int(cacheSize)),
		logger:  log.New("name", dbName),
	}
}

func (d *IndexDB) ChainID() *big.Int {
	return d.chainID
}

func (d *IndexDB) Count(filter ...q.Matcher) int {
	count, _ := d.db.Select(filter...).Count(&CrossTransactionIndexed{})
	return count
}

func (d *IndexDB) Load() error {
	return nil
}

//func (d *IndexDB) Height() uint64 {
//	var ctxs []*CrossTransactionIndexed
//	if err := d.db.AllByIndex(BlockNumField, &ctxs, storm.Limit(1), storm.Reverse()); err != nil || len(ctxs) == 0 {
//		return 0
//	}
//	return ctxs[0].BlockNum
//}

func (d *IndexDB) Repair() error {
	return d.db.ReIndex(&CrossTransactionIndexed{})
}

func (d *IndexDB) Clean() error {
	return d.db.Drop(&CrossTransactionIndexed{})
}

func (d *IndexDB) Close() error {
	return d.db.Commit()
}

func (d *IndexDB) Write(ctx *core.CrossTransaction) error {
	if err := d.Writes([]*core.CrossTransaction{ctx}, true); err != nil {
		return err
	}
	if d.cache != nil {
		d.cache.Put(CtxIdIndex, ctx.ID(), NewCrossTransactionIndexed(ctx))
	}
	return nil
}

func (d *IndexDB) Writes(ctxList []*core.CrossTransaction, replaceable bool) (err error) {
	d.logger.Debug("write cross transaction", "count", len(ctxList), "replaceable", replaceable)
	tx, err := d.db.Begin(true)
	if err != nil {
		return ErrCtxDbFailure{"begin transaction failed", err}
	}
	defer tx.Rollback()

	canReplace := func(old, new *CrossTransactionIndexed) bool {
		if !replaceable {
			return false
		}
		//if new.Status == uint8(cc.CtxStatusPending) {
		//	return false
		//}
		//if new.BlockNum < old.BlockNum {
		//	return false
		//}
		//if new.Status <= old.Status { //TODO:无法解决同步其他节点时，其他节点回滚的状态
		//	return false
		//}
		return true
	}

	for _, ctx := range ctxList {
		//if d.txLog.IsFinish(ctx.ID()) {
		//	continue
		//}
		new := NewCrossTransactionIndexed(ctx)
		var old CrossTransactionIndexed
		err = tx.One(CtxIdIndex, ctx.ID(), &old)
		if err == storm.ErrNotFound {
			//d.logger.Trace("add new cross transaction",
			//	"id", ctx.ID().String(), "status", ctx.Status.String(), "number", ctx.BlockNum)

			if err = tx.Save(new); err != nil {
				return err
			}

		} else if canReplace(&old, new) {
			//d.logger.Trace("replace cross transaction", "id", ctx.ID().String(),
			//	"old_status", cc.CtxStatus(old.Status).String(), "new_status", ctx.Status.String(),
			//	"old_height", old.BlockNum, "new_height", ctx.BlockNum)

			new.PK = old.PK
			if err = tx.Update(new); err != nil {
				return err
			}

		} else {
			//d.logger.Trace("can't add or replace cross transaction", "id", ctx.ID().String(),
			//	"old_status", cc.CtxStatus(old.Status).String(), "new_status", ctx.Status.String(),
			//	"old_height", old.BlockNum, "new_height", ctx.BlockNum, "replaceable", replaceable)

			continue
		}

		if d.cache != nil {
			d.cache.Remove(CtxIdIndex, ctx.ID())
			d.cache.Remove(CtxIdIndex, ctx.Data.TxHash)
		}
	}

	return tx.Commit()
}

func (d *IndexDB) Read(ctxId common.Hash) (*core.CrossTransaction, error) {
	ctx, err := d.get(ctxId)
	if err != nil {
		return nil, err
	}
	return ctx.ToCrossTransaction(), nil
}

func (d *IndexDB) One(field FieldName, key interface{}) *core.CrossTransaction {
	if d.cache != nil {
		ctx := d.cache.Get(field, key)
		if ctx != nil {
			return ctx.ToCrossTransaction()
		}
	}
	var ctx CrossTransactionIndexed
	if err := d.db.One(field, key, &ctx); err != nil {
		return nil
	}
	if d.cache != nil {
		d.cache.Put(field, key, &ctx)
	}
	return ctx.ToCrossTransaction()
}

func (d *IndexDB) get(ctxId common.Hash) (*CrossTransactionIndexed, error) {
	if d.cache != nil {
		ctx := d.cache.Get(CtxIdIndex, ctxId)
		if ctx != nil {
			return ctx, nil
		}
	}

	var ctx CrossTransactionIndexed
	if err := d.db.One(CtxIdIndex, ctxId, &ctx); err != nil {
		return nil, ErrCtxDbFailure{fmt.Sprintf("get ctx:%s failed", ctxId.String()), err}
	}

	if d.cache != nil {
		d.cache.Put(CtxIdIndex, ctxId, &ctx)
	}

	return &ctx, nil
}

func (d *IndexDB) Update(id common.Hash, updater func(ctx *CrossTransactionIndexed)) error {
	return d.Updates([]common.Hash{id}, []func(ctx *CrossTransactionIndexed){updater})
}

func (d *IndexDB) Updates(idList []common.Hash, updaters []func(ctx *CrossTransactionIndexed)) (err error) {
	if len(idList) != len(updaters) {
		return ErrCtxDbFailure{err: errors.New("invalid updates params")}
	}
	tx, err := d.db.Begin(true)
	if err != nil {
		return ErrCtxDbFailure{"begin transaction failed", err}
	}
	defer tx.Rollback()

	for i, id := range idList {
		var ctx CrossTransactionIndexed
		if err = tx.One(CtxIdIndex, id, &ctx); err != nil {
			return ErrCtxDbFailure{"transaction want to be updated is not exist", err}
		}
		updaters[i](&ctx)
		if err = tx.Update(&ctx); err != nil {
			return ErrCtxDbFailure{"transaction update failed", err}
		}
		if d.cache != nil {
			d.cache.Remove(CtxIdIndex, id)
			d.cache.Remove(TxHashIndex, ctx.TxHash)
		}
	}
	return tx.Commit()
}

func (d *IndexDB) Deletes(idList []common.Hash) (err error) {
	tx, err := d.db.Begin(true)
	if err != nil {
		return ErrCtxDbFailure{"begin transaction failed", err}
	}
	defer tx.Rollback()
	for _, id := range idList {
		var ctx CrossTransactionIndexed
		if err = tx.One(CtxIdIndex, id, &ctx); err != nil {
			continue
		}
		if d.cache != nil {
			d.cache.Remove(CtxIdIndex, id)
			d.cache.Remove(TxHashIndex, ctx.TxHash)
		}
		if err = tx.DeleteStruct(&ctx); err != nil {
			return ErrCtxDbFailure{"transaction delete failed", err}
		}
	}

	return tx.Commit()
}

func (d *IndexDB) Has(id common.Hash) bool {
	_, err := d.get(id)
	return err == nil
}

func (d *IndexDB) Query(pageSize int, startPage int, orderBy []FieldName, reverse bool, filter ...q.Matcher) []*core.CrossTransaction {
	if pageSize > 0 && startPage <= 0 {
		return nil
	}
	var ctxs []*CrossTransactionIndexed
	query := d.db.Select(filter...)
	if len(orderBy) > 0 {
		query.OrderBy(orderBy...)
	}
	if reverse {
		query.Reverse()
	}
	if pageSize > 0 {
		query.Limit(pageSize).Skip(pageSize * (startPage - 1))
	}
	query.Find(&ctxs)

	results := make([]*core.CrossTransaction, len(ctxs))
	for i, ctx := range ctxs {
		results[i] = ctx.ToCrossTransaction()
	}
	return results
}

//func (d *IndexDB) RangeByNumber(begin, end uint64, limit int) []*cc.CrossTransactionWithSignatures {
//	var (
//		ctxs    []*CrossTransactionIndexed
//		options []func(*index.Options)
//	)
//	if limit > 0 {
//		options = append(options, storm.Limit(limit))
//	}
//	d.db.Range(BlockNumField, begin, end, &ctxs, options...)
//	if ctxs == nil {
//		return nil
//	}
//	//把最后一笔ctx所在高度的所有ctx取出来
//	var lasts []*CrossTransactionIndexed
//	d.db.Find(BlockNumField, ctxs[len(ctxs)-1].BlockNum, &lasts)
//	for i, tx := range ctxs {
//		if tx.BlockNum == ctxs[len(ctxs)-1].BlockNum {
//			ctxs = ctxs[:i]
//			break
//		}
//	}
//	ctxs = append(ctxs, lasts...)
//
//	results := make([]*cc.CrossTransactionWithSignatures, len(ctxs))
//	for i, ctx := range ctxs {
//		results[i] = ctx.ToCrossTransaction()
//	}
//	return results
//}
