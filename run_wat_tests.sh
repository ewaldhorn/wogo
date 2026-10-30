#!/bin/sh
set -e

# Runs all the WAT related tests by default
# Does not bother compiling to wasm, just parses to WAT

if [ $# -gt 0 ]; then
    go run ./cmd/testing -wat "$@"
else
    go run ./cmd/testing -wat testing/*
fi
