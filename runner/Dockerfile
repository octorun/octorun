FROM ubuntu:22.04

ENV DEBIAN_FRONTEND=noninteractive
ARG RUNNER_VERSION
ARG RUNNER_SEMANTIC_VERSION

RUN echo "APT::Get::Assume-Yes \"true\";" > /etc/apt/apt.conf.d/90assumeyes
RUN apt-get update -y && apt-get install -y --no-install-recommends \
    build-essential \
    ca-certificates \
    curl \
    file \
    git \
    jq \
    openssh-client \
    pkg-config \
    tzdata \
    xz-utils \
    && apt-get clean && apt-get autoclean && rm -rf /var/lib/apt/lists/*

RUN useradd -u 1000 -ms /bin/bash -d /runner runner
WORKDIR /runner
RUN curl -L https://github.com/actions/runner/releases/download/${RUNNER_VERSION}/actions-runner-linux-x64-${RUNNER_SEMANTIC_VERSION}.tar.gz | tar --overwrite -xz \
    && ./bin/installdependencies.sh && apt-get clean && apt-get autoclean && rm -rf /var/lib/apt/lists/*

COPY entrypoint.sh /runner/entrypoint.sh
RUN chown -R runner:runner /runner
USER runner
ENTRYPOINT [ "/runner/entrypoint.sh" ]
