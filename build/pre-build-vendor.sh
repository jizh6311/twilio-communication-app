#!/bin/bash

cat pre-build-deps.txt | xargs go build -i -v
cat pre-build-deps.txt | xargs go build -race -i -v
