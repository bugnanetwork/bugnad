package model

import (
	"github.com/bugnanetwork/bugnad/domain/consensus/model/externalapi"
)

// TransactionProcessor processes Transaction
type TransactionProcessor interface {
	Excute(
		stagingArea *StagingArea,
		povBlockHash *externalapi.DomainHash,
		blockDaaScore uint64,
		tx *externalapi.DomainTransaction,
	) error
}
