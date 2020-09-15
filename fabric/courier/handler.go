package courier

import (
	"github.com/simplechain-org/crosshub/fabric/courier/client"

	"github.com/asdine/storm/v3"
	"github.com/simplechain-org/go-simplechain/crypto/ecdsa"
)

type Handler struct {
	blkSync *BlockSync
	rootDB  *storm.DB
	txm     *TxManager

	stopCh chan struct{}
}

func New(cfg *client.Config, ocli client.OutChainClient) (*Handler, error) {
	fabCli := client.NewFabCli(cfg)

	rootDB, err := OpenStormDB(cfg.DataDir())
	if err != nil {
		return nil, err
	}

	store, err := NewStore(rootDB)
	if err != nil {
		return nil, err
	}

	txm := NewTxManager(fabCli, ocli, store)
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

	h.txm.Stop()

	h.rootDB.Close()
}

func (h *Handler) SetPrivateKey(key *ecdsa.PrivateKey) {
	h.txm.privateKey = key
}

func (h *Handler) SetOutChainFlag(flag bool) {
	h.txm.outchain = flag
}
