#!/usr/bin/env bash

# Build and copy the latest python package from tools
echo "### Create and copy latest tools package to docker images sources..."
cd ../anomaly-detection
make package-wheel
cd -
mkdir -p ./dependencies
cp ../anomaly-detection/dist/*.whl ./dependencies
echo "########################### Available Wheels:"
echo
echo $(ls ./dependencies)
echo
echo "########################### Complete"
