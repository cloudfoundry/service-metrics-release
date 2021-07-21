#!/bin/bash

set -eo pipefail

time=$(date +%s%N)
cf install-plugin "log-cache" -f
sleep 60
cf tail service-metrics -t gauge -n 1000 --start-time="${time}" | grep service_dummy


