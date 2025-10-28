#!/bin/bash

go install golang.org/x/vuln/cmd/govulncheck@latest
go install golang.org/x/tools/cmd/deadcode@latest
go install github.com/mgechev/revive@latest

go env -w GOOS=js GOARCH=wasm

gofmt -s -w .

revive ./...

echo gocyclo begin
gocyclo -over 15 .
echo gocyclo end

go mod tidy

govulncheck ./...

deadcode ./cmd/*

#go env -w CGO_ENABLED=1
#go test -race ./...
#go env -w CGO_ENABLED=0

# wasm
GOOS=js GOARCH=wasm go build -o www/main.wasm ./cmd/hpademo

# javascript support file
cp "$(go env GOROOT)/lib/wasm/wasm_exec.js" www

#go env -u CGO_ENABLED
go env -u GOOS GOARCH
