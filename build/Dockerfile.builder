FROM golang:1.10-stretch as builder

WORKDIR /go/src/github.com/aspenmesh/tce

#
# Enable https apt packages.
#
RUN apt-get update \
 && apt-get install -y \
    apt-transport-https \
 && rm -rf /var/lib/apt/lists/*

# Update all packages and install new ones
RUN apt-get update \
 && apt-get dist-upgrade -y \
 && apt-get install -y \
    apt-transport-https \
    build-essential \
    ca-certificates \
    lsb-release \
    unzip \
 && rm -rf /var/lib/apt/lists/*

RUN curl -s -L \
    https://github.com/golang/dep/releases/download/v0.5.0/dep-linux-amd64 \
    > /go/bin/dep \
 && echo "287b08291e14f1fae8ba44374b26a2b12eb941af3497ed0ca649253e21ba2f83 /go/bin/dep" | sha256sum -c - \
 && chmod +x /go/bin/dep

# Optimize build cache
# Install the locked go deps into vendor
COPY Gopkg.lock Gopkg.toml ./
RUN dep ensure -vendor-only

# Install build deps
COPY build/dev-setup.sh ./
RUN bash dev-setup.sh

# Attempt to build the vendorized deps
# This will speed up later steps, but won't affect the results
COPY build/pre-build* ./
RUN bash pre-build-vendor.sh || true

# Copy in all the source. Everything below this gets run a lot
# 
COPY . .

RUN make clean
RUN make 
RUN SKIP_TESTS_THAT_REQUIRE_DOCKER=nonempty make test-ci coverage
RUN make cross

# This is only to be used as an intermediate image.
# Force other use case to fail
FROM scratch
