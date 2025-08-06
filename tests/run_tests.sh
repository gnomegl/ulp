#!/bin/bash

# Test runner script for ULP

set -e

echo "Building ULP..."
cd ..
go build -o ulp-go main.go
cd tests

echo -e "\n=== Running Integration Tests ==="
cd integration
go test -v test.go
cd ..

echo -e "\n=== Running Unit Tests ==="
cd ..
go test -v ./pkg/...

echo -e "\n=== Running Benchmarks ==="
cd tests/integration
go test -bench=. test.go

echo -e "\n=== All tests completed ==="