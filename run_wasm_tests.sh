#!/bin/sh
set -e

# Runs all the WASM related tests by default
# Parses WAT, compiles WASM and tests the resulting binary

if [ $# -gt 0 ]; then
    go run ./cmd/testing -wasm "$@"
else
    go run ./cmd/testing -wasm testing/*
fi
