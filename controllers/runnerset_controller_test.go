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
	"os"
	"time"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/utils/pointer"
	"sigs.k8s.io/controller-runtime/pkg/client"

	octorunv1alpha1 "octorun.github.io/octorun/api/v1alpha1"
	"octorun.github.io/octorun/util"
)

var _ = Describe("RunnerSetReconciler", func() {
	const (
		timeout  = time.Second * 120
		interval = time.Millisecond * 250
	)

	var (
		ctx       = context.Background()
		selector  metav1.LabelSelector
		runnerset *octorunv1alpha1.RunnerSet
	)

	BeforeEach(func() {
		selector = metav1.LabelSelector{
			MatchLabels: map[string]string{
				"octorun.github.io/runnerset": "myrunnerset",
			},
		}

		runnerset = &octorunv1alpha1.RunnerSet{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "test-runnerset-" + util.RandomString(6),
				Namespace: testns.GetName(),
			},
			Spec: octorunv1alpha1.RunnerSetSpec{
				Runners:  pointer.Int32(3),
				Selector: selector,
				Template: octorunv1alpha1.RunnerTemplateSpec{
					ObjectMeta: metav1.ObjectMeta{
						Labels: selector.MatchLabels,
					},
					Spec: octorunv1alpha1.RunnerSpec{
						URL: os.Getenv("TEST_GITHUB_URL"),
						Image: octorunv1alpha1.RunnerImage{
							Name:       "ghcr.io/octorun/runner",
							PullPolicy: corev1.PullIfNotPresent,
						},
					},
				},
			},
		}
	})

	JustBeforeEach(func() {
		By("Creating a new RunnerSet")
		Expect(crclient.Create(ctx, runnerset)).To(Succeed())
	})

	AfterEach(func() {
		By("Deleting RunnerSet")
		Expect(client.IgnoreNotFound(crclient.Delete(ctx, runnerset))).To(Succeed())
		Eventually(func() bool {
			err := crclient.Get(ctx, client.ObjectKeyFromObject(runnerset), runnerset)
			return apierrors.IsNotFound(err)
		}, timeout, interval).Should(BeTrue())
	})

	Describe("Reconcile", func() {

		WaitRunnersToBeSynced := func(runnerset *octorunv1alpha1.RunnerSet) {
			Eventually(func() bool {
				var runnerCount int32
				runnerList := &octorunv1alpha1.RunnerList{}
				Expect(crclient.List(ctx, runnerList, client.MatchingLabels(selector.MatchLabels))).To(Succeed())
				for i := range runnerList.Items {
					runner := &runnerList.Items[i]
					if !metav1.IsControlledBy(runner, runnerset) {
						continue
					}
					runnerCount += 1
				}
				return runnerCount == *runnerset.Spec.Runners
			}, timeout, interval).Should(BeTrue())
		}

		WaitRunnersBecomeIdle := func(runnerset *octorunv1alpha1.RunnerSet) {
			Eventually(func() bool {
				runnerList := &octorunv1alpha1.RunnerList{}
				Expect(crclient.List(ctx, runnerList, client.MatchingLabels(selector.MatchLabels))).To(Succeed())
				for i := range runnerList.Items {
					runner := &runnerList.Items[i]
					if !metav1.IsControlledBy(runner, runnerset) {
						continue
					}

					if runner.Status.Phase == octorunv1alpha1.RunnerIdlePhase {
						continue
					}
					return false
				}

				return true
			}, timeout, interval).Should(BeTrue())
		}

		Context("When RunnerSet just created", func() {
			It("Should reconcile until has Set of Runners according to the how many runners and template", func() {
				Eventually(func() error {
					return crclient.Get(ctx, client.ObjectKeyFromObject(runnerset), runnerset)
				}, timeout, interval).ShouldNot(HaveOccurred())

				By("Waiting Runners to be created")
				WaitRunnersToBeSynced(runnerset)

				By("Waiting Runners become Idle")
				WaitRunnersBecomeIdle(runnerset)
			})
		})

		Context("When up scale RunnerSet", func() {
			It("Should scaling up RunnerSet runners", func() {
				Eventually(func() error {
					return crclient.Get(ctx, client.ObjectKeyFromObject(runnerset), runnerset)
				}, timeout, interval).ShouldNot(HaveOccurred())

				By("Waiting Runners created")
				WaitRunnersToBeSynced(runnerset)

				By("Waiting created Runners become Idle")
				WaitRunnersBecomeIdle(runnerset)

				By("Scaling the RunnerSet to 5")
				newRunnerSet := runnerset.DeepCopy()
				newRunnerSet.Spec.Runners = pointer.Int32(5)
				Expect(crclient.Patch(ctx, newRunnerSet, client.MergeFrom(runnerset))).To(Succeed())

				By("Waiting for new Runners created")
				WaitRunnersToBeSynced(newRunnerSet)

				By("Waiting all Runners become Idle")
				WaitRunnersBecomeIdle(runnerset)
			})
		})

		Context("When down scale RunnerSet", func() {
			It("Should scaling down RunnerSet runners", func() {
				Eventually(func() error {
					return crclient.Get(ctx, client.ObjectKeyFromObject(runnerset), runnerset)
				}, timeout, interval).ShouldNot(HaveOccurred())

				By("Waiting Runners created")
				WaitRunnersToBeSynced(runnerset)

				By("Waiting created Runners become Idle")
				WaitRunnersBecomeIdle(runnerset)

				By("Scaling the RunnerSet to 1")
				newRunnerSet := runnerset.DeepCopy()
				newRunnerSet.Spec.Runners = pointer.Int32(1)
				Expect(crclient.Patch(ctx, newRunnerSet, client.MergeFrom(runnerset))).To(Succeed())

				By("Waiting for Runners to be synced")
				WaitRunnersToBeSynced(newRunnerSet)

				By("Waiting all Runners become Idle")
				WaitRunnersBecomeIdle(runnerset)
			})
		})
	})
})
