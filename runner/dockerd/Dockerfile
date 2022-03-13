FROM ghcr.io/octorun/runner:latest

USER root
RUN apt-get update -y && apt-get install -y --no-install-recommends \
    btrfs-progs \
    e2fsprogs \
    iproute2 \
    iptables \
    jq \
    kmod \
    pigz \
    uidmap \
    xfsprogs \
    && rm -rf /var/lib/apt/lists/*

RUN mkdir /run/user && chmod 1777 /run/user \
    && mkdir -p /runner/.local/share/docker \
    && chown -R runner:runner /runner

RUN apt-get clean \
    && apt-get autoclean

COPY --from=docker:dind-rootless /usr/local/bin /usr/local/bin
USER runner
