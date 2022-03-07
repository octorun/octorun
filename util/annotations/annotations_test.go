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
	"reflect"
	"testing"
	"time"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	octorunv1alpha1 "octorun.github.io/octorun/api/v1alpha1"
)

func TestAnnotateTokenExpires(t *testing.T) {
	tests := []struct {
		name           string
		runner         *octorunv1alpha1.Runner
		tokenExpiresAt string
	}{
		{
			name: "can_set_token_expire_annotation",
			runner: &octorunv1alpha1.Runner{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-runner",
					Namespace: "test-namespace",
				},
			},
			tokenExpiresAt: "2006-01-02T15:04:05Z07:00",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			AnnotateTokenExpires(tt.runner, tt.tokenExpiresAt)
		})

		annotation := tt.runner.GetAnnotations()
		tokenExpiresAtAnnotation := annotation[octorunv1alpha1.AnnotationRunnerTokenExpiresAt]
		if !reflect.DeepEqual(tokenExpiresAtAnnotation, tt.tokenExpiresAt) {
			t.Errorf("Expected tokenExpiresAtAnnotation to be the same tokenExpiresAt\n tokenExpiresAtAnnotation: %v\n tokenExpiresAt: %v\n", tokenExpiresAtAnnotation, tt.tokenExpiresAt)
		}
	}
}

func TestIsTokenExpired(t *testing.T) {
	tests := []struct {
		name   string
		runner *octorunv1alpha1.Runner
		want   bool
	}{
		{
			name: "invalid_time_format",
			runner: &octorunv1alpha1.Runner{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-runner",
					Namespace: "test-namespace",
					Annotations: map[string]string{
						octorunv1alpha1.AnnotationRunnerTokenExpiresAt: time.Now().Format(time.RFC1123),
					},
				},
			},
			want: true,
		},
		{
			name: "runner_with_registration_token_expire_1h_ago",
			runner: &octorunv1alpha1.Runner{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-runner",
					Namespace: "test-namespace",
					Annotations: map[string]string{
						octorunv1alpha1.AnnotationRunnerTokenExpiresAt: time.Now().Add(-1 * time.Hour).Format(time.RFC3339),
					},
				},
			},
			want: true,
		},

		{
			name: "runner_with_registration_token_expire_in_2m",
			runner: &octorunv1alpha1.Runner{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-runner",
					Namespace: "test-namespace",
					Annotations: map[string]string{
						octorunv1alpha1.AnnotationRunnerTokenExpiresAt: time.Now().Add(2 * time.Minute).Format(time.RFC3339),
					},
				},
			},
			want: true,
		},
		{
			name: "runner_with_registration_token_expire_in_6m",
			runner: &octorunv1alpha1.Runner{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-runner",
					Namespace: "test-namespace",
					Annotations: map[string]string{
						octorunv1alpha1.AnnotationRunnerTokenExpiresAt: time.Now().Add(6 * time.Minute).Format(time.RFC3339),
					},
				},
			},
			want: false,
		},
		{
			name: "runner_with_registration_token_expire_in_1h",
			runner: &octorunv1alpha1.Runner{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-runner",
					Namespace: "test-namespace",
					Annotations: map[string]string{
						octorunv1alpha1.AnnotationRunnerTokenExpiresAt: time.Now().Add(1 * time.Hour).Format(time.RFC3339),
					},
				},
			},
			want: false,
		},
		{
			name: "runner_without_registration_token_expire_annotation",
			runner: &octorunv1alpha1.Runner{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-runner",
					Namespace: "test-namespace",
				},
			},
			want: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := IsTokenExpired(tt.runner); got != tt.want {
				t.Errorf("IsTokenExpired() = %v, want %v", got, tt.want)
			}
		})
	}
}
