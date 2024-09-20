package bvmstore

import (
	"github.com/bugnanetwork/bugnad/domain/bvm/state"
	"github.com/bugnanetwork/bugnad/domain/bvm/vm"
	"github.com/bugnanetwork/bugnad/domain/consensus/database/serialization"
	"github.com/bugnanetwork/bugnad/domain/consensus/model"
	"github.com/bugnanetwork/bugnad/domain/consensus/model/externalapi"
	"github.com/bugnanetwork/bugnad/domain/consensus/utils/lrucache"
	"github.com/bugnanetwork/bugnad/util/staging"
	"github.com/golang/protobuf/proto"
)

var (
	bucketName        = []byte("bvm")
	accountBucketName = []byte("accounts")
	codeBucketName    = []byte("codes")
	storageBucketName = []byte("storage")
)

// bvmStore represents a store of bvms
type bvmStore struct {
	shardID model.StagingShardID
	cache   *lrucache.LRUCache
	bucket  model.DBBucket
}

// New instantiates a new bvmStore
func New(prefixBucket model.DBBucket, cacheSize int, preallocate bool) model.BVMStore {
	return &bvmStore{
		shardID: staging.GenerateShardingID(),
		cache:   lrucache.New(cacheSize, preallocate),
		bucket:  prefixBucket.Bucket(bucketName),
	}
}

func (ms *bvmStore) StateDBWrapper(dbContext model.DBReader, stagingArea *model.StagingArea) vm.StateDB {
	return ms.stagingShard(dbContext, stagingArea).stateDB
}

func (ms *bvmStore) IsStaged(dbContext model.DBReader, stagingArea *model.StagingArea) bool {
	return ms.stagingShard(dbContext, stagingArea).isStaged()
}

// Get gets the bvm associated with the given blockHash
func (ms *bvmStore) get(dbContext model.DBReader, stagingArea *model.StagingArea, key model.DBKey) ([]byte, error) {
	stagingShard := ms.stagingShard(dbContext, stagingArea)

	if value, ok := stagingShard.toAdd[key]; ok {
		return value, nil
	}

	k, err := externalapi.NewDomainHashFromByteSlice(vm.Keccak256(key.Bytes()))
	if err != nil {
		return nil, err
	}

	if value, ok := ms.cache.Get(k); ok {
		return value.([]byte), nil
	}

	value, err := dbContext.Get(key)
	if err != nil {
		return nil, err
	}

	ms.cache.Add(k, value)
	return value, nil
}

// Delete deletes the bvm associated with the given blockHash
func (ms *bvmStore) delete(dbContext model.DBReader, stagingArea *model.StagingArea, key model.DBKey) {
	stagingShard := ms.stagingShard(dbContext, stagingArea)

	if _, ok := stagingShard.toAdd[key]; ok {
		delete(stagingShard.toAdd, key)
		return
	}
	stagingShard.toDelete[key] = struct{}{}
}

func (ms *bvmStore) set(dbContext model.DBReader, stagingArea *model.StagingArea, key model.DBKey, value []byte) {
	stagingShard := ms.stagingShard(dbContext, stagingArea)
	stagingShard.toAdd[key] = value
}

func (ms *bvmStore) codeKey(hash vm.Hash) model.DBKey {
	return ms.bucket.Bucket(codeBucketName).Key(hash.Bytes())
}

func (ms *bvmStore) storageKey(addr vm.Address, key vm.Hash) model.DBKey {
	return ms.bucket.
		Bucket(storageBucketName).
		Bucket(addr.Bytes()).
		Key(key.Bytes())
}

func (ms *bvmStore) accountKey(addr vm.Address) model.DBKey {
	return ms.bucket.Bucket(accountBucketName).Key(addr.Bytes())
}

func (ms *bvmStore) serializeAccount(account *state.Account) ([]byte, error) {
	return proto.Marshal(serialization.AccountToDBAccount(account))
}

func (ms *bvmStore) deserializeAccount(accountBytes []byte) (*state.Account, error) {
	dbaccount := &serialization.DbAccount{}
	err := proto.Unmarshal(accountBytes, dbaccount)
	if err != nil {
		return nil, err
	}

	return serialization.DBAccountToAccount(dbaccount)
}
