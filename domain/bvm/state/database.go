package state

import "github.com/bugnanetwork/bugnad/domain/bvm/vm"

type Database interface {
	ReadCode(hash vm.Hash) []byte
	WriteCode(hash vm.Hash, code []byte)

	DelStorage(addr vm.Address, key vm.Hash)
	AddStorage(addr vm.Address, key vm.Hash, value []byte)
	GetStorage(addr vm.Address, key vm.Hash) ([]byte, error)

	AddAccount(addr vm.Address, data *Account)
	DeleteAccount(addr vm.Address, data *Account)
	GetAccount(address vm.Address) (*Account, error)
}
