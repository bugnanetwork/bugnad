package consensus

import (
	"github.com/bugnanetwork/bugnad/infrastructure/logger"
	"github.com/bugnanetwork/bugnad/util/panics"
)

var log = logger.RegisterSubSystem("BDAG")
var spawn = panics.GoroutineWrapperFunc(log)
