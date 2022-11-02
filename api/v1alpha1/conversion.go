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
	out.URL = in.URL
	out.ID = in.ID
	out.OS = in.OS
	out.Group = in.Group
	out.Workdir = in.Workdir
	if err := Convert_v1alpha2_RunnerImage_To_v1alpha1_RunnerImage(&in.Image, &out.Image, scope); err != nil {
		return err
	}
	if err := Convert_v1alpha2_RunnerPlacement_To_v1alpha1_RunnerPlacement(&in.Placement, &out.Placement, scope); err != nil {
		return err
	}
	out.Resources = in.Resources
	out.ServiceAccountName = in.ServiceAccountName
	out.SecurityContext = in.SecurityContext
	out.RuntimeClassName = in.RuntimeClassName
	out.Volumes = in.Volumes
	out.VolumeMounts = in.VolumeMounts
	return nil
}
