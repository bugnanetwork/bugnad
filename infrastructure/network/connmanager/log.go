package connmanager

import (
	"github.com/bugnanetwork/bugnad/infrastructure/logger"
	"github.com/bugnanetwork/bugnad/util/panics"
)

var log = logger.RegisterSubSystem("CMGR")
var spawn = panics.GoroutineWrapperFunc(log)
