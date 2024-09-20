package state

import (
	"bytes"
	"fmt"

	"github.com/bugnanetwork/bugnad/domain/bvm/vm"
	"github.com/bugnanetwork/bugnad/domain/consensus/model/externalapi"
)

type Code []byte

func (c Code) String() string {
	return string(c) //strings.Join(Disassemble(c), " ")
}

type Storage map[vm.Hash]vm.Hash

func (s Storage) String() (str string) {
	for key, value := range s {
		str += fmt.Sprintf("%X : %X\n", key, value)
	}

	return
}

func (s Storage) Copy() Storage {
	cpy := make(Storage)
	for key, value := range s {
		cpy[key] = value
	}

	return cpy
}

// stateObject represents an Ethereum account which is being modified.
//
// The usage pattern is as follows:
// First you need to obtain a state object.
// Account values can be accessed and modified through the object.
// Finally, call CommitTrie to write the modified storage trie into a database.
type stateObject struct {
	address vm.Address
	data    Account
	db      *StateDB

	// DB error.
	// State objects are used by the consensus core and VM which are
	// unable to deal with database-level errors. Any error that occurs
	// during a database read is memoized here and will eventually be returned
	// by StateDB.Commit.
	dbErr error

	// Write caches.
	code Code // contract bytecode, which gets set when code is loaded

	originStorage  Storage // Storage cache of original entries to dedup rewrites, reset for every transaction
	pendingStorage Storage // Storage entries that need to be flushed to disk, at the end of an entire block
	dirtyStorage   Storage // Storage entries that have been modified in the current transaction execution
	fakeStorage    Storage // Fake storage which constructed by caller for debugging purpose.

	// Cache flags.
	// When an object is marked suicided it will be delete from the trie
	// during the "update" phase of the state transition.
	dirtyCode bool // true if the code was updated
	suicided  bool
	deleted   bool
}

// empty returns whether the account is considered empty.
func (s *stateObject) empty() bool {
	return s.data.Nonce == 0 && bytes.Equal(s.data.CodeHash, vm.EmptyCodeHash.Bytes())
}

// Account is the Ethereum consensus representation of accounts.
// These objects are stored in the main account trie.
type Account struct {
	Nonce           uint64
	CodeHash        []byte
	ScriptPublicKey *externalapi.ScriptPublicKey
}

// newObject creates a state object.
func newObject(db *StateDB, address vm.Address, data Account) *stateObject {
	if data.CodeHash == nil {
		data.CodeHash = vm.EmptyCodeHash.Bytes()
	}
	return &stateObject{
		db:             db,
		address:        address,
		data:           data,
		originStorage:  make(Storage),
		pendingStorage: make(Storage),
		dirtyStorage:   make(Storage),
	}
}

// setError remembers the first non-nil error it is called with.
func (s *stateObject) setError(err error) {
	if s.dbErr == nil {
		s.dbErr = err
	}
}

func (s *stateObject) markSuicided() {
	s.suicided = true
}

func (s *stateObject) touch() {
	s.db.journal.append(touchChange{
		account: &s.address,
	})
}

// GetState retrieves a value from the account storage trie.
func (s *stateObject) GetState(key vm.Hash) vm.Hash {
	// If the fake storage is set, only lookup the state here(in the debugging mode)
	if s.fakeStorage != nil {
		return s.fakeStorage[key]
	}
	// If we have a dirty value for this state entry, return it
	value, dirty := s.dirtyStorage[key]
	if dirty {
		return value
	}
	// Otherwise return the entry's original value
	return s.GetCommittedState(key)
}

// GetCommittedState retrieves a value from the committed account storage trie.
func (s *stateObject) GetCommittedState(key vm.Hash) vm.Hash {
	// If the fake storage is set, only lookup the state here(in the debugging mode)
	if s.fakeStorage != nil {
		return s.fakeStorage[key]
	}
	// If we have a pending write or clean cached, return that
	if value, pending := s.pendingStorage[key]; pending {
		return value
	}
	if value, cached := s.originStorage[key]; cached {
		return value
	}

	enc, err := s.db.db.GetStorage(s.address, key)
	if err != nil {
		// s.setError(err)
		return vm.Hash{}
	}
	var value vm.Hash
	if len(enc) > 0 {
		value.SetBytes(enc)
	}
	s.originStorage[key] = value
	return value
}

// SetState updates a value in account storage.
func (s *stateObject) SetState(key, value vm.Hash) {
	// If the fake storage is set, put the temporary state update here.
	if s.fakeStorage != nil {
		s.fakeStorage[key] = value
		return
	}
	// If the new value is the same as old, don't set
	prev := s.GetState(key)
	if prev == value {
		return
	}
	// New value is different, update and journal the change
	s.db.journal.append(storageChange{
		account:  &s.address,
		key:      key,
		prevalue: prev,
		value:    value,
	})
	s.setState(key, value)
}

// SetStorage replaces the entire state storage with the given one.
//
// After this function is called, all original state will be ignored and state
// lookup only happens in the fake state storage.
//
// Note this function should only be used for debugging purpose.
func (s *stateObject) SetStorage(storage map[vm.Hash]vm.Hash) {
	// Allocate fake storage if it's nil.
	if s.fakeStorage == nil {
		s.fakeStorage = make(Storage)
	}
	for key, value := range storage {
		s.fakeStorage[key] = value
	}
	// Don't bother journal since this function should only be used for
	// debugging and the `fake` storage won't be committed to database.
}

func (s *stateObject) setState(key, value vm.Hash) {
	s.dirtyStorage[key] = value
}

// finalise moves all dirty storage slots into the pending area to be hashed or
// committed later. It is invoked at the end of every transaction.
func (s *stateObject) finalise() {
	for key, value := range s.dirtyStorage {
		s.pendingStorage[key] = value
	}
	if len(s.dirtyStorage) > 0 {
		s.dirtyStorage = make(Storage)
	}
}

func (s *stateObject) commit() {
	// Make sure all dirty slots are finalized into the pending storage area
	s.finalise()

	for key, value := range s.pendingStorage {
		// Skip noop changes, persist actual changes
		if value == s.originStorage[key] {
			continue
		}
		s.originStorage[key] = value

		if (value == vm.Hash{}) {
			s.db.db.DelStorage(s.address, key)
			continue
		}

		s.db.db.AddStorage(s.address, key, value.Bytes())
	}
	if len(s.pendingStorage) > 0 {
		s.pendingStorage = make(Storage)
	}
}

func (s *stateObject) Commit() error {
	s.commit()
	if s.dbErr != nil {
		return s.dbErr
	}
	return nil
}

func (s *stateObject) deepCopy(db *StateDB) *stateObject {
	stateObject := newObject(db, s.address, s.data)
	stateObject.code = s.code
	stateObject.dirtyStorage = s.dirtyStorage.Copy()
	stateObject.originStorage = s.originStorage.Copy()
	stateObject.pendingStorage = s.pendingStorage.Copy()
	stateObject.suicided = s.suicided
	stateObject.dirtyCode = s.dirtyCode
	stateObject.deleted = s.deleted
	return stateObject
}

//
// Attribute accessors
//

// Returns the address of the contract/account
func (s *stateObject) Address() vm.Address {
	return s.address
}

// Code returns the contract code associated with this object, if any.
func (s *stateObject) Code() []byte {
	if s.code != nil {
		return s.code
	}
	if bytes.Equal(s.CodeHash(), vm.EmptyCodeHash.Bytes()) {
		return nil
	}
	code := s.db.db.ReadCode(vm.BytesToHash(s.data.CodeHash))
	s.code = code
	return code
}

func (s *stateObject) SetCode(codeHash vm.Hash, code []byte) {
	s.db.journal.append(codeChange{
		account: &s.address,
	})
	s.setCode(codeHash, code)
}

func (s *stateObject) setCode(codeHash vm.Hash, code []byte) {
	s.code = code
	s.data.CodeHash = codeHash[:]
	s.dirtyCode = true
}

func (s *stateObject) SetNonce(nonce uint64) {
	s.db.journal.append(nonceChange{
		account: &s.address,
		prev:    s.data.Nonce,
	})
	s.setNonce(nonce)
}

func (s *stateObject) setNonce(nonce uint64) {
	s.data.Nonce = nonce
}

func (s *stateObject) setScriptPublicKey(scriptPublicKey *externalapi.ScriptPublicKey) {
	s.data.ScriptPublicKey = scriptPublicKey
}

func (s *stateObject) CodeHash() []byte {
	return s.data.CodeHash
}

func (s *stateObject) Nonce() uint64 {
	return s.data.Nonce
}

func (s *stateObject) ScriptPublicKey() *externalapi.ScriptPublicKey {
	return s.data.ScriptPublicKey
}
