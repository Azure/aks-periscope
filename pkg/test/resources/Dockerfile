FROM docker:20.10.16-alpine3.15@sha256:c24538b2a7a081efc81185772bd8066d33cbf1f3e1a8249657395bdad4d5f844

ARG OSM_VERSION

# Add binaries/archives
RUN apk add gettext && \
    wget -O /usr/local/bin/kind https://kind.sigs.k8s.io/dl/v0.12.0/kind-linux-amd64 && \
    wget -O /helm.tar.gz https://get.helm.sh/helm-v3.8.2-linux-amd64.tar.gz && \
    wget -O /usr/local/bin/kubectl https://dl.k8s.io/release/v1.23.5/bin/linux/amd64/kubectl && \
    wget -O /osm.tar.gz https://github.com/openservicemesh/osm/releases/download/v$OSM_VERSION/osm-v$OSM_VERSION-linux-amd64.tar.gz

# Set file modes and extract
RUN chmod 755 /usr/local/bin/kind && \
    chmod 755 /usr/local/bin/kubectl && \
    tar -zxvf /helm.tar.gz && mv /linux-amd64/helm /usr/local/bin/helm && \
    tar -zxvf /osm.tar.gz && mv /linux-amd64/osm /usr/local/bin/osm

# Copy resources
ADD tools-resources /resources
ADD deployment /deployment
