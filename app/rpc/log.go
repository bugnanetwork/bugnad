package rpc

import (
	"github.com/bugnanetwork/bugnad/infrastructure/logger"
	"github.com/bugnanetwork/bugnad/util/panics"
)

var log = logger.RegisterSubSystem("RPCS")
var spawn = panics.GoroutineWrapperFunc(log)
