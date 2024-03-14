#!/bin/bash
rm -rf /tmp/bugnad-temp

bugnad --devnet --appdir=/tmp/bugnad-temp --profile=6061 &
BGAPAD_PID=$!

sleep 1

infra-level-garbage --devnet -alocalhost:16611 -m messages.dat --profile=7000
TEST_EXIT_CODE=$?

kill $BGAPAD_PID

wait $BGAPAD_PID
BGAPAD_EXIT_CODE=$?

echo "Exit code: $TEST_EXIT_CODE"
echo "Bugnad exit code: $BGAPAD_EXIT_CODE"

if [ $TEST_EXIT_CODE -eq 0 ] && [ $BGAPAD_EXIT_CODE -eq 0 ]; then
  echo "infra-level-garbage test: PASSED"
  exit 0
fi
echo "infra-level-garbage test: FAILED"
exit 1
