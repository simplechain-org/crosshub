package main

import (
	"bytes"
	"crypto/x509"
	"encoding/json"
	"encoding/pem"
	"fmt"
	"strings"

	"github.com/hyperledger/fabric/core/chaincode/shim"
	pb "github.com/hyperledger/fabric/protos/peer"
)

// TODO: 锁住所有与precommit提交字段中key相同的资源,直到commit完成
func (t *SimpleChaincode) precommit(stub shim.ChaincodeStubInterface, args []string) pb.Response {
	callArgs := strings.Split(args[4], " ")
	core := ContractCore{
		Address:     args[0],
		Value:       args[1],
		Description: args[2],
		ToCallFunc:  args[3],
		Creator:     t.creator(stub),
		Args:        callArgs,
		Owner:       callArgs[0],
	}

	id, err := core.genContractID(stub.GetTxID())
	if err != nil {
		return shim.Error(fmt.Sprintf("get contract id err: %v", err))
	}

	contract := Contract{
		&PrecommitContract{
			Status:       Init,
			ContractID:   id,
			ContractCore: core,
		},
	}

	rawContract, err := json.Marshal(&contract)
	if err != nil {
		return shim.Error(err.Error())
	}

	// store to ledger
	if err = stub.PutState(id, rawContract); err != nil {
		return shim.Error(err.Error())
	}

	// send event
	if err = stub.SetEvent("precommit", rawContract); err != nil {
		return shim.Error(err.Error())
	}

	return shim.Success(nil)
}

func (t *SimpleChaincode) commit(stub shim.ChaincodeStubInterface, args []string) pb.Response {
	switch args[0] {
	case "outchain":
		if err := t.outchainCommit(args[1:], stub); err != nil {
			return shim.Error(fmt.Sprintf("outchainCommit %v, err: %s", args, err.Error()))
		}
	default:
		return t.regularCommit(args, stub)
	}

	return shim.Success(nil)
}

// args: "outchain", <crossID>, "invoke", <userA> <userB> <amount>
// example: {"outchain","998ad57079b2f8f5c280eb81b7c2076437f631716f9a8a0637be74c16fa50ce2", "invoke","a","b","10"}
func (t *SimpleChaincode) outchainCommit(args []string, stub shim.ChaincodeStubInterface) error {
	switch args[1] {
	case "invoke":
		err := t.doInvoke(stub, args[2:])
		if err != nil {
			return fmt.Errorf("doInvoke args %v, err: %v", args, err)
		}

		commit := Contract{
			&CommitContract{
				Status:     OutOnceCompleted,
				ContractID: args[0],
			}}
		rawCommit, err := json.Marshal(&commit)
		if err != nil {
			return err
		}

		// send event
		return stub.SetEvent("commit", rawCommit)
	default:
		return fmt.Errorf("Unsupported call for outchain ")
	}
}

// args : <contractID> <receipt>
// example: {"c55f273ed39e36cfdba7f6e5b0982b7fd4c81fb2cbf8965a562a0e12b1ab8e7a", "964b4adc1ed78fce2cefc8f1c5e33b98a5ea73978122a66cbae1d3e13b9b229e"}
func (t *SimpleChaincode) regularCommit(args []string, stub shim.ChaincodeStubInterface) pb.Response {
	contractID := args[0]
	receipt := args[1]

	rawContract, err := stub.GetState(contractID)
	if err != nil {
		return shim.Error(fmt.Sprintf("get contract by %s, err: %v", contractID, err))
	} else if rawContract == nil {
		return shim.Error(fmt.Sprintf("invalid contractid %s", contractID))
	}

	var contract Contract
	if err = json.Unmarshal(rawContract, &contract); err != nil {
		return shim.Error(fmt.Sprintf("parse contract with %s, err: %v", contractID, err))
	}
	if contract.GetStatus() == Finished {
		return shim.Success([]byte("replicate call commit"))
	}

	preCommit, ok := contract.IContract.(*PrecommitContract)
	if !ok {
		return shim.Error(fmt.Sprintf("assert contract.IContract.(*PrecommitContract) failed"))
	}
	if err = t.doCommit(stub, preCommit); err != nil {
		return shim.Error(fmt.Sprintf("doCommit err: %v", err))
	}
	preCommit.UpdateStatus(Finished)
	preCommit.UpdateReceipt(receipt)
	updateData, err := json.Marshal(&contract)
	if err != nil {
		return shim.Error(err.Error())
	}

	// store to ledger
	if err = stub.PutState(contractID, updateData); err != nil {
		return shim.Error(err.Error())
	}

	commit := Contract{
		&CommitContract{
			Status:     Finished,
			ContractID: contractID,
		}}
	rawCommit, err := json.Marshal(commit)
	if err != nil {
		return shim.Error(err.Error())
	}

	// send event
	if err = stub.SetEvent("commit", rawCommit); err != nil {
		return shim.Error(fmt.Sprintf("send commit event err: %v", err.Error()))
	}

	return shim.Success(nil)
}

func (t *SimpleChaincode) doCommit(stub shim.ChaincodeStubInterface, c *PrecommitContract) error {
	switch c.ToCallFunc {
	case "invoke":
		return t.doInvoke(stub, c.Args)
	default:
		return fmt.Errorf("undefined %s", c.ToCallFunc)
	}
}

func (t *SimpleChaincode) creator(stub shim.ChaincodeStubInterface) string {
	creatorByte, _ := stub.GetCreator()
	certStart := bytes.IndexAny(creatorByte, "-----BEGIN")
	if certStart == -1 {
		shim.Error("No certificate found")
	}
	certText := creatorByte[certStart:]
	bl, _ := pem.Decode(certText)
	if bl == nil {
		shim.Error("Could not decode the PEM structure")
	}
	cert, err := x509.ParseCertificate(bl.Bytes)
	if err != nil {
		shim.Error("ParseCertificate failed")
	}

	return cert.Subject.CommonName
}
