#!/bin/bash

# Test script for K8s dependency upgrade to v0.27
echo "Testing K8s dependency upgrade to v0.27..."

# Check if go is installed
if ! command -v go &> /dev/null; then
    echo "Error: Go is not installed. Please install Go 1.21 or later."
    exit 1
fi

# Check Go version
echo "Go version: $(go version)"

# Clean module cache
echo "Cleaning module cache..."
go clean -modcache

# Download dependencies
echo "Downloading dependencies..."
go mod download

# Verify dependencies
echo "Verifying dependencies..."
go mod verify

# Run tests
echo "Running tests..."
make test

# Run linting
echo "Running linting..."
make lint

# Build the project
echo "Building the project..."
make build

echo "Upgrade test completed successfully!"
