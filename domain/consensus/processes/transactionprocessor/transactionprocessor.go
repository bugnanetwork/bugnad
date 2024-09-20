package transactionprocessor

import (
	"github.com/bugnanetwork/bugnad/domain/consensus/model"
)

type transactionProcessor struct {
	databaseContext model.DBReader

	bvmStore model.BVMStore
}

// New instantiates a new TransactionProcessor
func New(
	databaseContext model.DBReader,
	bvmStore model.BVMStore,
) model.TransactionProcessor {

	return &transactionProcessor{
		databaseContext: databaseContext,
		bvmStore:        bvmStore,
	}
}
