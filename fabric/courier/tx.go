package courier

import (
	"encoding/json"
	"github.com/simplechain-org/crosshub/fabric/courier/utils"
	"github.com/simplechain-org/go-simplechain/common"
	"math/big"

	"github.com/simplechain-org/crosshub/core"
	"github.com/simplechain-org/crosshub/fabric/courier/contractlib"

	"github.com/golang/protobuf/ptypes/timestamp"
)

const (
	Fabric      uint8 = 2
	SimpleChain uint8 = 5
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

func toCrossHubTx(tx *CrossTx) *core.CrossTransaction {
	pre, ok := tx.IContract.(*contractlib.PrecommitContract)
	if !ok {
		return nil
	}

	payload, err := json.Marshal(pre.GetCoreInfo())
	if err != nil {
		return nil
	}

	val, ok := new(big.Int).SetString(pre.Value, 10)
	if !ok {
		val = new(big.Int)
	}
	charge := new(big.Int)

	ctxID := common.HexToHash(tx.CrossID)
	txID := common.HexToHash(tx.TxID)
	blkHash := common.BigToHash(new(big.Int).SetUint64(tx.BlockNumber))

	from := pre.Address
	to := ""

	return core.NewCrossTransaction(val, charge, from, to, Fabric, SimpleChain, ctxID, txID, blkHash, payload)
}

type CrossChannel chan interface{}

func (c CrossChannel) Send(ctx *core.CrossTransaction) error {
	c <- ctx
	utils.Logger.Debug("[courier.CrossChannel] Send ", "crossID", ctx.ID().String())
	return nil
}

func (c CrossChannel) Close() {

}
