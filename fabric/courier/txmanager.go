package courier

import (
	"encoding/json"
	"errors"
	"sync"

	"github.com/simplechain-org/crosshub/fabric/courier/client"
	"github.com/simplechain-org/crosshub/fabric/courier/contractlib"
	"github.com/simplechain-org/crosshub/fabric/courier/utils"
	"github.com/simplechain-org/crosshub/fabric/courier/utils/prque"

	"github.com/asdine/storm/v3"
	"github.com/asdine/storm/v3/q"
	"github.com/golang/protobuf/ptypes/timestamp"
)

type CrossTx struct {
	contractlib.Contract
	PK          int64                `storm:"id,increment"`
	CrossID     string               `storm:"unique"`
	TxID        string               `storm:"index"`
	BlockNumber uint64               `storm:"index"`
	TimeStamp   *timestamp.Timestamp `storm:"index"`
}

func (c *CrossTx) UnmarshalJSON(bytes []byte) (err error) {
	var errList []error

	var objMap map[string]*json.RawMessage
	errList = append(errList, json.Unmarshal(bytes, &objMap))
	errList = append(errList, json.Unmarshal(*objMap["PK"], &c.PK))
	errList = append(errList, json.Unmarshal(*objMap["CrossID"], &c.CrossID))
	errList = append(errList, json.Unmarshal(*objMap["TxID"], &c.TxID))
	errList = append(errList, json.Unmarshal(*objMap["BlockNumber"], &c.BlockNumber))
	errList = append(errList, json.Unmarshal(*objMap["TimeStamp"], &c.TimeStamp))

	c.IContract, err = contractlib.RebuildIContract(*objMap["IContract"])
	errList = append(errList, err)

	for _, err := range errList {
		if err != nil {
			return err
		}
	}

	return nil
}

type CrossTxReceipt struct {
	CrossID  string
	Receipt  string
	Sequence int64
}

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
	t.wg.Add(2)
	go t.ProcessCrossTxs()
	go t.ProcessCrossTxReceipts()

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

	utils.Logger.Debug("[courier.TxManager] reload completed")
	//TODO: executed queue
}

func (t *TxManager) AddCrossTxs(txs []*CrossTx) error {
	// pick up the precommit contract txs
	t.pending.mu.Lock()
	for _, tx := range txs {
		if tx.Contract.GetStatus() != contractlib.Finished {
			t.pending.prq.Push(tx, -tx.TimeStamp.Seconds)
		}
	}
	t.pending.mu.Unlock()

	// store to db
	if err := t.DB.Save(txs); err != nil {
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
				raw, err := json.Marshal(tx)
				if err != nil {
					utils.Logger.Error("[courier.TxManager] marshal tx", "crossID", tx.CrossID, "status", tx.GetStatus(), "err", err)
					continue
				}

				// TODO: batch send, MaxBatchSize = 64
				if err := t.oClient.Send(raw); err != nil {
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

func (t *TxManager) AddCrossTxReceipts(ctrs []CrossTxReceipt) error {
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

			if err := t.AddCrossTxReceipts(executed); err != nil {
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
