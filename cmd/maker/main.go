package main

import (
	"bytes"
	"context"
	"flag"
	"fmt"
	"github.com/simplechain-org/crosshub/chainview"
	"log"
	"math/big"

	"github.com/simplechain-org/go-simplechain/accounts/abi"
	"github.com/simplechain-org/go-simplechain/common"
	"github.com/simplechain-org/go-simplechain/common/hexutil"
	"github.com/simplechain-org/go-simplechain/rpc"
)

var (
	rawurlVar = flag.String("rawurl", "http://192.168.3.137:8545", "rpc url")

	contract = flag.String("contract", "0x71B4B8fd103dcDA2b971b1677ec70a96Ad24FB38", "合约地址")

	value = flag.Uint64("value", 100, "转入合约的数量")

	destValue = flag.Uint64("destValue", 1, "兑换数量")

	chainId = flag.Uint("chainId", 5, "目的链id")

	fromVar = flag.String("from", "0xa3213ef69420fb5a4b804197a7de9e7d5c8e43f4", "发起人地址")

	focusVar = flag.String("to", "", "focus addr")

	gaslimitVar = flag.Uint64("gaslimit", 120000, "gas最大值")

	countTx = flag.Int("count", 1, "交易数")
)

type SendTxArgs struct {
	From     common.Address  `json:"from"`
	To       *common.Address `json:"to"`
	Gas      *hexutil.Uint64 `json:"gas"`
	GasPrice *hexutil.Big    `json:"gasPrice"`
	Value    *hexutil.Big    `json:"value"`
	Nonce    *hexutil.Uint64 `json:"nonce"`
	Data     *hexutil.Bytes  `json:"data"`
	Input    *hexutil.Bytes  `json:"input"`
}

//跨链交易发起人
func main() {
	flag.Parse()
	maker()
}

func maker() {
	client, err := rpc.Dial(*rawurlVar)
	if err != nil {
		fmt.Println("dial", "err", err)
		return
	}
	data, err := hexutil.Decode(chainview.CrossAbi)
	if err != nil {
		fmt.Println(err)
		return
	}

	from := common.HexToAddress(*fromVar)
	//focusAddr := common.HexToAddress(*focusVar)
	to := common.HexToAddress(*contract)
	gas := hexutil.Uint64(*gaslimitVar)
	value := hexutil.Big(*new(big.Int).SetUint64(*value))
	price := hexutil.Big(*big.NewInt(1e9))

	abi, err := abi.JSON(bytes.NewReader(data))
	if err != nil {
		log.Fatalln(err)
	}

	des := new(big.Int).SetUint64(*destValue)

	//out, err := abi.Pack("makerStart",remoteChainId ,des,[]byte("In the end, it’s not the years in your life that count. It’s the life in your years."))
	out, err := abi.Pack("makerStart", des, uint8(*chainId), []string{"b",""}, []byte{})
	if err != nil {
		fmt.Println(err)
		return
	}
	input := hexutil.Bytes(out)

	for i := 0; i < *countTx; i++ {
		var result common.Hash
		if err = client.CallContext(context.Background(), &result, "eth_sendTransaction", &SendTxArgs{
			From:     from,
			To:       &to,
			Gas:      &gas,
			GasPrice: &price,
			Value:    &value,
			Input:    &input,
		}); err != nil {
			fmt.Println("CallContext", "err", err)
			return
		}

		fmt.Println("result=", result.Hex())
	}
}
