#!/bin/bash
set -ex

# Build the instrumentation tool from your local changes
echo "Building the instrumentation tool..."
go build -o otel

# Ensure dependencies are up to date
echo "Updating demo dependencies..."
cd demo
go mod tidy

# Initialize the otel imports using your local build
echo "Initializing otel imports..."
../otel init .

# Build the demo app with instrumentation
echo "Building with instrumentation..."
go build -toolexec=$(pwd)/../otel

# Run the demo app
echo "Running the demo app..."
./demo

# Show the modified source code
echo "Modified source code:"
MODIFIED_FILE=$(find "$(go env GOCACHE)" -name "modified.go" | grep -v "work" | head -1)
if [ -n "$MODIFIED_FILE" ]; then
  echo "Found modified source at: $MODIFIED_FILE"
  cat -n "$MODIFIED_FILE"
else
  echo "No modified.go found"
fi