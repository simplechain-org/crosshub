package courier

import (
	"errors"
	"fmt"
	"math/big"
	"sync"

	"github.com/simplechain-org/crosshub/core"
	"github.com/simplechain-org/crosshub/fabric/courier/client"
	"github.com/simplechain-org/crosshub/fabric/courier/contractlib"
	"github.com/simplechain-org/crosshub/fabric/courier/utils"
	"github.com/simplechain-org/crosshub/fabric/courier/utils/prque"

	"github.com/asdine/storm/v3"
	"github.com/asdine/storm/v3/q"
	"github.com/simplechain-org/go-simplechain/common"
	"github.com/simplechain-org/go-simplechain/crypto"
	"github.com/simplechain-org/go-simplechain/crypto/ecdsa"
)

type Prqueue struct {
	prq     *prque.Prque
	process chan struct{}
	mu      sync.Mutex
}

type TxManager struct {
	DB
	oClient client.OutChainClient
	fClient client.FabricClient

	wg     sync.WaitGroup
	stopCh chan struct{}

	pending  Prqueue
	executed Prqueue

	//p2p client private key
	privateKey *ecdsa.PrivateKey
	//if true, handle cross transaction from outchain, default not handle
	outchain bool
}

func NewTxManager(fabCli client.FabricClient, outCli client.OutChainClient, db DB) *TxManager {
	return &TxManager{
		DB:       db,
		stopCh:   make(chan struct{}),
		pending:  Prqueue{prq: prque.New(nil), process: make(chan struct{}, 4)},
		executed: Prqueue{prq: prque.New(nil), process: make(chan struct{}, 8)},
		oClient:  outCli,
		fClient:  fabCli,
	}
}

func (t *TxManager) Start() {
	utils.Logger.Info("[courier.TxManager] starting")
	t.wg.Add(3)
	go t.ProcessCrossTxs()
	go t.ProcessCrossTxReceipts()
	go t.receive()

	t.reload()
	utils.Logger.Info("[courier.TxManager] started")
}

func (t *TxManager) Stop() {
	utils.Logger.Info("[courier.TxManager] stopping")
	close(t.stopCh)
	t.wg.Wait()

	t.fClient.Close()
	utils.Logger.Info("[courier.TxManager] fClient closed")

	t.oClient.Close()
	utils.Logger.Info("[courier.TxManager] oClient closed")

	utils.Logger.Info("[courier.TxManager] stopped")
}

func (t *TxManager) reload() {
	utils.Logger.Debug("[courier.TxManager] reloading")
	toPending := t.DB.Query(0, 0, []FieldName{TimestampField}, false, q.Eq(StatusField, contractlib.Init))
	t.pending.mu.Lock()
	for _, tx := range toPending {
		t.pending.prq.Push(tx, -tx.TimeStamp.Seconds)
	}
	t.pending.mu.Unlock()

	t.pending.process <- struct{}{}

	fromExecuted := t.DB.Query(0, 0, []FieldName{TimestampField}, false, q.Eq(StatusField, contractlib.Executed))
	t.executed.mu.Lock()
	for _, tx := range fromExecuted {
		t.executed.prq.Push(tx, -tx.TimeStamp.Seconds)
	}
	t.executed.mu.Unlock()

	utils.Logger.Debug("[courier.TxManager] reload completed")
}

func (t *TxManager) AddCrossTxs(txs []*CrossTx) error {
	// split the Init, Finished and OutOnceCompleted txs
	var storeTxs, outTxs []*CrossTx

	t.pending.mu.Lock()
	for _, tx := range txs {
		switch {
		case tx.Contract.IsFinished():
			storeTxs = append(storeTxs, tx)
		case tx.Contract.IsOutOnceCompleted():
			outTxs = append(outTxs, tx)
		default:
			storeTxs = append(storeTxs, tx)
			t.pending.prq.Push(tx, -tx.TimeStamp.Seconds)
		}
	}
	t.pending.mu.Unlock()

	for _, tx := range outTxs {
		t.wg.Add(1)
		go func() {
			defer t.wg.Done()

			t.processOutChainCtxResp(tx.CrossID, tx.TxID)
		}()
	}

	// store to db
	if err := t.DB.Save(storeTxs); err != nil {
		return err
	}

	// start send
	if t.pending.prq.Size() != 0 {
		t.pending.process <- struct{}{}
	}

	return nil
}

func (t *TxManager) ProcessCrossTxs() {
	defer func() {
		t.wg.Done()
		utils.Logger.Info("[courier.TxManager] process crossTx stopped")
	}()

	utils.Logger.Info("[courier.TxManager] process crossTx started")
	for {
		select {
		case <-t.pending.process:
			var pending = make([]*CrossTx, 0)

			t.pending.mu.Lock()
			for !t.pending.prq.Empty() {
				item, _ := t.pending.prq.Pop()
				tx := item.(*CrossTx)
				pending = append(pending, tx)
			}
			t.pending.mu.Unlock()

			successList := make([]string, 0)
			updaters := make([]func(c *CrossTx), 0)

			for _, tx := range pending {
				ctx := toCrossHubTx(tx)
				if ctx == nil {
					continue
				}

				signed, err := t.signTx(ctx)
				if err != nil {
					utils.Logger.Error("[courier.TxManager] signed ctx", "crossID", tx.CrossID, "status", tx.GetStatus(), "err", err)
					t.pending.prq.Push(tx, -tx.TimeStamp.Seconds)
					continue
				}

				if err := t.oClient.Send(signed.(*core.CrossTransaction)); err != nil {
					utils.Logger.Error("[courier.TxManager] send tx to OutChain", "crossID", tx.CrossID, "status", tx.GetStatus(), "err", err)
					t.pending.prq.Push(tx, -tx.TimeStamp.Seconds)
					continue
				}

				successList = append(successList, tx.CrossID)
				updaters = append(updaters, func(c *CrossTx) {
					c.UpdateStatus(contractlib.Pending)
				})
			}

			go func() {
				if err := t.DB.Updates(successList, updaters); err != nil {
					utils.Logger.Debug("[courier.TxManager] update Init to Pending", "len(successList)", len(successList), "err", err)
					panic(err)
				}
			}()

			utils.Logger.Info("[courier.TxManager] update Init to Pending", "len(successList)", len(successList))

		case <-t.stopCh:
			return
		}
	}
}

func (t *TxManager) UpdateCrossTx(ctrs []CrossTxReceipt) error {
	var updaters []func(c *CrossTx)
	var ids []string

	for _, ctr := range ctrs {
		ids = append(ids, ctr.CrossID)
		updaters = append(updaters, func(c *CrossTx) {
			c.UpdateStatus(contractlib.Executed)
			pc, ok := c.IContract.(*contractlib.PrecommitContract)
			if ok {
				pc.UpdateReceipt(ctr.Receipt)
			}
		})
	}

	utils.Logger.Debug("[courier.TxManager] handle receipt", "ids", ids)

	return t.DB.Updates(ids, updaters)
}

func (t *TxManager) ProcessCrossTxReceipts() {
	defer func() {
		t.wg.Done()
		utils.Logger.Info("[courier.TxManager] process crossTx receipts stopped")
	}()

	utils.Logger.Info("[courier.TxManager] process crossTx receipts started")
	for {
		select {
		case <-t.executed.process:
			var executed = make([]CrossTxReceipt, 0)

			t.executed.mu.Lock()
			for !t.executed.prq.Empty() {
				item, _ := t.executed.prq.Pop()
				req := item.(CrossTxReceipt)
				executed = append(executed, req)
			}
			t.executed.mu.Unlock()

			if err := t.UpdateCrossTx(executed); err != nil {
				if errors.Is(err, storm.ErrNotFound) {
					utils.Logger.Info("[courier.TxManager] discard receipts", "receipts", executed)
					break
				}

				utils.Logger.Warn("[courier.TxManager] handle receipt", "err", err)

				for _, ctr := range executed {
					t.executed.prq.Push(ctr, -ctr.Sequence)
				}
				break
			}

			utils.Logger.Info("[courier.TxManager] update Pending to Executed", "len(successList)", len(executed))

			t.wg.Add(1)
			go func() {
				t.wg.Done()

				for _, ctr := range executed {
					_, err := t.fClient.InvokeChainCode("commit", []string{ctr.CrossID, ctr.Receipt})
					if err != nil {
						utils.Logger.Error("[courier.TxManager] send tx to fabric", "InvokeChainCode err", err)
					}

					t.executed.prq.Push(ctr, -ctr.Sequence)
				}
			}()

		case <-t.stopCh:
			return
		}
	}
}

func (t *TxManager) signTx(ctx interface{}) (interface{}, error) {
	switch ctx.(type) {
	case *core.CrossTransaction:
		return core.SignCtx(ctx.(*core.CrossTransaction), core.MakeCtxSigner(big.NewInt(11)), func(hash []byte) ([]byte, error) {
			return crypto.Sign(hash, t.privateKey.K)
		})
	case *core.ReceptTransaction:
		return core.SignRtx(ctx.(*core.ReceptTransaction), core.MakeRtxSigner(big.NewInt(11)), func(hash []byte) ([]byte, error) {
			return crypto.Sign(hash, t.privateKey.K)
		})
	default:
		return nil, fmt.Errorf("[courire.TxManager] signTx unsupported type transaction")
	}
}

func (t *TxManager) receive() {
	t.wg.Add(1)

	defer func() {
		if r := recover(); r != nil {
			utils.Logger.Error("[courier.CrossChannel] receive ", "panic", r)
		}

		t.wg.Done()
		utils.Logger.Info("[courier.TxManager] receive stopped")
	}()

	utils.Logger.Info("[courier.TxManager] receive started")

	for {
		select {
		case recv := <-t.oClient.Recv():
			utils.Logger.Debug("[courier.TxManager] receive", "receipt", recv)

			switch recv.(type) {
			case *core.ReceptTransaction:
				t.AddCrossTxReceipt(CrossTxReceipt{
					CrossID: recv.(*core.ReceptTransaction).ID().String()[2:], // ignore '0x' prefix
					Receipt: recv.(*core.ReceptTransaction).Data.TxHash.String()[2:],
				})
			case *core.CrossTransaction:
				if !t.outchain {
					break
				}

				req := recv.(*core.CrossTransaction)

				t.wg.Add(1)
				go func() {
					defer t.wg.Done()
					req.Data.To = testFabricAccount
					t.processOutChainCtxReq(req)
				}()
			default:
				utils.Logger.Warn("[courier.TxManager] discard received unsupported type transaction")
			}

		case <-t.stopCh:
			return
		}
	}
}

func (t *TxManager) AddCrossTxReceipt(ctr CrossTxReceipt) {
	t.wg.Add(1)
	go func() {
		defer t.wg.Done()

		t.executed.mu.Lock()
		t.executed.prq.Push(ctr, -ctr.Sequence)
		t.executed.mu.Unlock()

		select {
		case t.executed.process <- struct{}{}:
		case <-t.stopCh:
			return
		}
	}()
}

func (t *TxManager) processOutChainCtxReq(req *core.CrossTransaction) {
	// 1. create new ReceptTransaction and store to db

	var pendingReceipt = core.NewReceptTransaction(
		req.Data.CTxId,
		common.Hash{},
		req.Data.From,
		req.Data.To,
		testSimpleChainAddress,
		Fabric,
		SimpleChain,
		req.Data.Payload,
	)

	if err := t.DB.Set("outchain", pendingReceipt.ID().String(), pendingReceipt); err != nil {
		utils.Logger.Warn("[courier.TxManager] store outchain request", "err", err)
	}

	utils.Logger.Debug("[courier.TxManager] processOutChainCtxReq",
		"crossID", req.ID().String(), "from", req.Data.From, "to", req.Data.To, "charge", req.Data.Charge.String())

	// 2. parse and send to fabric
	_, err := t.fClient.InvokeChainCode("commit", []string{testChainCodePrefix, req.ID().String(), testFabricinvoke, req.Data.To, req.Data.From, req.Data.Charge.String()})
	if err != nil {
		utils.Logger.Error("[courier.TxManager] send processOutChainCtxReq to fabric", "err", err)
	}
	//TODO 并发err
}

func (t *TxManager) processOutChainCtxResp(crossID string, receipt string) {
	// 1. update
	var pendingReceipt = new(core.ReceptTransaction)
	t.DB.Get("outchain", crossID, pendingReceipt)
	pendingReceipt.Data.TxHash = common.HexToHash(receipt)

	if err := t.DB.Set("outchain", crossID, pendingReceipt); err != nil {
		utils.Logger.Warn("[courier.TxManager] store outchain response", "err", err)
	}

	utils.Logger.Debug("[courier.TxManager] processOutChainCtxResp", "crossID", crossID, "receipt", receipt)

	// 2. send to outchain
	signCtx, err := t.signTx(pendingReceipt)
	if err != nil {
		utils.Logger.Warn("[courier.TxManager] sign processOutChainCtxResp", "err", err)
	}

	if err := t.oClient.Send(signCtx.(*core.ReceptTransaction)); err != nil {
		utils.Logger.Error("[courier.TxManager] send processOutChainCtxResp to outchain", "err", err)
	}

}
