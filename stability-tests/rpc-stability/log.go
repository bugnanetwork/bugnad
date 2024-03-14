package main

import (
	"github.com/bugnanetwork/bugnad/infrastructure/logger"
	"github.com/bugnanetwork/bugnad/util/panics"
)

var (
	backendLog = logger.NewBackend()
	log        = backendLog.Logger("JSTT")
	spawn      = panics.GoroutineWrapperFunc(log)
)
