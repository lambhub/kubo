#!/usr/bin/env bash

SOURCE="${BASH_SOURCE[0]}"
while [ -h "$SOURCE" ] ; do SOURCE="$(readlink "$SOURCE")"; done
DIR="$( cd -P "$( dirname "$SOURCE" )/.." && pwd )"

# Change into that directory
cd "$DIR"

echo "==> Removing old directory..."
rm -f build/*
mkdir -p build/

FLAGS=$1
go build $FLAGS -o build/tower cmd/tower/main.go

echo "==> Results:"
ls -hl build/