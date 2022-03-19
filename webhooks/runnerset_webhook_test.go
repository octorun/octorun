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
	"testing"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	"k8s.io/utils/pointer"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"

	octorunv1alpha1 "octorun.github.io/octorun/api/v1alpha1"
	"octorun.github.io/octorun/util"
)

var _ = Describe("RunnerSetWebhook", func() {
	var (
		ctx       = context.Background()
		selector  metav1.LabelSelector
		runnerset *octorunv1alpha1.RunnerSet
	)

	BeforeEach(func() {
		selector = metav1.LabelSelector{
			MatchLabels: map[string]string{
				"runners": "my-runners-" + util.RandomString(6),
			},
		}

		runnerset = &octorunv1alpha1.RunnerSet{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "test-runnerset-" + util.RandomString(6),
				Namespace: "default",
			},
			Spec: octorunv1alpha1.RunnerSetSpec{
				Runners:  pointer.Int32(3),
				Selector: selector,
				Template: octorunv1alpha1.RunnerTemplateSpec{
					ObjectMeta: metav1.ObjectMeta{
						Labels: selector.MatchLabels,
					},
					Spec: octorunv1alpha1.RunnerSpec{
						URL: "https://github.com/org/repo",
						Image: octorunv1alpha1.RunnerImage{
							Name: "ghcr.io/octorun/runner:v2.288.1",
						},
					},
				},
			},
		}
	})

	Describe("Default", func() {
		Context("When just create the RunnerSet", func() {
			It("Should add well known runnerset template label", func() {
				By("Creating RunnerSet")
				Expect(crclient.Create(ctx, runnerset)).To(Succeed())
				Expect(crclient.Get(ctx, client.ObjectKeyFromObject(runnerset), runnerset)).ToNot(HaveOccurred())

				By("Assert the RunnerSet spec template labels")
				Expect(runnerset.Spec.Template.Labels).NotTo(BeEmpty())
			})
		})
	})

	Describe("ValidateCreate", func() {
		Context("When spec URL is invalid", func() {
			BeforeEach(func() {
				runnerset.Spec.Template.Spec.URL = "https://google.com/org/repo/foo"
			})

			It("Should returns an error", func() {
				Expect(crclient.Create(ctx, runnerset)).ToNot(Succeed())
			})
		})
	})

	Describe("ValidateUpdate", func() {
		Context("When spec URL is valid", func() {
			It("Should returns an error", func() {
				By("Creating the RunnerSet")
				Expect(crclient.Create(ctx, runnerset)).To(Succeed())
				Expect(crclient.Get(ctx, client.ObjectKeyFromObject(runnerset), runnerset)).ToNot(HaveOccurred())

				runnerset.Spec.Template.Spec.URL = "https://github.com/org/repo"
				By("Updating the RunnerSet")
				Expect(crclient.Update(ctx, runnerset)).To(Succeed())
			})
		})

		Context("When spec URL is invalid", func() {
			It("Should returns an error", func() {
				By("Creating the RunnerSet")
				Expect(crclient.Create(ctx, runnerset)).To(Succeed())
				Expect(crclient.Get(ctx, client.ObjectKeyFromObject(runnerset), runnerset)).ToNot(HaveOccurred())

				runnerset.Spec.Template.Spec.URL = "https://google.com/org/repo"
				By("Updating the RunnerSet")
				Expect(crclient.Update(ctx, runnerset)).To(HaveOccurred())
			})
		})
	})
})

func TestRunnerSetWebhook_Default(t *testing.T) {
	scheme := runtime.NewScheme()
	utilruntime.Must(octorunv1alpha1.AddToScheme(scheme))

	tests := []struct {
		name    string
		obj     runtime.Object
		wantErr bool
	}{
		{
			name:    "obj_is_not_runnerset",
			obj:     &octorunv1alpha1.Runner{},
			wantErr: true,
		},
		{
			name: "runnerset_defaulting_successful",
			obj: &octorunv1alpha1.RunnerSet{
				ObjectMeta: metav1.ObjectMeta{
					Name: "runnerset-test",
				},
			},
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rsw := &RunnerSetWebhook{
				Client: fake.NewClientBuilder().
					WithScheme(scheme).
					Build(),
			}

			if err := rsw.Default(context.Background(), tt.obj); (err != nil) != tt.wantErr {
				t.Errorf("RunnerSetWebhook.Default() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestRunnerSetWebhook_ValidateCreate(t *testing.T) {
	scheme := runtime.NewScheme()
	utilruntime.Must(octorunv1alpha1.AddToScheme(scheme))

	tests := []struct {
		name    string
		obj     runtime.Object
		wantErr bool
	}{
		{
			name:    "obj_is_not_runnerset",
			obj:     &octorunv1alpha1.Runner{},
			wantErr: true,
		},
		{
			name: "runnerset_with_invalid_template_url_spec",
			obj: &octorunv1alpha1.RunnerSet{
				ObjectMeta: metav1.ObjectMeta{
					Name: "runnerset-test",
				},
				Spec: octorunv1alpha1.RunnerSetSpec{
					Template: octorunv1alpha1.RunnerTemplateSpec{
						Spec: octorunv1alpha1.RunnerSpec{
							URL: "https://google.com",
						},
					},
				},
			},
			wantErr: true,
		},
		{
			name: "runnerset_with_valid_spec",
			obj: &octorunv1alpha1.RunnerSet{
				ObjectMeta: metav1.ObjectMeta{
					Name: "runnerset-test",
				},
				Spec: octorunv1alpha1.RunnerSetSpec{
					Template: octorunv1alpha1.RunnerTemplateSpec{
						Spec: octorunv1alpha1.RunnerSpec{
							URL: "https://github.com/octorun",
						},
					},
				},
			},
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rsw := &RunnerSetWebhook{
				Client: fake.NewClientBuilder().
					WithScheme(scheme).
					Build(),
			}

			if err := rsw.ValidateCreate(context.Background(), tt.obj); (err != nil) != tt.wantErr {
				t.Errorf("RunnerSetWebhook.ValidateCreate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestRunnerSetWebhook_ValidateUpdate(t *testing.T) {
	scheme := runtime.NewScheme()
	utilruntime.Must(octorunv1alpha1.AddToScheme(scheme))

	tests := []struct {
		name    string
		oldObj  runtime.Object
		newObj  runtime.Object
		wantErr bool
	}{
		{
			name:    "oldObj_is_not_runnerset",
			oldObj:  &octorunv1alpha1.Runner{},
			wantErr: true,
		},
		{
			name:    "newObj_is_not_runnerset",
			oldObj:  &octorunv1alpha1.RunnerSet{},
			newObj:  &octorunv1alpha1.Runner{},
			wantErr: true,
		},
		{
			name:   "newObj_with_invalid_template_url_spec",
			oldObj: &octorunv1alpha1.RunnerSet{},
			newObj: &octorunv1alpha1.RunnerSet{
				ObjectMeta: metav1.ObjectMeta{
					Name: "runnerset-test",
				},
				Spec: octorunv1alpha1.RunnerSetSpec{
					Template: octorunv1alpha1.RunnerTemplateSpec{
						Spec: octorunv1alpha1.RunnerSpec{
							URL: "https://google.com",
						},
					},
				},
			},
			wantErr: true,
		},
		{
			name:   "newObj_with_valid_template_url_spec",
			oldObj: &octorunv1alpha1.RunnerSet{},
			newObj: &octorunv1alpha1.RunnerSet{
				ObjectMeta: metav1.ObjectMeta{
					Name: "runnerset-test",
				},
				Spec: octorunv1alpha1.RunnerSetSpec{
					Template: octorunv1alpha1.RunnerTemplateSpec{
						Spec: octorunv1alpha1.RunnerSpec{
							URL: "https://github.com/octorun",
						},
					},
				},
			},
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rsw := &RunnerSetWebhook{
				Client: fake.NewClientBuilder().
					WithScheme(scheme).
					Build(),
			}

			if err := rsw.ValidateUpdate(context.Background(), tt.oldObj, tt.newObj); (err != nil) != tt.wantErr {
				t.Errorf("RunnerSetWebhook.ValidateUpdate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestRunnerSetWebhook_ValidateDelete(t *testing.T) {
	scheme := runtime.NewScheme()
	utilruntime.Must(octorunv1alpha1.AddToScheme(scheme))

	tests := []struct {
		name    string
		obj     runtime.Object
		wantErr bool
	}{
		// nothing to test.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rsw := &RunnerSetWebhook{
				Client: fake.NewClientBuilder().
					WithScheme(scheme).
					Build(),
			}

			if err := rsw.ValidateDelete(context.Background(), tt.obj); (err != nil) != tt.wantErr {
				t.Errorf("RunnerSetWebhook.ValidateDelete() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}
