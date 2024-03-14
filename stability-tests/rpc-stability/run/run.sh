#!/bin/bash
rm -rf /tmp/bugnad-temp

bugnad --devnet --appdir=/tmp/bugnad-temp --profile=6061 --loglevel=debug &
BGAPAD_PID=$!

sleep 1

rpc-stability --devnet -p commands.json --profile=7000
TEST_EXIT_CODE=$?

kill $BGAPAD_PID

wait $BGAPAD_PID
BGAPAD_EXIT_CODE=$?

echo "Exit code: $TEST_EXIT_CODE"
echo "Bugnad exit code: $BGAPAD_EXIT_CODE"

if [ $TEST_EXIT_CODE -eq 0 ] && [ $BGAPAD_EXIT_CODE -eq 0 ]; then
  echo "rpc-stability test: PASSED"
  exit 0
fi
echo "rpc-stability test: FAILED"
exit 1
