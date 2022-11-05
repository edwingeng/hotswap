#!/usr/bin/env bash

[[ "$TRACE" ]] && set -x
OWD=`pwd`
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

PROGRAM="slink"
PROGRAM_BUILD_OUTPUT_DIR="bin/$xOS/$PROGRAM"
PROGRAM_EXE="bin/$xOS/$PROGRAM"

read -p "Link plugin statically (not reloadable)? [y|n] " -r
if ! [[ $REPLY =~ ^[YyNn]$ ]]; then
    echo "Invalid input."
    exit 1
fi

staticLinking=
if [[ $REPLY =~ ^[Yy]$ ]]; then
    plugin/dog/slink.sh
    [[ $? -ne 0 ]] && exit 1
    staticLinking='--staticLinking'
    echo
fi

printf "Building $PROGRAM...\n"
echo

CGO_ENABLED=1 GOARCH=amd64 go build -trimpath -o "$PROGRAM_BUILD_OUTPUT_DIR"
[[ $? -ne 0 ]] && exit 1

if [[ $REPLY =~ ^[Nn]$ ]]; then
    plugin/dog/build.sh
    [[ $? -ne 0 ]] && exit 1
    echo
fi

signalFile="bin/$xOS/$PROGRAM.reload"
if [[ -f "$signalFile" ]]; then
    rm "$signalFile"
fi

printf "Starting $PROGRAM...\n\n"
"$PROGRAM_EXE" --pluginDir="bin/$xOS/plugin/$PROGRAM" --pidFile="bin/$xOS/$PROGRAM.pid" --signalFile="$signalFile" "$staticLinking"
