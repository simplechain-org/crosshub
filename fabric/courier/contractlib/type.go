package contractlib

// copy from chaincode/chaincode_example02/go/type.go

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
)

type CStatus uint8

const (
	// Init is the fabric precommit contract transaction status flag, generate on fabric chaincode
	Init CStatus = 1 << (8 - 1 - iota)
	// Pending is the fabric precommit contract transaction status flag, change by courier
	Pending
	// Executed is the fabric precommit contract transaction status flag, change by courier
	Executed
	// Finished is the fabric commit contract transaction status flag, generate on fabric chaincode
	Finished
	// Completed is the fabric commit contract transaction status flag, change by courier
	Completed
	// OutOnceCompleted is the fabric commit contract transaction status flag, change by chaincode, handle once outchain request
	OutOnceCompleted
)

func (c CStatus) String() string {
	switch c {
	case Init:
		return "Init"
	case Pending:
		return "Pending"
	case Executed:
		return "Executed"
	case Finished:
		return "Finished"
	case Completed:
		return "Completed"
	case OutOnceCompleted:
		return "OutOnceCompleted"
	default:
		return "UnSupported"
	}
}

func ParseCStatus(c string) (CStatus, error) {
	switch c {
	case "Init":
		return Init, nil
	case "Pending":
		return Pending, nil
	case "Executed":
		return Executed, nil
	case "Finished":
		return Finished, nil
	case "Completed":
		return Completed, nil
	case "OutOnceCompleted":
		return OutOnceCompleted, nil
	}

	var status CStatus
	return status, fmt.Errorf("not a valid cstatus flag: %s", c)
}

func (c CStatus) MarshalText() ([]byte, error) {
	return []byte(c.String()), nil
}

func (c *CStatus) UnmarshalText(text []byte) error {
	status, err := ParseCStatus(string(text))
	if err != nil {
		return err
	}

	*c = status
	return nil
}

type Contract struct {
	IContract
}

func (c *Contract) UnmarshalJSON(bytes []byte) (err error) {
	var objMap map[string]*json.RawMessage
	err = json.Unmarshal(bytes, &objMap)
	if err != nil {
		return err
	}

	c.IContract, err = RebuildIContract(*objMap["IContract"])

	return err
}

func RebuildIContract(bytes json.RawMessage) (c IContract, err error) {
	var contractMap map[string]*json.RawMessage
	err = json.Unmarshal(bytes, &contractMap)
	if err != nil {
		return nil, err
	}

	var typ string
	err = json.Unmarshal(*contractMap["status"], &typ)
	if err != nil {
		return nil, err
	}

	switch typ {
	case "Init":
		fallthrough
	case "Pending":
		fallthrough
	case "Executed":
		fallthrough
	case "Completed":
		var pc PrecommitContract
		err = json.Unmarshal(bytes, &pc)
		c = &pc
	case "Finished":
	case "OutOnceCompleted":
		var cc CommitContract
		err = json.Unmarshal(bytes, &cc)
		c = &cc
	default:
		return nil, fmt.Errorf("unsupported contract type: %s", typ)
	}

	return c, nil
}

type IContract interface {
	GetContractID() string
	GetStatus() CStatus
	GetCoreInfo() *ContractCore
	UpdateStatus(CStatus)
	IsInit() bool
	IsPending() bool
	IsExecuted() bool
	IsFinished() bool
	IsCompleted() bool
	IsOutOnceCompleted() bool
}

type ContractCore struct {
	Address     string   `json:"address"`
	Value       string   `json:"value"`
	Description string   `json:"description"`
	Owner       string   `json:"owner"`
	ToCallFunc  string   `json:"to_call"`
	Args        []string `json:"args"`
	Creator     string   `json:"creator"`
}

func (core *ContractCore) genContractID(txid string) (string, error) {
	rawData, err := json.Marshal(core)
	if err != nil {
		return "", err
	}

	var hash [32]byte

	h := sha256.New()
	h.Write(rawData)
	h.Write([]byte(txid))
	h.Sum(hash[:0])

	return hex.EncodeToString(hash[:]), nil
}

type PrecommitContract struct {
	Status     CStatus `json:"status" storm:"index"`
	ContractID string  `json:"contract_id"`
	Receipt    string  `json:"receipt" storm:"index"`
	ContractCore
}

func (c *PrecommitContract) GetCoreInfo() *ContractCore {
	return &c.ContractCore
}

func (c *PrecommitContract) GetContractID() string {
	return c.ContractID
}

func (c *PrecommitContract) GetStatus() CStatus {
	return c.Status
}

func (c *PrecommitContract) UpdateStatus(s CStatus) {
	c.Status = s
}

func (c *PrecommitContract) UpdateReceipt(receipt string) {
	c.Receipt = receipt
}

func (c *PrecommitContract) IsInit() bool {
	return c.Status == Init
}

func (c *PrecommitContract) IsPending() bool {
	return c.Status == Pending
}

func (c *PrecommitContract) IsExecuted() bool {
	return c.Status == Executed
}

func (c *PrecommitContract) IsFinished() bool {
	return c.Status == Finished
}

func (c *PrecommitContract) IsCompleted() bool {
	return c.Status == Completed
}

func (c *PrecommitContract) IsOutOnceCompleted() bool {
	return c.Status == OutOnceCompleted
}

type CommitContract struct {
	Status     CStatus `json:"status" storm:"index"`
	ContractID string  `json:"contract_id"`
}

func (c *CommitContract) GetContractID() string {
	return c.ContractID
}

func (c *CommitContract) GetStatus() CStatus {
	return c.Status
}

func (c *CommitContract) UpdateStatus(s CStatus) {
	c.Status = s
}

func (c *CommitContract) GetCoreInfo() *ContractCore {
	return nil
}

func (c *CommitContract) IsInit() bool {
	return c.Status == Init
}

func (c *CommitContract) IsPending() bool {
	return c.Status == Pending
}

func (c *CommitContract) IsExecuted() bool {
	return c.Status == Executed
}

func (c *CommitContract) IsFinished() bool {
	return c.Status == Finished
}

func (c *CommitContract) IsCompleted() bool {
	return c.Status == Completed
}

func (c *CommitContract) IsOutOnceCompleted() bool {
	return c.Status == OutOnceCompleted
}
