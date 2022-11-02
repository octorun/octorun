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

package annotations

import (
	"time"

	"sigs.k8s.io/controller-runtime/pkg/client"

	octorunv1 "octorun.github.io/octorun/api/v1alpha2"
)

// AnnotateTokenExpires give an annotation to given runner
// about registration token expires time.
func AnnotateTokenExpires(obj client.Object, tokenExpiresAt string) {
	annotations := obj.GetAnnotations()
	if annotations == nil {
		annotations = make(map[string]string)
	}

	annotations[octorunv1.AnnotationRunnerTokenExpiresAt] = tokenExpiresAt
	obj.SetAnnotations(annotations)
}

// IsTokenExpired determines if the current registration token already expired.
// It will retrieve token expiration time from annotation and compare it
// with the current time plus 5 minutes. That's mean if token will expires
// in the next 5 minutes it will be considered expired.
//
// If there is no token-expires-at annotation or format is invalid it will considered as expired
// so the controller will request new token and hopefully set the annotation.
func IsTokenExpired(obj client.Object) bool {
	n := time.Now().UTC()
	annotations := obj.GetAnnotations()
	tokenExpire, ok := annotations[octorunv1.AnnotationRunnerTokenExpiresAt]
	if !ok {
		return true
	}

	exp, err := time.Parse(time.RFC3339, tokenExpire)
	if err != nil {
		return true
	}

	return exp.Before(n.Add(5 * time.Minute))
}
