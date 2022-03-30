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

import (
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const (
	RunnerConditionOnline string = "runner.octorun.github.io/Online"
)

const (
	RunnerBusyReason         string = "RunnerBusy"
	RunnerOnlineReason       string = "RunnerOnline"
	RunnerOfflineReason      string = "RunnerOffline"
	RunnerPodPendingReason   string = "RunnerPodPending"
	RunnerPodSucceededReason string = "RunnerPodSucceeded"
	RunnerSecretFailedReason string = "RunnerSecretFailed"
)

type RunnerPhase string

// These are the valid phases of runners.
const (
	// Pending means the runner is in the initialization process.
	// eg: creating runner pod, registering the runner, etc.
	RunnerPendingPhase RunnerPhase = "Pending"
	// Idle means the runner has successfully initialized but has no
	// job assigned to this runner yet.
	RunnerIdlePhase RunnerPhase = "Idle"
	// Active means the runner has an assigned job.
	RunnerActivePhase RunnerPhase = "Active"
	// Complete means the runner has already completed his job.
	RunnerCompletePhase RunnerPhase = "Complete"
)

type RunnerImage struct {
	// Runner Container image name.
	Name string `json:"name,omitempty"`

	// Image pull policy.
	// One of Always, Never, IfNotPresent.
	// +kubebuilder:validation:Enum=Always;Never;IfNotPresent
	// +optional
	PullPolicy corev1.PullPolicy `json:"pullPolicy,omitempty"`

	// An optional list of references to secrets in the same namespace to use for pulling any of the images used by this PodSpec.
	// If specified, these secrets will be passed to individual puller implementations for them to use. For example,
	// in the case of docker, only DockerConfig type secrets are honored.
	// More info: https://kubernetes.io/docs/concepts/containers/images#specifying-imagepullsecrets-on-a-pod
	// +optional
	PullSecrets []corev1.LocalObjectReference `json:"pullSecrets,omitempty"`
}

type RunnerPlacement struct {
	// A selector which must be true for the pod to fit on a node.
	// Selector which must match a node's labels for the pod to be scheduled on that node.
	// More info: https://kubernetes.io/docs/concepts/configuration/assign-pod-node/
	// +optional
	NodeSelector map[string]string `json:"nodeSelector,omitempty"`

	// If specified, the pod's tolerations.
	// +optional
	Tolerations []corev1.Toleration `json:"tolerations,omitempty"`

	// If specified, the pod's scheduling constraints
	// +optional
	Affinity *corev1.Affinity `json:"affinity,omitempty"`
}

// RunnerTemplateSpec describes the data a runner should have when created from a template
type RunnerTemplateSpec struct {
	// Standard object's metadata.
	// More info: https://git.k8s.io/community/contributors/devel/sig-architecture/api-conventions.md#metadata
	// +optional
	metav1.ObjectMeta `json:"metadata,omitempty"`

	// Specification of the desired behavior of the runner.
	// More info: https://git.k8s.io/community/contributors/devel/sig-architecture/api-conventions.md#spec-and-status
	// +optional
	Spec RunnerSpec `json:"spec,omitempty"`
}

// RunnerSpec defines the desired state of Runner
type RunnerSpec struct {
	// The github Organization or Repository URL for this runner.
	// Must be a valid Github Org or Repository URL.
	// eg:
	// 	- "https://github.com/org"
	// 	- "https://github.com/org/repo"
	URL string `json:"url"`

	// ID of the runner assigned by Github, basically it is sequential number.
	// Read-only.
	// +optional
	ID *int64 `json:"id,omitempty"`

	// OS type of the runner. Populated by the system.
	// Read-only.
	// +optional
	OS string `json:"os,omitempty"`

	// Name of the runner group to add to this runner.
	// Defaults to Default.
	// +optional
	Group string `json:"group,omitempty"`

	// Relative runner work directory.
	// +optional
	Workdir string `json:"workdir,omitempty"`

	// Runner container image specification
	// +optional
	Image RunnerImage `json:"image,omitempty"`

	// Placement configuration to pass to kubernetes pod (affinity, node selector, etc).
	// +optional
	Placement RunnerPlacement `json:"placement,omitempty"`

	// Compute resources required by runner container.
	// +optional
	Resources corev1.ResourceRequirements `json:"resources,omitempty"`

	// ServiceAccountName is the name of the ServiceAccount to use to run this runner pod.
	// More info: https://kubernetes.io/docs/tasks/configure-pod-container/configure-service-account/
	// +optional
	ServiceAccountName string `json:"serviceAccountName,omitempty"`

	// SecurityContext holds security configuration that will be applied to the runner container.
	// +optional
	SecurityContext *corev1.SecurityContext `json:"securityContext,omitempty"`

	// RuntimeClassName refers to a RuntimeClass object in the node.k8s.io group, which should be used
	// to run this runner pod.  If no RuntimeClass resource matches the named class, the pod will not be run.
	// If unset or empty, the "legacy" RuntimeClass will be used, which is an implicit class with an
	// empty definition that uses the default runtime handler.
	// +optional
	RuntimeClassName *string `json:"runtimeClassName,omitempty"`

	// List of volumes that can be mounted by runner container belonging to the runner pod.
	// +optional
	Volumes []corev1.Volume `json:"volumes,omitempty"`

	// Runner pod volumes to mount into the runner container filesystem.
	// +optional
	VolumeMounts []corev1.VolumeMount `json:"volumeMounts,omitempty"`
}

// RunnerStatus defines the observed state of Runner
type RunnerStatus struct {
	// Phase represents the current phase of runner.
	// +optional
	Phase RunnerPhase `json:"phase,omitempty"`

	// Conditions defines current service state of the runner.
	// +optional
	Conditions []metav1.Condition `json:"conditions,omitempty"`
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
// +kubebuilder:printcolumn:name="RunnerID",type="string",description="ID of the runner assigned by Github, basically it is sequential number.",JSONPath=".spec.id"
// +kubebuilder:printcolumn:name="Status",type="string",description="Represents the current phase of the runner.",JSONPath=".status.phase"
// +kubebuilder:printcolumn:name="Online",type="string",description="Represents the current Online status of the runner.",JSONPath=".status.conditions[?(@.type==\"runner.octorun.github.io/Online\")].status"
// +kubebuilder:printcolumn:name="URL",type="string",description="The github Organization or Repository URL for this runner.",JSONPath=".spec.url",priority=10
// +kubebuilder:printcolumn:name="RunnerGroup",type="string",description="RunnerGroup of the runner",JSONPath=".spec.group",priority=10
// +kubebuilder:printcolumn:name="Age",type="date",description="Time duration since creation of Runner",JSONPath=".metadata.creationTimestamp"

// Runner is the Schema for the runners API
type Runner struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   RunnerSpec   `json:"spec,omitempty"`
	Status RunnerStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// RunnerList contains a list of Runner
type RunnerList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []Runner `json:"items"`
}

func init() {
	SchemeBuilder.Register(&Runner{}, &RunnerList{})
}
