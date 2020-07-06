#!/usr/bin/env bash

# Build and copy the latest python package from tools
echo "### Create and copy latest tools package to docker images sources..."
cd ../tools
rm dist/*.gz
source .env/bin/activate
python setup.py sdist
deactivate
cd ../amp_simulation

mkdir -p ./amp_model/dependencies
cp ../tools/dist/*.gz  ./amp_model/dependencies

mkdir -p ./amp_model_api/dependencies
cp ../tools/dist/*.gz  ./amp_model_api/dependencies

# Copy reusable configs
echo "### Copy resuable configurations to their proper locations..."
cp ./common_configs/model_api.yml ./amp_model/models/
cp ./common_configs/model_api.yml ./amp_model_api/api/

echo "### Complete"
