FROM ubuntu:20.04

ENV DEBIAN_FRONTEND=noninteractive

RUN echo "APT::Get::Assume-Yes \"true\";" > /etc/apt/apt.conf.d/90assumeyes
RUN apt-get update -y && apt-get install -y --no-install-recommends \
    ca-certificates \
    curl \
    git \
    jq \
    openssl \
    && rm -rf /var/lib/apt/lists/*

RUN useradd -u 1000 -ms /bin/bash -d /runner runner
WORKDIR /runner
RUN curl -L https://github.com/actions/runner/releases/download/v2.288.1/actions-runner-linux-x64-2.288.1.tar.gz | tar --overwrite -xz \
    && ./bin/installdependencies.sh && rm -rf /var/lib/apt/lists/*

RUN apt-get clean \
    && apt-get autoclean

COPY entrypoint.sh /runner/entrypoint.sh
RUN chown -R runner:runner /runner
USER runner
ENTRYPOINT [ "/runner/entrypoint.sh" ]
