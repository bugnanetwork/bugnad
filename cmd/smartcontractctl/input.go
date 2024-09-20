package main

import (
	"github.com/bugnanetwork/bugnad/util"
)

type SmartContractInput interface {
	Address() util.Address
	RedeemScript(signature []byte) ([]byte, error)
}
