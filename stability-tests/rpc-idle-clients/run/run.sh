#!/bin/bash
rm -rf /tmp/bugnad-temp

NUM_CLIENTS=128
bugnad --devnet --appdir=/tmp/bugnad-temp --profile=6061 --rpcmaxwebsockets=$NUM_CLIENTS &
BGAPAD_PID=$!
BGAPAD_KILLED=0
function killBugnadIfNotKilled() {
  if [ $BGAPAD_KILLED -eq 0 ]; then
    kill $BGAPAD_PID
  fi
}
trap "killBugnadIfNotKilled" EXIT

sleep 1

rpc-idle-clients --devnet --profile=7000 -n=$NUM_CLIENTS
TEST_EXIT_CODE=$?

kill $BGAPAD_PID

wait $BGAPAD_PID
BGAPAD_EXIT_CODE=$?
BGAPAD_KILLED=1

echo "Exit code: $TEST_EXIT_CODE"
echo "Bugnad exit code: $BGAPAD_EXIT_CODE"

if [ $TEST_EXIT_CODE -eq 0 ] && [ $BGAPAD_EXIT_CODE -eq 0 ]; then
  echo "rpc-idle-clients test: PASSED"
  exit 0
fi
echo "rpc-idle-clients test: FAILED"
exit 1
