export PACKAGE  := github.com/aspenmesh/tce

GOOS     = $(shell go env GOOS)
GOARCH   = $(shell go env GOARCH)
GOBIN = $(GOPATH)/bin/$(GOOS)-$(GOARCH)
BOILERPLATE := pkg/api/aspenmesh-boilerplate.go.txt
GROUP_VERSIONS := "networking:v1alpha3"

GO_BUILD_FLAGS=-v

# Generate local package lists
#   replace prefix "_$(dir $(CURDIR)" because that is prepended when GOPATH isn't set
#   Exclude all local vendor, becuase we don't want to explicitly build those
# ALL_PKGS      := $(shell go list ./...| sed 's,^_$(CURDIR),$(PACKAGE),' |  grep -v "^$(PACKAGE)/vendor")
ALL_PKGS      := ./cmd/... ./pkg/...
# TEST_PKGS     := $(shell go list -f '{{ if or .TestGoFiles .XTestGoFiles }}{{ .ImportPath }}{{ end }}' $(ALL_PKGS) | sed 's,^_$(CURDIR),$(PACKAGE),')
TEST_PKGS     := ./pkg/...

PROTO_FILES  := pkg/api/networking/v1alpha3/trafficclaim.proto


# Empty string to allow clean multi-line lists
LIST_END :=

#
# Cross-compile
#

# See https://golang.org/doc/install/source#environment
CROSS_PLATFORMS := \
	linux-amd64 \
	darwin-amd64 \
	$(LIST_END)

CROSS_CMDS := \
	tce \
	$(LIST_END)

ALL_CROSS := $(foreach plat,$(CROSS_PLATFORMS),$(foreach cmd,$(CROSS_CMDS),$(cmd)-$(plat)))


# Disable builtin implicit rules
.SUFFIXES:

#
# Normal build for developers. Just builds source.
#
all: go-build

image: tce-image

canonical-artifacts: _build/canonical/.ok

DEFAULT_TCE_TAG = traffic-claim-enforcer
ifeq ($(BRANCH_NAME)$(BUILD_ID),)
  ENV_TCE_TAG := $(DEFAULT_TCE_TAG)
else
  ENV_TCE_TAG := quay.io/aspenmesh/traffic-claim-enforcer:${BRANCH_NAME}-${BUILD_ID}
endif
TCE_TAG ?= $(ENV_TCE_TAG)

tce-image: build/pre-build-deps.txt always-build
	@echo "Building docker image $(TCE_TAG)"
	docker build -t $(TCE_TAG) ./

cross: $(addprefix _build/cross/,$(ALL_CROSS))

push-image: image
	[ "$(TCE_TAG)" != "$(DEFAULT_TCE_TAG)" ]
	docker push $(TCE_TAG)

#
# Generate mocks for testing
#
MOCK_PATHS = pkg/trafficclaim/trafficclaim.go

_build/generate-mocks.ok: _build/dev-setup.ok $(MOCK_PATHS)
	go generate ./pkg/...
	touch _build/generate-mocks.ok

#
# Run unit tests
#
test: go-test

# Like dev test, but only what can run in CI
test-ci: go-test

precommit: go-build images fmt


#
# Prod build, with all verification
#
prod: go-build go-test coverage

#
# Run tests, and genear
#
coverage: _build/coverage.html

#
# Debug build, with all verification
#
debug: GO_BUILD_FLAGS=-v -race -gcflags 'all=-N -l'
debug: go-build

#
# Verify and/or install dev depenedencies
#
dev-setup: _build/dev-setup.ok

_build/dev-setup.ok: Gopkg.toml Gopkg.lock build/dev-setup.sh
	./build/dev-setup.sh

# Build a list of direct dependencies to optimize cached docker build
#
# NOTE:  This file wil NOT be regenerated. This is a feature to maximize cacheability.
# If you change dependencies dramatically, run make clean.
# A stale pre-build-deps.txt will only make later docker build steps take longer, they will still be correct
build/pre-build-deps.txt:
	go list -f '{{ join .Imports "\n" }}' ./pkg/... ./cmd/... |  \
		grep '^$(PACKAGE)/vendor' | sort -u  > $@

#
# Remove all build artifacts
#
clean:
	rm -rf _build; mkdir _build
	rm -rf pkg/generated
	rm -rf pkg/client

info:
	@echo "ALL_PKGS: $(ALL_PKGS)"
	@echo "TEST_PKGS: $(TEST_PKGS)"
	@echo "GENERATED: $(GENERATED)"
	@echo "ALL_CROSS: $(ALL_CROSS)"

############################################################################
# NOTE:
#   The following targets are supporting targets for the publicly maintained
#   targets above. Publicly maintained targets above are always provided.
############################################################################
go-build: _build/dev-setup.ok _build/generated-crd _build/generate-mocks.ok
	go install $(GO_BUILD_FLAGS) $(ALL_PKGS)

# cross compile. Pattern is: CMD-GOOS-GOARCH
crosspart = $(shell echo "$(2)" | sed 's/\(.*\)-\(.*\)-\(.*\)/\$(1)/')
_build/cross/%: always-build
	mkdir -p $(@D)
	GOOS=$(call crosspart,2,$*) GOARCH=$(call crosspart,3,$*) go build -o $@ ./cmd/$(call crosspart,1,$*)

go-test: _build/coverage.out

_build/coverage.out: $(GENERATED) _build/dev-setup.ok _build/generate-mocks.ok always-build
	@mkdir -p $(@D)
	ginkgo -r  \
	  --randomizeAllSpecs \
	  --randomizeSuites \
	  --failOnPending \
	  --cover \
	  --trace \
	  --race \
	  --progress \
	  --outputdir $(CURDIR)/_build/
	gocovmerge $(@D)/*.coverprofile > $@
	@go tool cover -func=$@ | grep "^total:" | awk 'END { print "Total coverage:", $$3, "of statements" }'


fmt:
	@if git diff --exit-code; then true; \
	else echo "Can't format code with local changes";false; fi
	@echo "Files that need formating:"
	@go fmt $(ALL_PKGS)
	@git diff --exit-code


lint:
	golangci-lint run

_build/generated-crd: $(PROTO_FILES) $(BOILERPLATE)
	protoc --go_out=$(GOPATH)/src $(PROTO_FILES)
	./vendor/k8s.io/code-generator/generate-groups.sh all \
		$(PACKAGE)/pkg/client \
		$(PACKAGE)/pkg/api \
		$(GROUP_VERSIONS) \
		--go-header-file $(BOILERPLATE)
	touch _build/generated-crd

_build/coverage.html: _build/coverage.out
	go tool cover -html=$< -o $@
	go tool cover -func=$<

.PHONY: always-build docs image push-image go-test go-build fmt lint
