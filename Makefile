export PACKAGE  := github.com/aspenmesh/tce

GOOS     = $(shell go env GOOS)
GOARCH   = $(shell go env GOARCH)
GOBIN = $(GOPATH)/bin/$(GOOS)-$(GOARCH)

GO_BUILD_FLAGS=-v

# Generate local package lists
#   replace prefix "_$(dir $(CURDIR)" because that is prepended when GOPATH isn't set
#   Exclude all local vendor, becuase we don't want to explicitly build those
# ALL_PKGS      := $(shell go list ./...| sed 's,^_$(CURDIR),$(PACKAGE),' |  grep -v "^$(PACKAGE)/vendor")
ALL_PKGS      := ./cmd/... ./pkg/...
# TEST_PKGS     := $(shell go list -f '{{ if or .TestGoFiles .XTestGoFiles }}{{ .ImportPath }}{{ end }}' $(ALL_PKGS) | sed 's,^_$(CURDIR),$(PACKAGE),')
TEST_PKGS     := ./pkg/...


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

images: tce-image

canonical-artifacts: _build/canonical/.ok

DEFAULT_BUILDER_TAG = tce-builder
DEFAULT_TCE_TAG = tce
ifeq ($(BRANCH_NAME)$(BUILD_ID),)
  ENV_BUILDER_TAG := $(DEFAULT_BUILDER_TAG)
  ENV_TCE_TAG := $(DEFAULT_TCE_TAG)
else
  ENV_BUILDER_TAG := localhost:5000/tce-builder:${BRANCH_NAME}-${BUILD_ID}
  ENV_TCE_TAG := quay.io/aspenmesh/tce:${BRANCH_NAME}-${BUILD_ID}
endif
BUILDER_TAG ?= $(ENV_BUILDER_TAG)
TCE_TAG ?= $(ENV_TCE_TAG)

tce-image: _build/img/builder-id.txt
	@echo "Building docker image $(TCE_TAG)"
	docker build -t $(TCE_TAG) --build-arg BUILDER_ID=`cat $<` build/img/tce

cross: $(addprefix _build/cross/,$(ALL_CROSS))

push-images: images
	[ "$(TCE_TAG)" != "$(DEFAULT_TCE_TAG)" ]
	docker push $(TCE_TAG)

#
# Generate mocks for testing
#

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


# Make builder image, and save its id
# This builds all the binaries needed
# TODO: Use --iidfile when minikube updates docker to support it
_build/img/builder-id.txt: build/pre-build-deps.txt always-build
	@mkdir -p $(@D)
	@echo "Building docker builder image"
	#docker build --target=builder --iidfile $@ -f build/Dockerfile.builder .
	# Using quotes and $$ to access the environment variable directly
	# while preserving newlines.
	docker build --target=builder -t $(BUILDER_TAG) \
		-f build/Dockerfile.builder .
	docker inspect --format='{{.Id}}' $(BUILDER_TAG) > $@



# _build/canonical is reserved for artifacts build via the official builder image(s)
_build/canonical/.ok: _build/img/builder-id.txt
	@rm -rf _build/canonical
	@mkdir -p _build/canonical/coverage
	./build/docker-cp.sh $< /go/src/$(PACKAGE)/_build/coverage.out $(@D)/coverage/
	./build/docker-cp.sh $< /go/src/$(PACKAGE)/_build/coverage.html $(@D)/coverage/
	./build/docker-cp.sh $< /go/src/$(PACKAGE)/_build/cross $(@D)/cross
	cp $< $@


#
# Remove all build artifacts
#
clean:
	rm -rf _build; mkdir _build
	rm -rf pkg/generated

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
go-build: _build/dev-setup.ok
	go install $(GO_BUILD_FLAGS) $(ALL_PKGS)

# cross compile. Pattern is: CMD-GOOS-GOARCH
crosspart = $(shell echo "$(2)" | sed 's/\(.*\)-\(.*\)-\(.*\)/\$(1)/')
_build/cross/%: always-build
	mkdir -p $(@D)
	GOOS=$(call crosspart,2,$*) GOARCH=$(call crosspart,3,$*) go build -o $@ ./cmd/$(call crosspart,1,$*)

go-test: _build/coverage.out

_build/coverage.out: $(GENERATED) _build/dev-setup.ok always-build
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

.PHONY: always-build docs agent-image apiserver-image images push-images go-test go-build fmt lint

_build/coverage.html: _build/coverage.out
	go tool cover -html=$< -o $@
	go tool cover -func=$<
