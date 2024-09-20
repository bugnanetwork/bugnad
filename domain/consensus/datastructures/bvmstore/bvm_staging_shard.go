package bvmstore

import (
	"github.com/bugnanetwork/bugnad/domain/bvm/state"
	"github.com/bugnanetwork/bugnad/domain/bvm/vm"
	"github.com/bugnanetwork/bugnad/domain/consensus/model"
	"github.com/bugnanetwork/bugnad/domain/consensus/model/externalapi"
)

type bvmStagingShard struct {
	stateDB  *state.StateDB
	store    *bvmStore
	toAdd    map[model.DBKey][]byte
	toDelete map[model.DBKey]struct{}
}

func (ms *bvmStore) stagingShard(dbContext model.DBReader, stagingArea *model.StagingArea) *bvmStagingShard {
	return stagingArea.GetOrCreateShard(ms.shardID, func() model.StagingShard {
		rawDB := &rawDB{
			store:       ms,
			dbContext:   dbContext,
			stagingArea: stagingArea,
		}

		stateDB := state.New(rawDB)

		return &bvmStagingShard{
			stateDB:  stateDB,
			store:    ms,
			toAdd:    make(map[model.DBKey][]byte),
			toDelete: make(map[model.DBKey]struct{}),
		}
	}).(*bvmStagingShard)
}

func (mss *bvmStagingShard) Commit(dbTx model.DBTransaction) error {
	err := mss.stateDB.CommitAndFlush(true)
	if err != nil {
		return err
	}

	for hash, data := range mss.toAdd {
		err := dbTx.Put(hash, data)
		if err != nil {
			return err
		}
		k, err := externalapi.NewDomainHashFromByteSlice(vm.Keccak256(hash.Bytes()))
		if err != nil {
			return err
		}
		mss.store.cache.Add(k, data)
	}

	for key := range mss.toDelete {
		err := dbTx.Delete(key)
		if err != nil {
			return err
		}

		k, err := externalapi.NewDomainHashFromByteSlice(vm.Keccak256(key.Bytes()))
		if err != nil {
			return err
		}
		mss.store.cache.Remove(k)
	}

	return nil
}

func (mss *bvmStagingShard) isStaged() bool {
	return len(mss.toAdd) != 0 || len(mss.toDelete) != 0
}
