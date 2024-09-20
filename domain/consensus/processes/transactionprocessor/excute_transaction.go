package transactionprocessor

import (
	"fmt"
	"math/big"
	"time"

	"github.com/bugnanetwork/bugnad/domain/bvm/state"
	"github.com/bugnanetwork/bugnad/domain/bvm/vm"
	"github.com/bugnanetwork/bugnad/domain/consensus/model"
	"github.com/bugnanetwork/bugnad/domain/consensus/model/externalapi"
	"github.com/bugnanetwork/bugnad/domain/consensus/utils/consensushashing"
	"github.com/bugnanetwork/bugnad/domain/consensus/utils/txscript"
)

func (t *transactionProcessor) Excute(
	stagingArea *model.StagingArea,
	povBlockHash *externalapi.DomainHash,
	blockDaaScore uint64,
	tx *externalapi.DomainTransaction,
) error {
	s := stagingArea
	if povBlockHash.String() == "ffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff" {
		s = model.NewStagingArea()
	}

	for i, input := range tx.Inputs {
		if err := t.excuteTXInput(tx, blockDaaScore, s, input, i); err != nil {
			if err.Error() == "invalid script" {
				continue
			}
			return err
		}
	}

	return nil
}

func (t *transactionProcessor) excuteTXInput(tx *externalapi.DomainTransaction, blockDaaScore uint64, stagingArea *model.StagingArea, input *externalapi.DomainTransactionInput, inputIndex int) error {
	operator, action, toAddress, payload, err := extractSignatureScript(input.SignatureScript)
	if err != nil {
		return err
	}

	caller := vm.ScriptPubkeyToAddress(operator)
	stateDB := t.bvmStore.StateDBWrapper(t.databaseContext, stagingArea).(*state.StateDB)
	txID := consensushashing.TransactionID(tx)

	defer func() {
		logs := stateDB.GetLogs(vm.BytesToHash(txID.ByteSlice()))
		for _, log := range logs {
			topics := []externalapi.DomainHash{}
			for _, topic := range log.Topics {
				hash, _ := externalapi.NewDomainHashFromByteSlice(topic.Bytes())
				topics = append(topics, *hash)
			}

			tx.Logs = append(tx.Logs, &externalapi.DomainTransactionLog{
				ScriptPublicKey: log.Address.ScriptPublicKey(),
				Topics:          topics,
				Data:            log.Data,
				Index:           uint64(log.Index),
			})
		}

		journal := stateDB.DumpJournal()
		tx.Journal = journal

		stateDB.IntermediateRoot(true)
	}()

	context := CreateExecuteContext(blockDaaScore, caller, vm.BytesToHash(txID.ByteSlice()), uint32(inputIndex))
	chainConfig := CreateChainConfig()
	vmConfig := CreateVMDefaultConfig()

	evm := vm.NewEVM(context, stateDB, chainConfig, vmConfig)

	switch action {
	case ActionDeploy:
		_, _, _, err := evm.Create(vm.AccountRef(caller), payload, evm.GasLimit, big.NewInt(0))
		if err != nil {
			tx.Result = err.Error()
			return fmt.Errorf("err evm.Create: %w", err)
		}
	case ActionInteract:
		snapshot := stateDB.Snapshot()
		stateObject := stateDB.GetOrNewStateObject(vm.BytesToAddress(toAddress))
		if stateObject.ScriptPublicKey() == nil {
			stateDB.RevertToSnapshot(snapshot)
			tx.Result = "invalid contract address"
			return fmt.Errorf("invalid toAddress")
		}

		nonce := evm.StateDBHandler.GetNonce(caller)
		evm.StateDBHandler.SetNonce(caller, nonce+1)

		toAddr := vm.ScriptPubkeyToAddress(stateObject.ScriptPublicKey())
		_, _, err = evm.Call(vm.AccountRef(caller), toAddr, payload, evm.GasLimit, big.NewInt(0))
		if err != nil {
			tx.Result = err.Error()
			return fmt.Errorf("err evm.Call: %w", err)
		}
	default:
		return fmt.Errorf("invalid type: %s", action)
	}

	tx.Result = "Ok"

	return nil
}

func CreateLogTracer() *vm.StructLogger {
	logConf := vm.LogConfig{
		DisableMemory:  false,
		DisableStack:   false,
		DisableStorage: false,
		Debug:          false,
		Limit:          0,
	}
	return vm.NewStructLogger(&logConf)

}
func CreateChainConfig() *vm.ChainConfig {
	chainCfg := vm.ChainConfig{
		ChainID: big.NewInt(1),
	}
	return &chainCfg
}
func CreateExecuteContext(
	blockDaaScore uint64,
	caller vm.Address,
	txHash vm.Hash,
	txIndex uint32,
) vm.Context {
	context := vm.Context{
		Origin:      caller,
		GasPrice:    new(big.Int),
		TxHash:      txHash,
		TxIndex:     txIndex,
		GasLimit:    vm.MaxUint64,
		BlockNumber: big.NewInt(int64(blockDaaScore)),
		Time:        big.NewInt(time.Now().Unix()),
		Difficulty:  new(big.Int),
	}
	return context
}

func CreateVMDefaultConfig() vm.Config {
	return vm.Config{
		Debug:                   false,
		Tracer:                  CreateLogTracer(),
		NoRecursion:             false,
		EnablePreimageRecording: false,
	}

}

type Action byte

const (
	ActionDeploy   = 0x01
	ActionInteract = 0x02
)

func extractSignatureScript(script []byte) (operator *externalapi.ScriptPublicKey, action Action, toAddress []byte, payload []byte, err error) {
	operator, inputs, err := txscript.ExtractSignatureScriptToSmartcontractInputData(script)
	if err != nil {
		err = fmt.Errorf("invalid script")
		return
	}

	if len(inputs) < 2 {
		err = fmt.Errorf("invalid script")
		return
	}

	switch string(inputs[0]) {
	case "deploy":
		action = ActionDeploy
		for _, input := range inputs[1:] {
			payload = append(payload, input...)
		}
	case "interact":
		action = ActionInteract
		toAddress = inputs[1]
		for _, input := range inputs[2:] {
			payload = append(payload, input...)
		}
	default:
		err = fmt.Errorf("invalid script")
	}

	return
}
