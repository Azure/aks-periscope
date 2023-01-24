# golang builder image is multi-platform
ARG BUILDER_IMAGE=golang:1.19.5

# Base image depends on target platform.
# For Windows, use https://mcr.microsoft.com/en-us/product/windows/nanoserver/about
ARG BASE_IMAGE=mcr.microsoft.com/cbl-mariner/distroless/base:2.0

# For Windows, set to '.exe'
ARG EXTENSION=

# Builder
FROM $BUILDER_IMAGE AS builder

ENV GO111MODULE=on CGO_ENABLED=0

WORKDIR /build

COPY go.mod .
COPY go.sum .
RUN go mod download

COPY . .

RUN go build ./cmd/aks-periscope

# Runner
FROM $BASE_IMAGE

COPY --from=builder /build/aks-periscope$EXTENSION /

ENTRYPOINT ["/aks-periscope$EXTENSION"]
