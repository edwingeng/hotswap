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

printImportantMessage "It takes minutes to finish the tests."

go build -o cli/hotswap/hotswap github.com/edwingeng/hotswap/cli/hotswap
[[ $? -ne 0 ]] && exit 1

cli/hotswap/hotswap build --staticLinking cli/hotswap/trial/arya cli/hotswap/trial
[[ $? -ne 0 ]] && exit 1
cli/hotswap/hotswap build --staticLinking cli/hotswap/trial/snow cli/hotswap/trial
[[ $? -ne 0 ]] && exit 1
cli/hotswap/hotswap build --staticLinking cli/hotswap/trial/stubborn cli/hotswap/trial
[[ $? -ne 0 ]] && exit 1

go test -trimpath -v "$@"
