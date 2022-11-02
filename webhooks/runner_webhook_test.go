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

package webhooks

import (
	"context"
	"reflect"
	"testing"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"

	octorunv1 "octorun.github.io/octorun/api/v1alpha2"
	"octorun.github.io/octorun/util"
)

var _ = Describe("RunnerWebhook", func() {
	var (
		ctx    = context.Background()
		runner *octorunv1.Runner
	)

	BeforeEach(func() {
		runner = &octorunv1.Runner{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "test-runner-" + util.RandomString(6),
				Namespace: "default",
			},
			Spec: octorunv1.RunnerSpec{
				URL: "https://github.com/org/repo",
				Image: octorunv1.RunnerImage{
					Name: "ghcr.io/octorun/runner:v2.288.1",
				},
			},
		}
	})

	Describe("Default", func() {
		Context("When only required field is set", func() {
			It("Should defaulting the optional field", func() {
				By("Creating Runner with minimal spec")
				Expect(crclient.Create(ctx, runner)).To(Succeed())
				Expect(crclient.Get(ctx, client.ObjectKeyFromObject(runner), runner)).ToNot(HaveOccurred())

				By("Assert the Runner spec")
				Expect(runner.Spec.Group).NotTo(BeEmpty())
				Expect(runner.Spec.Workdir).NotTo(BeEmpty())
			})
		})
	})

	Describe("ValidateCreate", func() {
		Context("When spec URL is invalid", func() {
			BeforeEach(func() {
				runner.Spec.URL = "https://google.com/org/repo"
			})

			It("Should returns an error", func() {
				Expect(crclient.Create(ctx, runner)).ToNot(Succeed())
			})
		})
	})

	Describe("ValidateUpdate", func() {
		Context("When new spec is equal old spec", func() {
			It("Should success to update without any changes", func() {
				By("Creating the Runner")
				Expect(crclient.Create(ctx, runner)).To(Succeed())
				Expect(crclient.Get(ctx, client.ObjectKeyFromObject(runner), runner)).ToNot(HaveOccurred())

				runner.Spec.URL = "https://github.com/org/repo"
				By("Updating the Runner")
				Expect(crclient.Update(ctx, runner)).ToNot(HaveOccurred())
			})
		})

		Context("When new spec is not equal old spec", func() {
			It("Should fail to update", func() {
				By("Creating the Runner")
				Expect(crclient.Create(ctx, runner)).To(Succeed())
				Expect(crclient.Get(ctx, client.ObjectKeyFromObject(runner), runner)).ToNot(HaveOccurred())

				runner.Spec.URL = "https://github.com/org/repository"
				By("Updating the Runner")
				Expect(crclient.Update(ctx, runner)).To(HaveOccurred())
			})
		})
	})
})

func TestRunnerWebhook_Default(t *testing.T) {
	scheme := runtime.NewScheme()
	utilruntime.Must(octorunv1.AddToScheme(scheme))

	tests := []struct {
		name    string
		obj     runtime.Object
		want    runtime.Object
		wantErr bool
	}{
		{
			name:    "obj_is_not_runner",
			obj:     &octorunv1.RunnerSet{},
			wantErr: true,
		},
		{
			name: "runner_defaulting_successful",
			obj: &octorunv1.Runner{
				ObjectMeta: metav1.ObjectMeta{
					Name: "runner-test",
				},
			},
			want: &octorunv1.Runner{
				ObjectMeta: metav1.ObjectMeta{
					Name: "runner-test",
				},
				Spec: octorunv1.RunnerSpec{
					Workdir: "_work",
					Group:   "Default",
				},
			},
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rw := &RunnerWebhook{
				Client: fake.NewClientBuilder().
					WithScheme(scheme).
					Build(),
			}

			if err := rw.Default(context.Background(), tt.obj); (err != nil) != tt.wantErr {
				t.Errorf("RunnerWebhook.Default() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if !tt.wantErr && !reflect.DeepEqual(tt.obj.(*octorunv1.Runner).Spec, tt.want.(*octorunv1.Runner).Spec) {
				t.Errorf("obj spec = %v, want spec %v", tt.obj.(*octorunv1.Runner).Spec, tt.want.(*octorunv1.Runner).Spec)
				return
			}
		})
	}
}

func TestRunnerWebhook_ValidateCreate(t *testing.T) {
	scheme := runtime.NewScheme()
	utilruntime.Must(octorunv1.AddToScheme(scheme))

	tests := []struct {
		name    string
		obj     runtime.Object
		wantErr bool
	}{
		{
			name:    "obj_is_not_runner",
			obj:     &octorunv1.RunnerSet{},
			wantErr: true,
		},
		{
			name: "runner_missing_spec_url",
			obj: &octorunv1.Runner{
				ObjectMeta: metav1.ObjectMeta{
					Name: "runner-test",
				},
				Spec: octorunv1.RunnerSpec{},
			},
			wantErr: true,
		},
		{
			name: "runner_with_invalid_spec_url",
			obj: &octorunv1.Runner{
				ObjectMeta: metav1.ObjectMeta{
					Name: "runner-test",
				},
				Spec: octorunv1.RunnerSpec{
					URL: "https://google.com",
				},
			},
			wantErr: true,
		},
		{
			name: "runner_with_valid_spec",
			obj: &octorunv1.Runner{
				ObjectMeta: metav1.ObjectMeta{
					Name: "runner-test",
				},
				Spec: octorunv1.RunnerSpec{
					URL: "https://github.com/octorun",
				},
			},
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rw := &RunnerWebhook{
				Client: fake.NewClientBuilder().
					WithScheme(scheme).
					Build(),
			}

			if err := rw.ValidateCreate(context.Background(), tt.obj); (err != nil) != tt.wantErr {
				t.Errorf("RunnerWebhook.ValidateCreate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestRunnerWebhook_ValidateUpdate(t *testing.T) {
	scheme := runtime.NewScheme()
	utilruntime.Must(octorunv1.AddToScheme(scheme))

	tests := []struct {
		name    string
		oldObj  runtime.Object
		newObj  runtime.Object
		wantErr bool
	}{
		{
			name: "old_runner_and_new_runner_is_equal",
			oldObj: &octorunv1.Runner{
				ObjectMeta: metav1.ObjectMeta{
					Name: "runner-test",
				},
				Spec: octorunv1.RunnerSpec{
					URL: "https://github.com/octorun",
				},
			},
			newObj: &octorunv1.Runner{
				ObjectMeta: metav1.ObjectMeta{
					Name: "runner-test",
				},
				Spec: octorunv1.RunnerSpec{
					URL: "https://github.com/octorun",
				},
			},
			wantErr: false,
		},
		{
			name: "old_runner_and_new_runner_is_different",
			oldObj: &octorunv1.Runner{
				ObjectMeta: metav1.ObjectMeta{
					Name: "runner-test",
				},
				Spec: octorunv1.RunnerSpec{
					URL: "https://github.com/octorun",
				},
			},
			newObj: &octorunv1.Runner{
				ObjectMeta: metav1.ObjectMeta{
					Name: "runner-test",
				},
				Spec: octorunv1.RunnerSpec{
					URL: "https://github.com/octorun/repo",
				},
			},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rw := &RunnerWebhook{
				Client: fake.NewClientBuilder().
					WithScheme(scheme).
					Build(),
			}

			if err := rw.ValidateUpdate(context.Background(), tt.oldObj, tt.newObj); (err != nil) != tt.wantErr {
				t.Errorf("RunnerWebhook.ValidateUpdate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestRunnerWebhook_ValidateDelete(t *testing.T) {
	scheme := runtime.NewScheme()
	utilruntime.Must(octorunv1.AddToScheme(scheme))

	tests := []struct {
		name    string
		obj     runtime.Object
		wantErr bool
	}{
		// nothing to test.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rw := &RunnerWebhook{
				Client: fake.NewClientBuilder().
					WithScheme(scheme).
					Build(),
			}

			if err := rw.ValidateDelete(context.Background(), tt.obj); (err != nil) != tt.wantErr {
				t.Errorf("RunnerWebhook.ValidateDelete() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}
