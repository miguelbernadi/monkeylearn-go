#!/bin/bash

set -eu

LOG_FILE=${LOG_FILE:-results.log}
echo "Sequential time: $(awk '/took [0-9]+\.[0-9]+ms$/{sum+=$5} /took [0-9]+\.[0-9]+s$/{sum+=$5*1000}END{print sum}' $LOG_FILE)"
echo "Runtime: $(awk '/^real .*[0-9]+\.[0-9]+s$/{print $2}' $LOG_FILE)"
echo "Rate limit hits: $(awk '/Request got ratelimited/{hit+=1}END{print hit}' $LOG_FILE)"
