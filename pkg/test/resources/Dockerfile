FROM docker:20.10.14-alpine3.15
ADD https://kind.sigs.k8s.io/dl/v0.12.0/kind-linux-amd64 /usr/local/bin/kind
ADD https://get.helm.sh/helm-v3.8.2-linux-amd64.tar.gz /helm.tar.gz
RUN chmod 755 /usr/local/bin/kind && \
    tar -zxvf /helm.tar.gz && \
    mv /linux-amd64/helm /usr/local/bin/helm
