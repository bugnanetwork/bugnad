package common

import (
	"fmt"
	"os"
	"sync/atomic"
	"syscall"
	"testing"

	"github.com/bugnanetwork/bugnad/domain/dagconfig"
)

// RunBugnadForTesting runs bugnad for testing purposes
func RunBugnadForTesting(t *testing.T, testName string, rpcAddress string) func() {
	appDir, err := TempDir(testName)
	if err != nil {
		t.Fatalf("TempDir: %s", err)
	}

	bugnadRunCommand, err := StartCmd("BGAPAD",
		"bugnad",
		NetworkCliArgumentFromNetParams(&dagconfig.DevnetParams),
		"--appdir", appDir,
		"--rpclisten", rpcAddress,
		"--loglevel", "debug",
	)
	if err != nil {
		t.Fatalf("StartCmd: %s", err)
	}
	t.Logf("Bugnad started with --appdir=%s", appDir)

	isShutdown := uint64(0)
	go func() {
		err := bugnadRunCommand.Wait()
		if err != nil {
			if atomic.LoadUint64(&isShutdown) == 0 {
				panic(fmt.Sprintf("Bugnad closed unexpectedly: %s. See logs at: %s", err, appDir))
			}
		}
	}()

	return func() {
		err := bugnadRunCommand.Process.Signal(syscall.SIGTERM)
		if err != nil {
			t.Fatalf("Signal: %s", err)
		}
		err = os.RemoveAll(appDir)
		if err != nil {
			t.Fatalf("RemoveAll: %s", err)
		}
		atomic.StoreUint64(&isShutdown, 1)
		t.Logf("Bugnad stopped")
	}
}
