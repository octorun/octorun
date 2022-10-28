---
title: "API Reference"
description: "Generated Octorun API Reference"
lead: ""
date: 2022-03-23T23:27:26+07:00
lastmod: 2022-03-23T23:27:26+07:00
draft: false
images: []
menu:
  docs:
    parent: "reference"
weight: 500
toc: true
---

## Packages
- [octorun.github.io/v1alpha1](#octorungithubiov1alpha1)


### octorun.github.io/v1alpha1

Package v1alpha1 contains API Schema definitions for the  v1alpha1 API group

## Resource Types
- [Runner](#runner)
- [RunnerList](#runnerlist)
- [RunnerSet](#runnerset)
- [RunnerSetList](#runnersetlist)



### Runner



Runner is the Schema for the runners API

_Appears in:_
- [RunnerList](#runnerlist)

| Field | Description |
| --- | --- |
| `apiVersion` _string_ | `octorun.github.io/v1alpha1`
| `kind` _string_ | `Runner`
| `TypeMeta` _[TypeMeta](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.25/#typemeta-v1-meta)_ |  |
| `metadata` _[ObjectMeta](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.25/#objectmeta-v1-meta)_ | Refer to Kubernetes API documentation for fields of `metadata`. |
| `spec` _[RunnerSpec](#runnerspec)_ |  |
| `status` _[RunnerStatus](#runnerstatus)_ |  |


### RunnerImage





_Appears in:_
- [RunnerSpec](#runnerspec)

| Field | Description |
| --- | --- |
| `name` _string_ | Runner Container image name. |
| `pullPolicy` _[PullPolicy](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.25/#pullpolicy-v1-core)_ | Image pull policy. One of Always, Never, IfNotPresent. |
| `pullSecrets` _[LocalObjectReference](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.25/#localobjectreference-v1-core) array_ | An optional list of references to secrets in the same namespace to use for pulling any of the images used by this PodSpec. If specified, these secrets will be passed to individual puller implementations for them to use. For example, in the case of docker, only DockerConfig type secrets are honored. More info: https://kubernetes.io/docs/concepts/containers/images#specifying-imagepullsecrets-on-a-pod |


### RunnerList



RunnerList contains a list of Runner



| Field | Description |
| --- | --- |
| `apiVersion` _string_ | `octorun.github.io/v1alpha1`
| `kind` _string_ | `RunnerList`
| `TypeMeta` _[TypeMeta](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.25/#typemeta-v1-meta)_ |  |
| `metadata` _[ListMeta](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.25/#listmeta-v1-meta)_ | Refer to Kubernetes API documentation for fields of `metadata`. |
| `items` _[Runner](#runner) array_ |  |


### RunnerPlacement





_Appears in:_
- [RunnerSpec](#runnerspec)

| Field | Description |
| --- | --- |
| `nodeSelector` _object (keys:string, values:string)_ | A selector which must be true for the pod to fit on a node. Selector which must match a node's labels for the pod to be scheduled on that node. More info: https://kubernetes.io/docs/concepts/configuration/assign-pod-node/ |
| `tolerations` _[Toleration](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.25/#toleration-v1-core) array_ | If specified, the pod's tolerations. |
| `affinity` _[Affinity](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.25/#affinity-v1-core)_ | If specified, the pod's scheduling constraints |


### RunnerSet



RunnerSet is the Schema for the runnersets API

_Appears in:_
- [RunnerSetList](#runnersetlist)

| Field | Description |
| --- | --- |
| `apiVersion` _string_ | `octorun.github.io/v1alpha1`
| `kind` _string_ | `RunnerSet`
| `TypeMeta` _[TypeMeta](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.25/#typemeta-v1-meta)_ |  |
| `metadata` _[ObjectMeta](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.25/#objectmeta-v1-meta)_ | Refer to Kubernetes API documentation for fields of `metadata`. |
| `spec` _[RunnerSetSpec](#runnersetspec)_ |  |
| `status` _[RunnerSetStatus](#runnersetstatus)_ |  |


### RunnerSetList



RunnerSetList contains a list of RunnerSet



| Field | Description |
| --- | --- |
| `apiVersion` _string_ | `octorun.github.io/v1alpha1`
| `kind` _string_ | `RunnerSetList`
| `TypeMeta` _[TypeMeta](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.25/#typemeta-v1-meta)_ |  |
| `metadata` _[ListMeta](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.25/#listmeta-v1-meta)_ | Refer to Kubernetes API documentation for fields of `metadata`. |
| `items` _[RunnerSet](#runnerset) array_ |  |


### RunnerSetSpec



RunnerSetSpec defines the desired state of RunnerSet

_Appears in:_
- [RunnerSet](#runnerset)

| Field | Description |
| --- | --- |
| `runners` _integer_ | Runners is the number of desired runners. This is a pointer to distinguish between explicit zero and unspecified. Defaults to 1. |
| `selector` _[LabelSelector](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.25/#labelselector-v1-meta)_ | Selector is a label query over runners that should match the replica count. Label keys and values that must match in order to be controlled by this RunnerSet. It must match the runner template's labels. More info: https://kubernetes.io/docs/concepts/overview/working-with-objects/labels/#label-selectors |
| `template` _[RunnerTemplateSpec](#runnertemplatespec)_ | Template is the object that describes the runner that will be created if insufficient replicas are detected. |


### RunnerSetStatus



RunnerSetStatus defines the observed state of RunnerSet

_Appears in:_
- [RunnerSet](#runnerset)

| Field | Description |
| --- | --- |
| `runners` _integer_ | Runners is the most recently observed number of runners. |
| `idleRunners` _integer_ | The number of idle runners for this RunnerSet. |
| `activeRunners` _integer_ | The number of active runners for this RunnerSet. |
| `conditions` _[Condition](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.25/#condition-v1-meta) array_ | Conditions defines current service state of the runner. |
| `selector` _string_ | Selector is the same as the label selector but in the string format to avoid introspection by clients. The string will be in the same format as the query-param syntax. More info about label selectors: http://kubernetes.io/docs/user-guide/labels#label-selectors |


### RunnerSpec



RunnerSpec defines the desired state of Runner

_Appears in:_
- [Runner](#runner)
- [RunnerTemplateSpec](#runnertemplatespec)

| Field | Description |
| --- | --- |
| `url` _string_ | The github Organization or Repository URL for this runner. Must be a valid Github Org or Repository URL. eg: 	- "https://github.com/org" 	- "https://github.com/org/repo" |
| `id` _integer_ | ID of the runner assigned by Github, basically it is sequential number. Read-only. |
| `os` _string_ | OS type of the runner. Populated by the system. Read-only. |
| `group` _string_ | Name of the runner group to add to this runner. Defaults to Default. |
| `workdir` _string_ | Relative runner work directory. |
| `image` _[RunnerImage](#runnerimage)_ | Runner container image specification |
| `placement` _[RunnerPlacement](#runnerplacement)_ | Placement configuration to pass to kubernetes pod (affinity, node selector, etc). |
| `resources` _[ResourceRequirements](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.25/#resourcerequirements-v1-core)_ | Compute resources required by runner container. |
| `serviceAccountName` _string_ | ServiceAccountName is the name of the ServiceAccount to use to run this runner pod. More info: https://kubernetes.io/docs/tasks/configure-pod-container/configure-service-account/ |
| `securityContext` _[SecurityContext](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.25/#securitycontext-v1-core)_ | SecurityContext holds security configuration that will be applied to the runner container. |
| `runtimeClassName` _string_ | RuntimeClassName refers to a RuntimeClass object in the node.k8s.io group, which should be used to run this runner pod.  If no RuntimeClass resource matches the named class, the pod will not be run. If unset or empty, the "legacy" RuntimeClass will be used, which is an implicit class with an empty definition that uses the default runtime handler. |
| `volumes` _[Volume](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.25/#volume-v1-core) array_ | List of volumes that can be mounted by runner container belonging to the runner pod. |
| `volumeMounts` _[VolumeMount](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.25/#volumemount-v1-core) array_ | Runner pod volumes to mount into the runner container filesystem. |


### RunnerStatus



RunnerStatus defines the observed state of Runner

_Appears in:_
- [Runner](#runner)

| Field | Description |
| --- | --- |
| `phase` _RunnerPhase_ | Phase represents the current phase of runner. |
| `conditions` _[Condition](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.25/#condition-v1-meta) array_ | Conditions defines current service state of the runner. |


### RunnerTemplateSpec



RunnerTemplateSpec describes the data a runner should have when created from a template

_Appears in:_
- [RunnerSetSpec](#runnersetspec)

| Field | Description |
| --- | --- |
| `metadata` _[ObjectMeta](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.25/#objectmeta-v1-meta)_ | Refer to Kubernetes API documentation for fields of `metadata`. |
| `spec` _[RunnerSpec](#runnerspec)_ | Specification of the desired behavior of the runner. More info: https://git.k8s.io/community/contributors/devel/sig-architecture/api-conventions.md#spec-and-status |


