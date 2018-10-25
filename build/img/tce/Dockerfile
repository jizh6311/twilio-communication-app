ARG BUILDER_ID
FROM ${BUILDER_ID} as builder

FROM debian:stretch
WORKDIR /app

RUN apt-get update \
 && apt-get dist-upgrade -y \
 && apt-get install -y \
    curl \
    ca-certificates \
 && rm -rf /var/lib/apt/lists/*

COPY --from=builder /go/bin/tce /usr/local/bin

CMD ["tce"]
