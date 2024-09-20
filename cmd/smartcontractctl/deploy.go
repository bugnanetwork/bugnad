package main

import (
	"encoding/hex"
	"fmt"

	"github.com/bugnanetwork/bugnad/domain/consensus/utils/txscript"
	"github.com/bugnanetwork/bugnad/util"
)

func deploySmartContract(cfg *deployConfig) error {
	client, addr, myKeyPair, err := newCaller(&cfg.configFlags)
	if err != nil {
		return fmt.Errorf("Error parsing private key: %s", err)
	}

	contractCode, err := hex.DecodeString(cfg.ContractCode)
	if err != nil {
		return fmt.Errorf("Error decoding contract code: %s", err)
	}

	input, err := NewDeploySmartContractinput(addr, contractCode, cfg.configFlags.ActiveNetParams.Prefix)
	if err != nil {
		return fmt.Errorf("Error creating deploy smart contract input: %s", err)
	}

	return submit(&cfg.configFlags, client, myKeyPair, addr, input)
}

type DeploySmartContractinput struct {
	addr           util.Address
	secretContract []byte
	contractCode   []byte
}

func NewDeploySmartContractinput(
	caller util.Address,
	contractCode []byte,
	prefix util.Bech32Prefix,
) (*DeploySmartContractinput, error) {
	builder := txscript.NewScriptBuilder()

	// Verify their signature is being used to redeem the output
	builder.AddData(caller.ScriptAddress())
	builder.AddOp(txscript.OpCheckSig)

	for i := 0; i < len(contractCode); i += txscript.MaxScriptElementSize {
		builder.AddOp(txscript.OpDrop)
	}

	builder.AddOp(txscript.Op2Drop)
	secretContract, err := builder.Script()
	if err != nil {
		return nil, err
	}

	contractP2SHaddress, err := util.NewAddressScriptHash(secretContract, prefix)
	if err != nil {
		return nil, err
	}

	return &DeploySmartContractinput{
		addr:           contractP2SHaddress,
		secretContract: secretContract,
		contractCode:   contractCode,
	}, nil
}

func (d *DeploySmartContractinput) Address() util.Address {
	return d.addr
}

func (d *DeploySmartContractinput) RedeemScript(signature []byte) ([]byte, error) {
	builder := txscript.NewScriptBuilder()

	builder.AddData([]byte("bugna_script"))
	builder.AddData([]byte("deploy"))

	for i := 0; i < len(d.contractCode); i += txscript.MaxScriptElementSize {
		start := i
		end := i + txscript.MaxScriptElementSize
		if end > len(d.contractCode) {
			end = len(d.contractCode)
		}

		d := d.contractCode[start:end]
		builder.AddData(d)
	}

	builder.AddData(signature)
	builder.AddData(d.secretContract)
	return builder.Script()

}
