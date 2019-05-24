#!/bin/bash -eu
set -o pipefail

ginkgo -randomizeSuites=true -randomizeAllSpecs=true -keepGoing=true -r "$@"
