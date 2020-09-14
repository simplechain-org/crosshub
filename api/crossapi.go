package api

import (
	"github.com/simplechain-org/crosshub/core"
	db "github.com/simplechain-org/crosshub/database"
	"github.com/simplechain-org/go-simplechain/common"
	"github.com/simplechain-org/go-simplechain/common/hexutil"
)

type RPCCrossTransaction struct {
	CTxId     common.Hash 	`json:"ctxId"`
	TxHash    common.Hash 	`json:"txHash"`
	BlockHash common.Hash 	`json:"blockHash"`
	Value     *hexutil.Big	`json:"value"`
	Charge    *hexutil.Big	`json:"charge"`
	From      string      	`json:"from"`
	To        string      	`json:"to"`
	Origin    hexutil.Uint  `json:"origin"`
	Purpose   hexutil.Uint  `json:"purpose"`
	Payload   hexutil.Bytes	`json:"payload"`

	// Signature values
	V *hexutil.Big 		`json:"v"`
	R *hexutil.Big 		`json:"r"`
	S *hexutil.Big 		`json:"s"`
}

// newRPCCrossTransaction returns a transaction that will serialize to the RPC
// representation, with the given location metadata set (if available).
func newRPCCrossTransaction(tx *core.CrossTransaction) *RPCCrossTransaction {
	if tx == nil {
		return nil
	}
	result := &RPCCrossTransaction{
		CTxId:            tx.ID(),
		TxHash:           tx.Data.TxHash,
		BlockHash:        tx.Data.BlockHash,
		Value:            (*hexutil.Big)(tx.Data.Value),
		Charge:           (*hexutil.Big)(tx.Data.Charge),
		From:             tx.Data.From,
		To:               tx.Data.To,
		Origin:           hexutil.Uint(tx.Data.Origin),
		Purpose:          hexutil.Uint(tx.Data.Purpose),
		Payload:          tx.Data.Payload,

		V:    			  (*hexutil.Big)(tx.Data.V),
		R: 				  (*hexutil.Big)(tx.Data.R),
		S:                (*hexutil.Big)(tx.Data.S),
	}

	return result
}

type RPCPageCrossTransactions struct {
	Data map[uint8][]*RPCCrossTransaction `json:"data"`
	//Total int                               `json:"total"`
}

type CrossApi interface {
	CtxContentByPage(int, int, int, int) map[string]RPCPageCrossTransactions
	//CtxQuery(hash common.Hash) *RPCCrossTransaction
	//CtxQueryDestValue(value *hexutil.Big, pageSize, startPage int) *RPCPageCrossTransactions
	//CtxOwner(from common.Address) map[string]map[uint8][]*RPCCrossTransaction
	//CtxOwnerByPage(from common.Address, pageSize, startPage int) RPCPageCrossTransactions
	//CtxTakerByPage(to common.Address, pageSize, startPage int) RPCPageCrossTransactions
}

type CrossQueryApi struct {
	remoteDb *db.IndexDB
	localDb  *db.IndexDB
}

func NewPublicCrossQueryApi(db,idb *db.IndexDB) *CrossQueryApi {
	return &CrossQueryApi{remoteDb: db,localDb: idb}
}

func (s *CrossQueryApi) CtxContentByPage(localSize, localPage, remoteSize, remotePage int) map[string]RPCPageCrossTransactions {
	locals, remotes := s.QueryByPage(localSize, localPage, remoteSize, remotePage)
	content := map[string]RPCPageCrossTransactions{
		"local": {
			Data: make(map[uint8][]*RPCCrossTransaction),
			//Total: localTotal,
		},
		"remote": {
			Data: make(map[uint8][]*RPCCrossTransaction),
			//Total: remoteTotal,
		},
	}
	for s, txs := range locals {
		for _, tx := range txs {
			content["local"].Data[s] = append(content["local"].Data[s], newRPCCrossTransaction(tx))
		}
	}
	for k, txs := range remotes {
		for _, tx := range txs {
			content["remote"].Data[k] = append(content["remote"].Data[k], newRPCCrossTransaction(tx))
		}
	}
	return content
}





