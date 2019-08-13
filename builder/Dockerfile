FROM golang AS builder
RUN mkdir /app
ADD . /app
WORKDIR /app
RUN CGO_ENABLED=0 GOOS=linux go build github.com/Azure/aks-diagnostic-tool/cmd/aks-diagnostic

FROM alpine
RUN apk --no-cache add ca-certificates
RUN mkdir /app
WORKDIR /app
COPY --from=builder /app/aks-diagnostic .
CMD ["./aks-diagnostic"]