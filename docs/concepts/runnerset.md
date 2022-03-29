---
title: "RunnerSet"
description: ""
lead: ""
date: 2022-03-29T00:08:44+07:00
lastmod: 2022-03-29T00:08:44+07:00
draft: false
images: []
menu:
  docs:
    parent: "concepts"
weight: 220
toc: true
mermaid: true
---

## Overview

A RunnerSet purpose is to maintain a set of Runners running at any given time. It is used to guarantee the availability of a specified number of identical Runners.

## Controller

The RunnerSet controller has main responsibilities to:

- Creating a Runner when actual owned runners is less than desired runners.
- Deleting a Runner when actual owned runners is more than desired runners.
- Deleting a Runner when its status phase is `Complete`
- Adopting unowned Runners that aren’t assigned to a RunnerSet

### Reconciliation Flow

```mermaid
stateDiagram-v2
    state enqueue <<choice>>
    state runner_has_no_owner <<choice>>
    state runner_has_complete_phase <<choice>>
    state runners_start_loop <<choice>>
    state runners_end_loop <<choice>>
    state too_few_runners <<choice>>
    state too_many_runners <<choice>>
    state has_reconcile_error <<choice>>
    [*] --> RunnerSetController
    RunnerSetController --> enqueue
    enqueue --> EnqueuesReconcileRequest
    EnqueuesReconcileRequest --> ListRunnersBySelector
    ListRunnersBySelector --> runners_start_loop
    runners_start_loop --> runner_has_no_owner : Runner Has No Owner?
    runner_has_no_owner --> AdoptRunner : True
    runner_has_no_owner --> IdentifyRunnerPhase : False
    IdentifyRunnerPhase --> runner_has_complete_phase: Runner Has Complete Phase
    runner_has_complete_phase --> DeleteCompleteRunner : True
    runner_has_complete_phase --> UpdateRunnersStatus : False
    DeleteCompleteRunner --> runners_end_loop : More Runners?
    AdoptRunner --> runners_end_loop : More Runners?
    UpdateRunnersStatus --> runners_end_loop : More Runners?
    runners_end_loop --> runners_start_loop : True
    runners_end_loop --> CountRunners : False
    CountRunners --> too_few_runners: Too few Runners?
    CountRunners --> too_many_runners: Too many Runners?
    too_few_runners --> CreateNewRunner : True
    too_many_runners --> DeleteRunner : True
    too_few_runners --> UpdateStatus
    too_many_runners --> UpdateStatus
    CreateNewRunner --> UpdateStatus
    DeleteRunner --> UpdateStatus
    UpdateStatus --> has_reconcile_error : Has Reconcile Error?
    has_reconcile_error --> enqueue: True
    has_reconcile_error --> [*] : False
```

## Example RunnerSet

```yaml
apiVersion: octorun.github.io/v1alpha1
kind: RunnerSet
metadata:
  name: octocat-runnerset
spec:
  runners: 3
  selector:
    matchLabels:
      octorun.github.io/runnerset: octocat-runnerset
  template:
    metadata:
      labels:
        octorun.github.io/runnerset: octocat-runnerset
    spec:
      url: https://github.com/octocat
      image:
        name: ghcr.io/octorun/runner:v2.288.1
```

In the example above, RunnerSet controller will create 3 Runners with same spec. Once one or more owned Runners has complete phase, The RunnerSet controller will delete them and create new Runners.
