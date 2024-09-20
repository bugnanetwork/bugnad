package main

import (
	"sync/atomic"

	"github.com/bugnanetwork/bugnad/infrastructure/os/signal"
	"github.com/bugnanetwork/bugnad/util/panics"
)

var shutdown int32 = 0

func main() {
	interrupt := signal.InterruptListener()
	defer panics.HandlePanic(log, "main", nil)

	cmd, cfg := parseCommandLine()

	var err error

	switch cmd {
	case deploySubCmd:
		deployCfg := cfg.(*deployConfig)
		err = deploySmartContract(deployCfg)
	case callContractSubCmd:
		callContractCfg := cfg.(*callContractConfig)
		err = callSmartContract(callContractCfg)
	case evmAddrSubCmd:
		evmAddrCfg := cfg.(*evmAddrConfig)
		err = evmaddress(evmAddrCfg)
	case queryContractSubCmd:
		queryContractCfg := cfg.(*queryContractConfig)
		err = queryContract(queryContractCfg)
	default:
		printErrorAndExit("Unknown command")
	}

	if err != nil {
		printErrorAndExit(err.Error())
	}

	<-interrupt
	atomic.AddInt32(&shutdown, 1)
}
