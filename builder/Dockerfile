FROM golang AS builder
RUN mkdir /app
ADD . /app
WORKDIR /app
RUN CGO_ENABLED=0 GOOS=linux go build -mod=vendor github.com/Azure/aks-periscope/cmd/aks-periscope

FROM alpine
RUN apk --no-cache add ca-certificates
RUN mkdir /app
WORKDIR /app
COPY --from=builder /app/aks-periscope .
CMD ["./aks-periscope"]