package courier

import (
	"sync"

	"github.com/simplechain-org/crosshub/fabric/courier/client"

	"github.com/asdine/storm/v3"
)

type Handler struct {
	blkSync *BlockSync
	rootDB  *storm.DB
	txm     *TxManager

	taskWg sync.WaitGroup

	stopCh chan struct{}
}

func New(cfg *client.Config) (*Handler, error) {
	fabCli := client.NewFabCli(cfg)

	rootDB, err := OpenStormDB(cfg.DataDir())
	if err != nil {
		return nil, err
	}

	store, err := NewStore(rootDB)
	if err != nil {
		return nil, err
	}

	txm := NewTxManager(fabCli, &client.MockOutChainClient{}, store)
	h := &Handler{
		blkSync: NewBlockSync(fabCli, txm),
		rootDB:  rootDB,
		txm:     txm,
		stopCh:  make(chan struct{}),
	}

	return h, nil
}

func (h *Handler) Start() {
	h.txm.Start()
	h.blkSync.Start()
}

func (h *Handler) Stop() {
	h.blkSync.Stop()

	close(h.stopCh)
	h.taskWg.Wait()

	h.txm.Stop()

	h.rootDB.Close()
}

func (h *Handler) RecvMsg(ctr CrossTxReceipt) {
	h.taskWg.Add(1)
	go func() {
		defer h.taskWg.Done()

		h.txm.executed.mu.Lock()
		h.txm.executed.prq.Push(ctr, -ctr.Sequence)
		h.txm.executed.mu.Unlock()

		select {
		case h.txm.executed.process <- struct{}{}:
		case <-h.stopCh:
			return
		}
	}()
}
