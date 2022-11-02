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
	"sigs.k8s.io/controller-runtime/pkg/conversion"

	octorunv1 "octorun.github.io/octorun/api/v1alpha2"
)

var _ conversion.Convertible = &Runner{}
var _ conversion.Convertible = &RunnerList{}

// ConvertTo converts this Runner to the Hub version (v1alpha2).
func (src *Runner) ConvertTo(dstRaw conversion.Hub) error {
	dst := dstRaw.(*octorunv1.Runner)
	return Convert_v1alpha1_Runner_To_v1alpha2_Runner(src, dst, nil)
}

// ConvertFrom converts from the Hub version (v1alpha2) to this version.
func (dst *Runner) ConvertFrom(srcRaw conversion.Hub) error {
	src := srcRaw.(*octorunv1.Runner)
	return Convert_v1alpha2_Runner_To_v1alpha1_Runner(src, dst, nil)
}

// ConvertTo converts this RunnerList to the Hub version (v1alpha2).
func (src *RunnerList) ConvertTo(dstRaw conversion.Hub) error {
	dst := dstRaw.(*octorunv1.RunnerList)
	return Convert_v1alpha1_RunnerList_To_v1alpha2_RunnerList(src, dst, nil)
}

// ConvertFrom converts from the Hub version (v1alpha2) to this version.
func (dst *RunnerList) ConvertFrom(srcRaw conversion.Hub) error {
	src := srcRaw.(*octorunv1.RunnerList)
	return Convert_v1alpha2_RunnerList_To_v1alpha1_RunnerList(src, dst, nil)
}
