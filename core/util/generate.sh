#!/usr/bin/env bash

function GET_GOPATH() {
  echo "$GOPATH"
}

GOPATH=$(GET_GOPATH)

echo $GOPATH

PREFIX="protoc --proto_path=..:.:$GOPATH/src/ --gogofaster_out=plugins=grpc:."

ignore_protos=()

generate() {
  for x in `ls ./*.proto | grep -v gogo`
  do
    local target="$PREFIX $x"
    echo $target
    eval $target
  done
}

$1
