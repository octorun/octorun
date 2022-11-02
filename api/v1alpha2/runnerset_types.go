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

package v1alpha2

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const (
	RunnerAdoptedReason string = "RunnerAdopted"
	RunnerCreatedReason string = "RunnerCreated"
	RunnerDeletedReason string = "RunnerDeleted"
)

// RunnerSetSpec defines the desired state of RunnerSet
type RunnerSetSpec struct {
	// Runners is the number of desired runners. This is a pointer
	// to distinguish between explicit zero and unspecified.
	// Defaults to 1.
	// +optional
	// +kubebuilder:default=1
	Runners *int32 `json:"runners,omitempty"`

	// Selector is a label query over runners that should match the replica count.
	// Label keys and values that must match in order to be controlled by this RunnerSet.
	// It must match the runner template's labels.
	// More info: https://kubernetes.io/docs/concepts/overview/working-with-objects/labels/#label-selectors
	Selector metav1.LabelSelector `json:"selector"`

	// Template is the object that describes the runner that will be created if
	// insufficient replicas are detected.
	// +optional
	Template RunnerTemplateSpec `json:"template"`
}

// RunnerSetStatus defines the observed state of RunnerSet
type RunnerSetStatus struct {
	// Runners is the most recently observed number of runners.
	// +optional
	Runners int32 `json:"runners"`

	// The number of idle runners for this RunnerSet.
	// +optional
	IdleRunners int32 `json:"idleRunners"`

	// The number of active runners for this RunnerSet.
	// +optional
	ActiveRunners int32 `json:"activeRunners"`

	// Conditions defines current service state of the runner.
	// +optional
	Conditions []metav1.Condition `json:"conditions,omitempty"`

	// Selector is the same as the label selector but in the string format to avoid introspection
	// by clients. The string will be in the same format as the query-param syntax.
	// More info about label selectors: http://kubernetes.io/docs/user-guide/labels#label-selectors
	// +optional
	Selector string `json:"selector,omitempty"`
}

// +kubebuilder:object:root=true
// +kubebuilder:storageversion
// +kubebuilder:subresource:status
// +kubebuilder:subresource:scale:specpath=.spec.runners,statuspath=.status.runners,selectorpath=.status.selector
// +kubebuilder:printcolumn:name="Runners",type="integer",description="Represents the current number of the runner.",JSONPath=".status.runners"
// +kubebuilder:printcolumn:name="Idle",type="integer",description="Represents the current number of the idle runner.",JSONPath=".status.idleRunners"
// +kubebuilder:printcolumn:name="Active",type="integer",description="Represents the current number of the active runner.",JSONPath=".status.activeRunners"
// +kubebuilder:printcolumn:name="Age",type="date",description="Time duration since creation of RunnerSet",JSONPath=".metadata.creationTimestamp"

// RunnerSet is the Schema for the runnersets API
type RunnerSet struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   RunnerSetSpec   `json:"spec,omitempty"`
	Status RunnerSetStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// RunnerSetList contains a list of RunnerSet
type RunnerSetList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []RunnerSet `json:"items"`
}

func init() {
	SchemeBuilder.Register(&RunnerSet{}, &RunnerSetList{})
}
