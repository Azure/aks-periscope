# Base image depends on target platform.
# For Windows, use https://mcr.microsoft.com/en-us/product/windows/nanoserver/about
ARG BASE_IMAGE=mcr.microsoft.com/cbl-mariner/distroless/base:2.0

# Builder
# golang builder image is multi-platform
FROM golang:1.19.5 AS builder

ENV GO111MODULE=on CGO_ENABLED=0

WORKDIR /build

COPY go.mod .
COPY go.sum .
RUN go mod download

COPY . .

RUN go build ./cmd/aks-periscope

# Runner
FROM $BASE_IMAGE

# Use wildcard in case there is an '.exe' extension
COPY --from=builder /build/aks-periscope* /

ENTRYPOINT ["/aks-periscope"]
