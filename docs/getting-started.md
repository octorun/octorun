---
title: "Getting Started"
date: 2022-03-23T20:44:51+07:00
lastmod: 2022-03-23T20:44:51+07:00
draft: false
images: []
toc: true
weight: 100
toc: true
---

## Prerequisites

- Install [kubectl][kubectl] v1.25+
- A Kubernetes Clsuter with installed:
    - [cert-manager][cert-manager]: To issuing certificates for `admission-webhook` service
    - [prometheus][prometheus] and [prometheus-adapter][prometheus-adapter]: To collect Octorun state metrics and serve it as Kubernetes [custom.metrics.k8s.io][custom-metrics] API. [kube-prometheus][kube-prometheus] stack is easy way to install them. This stack is used for `HorizontalPodAutoscaler` work for `RunnerSet`
    resources
- A Github [Personal Access Token][gh-pat]
- A Github [Webhook][gh-webhook] with `workflow_job` event for Repository or Organization level.


## Installation

Prepare Github credentials as environment variable. 

```bash
export GITHUB_ACCESS_TOKEN=[YOUR_GITHUB_ACCESS_TOKEN]
export GITHUB_WEBHOOK_SECRET=[YOUR_GITHUB_WEBHOOK_SECRET]
```

To install octorun simply run.

```bash
kubectl apply -k https://github.com/octorun/octorun.git//config/default
```

By default, namespaced octorun component will be installed under `ocotorun-system` namespace. You can override it by specifying `--namespace` flag.

## Verify the installation

```bash
kubectl get all -n octorun-system
NAME                                   READY   STATUS    RESTARTS   AGE
pod/octorun-manager-5f7b78c6cb-pmgvv   2/2     Running   0          23s

NAME                              TYPE        CLUSTER-IP      EXTERNAL-IP   PORT(S)    AGE
service/octorun-manager           ClusterIP   10.98.168.233   <none>        9090/TCP   23s
service/octorun-manager-metrics   ClusterIP   10.98.123.186   <none>        8443/TCP   23s
service/octorun-manager-webhook   ClusterIP   10.103.21.57    <none>        443/TCP    23s

NAME                              READY   UP-TO-DATE   AVAILABLE   AGE
deployment.apps/octorun-manager   1/1     1            1           23s

NAME                                         DESIRED   CURRENT   READY   AGE
replicaset.apps/octorun-manager-5f7b78c6cb   1         1         1       23s
```

## Post Installation

### Allow prometheus-adapter to view Octorun Objects

Sometime prometheus-adapter need to query the Kubernetes Objects. To allow prometheus-adapter query the Octorun Object you need to give prometheus-adapter serviceaccount permission for that by creating a Kubernetes `ClusterRoleBinding`.

```bash
kubectl create clusterrolebinding prometheus-adapter-octorun-runner-viewer --clusterrole=octorun-runner-viewer-role --serviceaccount=[PROMETHEUS_ADAPTER_NAMESPACE]:[PROMETHEUS_ADAPTER_SERIVCEACCOUNT]
kubectl create clusterrolebinding prometheus-adapter-octorun-runnerset-viewer --clusterrole=octorun-runnerset-viewer-role --serviceaccount=[PROMETHEUS_ADAPTER_NAMESPACE]:[PROMETHEUS_ADAPTER_SERIVCEACCOUNT]
```

### Expose Octorun Github Webhook Service

Since Github need an accessible URL for publishing Webhook event. You need to publish `octorun-manager` service either by patching its service to be `LoadBalancer` type or by creating a [Kubernetes Ingress][kubernetes-ingress].

To patch the `octorun-manager` service to be `Loadbalancer` type.

```bash
kubectl patch -n octorun-system service/octorun-manager --type='json' -p='[{"op": "replace", "path": "/spec/type", "value":"LoadBalancer"}]'
```

And finally you can create a Github Webhook with the Kubernetes Service External IP or Kubernetes Ingress Address.

```bash
kubectl get -n octorun-system service/octorun-manager
NAME              TYPE           CLUSTER-IP      EXTERNAL-IP      PORT(S)          AGE
octorun-manager   LoadBalancer   10.98.168.233   172.16.223.192   9090:30944/TCP   14m
```

*See: How to create a Github personal access token and webhooks*
- [Creating a personal access token][gh-pat]
- [Creating webhooks][gh-webhook]

<!-- References -->
[cert-manager]: https://cert-manager.io/docs/installation/
[custom-metrics]: https://github.com/kubernetes/design-proposals-archive/blob/main/instrumentation/custom-metrics-api.md
[gh-pat]: https://docs.github.com/en/authentication/keeping-your-account-and-data-secure/creating-a-personal-access-token
[gh-webhook]: https://docs.github.com/en/developers/webhooks-and-events/webhooks/creating-webhooks
[kubectl]: https://kubernetes.io/docs/tasks/tools/install-kubectl/
[kubernetes-ingress]: https://kubernetes.io/docs/concepts/services-networking/ingress/
[kube-prometheus]: https://github.com/prometheus-operator/kube-prometheus
[prometheus]: https://prometheus.io/
[prometheus-adapter]: https://github.com/kubernetes-sigs/prometheus-adapter
