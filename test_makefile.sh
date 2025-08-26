#!/bin/bash

# Test script to verify Makefile targets work correctly
# This script tests each Makefile target and verifies they behave as expected

set -e  # Exit on any error

echo "Testing Makefile targets..."
echo "=========================="

# Function to print test results
print_result() {
    local test_name="$1"
    local result="$2"
    if [ "$result" = "PASS" ]; then
        echo "✅ $test_name: PASS"
    else
        echo "❌ $test_name: FAIL"
        exit 1
    fi
}

# Test 1: help target should show help without errors
echo "Testing 'make help'..."
if make help > /dev/null 2>&1; then
    print_result "make help" "PASS"
else
    print_result "make help" "FAIL"
fi

# Test 2: test target should run unit tests
echo "Testing 'make test'..."
if make test > /dev/null 2>&1; then
    print_result "make test" "PASS"
else
    print_result "make test" "FAIL"
fi

# Test 3: fmt target should format code without errors
echo "Testing 'make fmt'..."
if make fmt > /dev/null 2>&1; then
    print_result "make fmt" "PASS"
else
    print_result "make fmt" "FAIL"
fi

# Test 4: mod-tidy target should tidy modules without errors
echo "Testing 'make mod-tidy'..."
if make mod-tidy > /dev/null 2>&1; then
    print_result "make mod-tidy" "PASS"
else
    print_result "make mod-tidy" "FAIL"
fi

# Test 5: clean target should clean without errors
echo "Testing 'make clean'..."
if make clean > /dev/null 2>&1; then
    print_result "make clean" "PASS"
else
    print_result "make clean" "FAIL"
fi

# Test 6: benchmark target should run benchmarks
echo "Testing 'make benchmark'..."
if make benchmark > /dev/null 2>&1; then
    print_result "make benchmark" "PASS"
else
    print_result "make benchmark" "FAIL"
fi

# Test 7: test-integration target should run integration tests
echo "Testing 'make test-integration'..."
if make test-integration > /dev/null 2>&1; then
    print_result "make test-integration" "PASS"
else
    print_result "make test-integration" "FAIL"
fi

# Test 8: test-all target should run all tests
echo "Testing 'make test-all'..."
if make test-all > /dev/null 2>&1; then
    print_result "make test-all" "PASS"
else
    print_result "make test-all" "FAIL"
fi

# Test 9: check target should run full validation
echo "Testing 'make check'..."
if make check > /dev/null 2>&1; then
    print_result "make check" "PASS"
else
    print_result "make check" "FAIL"
fi

# Test 10: Verify error handling - test with non-existent target
echo "Testing error handling with invalid target..."
if make invalid-target > /dev/null 2>&1; then
    print_result "error handling" "FAIL"
else
    print_result "error handling" "PASS"
fi

echo ""
echo "🎉 All Makefile targets tested successfully!"
echo "The Makefile is ready for development use."