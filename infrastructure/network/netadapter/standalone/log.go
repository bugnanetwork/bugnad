package standalone

import (
	"github.com/bugnanetwork/bugnad/infrastructure/logger"
	"github.com/bugnanetwork/bugnad/util/panics"
)

var log = logger.RegisterSubSystem("NTAR")
var spawn = panics.GoroutineWrapperFunc(log)
