package prefixmanager

import (
	"github.com/bugnanetwork/bugnad/infrastructure/logger"
	"github.com/bugnanetwork/bugnad/util/panics"
)

var log = logger.RegisterSubSystem("PRFX")
var spawn = panics.GoroutineWrapperFunc(log)
