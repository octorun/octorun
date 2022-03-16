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

package controllers

import (
	"context"
	"encoding/base64"
	"os"
	"time"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/meta"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"

	octorunv1alpha1 "octorun.github.io/octorun/api/v1alpha1"
	"octorun.github.io/octorun/util"
	"octorun.github.io/octorun/util/pod"
)

var _ = Describe("RunnerReconciler", func() {
	const (
		timeout  = time.Second * 60
		interval = time.Millisecond * 250
	)

	var (
		ctx    = context.Background()
		runner *octorunv1alpha1.Runner
	)

	BeforeEach(func() {
		runner = &octorunv1alpha1.Runner{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "test-runner-" + util.RandomString(6),
				Namespace: testns.GetName(),
				Labels: map[string]string{
					"octorun.github.io/runners": "test-runner",
				},
			},
			Spec: octorunv1alpha1.RunnerSpec{
				URL: os.Getenv("TEST_GITHUB_URL"),
				Image: octorunv1alpha1.RunnerImage{
					Name:       "ghcr.io/octorun/runner",
					PullPolicy: corev1.PullIfNotPresent,
				},
				Group:   "Default",
				Workdir: "/work-dir",
				Resources: corev1.ResourceRequirements{
					Limits: corev1.ResourceList{
						corev1.ResourceCPU:    resource.MustParse("500m"),
						corev1.ResourceMemory: resource.MustParse("128Mi"),
					},
					Requests: corev1.ResourceList{
						corev1.ResourceCPU:    resource.MustParse("500m"),
						corev1.ResourceMemory: resource.MustParse("128Mi"),
					},
				},
				Volumes: []corev1.Volume{
					{
						Name: "work-dir",
						VolumeSource: corev1.VolumeSource{
							EmptyDir: &corev1.EmptyDirVolumeSource{},
						},
					},
				},
				VolumeMounts: []corev1.VolumeMount{
					{
						Name:      "work-dir",
						MountPath: "/work-dir",
					},
				},
			},
		}
	})

	JustBeforeEach(func() {
		By("Creating a new Runner")
		Expect(crclient.Create(ctx, runner)).To(Succeed())
	})

	AfterEach(func() {
		By("Cleanup runner")
		Expect(client.IgnoreNotFound(crclient.Delete(ctx, runner))).To(Succeed())
		Eventually(func() bool {
			err := crclient.Get(ctx, client.ObjectKeyFromObject(runner), runner)
			return apierrors.IsNotFound(err)
		}, timeout, interval).Should(BeTrue())
	})

	Describe("Reconcile", func() {
		Context("When Runner just created", func() {
			It("Should reconcile until Runner has Idle phase", func() {
				By("Ensuring new Runner has been created successfully")
				Eventually(func() error {
					return crclient.Get(ctx, client.ObjectKeyFromObject(runner), runner)
				}, timeout, interval).ShouldNot(HaveOccurred())

				secret := secretForRunner(runner)
				runnerpod := podForRunner(runner)

				By("Waiting Registration Token Secret created")
				Eventually(func() error {
					return crclient.Get(ctx, client.ObjectKeyFromObject(secret), secret)
				}, timeout, interval).ShouldNot(HaveOccurred())

				By("Waiting Runner Pod created")
				Eventually(func() error {
					return crclient.Get(ctx, client.ObjectKeyFromObject(runnerpod), runnerpod)
				}, timeout, interval).ShouldNot(HaveOccurred())

				By("Ensure Runner Pod in Pending phase")
				Eventually(func() bool {
					Expect(crclient.Get(ctx, client.ObjectKeyFromObject(runnerpod), runnerpod)).NotTo(HaveOccurred())
					return runnerpod.Status.Phase == corev1.PodPending
				}, timeout, interval).Should(BeTrue())

				By("Waiting until Runner Pod become Running and Ready")
				Eventually(func() bool {
					Expect(crclient.Get(ctx, client.ObjectKeyFromObject(runnerpod), runnerpod)).NotTo(HaveOccurred())
					return runnerpod.Status.Phase == corev1.PodRunning && pod.PodConditionIsReady(runnerpod)
				}, timeout, interval).Should(BeTrue())

				By("Ensure Runner has RunnerID")
				Eventually(func() bool {
					Expect(crclient.Get(ctx, client.ObjectKeyFromObject(runner), runner))
					return runner.Spec.ID != nil
				}, timeout, interval).Should(BeTrue())

				By("Ensure Runner is Online")
				Eventually(func() bool {
					Expect(crclient.Get(ctx, client.ObjectKeyFromObject(runner), runner))
					return meta.IsStatusConditionTrue(runner.Status.Conditions, octorunv1alpha1.RunnerConditionOnline)
				}, timeout, interval).Should(BeTrue())

				By("Ensure Runner become Idle phase")
				Eventually(func() bool {
					Expect(crclient.Get(ctx, client.ObjectKeyFromObject(runner), runner))
					return runner.Status.Phase == octorunv1alpha1.RunnerIdlePhase
				}, timeout, interval).Should(BeTrue())
			})
		})

		Context("When Runner registration token expired", func() {
			It("Should recreate registration token", func() {
				By("Ensuring new Runner has been created successfully")
				Eventually(func() error {
					return crclient.Get(ctx, client.ObjectKeyFromObject(runner), runner)
				}, timeout, interval).ShouldNot(HaveOccurred())

				secret := secretForRunner(runner)
				runnerpod := podForRunner(runner)

				By("Waiting Registration Token Secret created")
				Eventually(func() error {
					return crclient.Get(ctx, client.ObjectKeyFromObject(secret), secret)
				}, timeout, interval).ShouldNot(HaveOccurred())

				By("Waiting until Runner Pod become Running and Ready")
				Eventually(func() bool {
					Expect(crclient.Get(ctx, client.ObjectKeyFromObject(runnerpod), runnerpod)).NotTo(HaveOccurred())
					return runnerpod.Status.Phase == corev1.PodRunning && pod.PodConditionIsReady(runnerpod)
				}, timeout, interval).Should(BeTrue())

				oldsecret := secret.DeepCopy()
				By("Simulate token expires")
				secret.SetAnnotations(map[string]string{
					octorunv1alpha1.AnnotationRunnerTokenExpiresAt: time.Now().Add(-1 * time.Hour).Format(time.RFC3339),
				})
				Expect(crclient.Patch(ctx, secret, client.MergeFrom(oldsecret))).To(Succeed())

				By("Comparing old registration token with new registration token")
				oldtoken, _ := base64.StdEncoding.DecodeString(string(oldsecret.Data["token"]))
				Eventually(func() string {
					newsecret := &corev1.Secret{}
					Expect(crclient.Get(ctx, client.ObjectKeyFromObject(secret), newsecret)).NotTo(HaveOccurred())
					newtoken, _ := base64.StdEncoding.DecodeString(string(newsecret.Data["token"]))
					return string(newtoken)
				}, timeout, interval).ShouldNot(Equal(string(oldtoken)))
			})
		})

		Context("When Runner spec URL is not found or unaccessible", func() {
			BeforeEach(func() {
				runner.Spec.URL = "https://github.com/octonotfound"
			})

			It("Should terminate reconciliation", func() {
				By("Ensuring new Runner has been created successfully")
				Eventually(func() error {
					return crclient.Get(ctx, client.ObjectKeyFromObject(runner), runner)
				}, timeout, interval).ShouldNot(HaveOccurred())

				time.Sleep(2 * time.Second)
				runnerpod := podForRunner(runner)
				runnersecret := secretForRunner(runner)

				By("Ensuring Runner Secret not to created")
				Expect(apierrors.IsNotFound(crclient.Get(ctx, client.ObjectKeyFromObject(runnersecret), runnersecret))).To(BeTrue())

				By("Ensuring Runner Pod not to created")
				Expect(apierrors.IsNotFound(crclient.Get(ctx, client.ObjectKeyFromObject(runnerpod), runnerpod))).To(BeTrue())
			})
		})
	})
})
