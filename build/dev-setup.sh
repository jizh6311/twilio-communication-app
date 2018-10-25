#!/bin/bash

set -e

LOG() {
  printf "%s\n" "$*"
  "$@"
}

golangci_lint_url=https://github.com/golangci/golangci-lint/releases/download/v1.9.1/golangci-lint-1.9.1-$(go env GOOS)-amd64.tar.gz
tmpdir=$(mktemp -d)
curl -L "${golangci_lint_url}" | ( cd $tmpdir && tar xzf - )
mv $tmpdir/golangci-lint-1.9.1-$(go env GOOS)-amd64/golangci-lint $GOPATH/bin/
rm -rf "$tmpdir"

LOG dep ensure -vendor-only
LOG go install -v \
  ./vendor/github.com/wadey/gocovmerge
LOG go install -v \
  ./vendor/github.com/onsi/ginkgo/ginkgo

mkdir -p _build
touch _build/dev-setup.ok
