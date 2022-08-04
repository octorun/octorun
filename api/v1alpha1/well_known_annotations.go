/*
Copyright 2022 The Authors.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package v1alpha1

const (
	// AnnotationRunnerAssignedJobAt can be used to indicate that a runner
	// already assigned a Github Workflow Job. This annotation is also used by
	// github webhook handler to retrigger runner controller reconciliation.
	AnnotationRunnerAssignedJobAt = "runner.octorun.github.io/assigned-job-at"

	// AnnotationRunnerTokenExpiresAt is used to note when the registration token will expire.
	// The runner controller will refresh the token if needed based on this annotation.
	AnnotationRunnerTokenExpiresAt = "runner.octorun.github.io/token-expires-at"

	// AnnotationRunnerEvictionPolicy used to respect the Kubernetes cluster-autoscaler eviction.
	// Since the Pod created by Runner is not controlled by Kubernetes workload controller (eg: ReplicaSet, StatefulSet etc)
	// cluster-autoscaler unable to drain underutilized node.
	//
	// The value could be Never or IfNotActive. set this annotation to IfNotActive will annotate the runner Pod
	// with `cluster-autoscaler.kubernetes.io/safe-to-evict=true` once created and will be removed when Runner become Active (has assigned job)
	//
	// NOTE: this annotation is experimental and will be added to Runner Object field if possible in the next API version.
	AnnotationRunnerEvictionPolicy = "runner.octorun.github.io/eviction-policy"
)
