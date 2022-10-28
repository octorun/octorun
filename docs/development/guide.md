---
title: "Developer Guide"
date: 2022-03-23T21:33:52+07:00
lastmod: 2022-03-23T21:33:52+07:00
draft: false
images: []
menu:
  docs:
    parent: "development"
weight: 400
toc: true
---

Octorun is built using [Kubebuilder][kubebuilder], take a time to see the [Kubebuilder documentation][kubebuilder-intro] will be very helpful to get started with Octorun.

Most of development process can be done using `make` command. Run `make help` to see available make command targets.

## Prerequisites

- Install [go][go] v1.19+
- Install [docker][docker]
- Install [kubectl][kubectl] v1.22+
- Install Kubernetes cluster for development (eg: [kind][kind], [minikube][minikube], [docker-desktop][docker-desktop])
- Install [tilt][tilt]

## Getting Started

We are using [tilt.dev][tilt] for rapid iterative development. To start tilt it's quite simple just run `tilt up`. 

*See: <https://docs.tilt.dev/> for tilt documentations*

First, perpare your Github credentials as environment variable.

```bash
export GITHUB_ACCESS_TOKEN=[YOUR_GITHUB_ACCESS_TOKEN]
export GITHUB_WEBHOOK_SECRET=[YOUR_GITHUB_WEBHOOK_SECRET]
```

Besides directly invoking `tilt up` command we also have a Makefile target to start the tilt.

```bash
make dev
```

The command above will start the tilt and teardown the tilt when receive an EXIT signal (ctrl+c)

## Testing

To run tests locally, run

```bash
make test
```

<!-- References -->
[docker]: https://docs.docker.com/get-docker/
[docker-desktop]: https://docs.docker.com/desktop/kubernetes/
[go]: https://go.dev/doc/install
[kind]: https://kind.sigs.k8s.io/docs/user/quick-start/#installation
[kubectl]: https://kubernetes.io/docs/tasks/tools/install-kubectl/
[kubebuilder]: https://book.kubebuilder.io/
[kubebuilder-intro]: https://book.kubebuilder.io/introduction.html
[minikube]: https://minikube.sigs.k8s.io/docs/start/
[tilt]: https://docs.tilt.dev/install.html
