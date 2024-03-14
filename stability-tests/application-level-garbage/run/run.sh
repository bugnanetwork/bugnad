#!/bin/bash
rm -rf /tmp/bugnad-temp

bugnad --devnet --appdir=/tmp/bugnad-temp --profile=6061 --loglevel=debug &
BGAPAD_PID=$!
BGAPAD_KILLED=0
function killBugnadIfNotKilled() {
    if [ $BGAPAD_KILLED -eq 0 ]; then
      kill $BGAPAD_PID
    fi
}
trap "killBugnadIfNotKilled" EXIT

sleep 1

application-level-garbage --devnet -alocalhost:16611 -b blocks.dat --profile=7000
TEST_EXIT_CODE=$?

kill $BGAPAD_PID

wait $BGAPAD_PID
BGAPAD_KILLED=1
BGAPAD_EXIT_CODE=$?

echo "Exit code: $TEST_EXIT_CODE"
echo "Bugnad exit code: $BGAPAD_EXIT_CODE"

if [ $TEST_EXIT_CODE -eq 0 ] && [ $BGAPAD_EXIT_CODE -eq 0 ]; then
  echo "application-level-garbage test: PASSED"
  exit 0
fi
echo "application-level-garbage test: FAILED"
exit 1
