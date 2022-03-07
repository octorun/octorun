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

package pod

import (
	corev1 "k8s.io/api/core/v1"
)

// PodConditionIsReady returns true if a pod is ready.
func PodConditionIsReady(pod *corev1.Pod) bool {
	condition := PodCondition(&pod.Status, corev1.PodReady)
	if condition == nil {
		return false
	}

	return condition.Status == corev1.ConditionTrue
}

// PodCondition extracts the provided condition from the given status and returns that.
// Returns nil if the condition is not present.
func PodCondition(status *corev1.PodStatus, conditionType corev1.PodConditionType) *corev1.PodCondition {
	if status != nil {
		conditions := status.Conditions
		for i := range conditions {
			if conditions[i].Type == conditionType {
				return &conditions[i]
			}
		}
	}

	return nil
}
