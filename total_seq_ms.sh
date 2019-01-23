#!/bin/bash

set -eu

LOG_FILE=${LOG_FILE:-results.log}
awk '/took [0-9]+\.[0-9]+ms$/{sum+=$5} /took [0-9]+\.[0-9]+s$/{sum+=$5*1000}END{print sum}' $LOG_FILE
