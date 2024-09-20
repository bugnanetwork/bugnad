package main

import (
	"github.com/bugnanetwork/bugnad/domain/bvm/vm"
	"github.com/bugnanetwork/bugnad/domain/consensus/utils/txscript"
	"github.com/bugnanetwork/bugnad/util"
)

func evmaddress(cfg *evmAddrConfig) error {
	addr, err := util.DecodeAddress(cfg.BugnaAddress, cfg.ActiveNetParams.Prefix)
	if err != nil {
		return err
	}

	scriptPubKey, _ := txscript.PayToAddrScript(addr)
	evmAddr := vm.ScriptPubkeyToAddress(scriptPubKey)

	log.Infof("EVM address: %x", evmAddr)
	return nil
}
