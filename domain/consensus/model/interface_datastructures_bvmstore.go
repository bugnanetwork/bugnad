package model

import (
	"github.com/bugnanetwork/bugnad/domain/bvm/vm"
)

// MultisetStore represents a store of Multisets
type BVMStore interface {
	Store
	StateDBWrapper(dbContext DBReader, stagingArea *StagingArea) vm.StateDB
	IsStaged(dbContext DBReader, stagingArea *StagingArea) bool
}
