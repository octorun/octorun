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

package sortable

import (
	octorunv1 "octorun.github.io/octorun/api/v1alpha2"
)

const (
	mustDelete    float64 = 100.0
	couldDelete   float64 = 50.0
	mustNotDelete float64 = 0.0
)

// RunnersToDelete is sortable slice of Runner
// implement sort.Interface.
type RunnersToDelete []*octorunv1.Runner

func (r RunnersToDelete) Len() int      { return len(r) }
func (r RunnersToDelete) Swap(i, j int) { r[i], r[j] = r[j], r[i] }
func (r RunnersToDelete) Less(i, j int) bool {
	priority := func(runner *octorunv1.Runner) float64 {
		if !runner.GetDeletionTimestamp().IsZero() {
			return mustDelete
		}

		if runner.Status.Phase == octorunv1.RunnerActivePhase {
			return mustNotDelete
		}

		return couldDelete
	}

	return priority(r[j]) < priority(r[i])
}
