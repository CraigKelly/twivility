#!/bin/bash

# Source find_helpers.sh in the same dir as this script
SCRIPT_DIR="$( cd "$( dirname "${BASH_SOURCE[0]}" )" && pwd )"
cd $SCRIPT_DIR

# Configuration
export GOMAXPROCS=8
TIMEFILE=$SCRIPT_DIR/timing

# Setup color printing for our headers
source ~/bin/color_help.sh
function header() {
    printf "${YELLOW}${BRIGHT}$*${NORMAL}\n"
}

# Actual work
header TEST
go test -v -race
header BUILD
go build -v -o twivility *.go
header RUN
/usr/bin/time -v -o "$TIMEFILE" ./twivility $*
header TIMING
cat $TIMEFILE
