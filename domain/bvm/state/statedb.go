// Copyright 2014 The go-ethereum Authors
// This file is part of the go-ethereum library.
//
// The go-ethereum library is free software: you can redistribute it and/or modify
// it under the terms of the GNU Lesser General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// The go-ethereum library is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
// GNU Lesser General Public License for more details.
//
// You should have received a copy of the GNU Lesser General Public License
// along with the go-ethereum library. If not, see <http://www.gnu.org/licenses/>.

// Package state provides a caching layer atop the Ethereum state trie.
package state

import (
	"fmt"
	"sort"
	"strings"

	"github.com/bugnanetwork/bugnad/domain/bvm/vm"
	"github.com/bugnanetwork/bugnad/domain/consensus/model/externalapi"
)

// StateDBs within the ethereum protocol are used to store anything
// within the merkle trie. StateDBs take care of caching and storing
// nested states. It's the general query interface to retrieve:
// * Contracts
// * Accounts
type StateDB struct {
	db Database

	// This map holds 'live' objects, which will get modified while processing a state transition.
	stateObjects        map[vm.Address]*stateObject
	stateObjectsPending map[vm.Address]struct{} // State objects finalized but not yet written to the trie
	stateObjectsDirty   map[vm.Address]struct{} // State objects modified in the current execution

	// DB error.
	// State objects are used by the consensus core and VM which are
	// unable to deal with database-level errors. Any error that occurs
	// during a database read is memoized here and will eventually be returned
	// by StateDB.Commit.
	dbErr error

	thash   vm.Hash
	txIndex int
	logs    map[vm.Hash][]*vm.Log
	logSize uint

	preimages map[vm.Hash][]byte

	// Journal of state modifications. This is the backbone of
	// Snapshot and RevertToSnapshot.
	journal        *journal
	validRevisions []revision
	nextRevisionId int
}

// Create a new state from a given trie.
func New(db Database) *StateDB {
	return &StateDB{
		db:                  db,
		stateObjects:        make(map[vm.Address]*stateObject),
		stateObjectsPending: make(map[vm.Address]struct{}),
		stateObjectsDirty:   make(map[vm.Address]struct{}),
		logs:                make(map[vm.Hash][]*vm.Log),
		preimages:           make(map[vm.Hash][]byte),
		journal:             newJournal(),
	}
}

// setError remembers the first non-nil error it is called with.
func (s *StateDB) setError(err error) {
	if s.dbErr == nil {
		s.dbErr = err
	}
}

func (s *StateDB) Error() error {
	return s.dbErr
}

func (s *StateDB) Reset() {
	s.stateObjects = make(map[vm.Address]*stateObject)
	s.stateObjectsPending = make(map[vm.Address]struct{})
	s.stateObjectsDirty = make(map[vm.Address]struct{})
	s.thash = vm.Hash{}
	s.txIndex = 0
	s.logs = make(map[vm.Hash][]*vm.Log)
	s.logSize = 0
	s.preimages = make(map[vm.Hash][]byte)
	s.clearJournal()
}

func (s *StateDB) AddLog(log *vm.Log) {
	s.journal.append(addLogChange{txhash: s.thash})

	log.Index = s.logSize
	s.logs[s.thash] = append(s.logs[s.thash], log)
	s.logSize++
}

func (s *StateDB) GetLogs(hash vm.Hash) []*vm.Log {
	return s.logs[hash]
}

func (s *StateDB) Logs() []*vm.Log {
	var logs []*vm.Log
	for _, lgs := range s.logs {
		logs = append(logs, lgs...)
	}
	return logs
}

// AddPreimage records a SHA3 preimage seen by the VM.
func (s *StateDB) AddPreimage(hash vm.Hash, preimage []byte) {
	if _, ok := s.preimages[hash]; !ok {
		s.journal.append(addPreimageChange{hash: hash})
		pi := make([]byte, len(preimage))
		copy(pi, preimage)
		s.preimages[hash] = pi
	}
}

// Preimages returns a list of SHA3 preimages that have been submitted.
func (s *StateDB) Preimages() map[vm.Hash][]byte {
	return s.preimages
}

// Exist reports whether the given account address exists in the state.
// Notably this also returns true for suicided accounts.
func (s *StateDB) Exist(addr vm.Address) bool {
	return s.getStateObject(addr) != nil
}

// Empty returns whether the state object is either non-existent
// or empty according to the EIP161 specification (balance = nonce = code = 0)
func (s *StateDB) Empty(addr vm.Address) bool {
	so := s.getStateObject(addr)
	return so == nil || so.empty()
}

func (s *StateDB) GetNonce(addr vm.Address) uint64 {
	stateObject := s.getStateObject(addr)
	if stateObject != nil {
		return stateObject.Nonce()
	}

	return 0
}

// TxIndex returns the current transaction index set by Prepare.
func (s *StateDB) TxIndex() int {
	return s.txIndex
}

func (s *StateDB) GetCode(addr vm.Address) []byte {
	stateObject := s.getStateObject(addr)
	if stateObject != nil {
		return stateObject.Code()
	}
	return nil
}

func (s *StateDB) GetCodeSize(addr vm.Address) int {
	stateObject := s.getStateObject(addr)
	if stateObject == nil {
		return 0
	}
	if stateObject.code != nil {
		return len(stateObject.code)
	}

	code := stateObject.Code()
	return len(code)
}

func (s *StateDB) GetCodeHash(addr vm.Address) vm.Hash {
	stateObject := s.getStateObject(addr)
	if stateObject == nil {
		return vm.Hash{}
	}
	return vm.BytesToHash(stateObject.CodeHash())
}

// GetState retrieves a value from the given account's storage trie.
func (s *StateDB) GetState(addr vm.Address, hash vm.Hash) vm.Hash {
	stateObject := s.getStateObject(addr)
	if stateObject != nil {
		return stateObject.GetState(hash)
	}
	return vm.Hash{}
}

// GetCommittedState retrieves a value from the given account's committed storage trie.
func (s *StateDB) GetCommittedState(addr vm.Address, hash vm.Hash) vm.Hash {
	stateObject := s.getStateObject(addr)
	if stateObject != nil {
		return stateObject.GetCommittedState(hash)
	}
	return vm.Hash{}
}

func (s *StateDB) HasSuicided(addr vm.Address) bool {
	stateObject := s.getStateObject(addr)
	if stateObject != nil {
		return stateObject.suicided
	}
	return false
}

/*
 * SETTERS
 */

func (s *StateDB) SetNonce(addr vm.Address, nonce uint64) {
	stateObject := s.GetOrNewStateObject(addr)
	if stateObject != nil {
		stateObject.SetNonce(nonce)
	}
}

func (s *StateDB) SetCode(addr vm.Address, code []byte) {
	stateObject := s.GetOrNewStateObject(addr)
	if stateObject != nil {
		stateObject.SetCode(vm.Keccak256Hash(code), code)
	}
}

func (s *StateDB) SetState(addr vm.Address, key, value vm.Hash) {
	stateObject := s.GetOrNewStateObject(addr)
	if stateObject != nil {
		stateObject.SetState(key, value)
	}
}

// SetStorage replaces the entire storage for the specified account with given
// storage. This function should only be used for debugging.
func (s *StateDB) SetStorage(addr vm.Address, storage map[vm.Hash]vm.Hash) {
	stateObject := s.GetOrNewStateObject(addr)
	if stateObject != nil {
		stateObject.SetStorage(storage)
	}
}

// Suicide marks the given account as suicided.
// This clears the account balance.
//
// The account's state object is still available until the state is committed,
// getStateObject will return a non-nil account after Suicide.
func (s *StateDB) Suicide(addr vm.Address) bool {
	stateObject := s.getStateObject(addr)
	if stateObject == nil {
		return false
	}
	s.journal.append(suicideChange{
		account: &addr,
		prev:    stateObject.suicided,
	})
	stateObject.markSuicided()

	return true
}

//
// Setting, updating & deleting state object methods.
//

// updateStateObject writes the given object to the trie.
func (s *StateDB) updateStateObject(obj *stateObject) {
	s.db.AddAccount(obj.Address(), &obj.data)
}

// deleteStateObject removes the given object from the state trie.
func (s *StateDB) deleteStateObject(obj *stateObject) {
	s.db.DeleteAccount(obj.Address(), &obj.data)
}

// getStateObject retrieves a state object given by the address, returning nil if
// the object is not found or was deleted in this execution context. If you need
// to differentiate between non-existent/just-deleted, use getDeletedStateObject.
func (s *StateDB) getStateObject(addr vm.Address) *stateObject {
	if obj := s.getDeletedStateObject(addr); obj != nil && !obj.deleted {
		return obj
	}
	return nil
}

// getDeletedStateObject is similar to getStateObject, but instead of returning
// nil for a deleted state object, it returns the actual object with the deleted
// flag set. This is needed by the state journal to revert to the correct s-
// destructed object instead of wiping all knowledge about the state object.
func (s *StateDB) getDeletedStateObject(addr vm.Address) *stateObject {
	// Prefer live objects if any is available
	if obj := s.stateObjects[addr]; obj != nil {
		return obj
	}

	data, err := s.db.GetAccount(addr)
	if err != nil {
		if !strings.Contains(err.Error(), "not found") {
			s.setError(err)
		}
		return nil
	}

	// Insert into the live set
	obj := newObject(s, addr, *data)
	s.setStateObject(obj)
	return obj
}

func (s *StateDB) setStateObject(object *stateObject) {
	s.stateObjects[object.Address()] = object
}

// Retrieve a state object or create a new state object if nil.
func (s *StateDB) GetOrNewStateObject(addr vm.Address) *stateObject {
	stateObject := s.getStateObject(addr)
	if stateObject == nil {
		stateObject, _ = s.createObject(addr)
	}
	return stateObject
}

// createObject creates a new state object. If there is an existing account with
// the given address, it is overwritten and returned as the second return value.
func (s *StateDB) createObject(addr vm.Address) (newobj, prev *stateObject) {
	prev = s.getDeletedStateObject(addr) // Note, prev might have been deleted, we need that!

	newobj = newObject(s, addr, Account{})
	newobj.setNonce(0) // sets the object to dirty
	newobj.setScriptPublicKey(addr.ScriptPublicKey())

	if prev == nil {
		s.journal.append(createObjectChange{account: &addr})
	} else {
		s.journal.append(resetObjectChange{prev: prev})
	}
	s.setStateObject(newobj)
	return newobj, prev
}

// CreateAccount explicitly creates a state object. If a state object with the address
// already exists the balance is carried over to the new account.
//
// CreateAccount is called during the EVM CREATE operation. The situation might arise that
// a contract does the following:
//
//  1. sends funds to sha(account ++ (nonce + 1))
//  2. tx_create(sha(account ++ nonce)) (note that this gets the address of 1)
func (s *StateDB) CreateAccount(addr vm.Address) {
	_, _ = s.createObject(addr)
}

// Copy creates a deep, independent copy of the state.
// Snapshots of the copied state cannot be applied to the copy.
func (s *StateDB) Copy() *StateDB {
	// Copy all the basic fields, initialize the memory ones
	state := &StateDB{
		db:                  s.db,
		stateObjects:        make(map[vm.Address]*stateObject, len(s.journal.dirties)),
		stateObjectsPending: make(map[vm.Address]struct{}, len(s.stateObjectsPending)),
		stateObjectsDirty:   make(map[vm.Address]struct{}, len(s.journal.dirties)),
		logs:                make(map[vm.Hash][]*vm.Log, len(s.logs)),
		logSize:             s.logSize,
		preimages:           make(map[vm.Hash][]byte, len(s.preimages)),
		journal:             newJournal(),
	}
	// Copy the dirty states, logs, and preimages
	for addr := range s.journal.dirties {
		// As documented [here](https://github.com/ethereum/go-ethereum/pull/16485#issuecomment-380438527),
		// and in the Finalise-method, there is a case where an object is in the journal but not
		// in the stateObjects: OOG after touch on ripeMD prior to Byzantium. Thus, we need to check for
		// nil
		if object, exist := s.stateObjects[addr]; exist {
			// Even though the original object is dirty, we are not copying the journal,
			// so we need to make sure that anyside effect the journal would have caused
			// during a commit (or similar op) is already applied to the copy.
			state.stateObjects[addr] = object.deepCopy(state)

			state.stateObjectsDirty[addr] = struct{}{}   // Mark the copy dirty to force internal (code/state) commits
			state.stateObjectsPending[addr] = struct{}{} // Mark the copy pending to force external (account) commits
		}
	}
	// Above, we don't copy the actual journal. This means that if the copy is copied, the
	// loop above will be a no-op, since the copy's journal is empty.
	// Thus, here we iterate over stateObjects, to enable copies of copies
	for addr := range s.stateObjectsPending {
		if _, exist := state.stateObjects[addr]; !exist {
			state.stateObjects[addr] = s.stateObjects[addr].deepCopy(state)
		}
		state.stateObjectsPending[addr] = struct{}{}
	}
	for addr := range s.stateObjectsDirty {
		if _, exist := state.stateObjects[addr]; !exist {
			state.stateObjects[addr] = s.stateObjects[addr].deepCopy(state)
		}
		state.stateObjectsDirty[addr] = struct{}{}
	}
	for hash, logs := range s.logs {
		cpy := make([]*vm.Log, len(logs))
		for i, l := range logs {
			cpy[i] = new(vm.Log)
			*cpy[i] = *l
		}
		state.logs[hash] = cpy
	}
	for hash, preimage := range s.preimages {
		state.preimages[hash] = preimage
	}
	return state
}

// Snapshot returns an identifier for the current revision of the state.
func (s *StateDB) Snapshot() int {
	id := s.nextRevisionId
	s.nextRevisionId++
	s.validRevisions = append(s.validRevisions, revision{id, s.journal.length()})
	return id
}

// RevertToSnapshot reverts all state changes made since the given revision.
func (s *StateDB) RevertToSnapshot(revid int) {
	// Find the snapshot in the stack of valid snapshots.
	idx := sort.Search(len(s.validRevisions), func(i int) bool {
		return s.validRevisions[i].id >= revid
	})
	if idx == len(s.validRevisions) || s.validRevisions[idx].id != revid {
		panic(fmt.Errorf("revision id %v cannot be reverted", revid))
	}
	snapshot := s.validRevisions[idx].journalIndex

	// Replay the journal to undo changes and remove invalidated snapshots
	s.journal.revert(s, snapshot)
	s.validRevisions = s.validRevisions[:idx]
}

func (s *StateDB) DumpJournal() []externalapi.DomainTransactionJournal {
	changes := make([]externalapi.DomainTransactionJournal, 0)
	for _, entry := range s.journal.entries {
		switch e := entry.(type) {
		case createObjectChange:
			changes = append(changes, &externalapi.DomainTransactionJournalCreateObjectChange{
				ScriptPublicKey: e.account.ScriptPublicKey(),
			})
		case nonceChange:
			changes = append(changes, &externalapi.DomainTransactionJournalNonceChange{
				ScriptPublicKey: e.account.ScriptPublicKey(),
				PreviousNonce:   e.prev,
				NewNonce:        e.prev + 1,
			})
		case storageChange:
			key, _ := externalapi.NewDomainHashFromByteSlice(e.key.Bytes())
			changes = append(changes, &externalapi.DomainTransactionJournalStorageChange{
				ScriptPublicKey: e.account.ScriptPublicKey(),
				Key:             *key,
				PreviousValue:   e.prevalue.Bytes(),
				NewValue:        e.value.Bytes(),
			})
		}
	}

	return changes
}

// Finalise finalises the state by removing the s destructed objects and clears
// the journal as well as the refunds. Finalise, however, will not push any updates
// into the tries just yet. Only IntermediateRoot or Commit will do that.
func (s *StateDB) Finalise(deleteEmptyObjects bool) {
	for addr := range s.journal.dirties {
		obj, exist := s.stateObjects[addr]
		if !exist {
			continue
		}
		if obj.suicided || (deleteEmptyObjects && obj.empty()) {
			obj.deleted = true
		} else {
			obj.finalise()
		}
		s.stateObjectsPending[addr] = struct{}{}
		s.stateObjectsDirty[addr] = struct{}{}
	}
	// Invalidate journal because reverting across transactions is not allowed.
	s.clearJournal()
}

// It is called in between transactions to get the root hash that
// goes into transaction receipts.
func (s *StateDB) IntermediateRoot(deleteEmptyObjects bool) {
	// Finalise all the dirty storage states and write them into the tries
	s.Finalise(deleteEmptyObjects)

	for addr := range s.stateObjectsPending {
		obj := s.stateObjects[addr]
		if obj.deleted {
			s.deleteStateObject(obj)
		} else {
			err := obj.Commit()
			if err != nil {
				s.setError(err)
			}
			s.updateStateObject(obj)
		}
	}
	if len(s.stateObjectsPending) > 0 {
		s.stateObjectsPending = make(map[vm.Address]struct{})
	}
}

// Prepare sets the current transaction hash and index and block hash which is
// used when the EVM emits new state logs.
func (s *StateDB) Prepare(thash vm.Hash, ti int) {
	s.thash = thash
	s.txIndex = ti
}

func (s *StateDB) clearJournal() {
	if len(s.journal.entries) > 0 {
		s.journal = newJournal()
	}
	s.validRevisions = s.validRevisions[:0] // Snapshots can be created without journal entires
}

// Commit writes the state to the underlying in-memory
func (s *StateDB) CommitAndFlush(deleteEmptyObjects bool) error {
	// Finalize any pending changes and merge everything into the tries
	s.IntermediateRoot(deleteEmptyObjects)

	// Commit Code
	for addr := range s.stateObjectsDirty {
		if obj := s.stateObjects[addr]; !obj.deleted {
			// Write any contract code associated with the state object
			if obj.code != nil && obj.dirtyCode {
				obj.dirtyCode = false
				s.db.WriteCode(vm.BytesToHash(obj.CodeHash()), obj.code)
			}
		}
	}

	if len(s.stateObjectsDirty) > 0 {
		s.stateObjectsDirty = make(map[vm.Address]struct{})
	}

	err := s.dbErr
	s.dbErr = nil

	s.Reset()

	return err
}
