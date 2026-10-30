#!/bin/sh
set -e

# first the unit tests
go test ./...

# now the WAT tests
go run ./cmd/testing -wat testing/*

# now the WASM tests
go run ./cmd/testing -wasm testing/*
