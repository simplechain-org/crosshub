package contractlib

import (
	"encoding/json"
	"testing"
)

func TestPrecommitContract(t *testing.T) {
	precmmit := PrecommitContract{Status: Init}

	contract := Contract{&precmmit}

	raw, err := json.Marshal(contract)
	if err != nil {
		t.Error(err)
	}

	var contract2 Contract
	err = json.Unmarshal(raw, &contract2)
	if err != nil {
		t.Error(err)
	}

	if _, ok := contract2.IContract.(*PrecommitContract); !ok {
		t.Error("Contract UnmarshalJson failed")
	}
}

func TestCommitContract(t *testing.T) {
	commit := CommitContract{Status: Finished}

	contract := Contract{&commit}

	raw, err := json.Marshal(contract)
	if err != nil {
		t.Error(err)
	}

	var contract2 Contract
	err = json.Unmarshal(raw, &contract2)
	if err != nil {
		t.Error(err)
	}

	if _, ok := contract2.IContract.(*CommitContract); !ok {
		t.Error("Contract UnmarshalJson failed")
	}
}

type Test struct {
	Name string
	Contract
}

func (t *Test) UnmarshalJSON(bytes []byte) (err error) {
	var errList []error

	var objMap map[string]*json.RawMessage
	errList = append(errList, json.Unmarshal(bytes, &objMap))
	errList = append(errList, json.Unmarshal(*objMap["Name"], &t.Name))

	t.IContract, err = RebuildIContract(*objMap["IContract"])
	errList = append(errList, err)

	for _, err := range errList {
		if err != nil {
			return err
		}
	}

	return nil
}

func TestIncRebuildIContract(t *testing.T) {
	var t1 = Test{
		Name: "test",
		Contract: Contract{&PrecommitContract{
			Status:       Init,
			ContractID:   "test",
			Receipt:      "",
			ContractCore: ContractCore{},
		}},
	}
	raw, err := json.Marshal(t1)
	if err != nil {
		t.Error(err)
	}

	var t2 Test
	if err = json.Unmarshal(raw, &t2); err != nil {
		t.Error(err)
	}

	if t2.Name != t1.Name {
		t.Error(err)
	}
}

func TestParseCStatus(t *testing.T) {
	var contracts = []Contract{
		Contract{
			&PrecommitContract{
				Status:     Init,
				ContractID: "d2a6e261c941bb065b17ec09f8d8c5f16c08546b35420cfac4d9b324efb6d675",
			},
		},
		Contract{
			&CommitContract{
				Status:     Finished,
				ContractID: "d2a6e261c941bb065b17ec09f8d8c5f16c08546b35420cfac4d9b324efb6d675",
			},
		},
		Contract{
			&CommitContract{
				Status:     OutOnceCompleted,
				ContractID: "d2a6e261c941bb065b17ec09f8d8c5f16c08546b35420cfac4d9b324efb6d675",
			},
		},
	}

	for _, c := range contracts {
		rawContract, err := json.Marshal(c)
		if err != nil {
			t.Fatal(err)
		}

		var cc = Contract{}
		err = json.Unmarshal(rawContract, &cc)
		if err != nil {
			t.Fatal(err)
		}

		if cc.GetContractID() != c.GetContractID() {
			t.Fatalf("want: %s, got: %s", c.GetContractID(), cc.GetContractID())
		}

		if cc.GetStatus() != c.GetStatus() {
			t.Fatalf("want: %s, got: %s", c.GetStatus(), cc.GetStatus())
		}
	}
}
