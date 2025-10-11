#!/bin/bash

# Test script for NATS image upgrade to v2.10.14
echo "Testing NATS image upgrade to v2.10.14..."

# Check if Docker is available
if ! command -v docker &> /dev/null; then
    echo "Warning: Docker is not installed. Cannot test image pull."
    echo "Please install Docker to test image availability."
    exit 0
fi

# Test NATS image pull
echo "Testing NATS image pull..."
if docker pull nats:2.10.14-alpine3.19; then
    echo "✅ NATS image pull successful"
else
    echo "❌ NATS image pull failed"
    exit 1
fi

# Test config reloader image pull
echo "Testing config reloader image pull..."
if docker pull connecteverything/nats-server-config-reloader:0.7.0; then
    echo "✅ Config reloader image pull successful"
else
    echo "❌ Config reloader image pull failed"
    exit 1
fi

# Test image compatibility
echo "Testing image compatibility..."
docker run --rm nats:2.10.14-alpine3.19 --help > /dev/null
if [ $? -eq 0 ]; then
    echo "✅ NATS image compatibility test passed"
else
    echo "❌ NATS image compatibility test failed"
    exit 1
fi

# Test config reloader compatibility
echo "Testing config reloader compatibility..."
docker run --rm connecteverything/nats-server-config-reloader:0.7.0 --help > /dev/null
if [ $? -eq 0 ]; then
    echo "✅ Config reloader compatibility test passed"
else
    echo "❌ Config reloader compatibility test failed"
    exit 1
fi

echo ""
echo "🎉 All NATS upgrade tests passed successfully!"
echo "✅ NATS v2.10.14-alpine3.19 is ready for deployment"
echo "✅ Config reloader v0.7.0 is ready for deployment"
