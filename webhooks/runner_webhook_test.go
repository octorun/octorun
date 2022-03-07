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

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"

	octorunv1alpha1 "octorun.github.io/octorun/api/v1alpha1"
	"octorun.github.io/octorun/util"
)

var _ = Describe("RunnerWebhook", func() {
	var (
		ctx    = context.Background()
		runner *octorunv1alpha1.Runner
	)

	BeforeEach(func() {
		runner = &octorunv1alpha1.Runner{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "test-runner-" + util.RandomString(6),
				Namespace: "default",
			},
			Spec: octorunv1alpha1.RunnerSpec{
				URL: "github.com/org/repo",
				Image: octorunv1alpha1.RunnerImage{
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

	Describe("ValidateUpdate", func() {
		Context("When new spec is equal old spec", func() {
			It("Should success to update without any changes", func() {
				By("Creating the Runner")
				Expect(crclient.Create(ctx, runner)).To(Succeed())
				Expect(crclient.Get(ctx, client.ObjectKeyFromObject(runner), runner)).ToNot(HaveOccurred())

				runner.Spec.URL = "github.com/org/repo"
				By("Updating the Runner")
				Expect(crclient.Update(ctx, runner)).ToNot(HaveOccurred())
			})
		})

		Context("When new spec is not equal old spec", func() {
			It("Should fail to update", func() {
				By("Creating the Runner")
				Expect(crclient.Create(ctx, runner)).To(Succeed())
				Expect(crclient.Get(ctx, client.ObjectKeyFromObject(runner), runner)).ToNot(HaveOccurred())

				runner.Spec.URL = "github.com/org/repository"
				By("Updating the Runner")
				Expect(crclient.Update(ctx, runner)).To(HaveOccurred())
			})
		})
	})
})
