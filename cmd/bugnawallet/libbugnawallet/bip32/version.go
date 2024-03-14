package bip32

import "github.com/pkg/errors"

// BitcoinMainnetPrivate is the version that is used for
// bitcoin mainnet bip32 private extended keys.
// Ecnodes to xprv in base58.
var BitcoinMainnetPrivate = [4]byte{
	0x04,
	0x88,
	0xad,
	0xe4,
}

// BitcoinMainnetPublic is the version that is used for
// bitcoin mainnet bip32 public extended keys.
// Ecnodes to xpub in base58.
var BitcoinMainnetPublic = [4]byte{
	0x04,
	0x88,
	0xb2,
	0x1e,
}

// BugnaMainnetPrivate is the version that is used for
// bugna mainnet bip32 private extended keys.
// Ecnodes to xprv in base58.
var BugnaMainnetPrivate = [4]byte{
	0x03,
	0x8f,
	0x2e,
	0xf4,
}

// BugnaMainnetPublic is the version that is used for
// bugna mainnet bip32 public extended keys.
// Ecnodes to kpub in base58.
var BugnaMainnetPublic = [4]byte{
	0x03,
	0x8f,
	0x33,
	0x2e,
}

// BugnaTestnetPrivate is the version that is used for
// bugna testnet bip32 public extended keys.
// Ecnodes to ktrv in base58.
var BugnaTestnetPrivate = [4]byte{
	0x03,
	0x90,
	0x9e,
	0x07,
}

// BugnaTestnetPublic is the version that is used for
// bugna testnet bip32 public extended keys.
// Ecnodes to ktub in base58.
var BugnaTestnetPublic = [4]byte{
	0x03,
	0x90,
	0xa2,
	0x41,
}

// BugnaDevnetPrivate is the version that is used for
// bugna devnet bip32 public extended keys.
// Ecnodes to kdrv in base58.
var BugnaDevnetPrivate = [4]byte{
	0x03,
	0x8b,
	0x3d,
	0x80,
}

// BugnaDevnetPublic is the version that is used for
// bugna devnet bip32 public extended keys.
// Ecnodes to xdub in base58.
var BugnaDevnetPublic = [4]byte{
	0x03,
	0x8b,
	0x41,
	0xba,
}

// BugnaSimnetPrivate is the version that is used for
// bugna simnet bip32 public extended keys.
// Ecnodes to ksrv in base58.
var BugnaSimnetPrivate = [4]byte{
	0x03,
	0x90,
	0x42,
	0x42,
}

// BugnaSimnetPublic is the version that is used for
// bugna simnet bip32 public extended keys.
// Ecnodes to xsub in base58.
var BugnaSimnetPublic = [4]byte{
	0x03,
	0x90,
	0x46,
	0x7d,
}

func toPublicVersion(version [4]byte) ([4]byte, error) {
	switch version {
	case BitcoinMainnetPrivate:
		return BitcoinMainnetPublic, nil
	case BugnaMainnetPrivate:
		return BugnaMainnetPublic, nil
	case BugnaTestnetPrivate:
		return BugnaTestnetPublic, nil
	case BugnaDevnetPrivate:
		return BugnaDevnetPublic, nil
	case BugnaSimnetPrivate:
		return BugnaSimnetPublic, nil
	}

	return [4]byte{}, errors.Errorf("unknown version %x", version)
}

func isPrivateVersion(version [4]byte) bool {
	switch version {
	case BitcoinMainnetPrivate:
		return true
	case BugnaMainnetPrivate:
		return true
	case BugnaTestnetPrivate:
		return true
	case BugnaDevnetPrivate:
		return true
	case BugnaSimnetPrivate:
		return true
	}

	return false
}
