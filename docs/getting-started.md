# Getting Started with Octorun

## Prerequisites

- Install [kubectl][kubectl] v1.22+
- Install [cert-manager][cert-manager] on your Kubernetes Cluster

## Installation

Prepare Github credentials as environment variable. 

```shell
export GITHUB_ACCESS_TOKEN=[YOUR_GITHUB_ACCESS_TOKEN]
export GITHUB_WEBHOOK_SECRET=[YOUR_GITHUB_WEBHOOK_SECRET]
```

To install octorun simply run.

```shell
kubectl apply -k https://github.com/octorun/octorun.git//config/default
```

By default, namespaced octorun component will be installed under `ocotorun-system` namespace. You can override it by specifying `--namespace` flag.

## Verify the installation

```shell
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

Since Github need an accessible URL for publishing Webhook event. You need to publish `octorun-manager` service either by patching its service to be `LoadBalancer` type or by creating a [Kubernetes Ingress][kubernetes-ingress].

To patch the `octorun-manager` service to be `Loadbalancer` type.

```shell
kubectl patch -n octorun-system service/octorun-manager --type='json' -p='[{"op": "replace", "path": "/spec/type", "value":"LoadBalancer"}]'
```

And finally you can create a Github Webhook with the Kubernetes Service External IP or Kubernetes Ingress Address.

```shell
kubectl get -n octorun-system service/octorun-manager
NAME              TYPE           CLUSTER-IP      EXTERNAL-IP      PORT(S)          AGE
octorun-manager   LoadBalancer   10.98.168.233   172.16.223.192   9090:30944/TCP   14m
```

*See: How to create a Github personal access token and webhooks*
- <https://docs.github.com/en/authentication/keeping-your-account-and-data-secure/creating-a-personal-access-token>
- <https://docs.github.com/en/developers/webhooks-and-events/webhooks/creating-webhooks>

<!-- References -->
[cert-manager]: https://cert-manager.io/docs/installation/
[kubectl]: https://kubernetes.io/docs/tasks/tools/install-kubectl/
[kubernetes-ingress]: https://kubernetes.io/docs/concepts/services-networking/ingress/
