package serialization

import (
	"math"

	"github.com/bugnanetwork/bugnad/domain/consensus/model/externalapi"
	"github.com/pkg/errors"
)

// DomainTransactionToDbTransaction converts DomainTransaction to DbTransaction
func DomainTransactionToDbTransaction(domainTransaction *externalapi.DomainTransaction) *DbTransaction {
	dbInputs := make([]*DbTransactionInput, len(domainTransaction.Inputs))
	for i, domainTransactionInput := range domainTransaction.Inputs {
		dbInputs[i] = &DbTransactionInput{
			PreviousOutpoint: DomainOutpointToDbOutpoint(&domainTransactionInput.PreviousOutpoint),
			SignatureScript:  domainTransactionInput.SignatureScript,
			Sequence:         domainTransactionInput.Sequence,
			SigOpCount:       uint32(domainTransactionInput.SigOpCount),
		}
	}

	dbOutputs := make([]*DbTransactionOutput, len(domainTransaction.Outputs))
	for i, domainTransactionOutput := range domainTransaction.Outputs {
		dbScriptPublicKey := ScriptPublicKeyToDBScriptPublicKey(domainTransactionOutput.ScriptPublicKey)
		dbOutputs[i] = &DbTransactionOutput{
			Value:           domainTransactionOutput.Value,
			ScriptPublicKey: dbScriptPublicKey,
		}
	}

	logs := make([]*DbTransactionLog, len(domainTransaction.Logs))
	for i, domainLog := range domainTransaction.Logs {
		topics := make([]*DbHash, len(domainLog.Topics))
		for j, topic := range domainLog.Topics {
			topics[j] = DomainHashToDbHash(&topic)
		}

		logs[i] = &DbTransactionLog{
			ScriptPublicKey: ScriptPublicKeyToDBScriptPublicKey(domainLog.ScriptPublicKey),
			Topics:          topics,
			Data:            domainLog.Data,
			Index:           domainLog.Index,
		}
	}

	journal := make([]*DbTransactionJournal, len(domainTransaction.Journal))
	for i, domainJournal := range domainTransaction.Journal {
		switch p := domainJournal.(type) {
		case *externalapi.DomainTransactionJournalCreateObjectChange:
			journal[i] = &DbTransactionJournal{
				Payload: &DbTransactionJournal_CreateObjectChange_{
					CreateObjectChange: &DbTransactionJournal_CreateObjectChange{
						ScriptPublicKey: ScriptPublicKeyToDBScriptPublicKey(p.ScriptPublicKey),
					},
				},
			}
		case *externalapi.DomainTransactionJournalNonceChange:
			journal[i] = &DbTransactionJournal{
				Payload: &DbTransactionJournal_NonceChange_{NonceChange: &DbTransactionJournal_NonceChange{
					ScriptPublicKey: ScriptPublicKeyToDBScriptPublicKey(p.ScriptPublicKey),
					PreviousNonce:   p.PreviousNonce,
					NewNonce:        p.NewNonce,
				}},
			}
		case *externalapi.DomainTransactionJournalStorageChange:
			journal[i] = &DbTransactionJournal{
				Payload: &DbTransactionJournal_StorageChange_{StorageChange: &DbTransactionJournal_StorageChange{
					ScriptPublicKey: ScriptPublicKeyToDBScriptPublicKey(p.ScriptPublicKey),
					Key:             DomainHashToDbHash(&p.Key),
					PreviousValue:   p.PreviousValue,
					NewValue:        p.NewValue,
				}},
			}
		}
	}

	return &DbTransaction{
		Version:      uint32(domainTransaction.Version),
		Inputs:       dbInputs,
		Outputs:      dbOutputs,
		LockTime:     domainTransaction.LockTime,
		SubnetworkID: DomainSubnetworkIDToDbSubnetworkID(&domainTransaction.SubnetworkID),
		Gas:          domainTransaction.Gas,
		Payload:      domainTransaction.Payload,
		Logs:         logs,
		Journal:      journal,
		Result:       domainTransaction.Result,
	}
}

// DbTransactionToDomainTransaction converts DbTransaction to DomainTransaction
func DbTransactionToDomainTransaction(dbTransaction *DbTransaction) (*externalapi.DomainTransaction, error) {
	domainSubnetworkID, err := DbSubnetworkIDToDomainSubnetworkID(dbTransaction.SubnetworkID)
	if err != nil {
		return nil, err
	}

	domainInputs := make([]*externalapi.DomainTransactionInput, len(dbTransaction.Inputs))
	for i, dbTransactionInput := range dbTransaction.Inputs {
		domainPreviousOutpoint, err := DbOutpointToDomainOutpoint(dbTransactionInput.PreviousOutpoint)
		if err != nil {
			return nil, err
		}
		domainInputs[i] = &externalapi.DomainTransactionInput{
			PreviousOutpoint: *domainPreviousOutpoint,
			SignatureScript:  dbTransactionInput.SignatureScript,
			Sequence:         dbTransactionInput.Sequence,
			SigOpCount:       byte(dbTransactionInput.SigOpCount),
		}
	}

	domainOutputs := make([]*externalapi.DomainTransactionOutput, len(dbTransaction.Outputs))
	for i, dbTransactionOutput := range dbTransaction.Outputs {
		scriptPublicKey, err := DBScriptPublicKeyToScriptPublicKey(dbTransactionOutput.ScriptPublicKey)
		if err != nil {
			return nil, err
		}
		domainOutputs[i] = &externalapi.DomainTransactionOutput{
			Value:           dbTransactionOutput.Value,
			ScriptPublicKey: scriptPublicKey,
		}
	}

	logs := make([]*externalapi.DomainTransactionLog, len(dbTransaction.Logs))
	for i, dbLog := range dbTransaction.Logs {
		topics := make([]externalapi.DomainHash, len(dbLog.Topics))
		for j, topic := range dbLog.Topics {
			hash, _ := DbHashToDomainHash(topic)
			topics[j] = *hash
		}

		scriptPublicKey, _ := DBScriptPublicKeyToScriptPublicKey(dbLog.ScriptPublicKey)
		logs[i] = &externalapi.DomainTransactionLog{
			ScriptPublicKey: scriptPublicKey,
			Topics:          topics,
			Data:            dbLog.Data,
			Index:           dbLog.Index,
		}
	}

	journal := make([]externalapi.DomainTransactionJournal, len(dbTransaction.Journal))
	for i, dbJournal := range dbTransaction.Journal {
		switch p := dbJournal.Payload.(type) {
		case *DbTransactionJournal_CreateObjectChange_:
			scriptPublicKey, _ := DBScriptPublicKeyToScriptPublicKey(p.CreateObjectChange.ScriptPublicKey)
			journal[i] = &externalapi.DomainTransactionJournalCreateObjectChange{
				ScriptPublicKey: scriptPublicKey,
			}
		case *DbTransactionJournal_NonceChange_:
			scriptPublicKey, _ := DBScriptPublicKeyToScriptPublicKey(p.NonceChange.ScriptPublicKey)
			journal[i] = &externalapi.DomainTransactionJournalNonceChange{
				ScriptPublicKey: scriptPublicKey,
				PreviousNonce:   p.NonceChange.PreviousNonce,
				NewNonce:        p.NonceChange.NewNonce,
			}
		case *DbTransactionJournal_StorageChange_:
			key, _ := DbHashToDomainHash(p.StorageChange.Key)
			scriptPublicKey, _ := DBScriptPublicKeyToScriptPublicKey(p.StorageChange.ScriptPublicKey)
			journal[i] = &externalapi.DomainTransactionJournalStorageChange{
				ScriptPublicKey: scriptPublicKey,
				Key:             *key,
				PreviousValue:   p.StorageChange.PreviousValue,
				NewValue:        p.StorageChange.NewValue,
			}
		}
	}

	if dbTransaction.Version > math.MaxUint16 {
		return nil, errors.Errorf("The transaction version is bigger then uint16.")
	}
	return &externalapi.DomainTransaction{
		Version:      uint16(dbTransaction.Version),
		Inputs:       domainInputs,
		Outputs:      domainOutputs,
		LockTime:     dbTransaction.LockTime,
		SubnetworkID: *domainSubnetworkID,
		Gas:          dbTransaction.Gas,
		Payload:      dbTransaction.Payload,
		Logs:         logs,
		Journal:      journal,
		Result:       dbTransaction.Result,
	}, nil
}
