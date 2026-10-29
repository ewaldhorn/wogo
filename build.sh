#!/bin/sh
set -e

cd "$(dirname "$0")"

# let's be strict on formatting
echo "==> gofmt"
unformatted="$(gofmt -l .)"
if [ -n "$unformatted" ]; then
	echo "gofmt: unformatted files (run gofmt -w):" >&2
	echo "$unformatted" >&2
	exit 1
fi

echo "==> go vet"
go vet ./...

echo "==> go test"
go test ./...

echo "==> go build"
go build -o wogo .
echo "built ./wogo"
