package externalapi

import (
	"bytes"
	"encoding/binary"
	"fmt"

	"github.com/pkg/errors"
)

// DomainTransaction represents a Bugna transaction
type DomainTransaction struct {
	Version      uint16
	Inputs       []*DomainTransactionInput
	Outputs      []*DomainTransactionOutput
	LockTime     uint64
	SubnetworkID DomainSubnetworkID
	Gas          uint64
	Payload      []byte

	Fee  uint64
	Mass uint64

	Logs    []*DomainTransactionLog
	Journal []DomainTransactionJournal
	Result  string

	// ID is a field that is used to cache the transaction ID.
	// Always use consensushashing.TransactionID instead of accessing this field directly
	ID *DomainTransactionID
}

// Clone returns a clone of DomainTransaction
func (tx *DomainTransaction) Clone() *DomainTransaction {
	payloadClone := make([]byte, len(tx.Payload))
	copy(payloadClone, tx.Payload)

	inputsClone := make([]*DomainTransactionInput, len(tx.Inputs))
	for i, input := range tx.Inputs {
		inputsClone[i] = input.Clone()
	}

	outputsClone := make([]*DomainTransactionOutput, len(tx.Outputs))
	for i, output := range tx.Outputs {
		outputsClone[i] = output.Clone()
	}

	var idClone *DomainTransactionID
	if tx.ID != nil {
		idClone = tx.ID.Clone()
	}

	logsClone := make([]*DomainTransactionLog, len(tx.Logs))
	for i, log := range tx.Logs {
		logsClone[i] = log.Clone()
	}

	journalClone := make([]DomainTransactionJournal, len(tx.Journal))
	for i, journal := range tx.Journal {
		journalClone[i] = journal.Clone()
	}

	return &DomainTransaction{
		Version:      tx.Version,
		Inputs:       inputsClone,
		Outputs:      outputsClone,
		LockTime:     tx.LockTime,
		SubnetworkID: *tx.SubnetworkID.Clone(),
		Gas:          tx.Gas,
		Payload:      payloadClone,
		Fee:          tx.Fee,
		Mass:         tx.Mass,
		Logs:         logsClone,
		Journal:      journalClone,
		Result:       tx.Result,
		ID:           idClone,
	}
}

// If this doesn't compile, it means the type definition has been changed, so it's
// an indication to update Equal and Clone accordingly.
var _ = DomainTransaction{0, []*DomainTransactionInput{}, []*DomainTransactionOutput{}, 0,
	DomainSubnetworkID{}, 0, []byte{}, 0, 0,
	[]*DomainTransactionLog{},
	[]DomainTransactionJournal{},
	"",
	&DomainTransactionID{}}

// Equal returns whether tx equals to other
func (tx *DomainTransaction) Equal(other *DomainTransaction) bool {
	if tx == nil || other == nil {
		return tx == other
	}

	if tx.Version != other.Version {
		return false
	}

	if len(tx.Inputs) != len(other.Inputs) {
		return false
	}

	for i, input := range tx.Inputs {
		if !input.Equal(other.Inputs[i]) {
			return false
		}
	}

	if len(tx.Outputs) != len(other.Outputs) {
		return false
	}

	for i, output := range tx.Outputs {
		if !output.Equal(other.Outputs[i]) {
			return false
		}
	}

	if len(tx.Logs) != len(other.Logs) {
		return false
	}

	for i, log := range tx.Logs {
		if !log.Equal(other.Logs[i]) {
			return false
		}
	}

	if len(tx.Journal) != len(other.Journal) {
		return false
	}

	for i, journal := range tx.Journal {
		if !journal.Equal(other.Journal[i].(*DomainTransactionJournalCreateObjectChange)) {
			return false
		}
	}

	if tx.LockTime != other.LockTime {
		return false
	}

	if !tx.SubnetworkID.Equal(&other.SubnetworkID) {
		return false
	}

	if tx.Gas != other.Gas {
		return false
	}

	if !bytes.Equal(tx.Payload, other.Payload) {
		return false
	}

	if tx.Result != other.Result {
		return false
	}

	if tx.Fee != 0 && other.Fee != 0 && tx.Fee != other.Fee {
		panic(errors.New("identical transactions should always have the same fee"))
	}

	if tx.Mass != 0 && other.Mass != 0 && tx.Mass != other.Mass {
		panic(errors.New("identical transactions should always have the same mass"))
	}

	if tx.ID != nil && other.ID != nil && !tx.ID.Equal(other.ID) {
		panic(errors.New("identical transactions should always have the same ID"))
	}

	return true
}

// DomainTransactionInput represents a Bugna transaction input
type DomainTransactionInput struct {
	PreviousOutpoint DomainOutpoint
	SignatureScript  []byte
	Sequence         uint64
	SigOpCount       byte

	UTXOEntry UTXOEntry
}

// If this doesn't compile, it means the type definition has been changed, so it's
// an indication to update Equal and Clone accordingly.
var _ = &DomainTransactionInput{DomainOutpoint{}, []byte{}, 0, 0, nil}

// Equal returns whether input equals to other
func (input *DomainTransactionInput) Equal(other *DomainTransactionInput) bool {
	if input == nil || other == nil {
		return input == other
	}

	if !input.PreviousOutpoint.Equal(&other.PreviousOutpoint) {
		return false
	}

	if !bytes.Equal(input.SignatureScript, other.SignatureScript) {
		return false
	}

	if input.Sequence != other.Sequence {
		return false
	}

	if input.SigOpCount != other.SigOpCount {
		return false
	}

	if input.UTXOEntry != nil && other.UTXOEntry != nil && !input.UTXOEntry.Equal(other.UTXOEntry) {
		panic(errors.New("identical inputs should always have the same UTXO entry"))
	}

	return true
}

// Clone returns a clone of DomainTransactionInput
func (input *DomainTransactionInput) Clone() *DomainTransactionInput {
	signatureScriptClone := make([]byte, len(input.SignatureScript))
	copy(signatureScriptClone, input.SignatureScript)

	return &DomainTransactionInput{
		PreviousOutpoint: *input.PreviousOutpoint.Clone(),
		SignatureScript:  signatureScriptClone,
		Sequence:         input.Sequence,
		SigOpCount:       input.SigOpCount,
		UTXOEntry:        input.UTXOEntry,
	}
}

// DomainOutpoint represents a Bugna transaction outpoint
type DomainOutpoint struct {
	TransactionID DomainTransactionID
	Index         uint32
}

// If this doesn't compile, it means the type definition has been changed, so it's
// an indication to update Equal and Clone accordingly.
var _ = DomainOutpoint{DomainTransactionID{}, 0}

// Equal returns whether op equals to other
func (op *DomainOutpoint) Equal(other *DomainOutpoint) bool {
	if op == nil || other == nil {
		return op == other
	}

	return *op == *other
}

// Clone returns a clone of DomainOutpoint
func (op *DomainOutpoint) Clone() *DomainOutpoint {
	return &DomainOutpoint{
		TransactionID: *op.TransactionID.Clone(),
		Index:         op.Index,
	}
}

// String stringifies an outpoint.
func (op DomainOutpoint) String() string {
	return fmt.Sprintf("(%s: %d)", op.TransactionID, op.Index)
}

// NewDomainOutpoint instantiates a new DomainOutpoint with the given id and index
func NewDomainOutpoint(id *DomainTransactionID, index uint32) *DomainOutpoint {
	return &DomainOutpoint{
		TransactionID: *id,
		Index:         index,
	}
}

// ScriptPublicKey represents a Bugnad ScriptPublicKey
type ScriptPublicKey struct {
	Script  []byte
	Version uint16
}

// Equal returns whether spk equals to other
func (spk *ScriptPublicKey) Equal(other *ScriptPublicKey) bool {
	if spk == nil || other == nil {
		return spk == other
	}

	if spk.Version != other.Version {
		return false
	}

	return bytes.Equal(spk.Script, other.Script)
}

// String stringifies a ScriptPublicKey.
func (spk *ScriptPublicKey) String() string {
	var versionBytes = make([]byte, 2) // uint16
	binary.LittleEndian.PutUint16(versionBytes, spk.Version)
	versionString := string(versionBytes)
	scriptString := string(spk.Script)
	return versionString + scriptString
}

// NewScriptPublicKeyFromString converts the given string to a scriptPublicKey
func NewScriptPublicKeyFromString(ScriptPublicKeyString string) *ScriptPublicKey {
	bytes := []byte(ScriptPublicKeyString)
	version := binary.LittleEndian.Uint16(bytes[:2])
	script := bytes[2:]
	return &ScriptPublicKey{Script: script, Version: version}
}

// DomainTransactionOutput represents a Bugnad transaction output
type DomainTransactionOutput struct {
	Value           uint64
	ScriptPublicKey *ScriptPublicKey
}

// If this doesn't compile, it means the type definition has been changed, so it's
// an indication to update Equal and Clone accordingly.
var _ = DomainTransactionOutput{0, &ScriptPublicKey{Script: []byte{}, Version: 0}}

// Equal returns whether output equals to other
func (output *DomainTransactionOutput) Equal(other *DomainTransactionOutput) bool {
	if output == nil || other == nil {
		return output == other
	}

	if output.Value != other.Value {
		return false
	}

	return output.ScriptPublicKey.Equal(other.ScriptPublicKey)
}

// Clone returns a clone of DomainTransactionOutput
func (output *DomainTransactionOutput) Clone() *DomainTransactionOutput {
	scriptPublicKeyClone := &ScriptPublicKey{
		Script:  make([]byte, len(output.ScriptPublicKey.Script)),
		Version: output.ScriptPublicKey.Version}
	copy(scriptPublicKeyClone.Script, output.ScriptPublicKey.Script)

	return &DomainTransactionOutput{
		Value:           output.Value,
		ScriptPublicKey: scriptPublicKeyClone,
	}
}

// DomainTransactionID represents the ID of a Bugna transaction
type DomainTransactionID DomainHash

// NewDomainTransactionIDFromByteArray constructs a new TransactionID out of a byte array
func NewDomainTransactionIDFromByteArray(transactionIDBytes *[DomainHashSize]byte) *DomainTransactionID {
	return (*DomainTransactionID)(NewDomainHashFromByteArray(transactionIDBytes))
}

// NewDomainTransactionIDFromByteSlice constructs a new TransactionID out of a byte slice
// Returns an error if the length of the byte slice is not exactly `DomainHashSize`
func NewDomainTransactionIDFromByteSlice(transactionIDBytes []byte) (*DomainTransactionID, error) {
	hash, err := NewDomainHashFromByteSlice(transactionIDBytes)
	if err != nil {
		return nil, err
	}
	return (*DomainTransactionID)(hash), nil
}

// NewDomainTransactionIDFromString constructs a new TransactionID out of a string
// Returns an error if the length of the string is not exactly `DomainHashSize * 2`
func NewDomainTransactionIDFromString(transactionIDString string) (*DomainTransactionID, error) {
	hash, err := NewDomainHashFromString(transactionIDString)
	if err != nil {
		return nil, err
	}
	return (*DomainTransactionID)(hash), nil
}

// String stringifies a transaction ID.
func (id DomainTransactionID) String() string {
	return DomainHash(id).String()
}

// Clone returns a clone of DomainTransactionID
func (id *DomainTransactionID) Clone() *DomainTransactionID {
	idClone := *id
	return &idClone
}

// Equal returns whether id equals to other
func (id *DomainTransactionID) Equal(other *DomainTransactionID) bool {
	return (*DomainHash)(id).Equal((*DomainHash)(other))
}

// Less returns true if id is less than other
func (id *DomainTransactionID) Less(other *DomainTransactionID) bool {
	return (*DomainHash)(id).Less((*DomainHash)(other))
}

// LessOrEqual returns true if id is smaller or equal to other
func (id *DomainTransactionID) LessOrEqual(other *DomainTransactionID) bool {
	return (*DomainHash)(id).LessOrEqual((*DomainHash)(other))
}

// ByteArray returns the bytes in this transactionID represented as a byte array.
// The transactionID bytes are cloned, therefore it is safe to modify the resulting array.
func (id *DomainTransactionID) ByteArray() *[DomainHashSize]byte {
	return (*DomainHash)(id).ByteArray()
}

// ByteSlice returns the bytes in this transactionID represented as a byte slice.
// The transactionID bytes are cloned, therefore it is safe to modify the resulting slice.
func (id *DomainTransactionID) ByteSlice() []byte {
	return (*DomainHash)(id).ByteSlice()
}

type DomainTransactionLog struct {
	ScriptPublicKey *ScriptPublicKey
	Topics          []DomainHash
	Data            []byte
	Index           uint64
}

func (l *DomainTransactionLog) Clone() *DomainTransactionLog {
	scriptPublicKeyClone := &ScriptPublicKey{
		Script:  make([]byte, len(l.ScriptPublicKey.Script)),
		Version: l.ScriptPublicKey.Version}
	copy(scriptPublicKeyClone.Script, l.ScriptPublicKey.Script)

	return &DomainTransactionLog{
		ScriptPublicKey: scriptPublicKeyClone,
		Topics:          l.Topics,
		Data:            l.Data,
		Index:           l.Index,
	}
}

func (l *DomainTransactionLog) Equal(other *DomainTransactionLog) bool {
	if l == nil || other == nil {
		return l == other
	}

	if !l.ScriptPublicKey.Equal(other.ScriptPublicKey) {
		return false
	}

	if len(l.Topics) != len(other.Topics) {
		return false
	}

	for i, topic := range l.Topics {
		if !topic.Equal(&other.Topics[i]) {
			return false
		}
	}

	if !bytes.Equal(l.Data, other.Data) {
		return false
	}

	if l.Index != other.Index {
		return false
	}

	return true
}

type DomainTransactionJournal interface {
	isDomainTransactionJournal()
	Clone() DomainTransactionJournal
	Equal(other DomainTransactionJournal) bool
}

type DomainTransactionJournalCreateObjectChange struct {
	ScriptPublicKey *ScriptPublicKey
}

func (DomainTransactionJournalCreateObjectChange) isDomainTransactionJournal() {}

func (j *DomainTransactionJournalCreateObjectChange) Clone() DomainTransactionJournal {
	scriptPublicKeyClone := &ScriptPublicKey{
		Script:  make([]byte, len(j.ScriptPublicKey.Script)),
		Version: j.ScriptPublicKey.Version}
	copy(scriptPublicKeyClone.Script, j.ScriptPublicKey.Script)

	return &DomainTransactionJournalCreateObjectChange{
		ScriptPublicKey: scriptPublicKeyClone,
	}
}

func (j *DomainTransactionJournalCreateObjectChange) Equal(o DomainTransactionJournal) bool {
	other, ok := o.(*DomainTransactionJournalCreateObjectChange)
	if !ok {
		return false
	}

	if j == nil || other == nil {
		return j == other
	}

	return j.ScriptPublicKey.Equal(other.ScriptPublicKey)
}

type DomainTransactionJournalNonceChange struct {
	ScriptPublicKey *ScriptPublicKey
	PreviousNonce   uint64
	NewNonce        uint64
}

func (DomainTransactionJournalNonceChange) isDomainTransactionJournal() {}

func (j *DomainTransactionJournalNonceChange) Clone() DomainTransactionJournal {
	scriptPublicKeyClone := &ScriptPublicKey{
		Script:  make([]byte, len(j.ScriptPublicKey.Script)),
		Version: j.ScriptPublicKey.Version}
	copy(scriptPublicKeyClone.Script, j.ScriptPublicKey.Script)

	return &DomainTransactionJournalNonceChange{
		ScriptPublicKey: scriptPublicKeyClone,
		PreviousNonce:   j.PreviousNonce,
		NewNonce:        j.NewNonce,
	}
}

func (j *DomainTransactionJournalNonceChange) Equal(o DomainTransactionJournal) bool {
	other, ok := o.(*DomainTransactionJournalNonceChange)
	if !ok {
		return false
	}

	if j == nil || other == nil {
		return j == other
	}

	return j.ScriptPublicKey.Equal(other.ScriptPublicKey) && j.PreviousNonce == other.PreviousNonce && j.NewNonce == other.NewNonce
}

type DomainTransactionJournalStorageChange struct {
	ScriptPublicKey *ScriptPublicKey
	Key             DomainHash
	PreviousValue   []byte
	NewValue        []byte
}

func (DomainTransactionJournalStorageChange) isDomainTransactionJournal() {}

func (j *DomainTransactionJournalStorageChange) Clone() DomainTransactionJournal {
	scriptPublicKeyClone := &ScriptPublicKey{
		Script:  make([]byte, len(j.ScriptPublicKey.Script)),
		Version: j.ScriptPublicKey.Version}
	copy(scriptPublicKeyClone.Script, j.ScriptPublicKey.Script)

	previousValueClone := make([]byte, len(j.PreviousValue))
	copy(previousValueClone, j.PreviousValue)

	newValueClone := make([]byte, len(j.NewValue))
	copy(newValueClone, j.NewValue)

	return &DomainTransactionJournalStorageChange{
		ScriptPublicKey: scriptPublicKeyClone,
		Key:             j.Key,
		PreviousValue:   previousValueClone,
		NewValue:        newValueClone,
	}
}

func (j *DomainTransactionJournalStorageChange) Equal(o DomainTransactionJournal) bool {
	other, ok := o.(*DomainTransactionJournalStorageChange)
	if !ok {
		return false
	}

	if j == nil || other == nil {
		return j == other
	}

	return j.ScriptPublicKey.Equal(other.ScriptPublicKey) &&
		j.Key.Equal(&other.Key) &&
		bytes.Equal(j.PreviousValue, other.PreviousValue) &&
		bytes.Equal(j.NewValue, other.NewValue)
}
