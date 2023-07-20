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
	apiconversion "k8s.io/apimachinery/pkg/conversion"

	octorunv1 "octorun.github.io/octorun/api/v1alpha2"
)

func Convert_v1alpha2_RunnerSpec_To_v1alpha1_RunnerSpec(in *octorunv1.RunnerSpec, out *RunnerSpec, scope apiconversion.Scope) error {
	return autoConvert_v1alpha2_RunnerSpec_To_v1alpha1_RunnerSpec(in, out, scope)
}

func Convert_v1alpha2_RunnerSetSpec_To_v1alpha1_RunnerSetSpec(in *octorunv1.RunnerSetSpec, out *RunnerSetSpec, scope apiconversion.Scope) error {
	return autoConvert_v1alpha2_RunnerSetSpec_To_v1alpha1_RunnerSetSpec(in, out, scope)
}

func Convert_v1alpha2_RunnerSetStatus_To_v1alpha1_RunnerSetStatus(in *octorunv1.RunnerSetStatus, out *RunnerSetStatus, scope apiconversion.Scope) error {
	return autoConvert_v1alpha2_RunnerSetStatus_To_v1alpha1_RunnerSetStatus(in, out, scope)
}
