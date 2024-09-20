package vm

import (
	"math/big"
)

// Context provides the EVM with auxiliary information. Once provided
// it shouldn't be modified.
type Context struct {
	// Message information
	Origin   Address  // Provides information for ORIGIN
	GasPrice *big.Int // Provides information for GASPRICE
	TxHash   Hash     // Provides information for TXHASH
	TxIndex  uint32

	// Block information
	GasLimit    uint64   // Provides information for GASLIMIT
	BlockNumber *big.Int // Provides information for NUMBER => DAA score
	Time        *big.Int // Provides information for TIME
	Difficulty  *big.Int // Provides information for DIFFICULTY
}
