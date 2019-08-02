FROM golang:1.11 AS builder
RUN apt-get update
RUN apt-get install libsystemd-dev
RUN apt-get install -y ca-certificates
RUN mkdir /app
ADD . /app/
WORKDIR /app
RUN GOOS=linux go build github.com/Azure/aks-diagnostic-tool/cmd/aks-diagnostic
CMD ["./aks-diagnostic"]