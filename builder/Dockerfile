FROM golang AS builder
RUN mkdir /app
ADD . /app
WORKDIR /app
RUN CGO_ENABLED=0 GOOS=linux go build -mod=vendor github.com/Azure/aks-periscope/cmd/aks-periscope

FROM alpine
RUN apk --no-cache add ca-certificates curl openssl bash
ADD https://storage.googleapis.com/kubernetes-release/release/v1.16.0/bin/linux/amd64/kubectl /usr/local/bin/kubectl
RUN chmod +x /usr/local/bin/kubectl
RUN curl -fsSL -o get_helm.sh https://raw.githubusercontent.com/helm/helm/master/scripts/get-helm-3 \
    && chmod +x get_helm.sh \
    && ./get_helm.sh
RUN mkdir /app
WORKDIR /app
COPY --from=builder /app/aks-periscope .
CMD ["./aks-periscope"]