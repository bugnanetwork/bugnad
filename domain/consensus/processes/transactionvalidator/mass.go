package transactionvalidator

import (
	"github.com/bugnanetwork/bugnad/domain/consensus/model/externalapi"
)

// PopulateMass calculates and populates the mass of the given transaction
func (v *transactionValidator) PopulateMass(transaction *externalapi.DomainTransaction) {
	if transaction.Mass != 0 {
		return
	}
	transaction.Mass = v.txMassCalculator.CalculateTransactionMass(transaction)
}
