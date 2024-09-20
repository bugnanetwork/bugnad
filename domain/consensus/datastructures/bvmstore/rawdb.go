package bvmstore

import (
	"log/slog"

	"github.com/bugnanetwork/bugnad/domain/bvm/state"
	"github.com/bugnanetwork/bugnad/domain/bvm/vm"
	"github.com/bugnanetwork/bugnad/domain/consensus/model"
)

type rawDB struct {
	store       *bvmStore
	dbContext   model.DBReader
	stagingArea *model.StagingArea
}

var _ state.Database = (*rawDB)(nil)

func (s *rawDB) ReadCode(hash vm.Hash) []byte {
	key := s.store.codeKey(hash)
	codes, _ := s.store.get(s.dbContext, s.stagingArea, key)

	return codes
}

func (s *rawDB) WriteCode(hash vm.Hash, code []byte) {
	key := s.store.codeKey(hash)
	s.store.set(s.dbContext, s.stagingArea, key, code)
}

func (s *rawDB) DelStorage(addr vm.Address, key vm.Hash) {
	k := s.store.storageKey(addr, key)
	s.store.delete(s.dbContext, s.stagingArea, k)
}
func (s *rawDB) AddStorage(addr vm.Address, key vm.Hash, value []byte) {
	k := s.store.storageKey(addr, key)
	s.store.set(s.dbContext, s.stagingArea, k, value)
}
func (s *rawDB) GetStorage(addr vm.Address, key vm.Hash) ([]byte, error) {
	k := s.store.storageKey(addr, key)
	return s.store.get(s.dbContext, s.stagingArea, k)
}
func (s *rawDB) AddAccount(addr vm.Address, data *state.Account) {
	accountBytes, _ := s.store.serializeAccount(data)
	accountKey := s.store.accountKey(addr)
	s.store.set(s.dbContext, s.stagingArea, accountKey, accountBytes)
}
func (s *rawDB) DeleteAccount(addr vm.Address, data *state.Account) {
	accountKey := s.store.accountKey(addr)
	s.store.delete(s.dbContext, s.stagingArea, accountKey)

	// get all storage keys
	storagePrefix := s.store.bucket.Bucket(storageBucketName).Bucket(addr.Bytes())
	cursor, err := s.dbContext.Cursor(storagePrefix)
	if err != nil {
		slog.Error("Failed to create cursor", "error", err)
		return
	}

	for cursor.Next() {
		key, err := cursor.Key()
		if err != nil {
			slog.Error("Failed to get key", "error", err)
			return
		}

		s.store.delete(s.dbContext, s.stagingArea, key)
	}
}
func (s *rawDB) GetAccount(address vm.Address) (*state.Account, error) {
	accountKey := s.store.accountKey(address)
	accountBytes, err := s.store.get(s.dbContext, s.stagingArea, accountKey)
	if err != nil {
		return nil, err
	}
	return s.store.deserializeAccount(accountBytes)
}
