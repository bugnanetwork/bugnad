#!/bin/bash

APPDIR=/tmp/bugnad-temp
BGAPAD_RPC_PORT=29587

rm -rf "${APPDIR}"

bugnad --simnet --appdir="${APPDIR}" --rpclisten=0.0.0.0:"${BGAPAD_RPC_PORT}" --profile=6061 &
BGAPAD_PID=$!

sleep 1

RUN_STABILITY_TESTS=true go test ../ -v -timeout 86400s -- --rpc-address=127.0.0.1:"${BGAPAD_RPC_PORT}" --profile=7000
TEST_EXIT_CODE=$?

kill $BGAPAD_PID

wait $BGAPAD_PID
BGAPAD_EXIT_CODE=$?

echo "Exit code: $TEST_EXIT_CODE"
echo "Bugnad exit code: $BGAPAD_EXIT_CODE"

if [ $TEST_EXIT_CODE -eq 0 ] && [ $BGAPAD_EXIT_CODE -eq 0 ]; then
  echo "mempool-limits test: PASSED"
  exit 0
fi
echo "mempool-limits test: FAILED"
exit 1
