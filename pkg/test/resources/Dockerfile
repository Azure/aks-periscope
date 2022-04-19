FROM docker:20.10.14-alpine3.15

# Add binaries/archives
ADD https://kind.sigs.k8s.io/dl/v0.12.0/kind-linux-amd64 /usr/local/bin/kind
ADD https://get.helm.sh/helm-v3.8.2-linux-amd64.tar.gz /helm.tar.gz
ADD https://dl.k8s.io/release/v1.23.5/bin/linux/amd64/kubectl /usr/local/bin/kubectl

# Set file modes and extract
RUN chmod 755 /usr/local/bin/kind && \
    chmod 755 /usr/local/bin/kubectl && \
    tar -zxvf /helm.tar.gz && \
    mv /linux-amd64/helm /usr/local/bin/helm

# Copy resources
ADD metrics-server /resources/metrics-server
ADD testchart /resources/testchart
