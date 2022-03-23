---
title: "Create Your First Runner"
date: 2022-03-23T21:01:13+07:00
lastmod: 2022-03-23T21:01:13+07:00
draft: false
images: []
menu:
  docs:
    parent: "tasks"
weight: 300
toc: true
---

## Before you begin

You need to have a Kubernetes cluster with octorun installed, and the kubectl command-line tool must be configured to communicate with your cluster.

## Creating and exploring a Runner

You can spawn and register Github self-hosted runner by creating a Runner object. For example, this YAML file describes a Runner called `runner-sample` that will spawn and registers to <https://github.com/octorun/test-repo> repository with label `runner: myrunner` and runs using ghcr.io/octorun/runner:v2.288.1 runner image

```yaml
# runner.yaml
apiVersion: octorun.github.io/v1alpha1
kind: Runner
metadata:
  name: runner-sample
  labels:
    octorun.github.io/runner: myrunner
spec:
  url: https://github.com/octorun/test-repo
  image:
    name: ghcr.io/octorun/runner:v2.288.1
```

1. Create a Runner based on the YAML file:

        kubectl apply -f runner.yaml

2. Querying the Runner:

        kubectl get runner runner-sample

The output is similar to this:

```bash
NAME            RUNNERID   STATUS   ONLINE   AGE
runner-sample   92         Idle     True     4m59s
```

3. Display information about the Runner:

        kubectl describe runner runner-sample

The output is similar to this:

```bash
Name:         runner-sample
Namespace:    default
Labels:       octorun.github.io/runner=myrunner
Annotations:  <none>
API Version:  octorun.github.io/v1alpha1
Kind:         Runner
Metadata:
  Creation Timestamp:  2022-03-08T19:21:23Z
  Finalizers:
    runner.octorun.github.io/controller
  Generation:  2
Spec:
  Group:  Default
  Id:     92
  Image:
    Name:  ghcr.io/octorun/runner:v2.288.1
  Os:      Linux
  URL:      https://github.com/octorun/test-repo
  Workdir:  _work
Status:
  Conditions:
    Last Transition Time:  2022-03-08T19:21:37Z
    Message:               Github Runner has Online status
    Reason:                RunnerOnline
    Status:                True
    Type:                  runner.octorun.github.io/Online
  Phase:                   Idle
Events:
  Type    Reason            Age                  From                                 Message
  ----    ------            ----                 ----                                 -------
  Normal  RunnerPodPending  2m (x3 over 2m)      runner.octorun.github.io/controller  Waiting for Pod to be Running.
  Normal  RunnerOnline      102s (x3 over 106s)  runner.octorun.github.io/controller  Runner wait for a job.
```

4. Verify Your First Runner in Github

On your browser, you can navigate to `/settings/actions/runners`. In our case <https://github.com/octorun/test-repo/settings/actions/runners>

![Runner on Github](/docs/images/runner-on-github.png)

## Trigger a Github Action Workflow

When your Runner assigned a Workflow Job its phase status will become `Active` and then change to `Complete` once it was finished his Workflow Job.

```bash
$ kubectl get runner runner-sample --watch
NAME            RUNNERID   STATUS   ONLINE   AGE
runner-sample   92         Idle     True     13m
runner-sample   92         Idle     True     14m
runner-sample   92         Active   True     14m
runner-sample   92         Complete   False    14m
```

## Cleanup

Delete the Runner by name:

    kubectl delete runner runner-sample
