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
	rawurlVar  = flag.String("rawurl", "http://127.0.0.1:60012", "rpc url")
	sendurlVar = flag.String("sendurl", "http://192.168.3.137:8545", "send rpc url")

	contract = flag.String("contract", "0x71B4B8fd103dcDA2b971b1677ec70a96Ad24FB38", "合约地址")

	fromVar = flag.String("from", "0xa3213ef69420fb5a4b804197a7de9e7d5c8e43f4", "接单人地址")

	gaslimitVar = flag.Uint64("gaslimit", 200000, "gas最大值")

	limit = flag.Uint64("count", 1, "接单数量")
)

type SendTxArgs struct {
	From     common.Address  `json:"From"`
	To       *common.Address `json:"to"`
	Gas      *hexutil.Uint64 `json:"gas"`
	GasPrice *hexutil.Big    `json:"gasPrice"`
	Value    *hexutil.Big    `json:"Value"`
	Nonce    *hexutil.Uint64 `json:"nonce"`
	Data     *hexutil.Bytes  `json:"Data"`
	Input    *hexutil.Bytes  `json:"input"`
}

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

type RPCPageCrossTransactions struct {
	Data map[uint64][]*RPCCrossTransaction `json:"data"`
	//Total int                               `json:"total"`
}

type Order struct {
	TxId 	common.Hash
	TxHash	common.Hash
 	BlockHash common.Hash
	Value     *big.Int
 	Charge    *big.Int
 	From    common.Address
 	To   	common.Address
    Origin  uint8
 	Purpose uint8
	Payload []byte
	V                []*big.Int
	R                [][32]byte
	S                [][32]byte
}

var signatures map[string]RPCPageCrossTransactions

func main() {
	flag.Parse()
	taker()
}

func taker() {
	data, err := hexutil.Decode(chainview.CrossAbi)
	if err != nil {
		fmt.Println(err)
		return
	}
	abi, err := abi.JSON(bytes.NewReader(data))
	if err != nil {
		log.Fatalln(err)
	}
	//账户地址
	from := common.HexToAddress(*fromVar)
	//合约地址
	//在子链上接单就要填写子链上的合约地址
	//在主链上接单就要填写主链上的合约地址
	to := common.HexToAddress(*contract)
	gas := hexutil.Uint64(*gaslimitVar)
	price := hexutil.Big(*big.NewInt(1e9))

	client, err := rpc.Dial(*rawurlVar)
	if err != nil {
		fmt.Println("dial", "err", err)
		return
	}

	err = client.CallContext(context.Background(), &signatures, "cross_ctxContentByPage", 10, 1, limit, 1)
	if err != nil {
		fmt.Println("CallContext", "err", err)
		return
	}

	client.Close()

	client, err = rpc.Dial(*sendurlVar)
	if err != nil {
		fmt.Println("dial", "err", err)
		return
	}

	for _, value := range signatures["remote"].Data {
		for _, v := range value {
			//if i <= 50 { //自动最多接10000单交易
			if v.To != "" && (common.HexToAddress(v.To)  != from && common.HexToAddress(v.To) != from) { //指定了接单地址并且不是from的直接跳过
				fmt.Printf("tx: %s need taker: %s\n", v.TxHash.String(), v.To)
				continue
			}
			r := make([][32]byte, 0, 1)
			s := make([][32]byte, 0, 1)
			vv := make([]*big.Int, 0, 1)

			for i := 0; i < 1; i++ {
				rone := common.LeftPadBytes(v.R.ToInt().Bytes(), 32)
				var a [32]byte
				copy(a[:], rone)
				r = append(r, a)
				sone := common.LeftPadBytes(v.S.ToInt().Bytes(), 32)
				var b [32]byte
				copy(b[:], sone)
				s = append(s, b)
				vv = append(vv, v.V.ToInt())
			}
			//在调用这个函数中调用的chainId其实就是表示的是发单的链id
			//也就是maker的源头，那条链调用了maker,这个链id就对应那条链的id
			//chainId := big.NewInt(int64(remoteId))

			ord := Order{
				TxId:v.CTxId,
				TxHash:v.TxHash,
				BlockHash:v.BlockHash,
				Value:v.Value.ToInt(),
				Charge:v.Charge.ToInt(),
				From:common.HexToAddress(v.From),
				To:common.Address{},
				Origin:uint8(v.Origin),
				Purpose:uint8(v.Purpose),
				Payload:v.Payload,
				V:                vv,
				R:                r,
				S:                s,
			}
			fmt.Println(ord.To.String())

			out, err := abi.Pack("taker", &ord,"b",[]byte("i am b!"))
			if err != nil {
				fmt.Println("abi.Pack err=", err)
				continue
			}

			input := hexutil.Bytes(out)

			var result common.Hash
			if err := client.CallContext(context.Background(), &result, "eth_sendTransaction", &SendTxArgs{
				From:     from,
				To:       &to,
				Gas:      &gas,
				GasPrice: &price,
				Value:    v.Charge,
				Input:    &input,
			}); err != nil {
				fmt.Println("SendTransaction", "err", err)
				return
			}

			fmt.Printf("eth_sendTransaction result=%s, ctxID=%s\n", result.Hex(), v.CTxId.String())
		}

		//}

	}

}
