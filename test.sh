#!/bin/bash

go get github.com/agnivade/wasmbrowsertest

GOOS=js GOARCH=wasm go test ./cmd/hpademo -exec=~/go/bin/wasmbrowsertest
