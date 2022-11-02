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

package hooks

import (
	"context"
	"reflect"
	"testing"

	"github.com/google/go-github/v41/github"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/utils/pointer"

	octorunv1 "octorun.github.io/octorun/api/v1alpha2"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
)

func TestGithubHook_runnerCompositeIndexer(t *testing.T) {
	tests := []struct {
		name string
		obj  client.Object
		want int
	}{
		{
			name: "obj_is_not_runner",
			obj: &corev1.Pod{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "foo",
					Namespace: "bar",
				},
			},
			want: 0,
		},
		{
			name: "runner_spec_id_is_null",
			obj: &octorunv1.Runner{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "foo",
					Namespace: "bar",
				},
				Spec: octorunv1.RunnerSpec{
					ID: nil,
				},
			},
			want: 0,
		},
		{
			name: "runner_spec_id_not_null",
			obj: &octorunv1.Runner{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "foo",
					Namespace: "bar",
				},
				Spec: octorunv1.RunnerSpec{
					ID: pointer.Int64(1),
				},
			},
			want: 1,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gh := &GithubHook{}
			if got := gh.runnerCompositeIndexer(tt.obj); !reflect.DeepEqual(len(got), tt.want) {
				t.Errorf("GithubHook.runnerCompositeIndexer() = %v, want %v", len(got), tt.want)
			}
		})
	}
}

func TestGithubHook_triggerRunnerReconciliation(t *testing.T) {
	scheme := runtime.NewScheme()
	if err := octorunv1.AddToScheme(scheme); err != nil {
		t.Errorf("unexpected AddToScheme error: %v", err)
	}

	fakec := fake.NewClientBuilder().
		WithScheme(scheme).
		WithObjects(&octorunv1.Runner{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "foo",
				Namespace: "default",
			},
		}).Build()

	type fields struct {
		client.Client
	}

	tests := []struct {
		name      string
		fields    fields
		runnerKey client.ObjectKey
		wantErr   bool
	}{
		{
			name: "runner_not_found_should_ignore_error",
			fields: fields{
				Client: fakec,
			},
			runnerKey: types.NamespacedName{Namespace: "default", Name: "barr"},
			wantErr:   false,
		},
		{
			name: "runner_found",
			fields: fields{
				Client: fakec,
			},
			runnerKey: types.NamespacedName{Namespace: "default", Name: "foo"},
			wantErr:   false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gh := &GithubHook{
				Client: tt.fields.Client,
			}
			ctx := context.Background()
			if err := gh.triggerRunnerReconciliation(ctx, tt.runnerKey); (err != nil) != tt.wantErr {
				t.Errorf("GithubHook.triggerRunnerReconciliation() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestGithubHook_processWorkflowJobEvent(t *testing.T) {
	type fields struct {
		Client client.Client
	}
	type args struct {
		ctx   context.Context
		event *github.WorkflowJobEvent
	}
	tests := []struct {
		name   string
		fields fields
		args   args
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gh := &GithubHook{
				Client: tt.fields.Client,
			}
			gh.processWorkflowJobEvent(tt.args.ctx, tt.args.event)
		})
	}
}
