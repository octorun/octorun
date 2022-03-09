# Create set of Runners using RunnerSet

## Before you begin

You need to have a Kubernetes cluster with octorun installed, and the kubectl command-line tool must be configured to communicate with your cluster.

## Creating a RunnerSet

You can spawn and register identically multiple Github self-hosted runners by creating a RunnerSet object. For example, this YAML file describes a RunnerSet called `runnerset-sample` that will spawn and registers 3 runners to <https://github.com/octorun/test-repo> repository with label `runnerset: myrunnerset`.

```yaml,editable
# runnerset.yaml
apiVersion: octorun.github.io/v1alpha1
kind: RunnerSet
metadata:
  name: runnerset-sample
spec:
  runners: 3
  selector:
    matchLabels:
      octorun.github.io/runnerset: myrunnerset
  template:
    metadata:
      labels:
        octorun.github.io/runnerset: myrunnerset
    spec:
      url: https://github.com/octorun/test-repo
      image:
        name: ghcr.io/octorun/runner:v2.288.1
```

1. Create a Runner based on the YAML file:

        kubectl apply -f runnerset.yaml

2. Querying the RunnerSet:

        kubectl get runnerset runnerset-sample

The output is similar to this:

```console
NAME               RUNNERS   IDLE   ACTIVE   AGE
runnerset-sample   3         3               87s
```

3. List the runners created by the RunnerSet:

        kubectl get runners -l octorun.github.io/runnerset=myrunnerset

The output is similar to this:

```console
NAME                     RUNNERID   STATUS   ONLINE   AGE
runnerset-sample-52sbk   94         Idle     True     3m34s
runnerset-sample-nkd65   95         Idle     True     3m34s
runnerset-sample-zfnsn   93         Idle     True     3m34s
```

4. Display information about the RunnerSet:

        kubectl describe runnersets runnerset-sample

The output is similar to this:

```console
Name:         runnerset-sample
Namespace:    default
Labels:       <none>
Annotations:  <none>
API Version:  octorun.github.io/v1alpha1
Kind:         RunnerSet
Metadata:
  Creation Timestamp:  2022-03-08T19:50:10Z
  Generation:          1
Spec:
  Runners:  3
  Selector:
    Match Labels:
      octorun.github.io/runnerset:  myrunnerset
  Template:
    Metadata:
      Labels:
        octorun.github.io/runnerset:  myrunnerset
    Spec:
      Image:
        Name:  ghcr.io/octorun/runner:v2.288.1
      Placement:
      Resources:
      URL:  https://github.com/octorun/test-repo
Status:
  Idle Runners:  3
  Runners:       3
Events:          <none>
```

## Scaling a RunnerSet

You can easily adjust the number of runners managed by RunnerSet using `kubectl scale` command.

    kubectl scale runnersets runnerset-sample --replicas 5

## Trigger a Github Action Workflow

Once one of your Runners managed by RunnerSet finishes his Workflow Job the RunnerSet controller will replace it. Because Runner itself is designed to be ephemeral so each Runners managed by RunnerSet is always ready to be assigned a Workflow Job.

## Deleting a RunnerSet

Delete the Runner by name:

    kubectl delete runnersets runnerset-sample
