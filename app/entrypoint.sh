#!/bin/bash
set -e

go test -v
go run $(find . -name "*.go" -and -not -name "*_test.go" -maxdepth 1)

$@
