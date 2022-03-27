---
title: "Architecture"
description: ""
lead: ""
date: 2022-03-27T20:19:48+07:00
lastmod: 2022-03-27T20:19:48+07:00
draft: false
images: []
menu:
  docs:
    parent: "concepts"
weight: 100
toc: true
mermaid: true
---

![Octorun Architecture](/docs/images/octorun-architecture.png)

## Overview

Octorun is a Kubernetes native application means Octorun specifically designed to run on Kubernetes by extending the Kubernetes API using [Custom Resources][custom-resources] and [Custom Controllers][custom-controllers] lately known as *[Operator Pattern][operator-pattern]*.

## Components

### Custom Resource Definitions (CRDs)

A CustomResourceDefinition is a built-in resource that lets you extend the Kubernetes API. Octorun provides and relies on several CustomResourceDefinitions:

- **Runner**: represents a single Github self-hosted runner. It holds several fields for Github self-hosted runner creation as well as Pod specification. Runner designed to be immutable: once they are created, they are never updated (except for labels, annotations and status), only deleted.

- **RunnerSet**: provides a declarative Runner deployment strategy. The purpose is to maintain sets of Runners that have the same specification.

### Controller

Octorun controller implements Octorun Resources defined by Custom Resource Definitions. Octorun controller works similarly to [Kubernetes controller][kubernetes-controller] but is only responsible for the Resources owned by Octorun. It does a *control-loop* logic that watches for create / update / delete events then make or request changes where needed that knowns as *reconciliation*.

Unlike Controllers in the *ModelViewController* pattern, Controllers in Kubernetes are run *asynchronously* after the Resources (Models) have been written to storage i.e. etcd.

### Admission Webhook

Octorun admission webhook is HTTP callbacks that receive admission requests, process it and return admission responses before Kubernetes API-Server store the Resouces in the etcd. Again Octorun Admission Webhook works similiarly to [Kubernetes admission webhook][kubernetes-webhook]. There are two types of admission webhooks: mutating admission webhook and validating admission webhook.

![Octorun Admission Webhook](/docs/images/octorun-admission-webhook.png)

### Github Webhook

Octorun uses Github Webhook to listen for [workflow_job][workflow-job-event] events. The purpose is to inform the controller when owned runner is assigned a [Workflow Job][workflow-job].

### State Metrics

Octorun state metrics is prometheus metric that export the state of Octorun Resources (i.e. Runner and RunnerSet). The implementation is similar to [kube-state-metrics][kube-state-metrics] except octorun state metrics use prometheus library to provide the metrics instead of a custom HTTP response writer.

Octorun state metrics work well with Kubernetes [prometheus-adapter][kube-prometheus-adapter] to provide Kubernetes metrics series. The purpose is to support Kubernetes `HorizonalPodAutoscaler` for `RunnerSet` resource through Kubernetes [custom.metrics.k8s.io][hpa-custom-metrics] API.


<!-- Link -->

[custom-resources]: https://kubernetes.io/docs/concepts/extend-kubernetes/api-extension/custom-resources/
[custom-controllers]: https://kubernetes.io/docs/concepts/extend-kubernetes/api-extension/custom-resources/
[hpa-custom-metrics]: https://kubernetes.io/docs/tasks/run-application/horizontal-pod-autoscale/#scaling-on-custom-metrics
[kubernetes-controller]: https://kubernetes.io/docs/concepts/architecture/controller/
[kubernetes-webhook]: https://kubernetes.io/docs/reference/access-authn-authz/extensible-admission-controllers/
[kube-prometheus-adapter]: https://github.com/kubernetes-sigs/prometheus-adapter
[kube-state-metrics]: https://github.com/kubernetes/kube-state-metrics
[operator-pattern]: https://kubernetes.io/docs/concepts/extend-kubernetes/operator/
[workflow-job]: https://docs.github.com/en/actions/using-jobs/using-jobs-in-a-workflow
[workflow-job-event]: https://docs.github.com/en/developers/webhooks-and-events/webhooks/webhook-events-and-payloads#workflow_job
