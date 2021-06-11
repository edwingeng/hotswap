#!/usr/bin/env bash

[[ "$TRACE" ]] && set -x
pushd `dirname "$0"` > /dev/null
trap __EXIT EXIT

colorful=false
tput setaf 7 > /dev/null 2>&1
if [[ $? -eq 0 ]]; then
    colorful=true
fi

function __EXIT() {
    popd > /dev/null
}

function printError() {
    $colorful && tput setaf 1
    >&2 echo "Error: $@"
    $colorful && tput setaf 7
}

function printImportantMessage() {
    $colorful && tput setaf 3
    >&2 echo "$@"
    $colorful && tput setaf 7
}

function printUsage() {
    $colorful && tput setaf 3
    >&2 echo "$@"
    $colorful && tput setaf 7
}

xOS="linux"
if [[ $OSTYPE == darwin* ]]; then
    xOS="darwin"
fi

PROGRAM="extension"

mkdir -p "../../bin/$xOS/plugin/$PROGRAM"
[[ $? -ne 0 ]] && exit 1

compileTimeString="`date +%s`-$RANDOM"
go run ../../../../cli/hotswap build "$@" . "../../bin/$xOS/plugin/$PROGRAM" \
    -- -ldflags "-X main.CompileTimeString=$compileTimeString"
