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
	"reflect"
	"strconv"
	"testing"
	"time"

	"github.com/google/uuid"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	"k8s.io/client-go/tools/record"
	"k8s.io/utils/pointer"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	octorunv1 "octorun.github.io/octorun/api/v1alpha2"
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
		runnerset *octorunv1.RunnerSet
	)

	BeforeEach(func() {
		selector = metav1.LabelSelector{
			MatchLabels: map[string]string{
				"octorun.github.io/runnerset": "myrunnerset",
			},
		}

		runnerset = &octorunv1.RunnerSet{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "test-runnerset-" + util.RandomString(6),
				Namespace: testns.GetName(),
			},
			Spec: octorunv1.RunnerSetSpec{
				Runners:  pointer.Int32(3),
				Selector: selector,
				Template: octorunv1.RunnerTemplateSpec{
					ObjectMeta: metav1.ObjectMeta{
						Labels: selector.MatchLabels,
					},
					Spec: octorunv1.RunnerSpec{
						URL: os.Getenv("TEST_GITHUB_URL"),
						Image: octorunv1.RunnerImage{
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

		WaitRunnersToBeSynced := func(runnerset *octorunv1.RunnerSet) {
			Eventually(func() bool {
				var runnerCount int32
				runnerList := &octorunv1.RunnerList{}
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

		WaitRunnersBecomeIdle := func(runnerset *octorunv1.RunnerSet) {
			Eventually(func() bool {
				runnerList := &octorunv1.RunnerList{}
				Expect(crclient.List(ctx, runnerList, client.MatchingLabels(selector.MatchLabels))).To(Succeed())
				for i := range runnerList.Items {
					runner := &runnerList.Items[i]
					if !metav1.IsControlledBy(runner, runnerset) {
						continue
					}

					if runner.Status.Phase == octorunv1.RunnerIdlePhase {
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

func TestRunnerSetReconciler_Reconcile(t *testing.T) {
	scheme := runtime.NewScheme()
	utilruntime.Must(octorunv1.AddToScheme(scheme))

	runnerListForRunnerSet := func(rs *octorunv1.RunnerSet) *octorunv1.RunnerList {
		runners := int(pointer.Int32Deref(rs.Spec.Runners, 0))
		items := make([]octorunv1.Runner, 0, runners)
		for i := 0; i < runners; i++ {
			items = append(items, octorunv1.Runner{
				ObjectMeta: metav1.ObjectMeta{
					Name:            rs.Name + "-" + strconv.Itoa(i),
					Namespace:       rs.Namespace,
					Labels:          rs.Spec.Selector.MatchLabels,
					OwnerReferences: rs.GetOwnerReferences(),
				},
				Spec: octorunv1.RunnerSpec{
					URL: rs.Spec.Template.Spec.URL,
				},
			})
		}

		return &octorunv1.RunnerList{
			Items: items,
		}
	}

	tests := []struct {
		name         string
		runnersetFn  func(rs *octorunv1.RunnerSet) *octorunv1.RunnerSet
		runnerListFn func(rs *octorunv1.RunnerSet) *octorunv1.RunnerList
		want         ctrl.Result
		wantErr      bool
	}{
		{
			name:         "runnerset_just_created",
			runnersetFn:  func(rs *octorunv1.RunnerSet) *octorunv1.RunnerSet { return rs },
			runnerListFn: func(rs *octorunv1.RunnerSet) *octorunv1.RunnerList { return &octorunv1.RunnerList{} },
			want:         reconcile.Result{},
			wantErr:      false,
		},
		{
			name:         "runnerset_not_found",
			runnersetFn:  func(rs *octorunv1.RunnerSet) *octorunv1.RunnerSet { return &octorunv1.RunnerSet{} },
			runnerListFn: func(rs *octorunv1.RunnerSet) *octorunv1.RunnerList { return &octorunv1.RunnerList{} },
			want:         reconcile.Result{},
			wantErr:      false,
		},
		{
			name: "runnerset_has_deletion_timestamp",
			runnersetFn: func(rs *octorunv1.RunnerSet) *octorunv1.RunnerSet {
				now := metav1.Now()
				rs.DeletionTimestamp = &now
				return rs
			},
			runnerListFn: func(rs *octorunv1.RunnerSet) *octorunv1.RunnerList { return &octorunv1.RunnerList{} },
			want:         reconcile.Result{},
			wantErr:      false,
		},
		{
			name:        "runners_has_idle_phase",
			runnersetFn: func(rs *octorunv1.RunnerSet) *octorunv1.RunnerSet { return rs },
			runnerListFn: func(rs *octorunv1.RunnerSet) *octorunv1.RunnerList {
				var items []octorunv1.Runner
				runnerList := runnerListForRunnerSet(rs)
				for _, item := range runnerList.Items {
					item.Status.Phase = octorunv1.RunnerIdlePhase
					items = append(items, item)
				}

				runnerList.Items = items
				return runnerList
			},
			want:    reconcile.Result{},
			wantErr: false,
		},
		{
			name:        "runners_has_active_phase",
			runnersetFn: func(rs *octorunv1.RunnerSet) *octorunv1.RunnerSet { return rs },
			runnerListFn: func(rs *octorunv1.RunnerSet) *octorunv1.RunnerList {
				var items []octorunv1.Runner
				runnerList := runnerListForRunnerSet(rs)
				for _, item := range runnerList.Items {
					item.Status.Phase = octorunv1.RunnerActivePhase
					items = append(items, item)
				}

				runnerList.Items = items
				return runnerList
			},
			want:    reconcile.Result{},
			wantErr: false,
		},
		{
			name:        "runners_has_complete_phase",
			runnersetFn: func(rs *octorunv1.RunnerSet) *octorunv1.RunnerSet { return rs },
			runnerListFn: func(rs *octorunv1.RunnerSet) *octorunv1.RunnerList {
				var items []octorunv1.Runner
				runnerList := runnerListForRunnerSet(rs)
				for _, item := range runnerList.Items {
					item.Status.Phase = octorunv1.RunnerCompletePhase
					items = append(items, item)
				}

				runnerList.Items = items
				return runnerList
			},
			want:    reconcile.Result{},
			wantErr: false,
		},
		{
			name:        "too_many_runners",
			runnersetFn: func(rs *octorunv1.RunnerSet) *octorunv1.RunnerSet { return rs },
			runnerListFn: func(rs *octorunv1.RunnerSet) *octorunv1.RunnerList {
				now := metav1.Now()
				var items []octorunv1.Runner
				runnerList := runnerListForRunnerSet(rs)
				items = append(items, runnerList.Items...)
				items = append(items, octorunv1.Runner{
					ObjectMeta: metav1.ObjectMeta{
						Name:            rs.Name + "-" + strconv.Itoa(len(items)+1),
						Namespace:       rs.Namespace,
						Labels:          rs.Spec.Selector.MatchLabels,
						OwnerReferences: rs.GetOwnerReferences(),
					},
					Spec: octorunv1.RunnerSpec{
						URL: rs.Spec.Template.Spec.URL,
					},
					Status: octorunv1.RunnerStatus{
						Phase: octorunv1.RunnerActivePhase,
					},
				})
				items = append(items, octorunv1.Runner{
					ObjectMeta: metav1.ObjectMeta{
						Name:              rs.Name + "-" + strconv.Itoa(len(items)+1),
						Namespace:         rs.Namespace,
						Labels:            rs.Spec.Selector.MatchLabels,
						OwnerReferences:   rs.GetOwnerReferences(),
						DeletionTimestamp: &now,
					},
					Spec: octorunv1.RunnerSpec{
						URL: rs.Spec.Template.Spec.URL,
					},
				})

				runnerList.Items = items
				return runnerList
			},
			want:    reconcile.Result{},
			wantErr: false,
		},
		{
			name:        "oneof_runners_has_ownerref_but_not_this_runnerset",
			runnersetFn: func(rs *octorunv1.RunnerSet) *octorunv1.RunnerSet { return rs },
			runnerListFn: func(rs *octorunv1.RunnerSet) *octorunv1.RunnerList {
				var items []octorunv1.Runner
				runnerList := runnerListForRunnerSet(rs)
				items = append(items, runnerList.Items...)
				items = append(items, octorunv1.Runner{
					ObjectMeta: metav1.ObjectMeta{
						Name:      rs.Name + "-" + strconv.Itoa(len(items)+1),
						Namespace: rs.Namespace,
						Labels:    rs.Spec.Selector.MatchLabels,
						OwnerReferences: []metav1.OwnerReference{
							{
								Kind:       rs.Kind,
								APIVersion: rs.APIVersion,
								Name:       "another-runnerset",
								UID:        types.UID(uuid.New().String()),
								Controller: pointer.Bool(true),
							},
						},
					},
					Spec: octorunv1.RunnerSpec{
						URL: rs.Spec.Template.Spec.URL,
					},
				})

				runnerList.Items = items
				return runnerList
			},
			want:    reconcile.Result{},
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			runnerset := tt.runnersetFn(&octorunv1.RunnerSet{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "runnerset-test",
					Namespace: "default",
				},
				Spec: octorunv1.RunnerSetSpec{
					Runners: pointer.Int32(3),
					Selector: metav1.LabelSelector{
						MatchLabels: map[string]string{
							"octorun.github.io/runnerset": "myrunnerset",
						},
					},
					Template: octorunv1.RunnerTemplateSpec{
						ObjectMeta: metav1.ObjectMeta{
							Labels: map[string]string{
								"octorun.github.io/runnerset": "myrunnerset",
							},
						},
						Spec: octorunv1.RunnerSpec{
							URL: "https://github.com/octorun",
						},
					},
				},
			})

			runnerList := tt.runnerListFn(runnerset)
			fakec := fake.NewClientBuilder().
				WithScheme(scheme).
				WithObjects(runnerset).
				WithLists(runnerList).
				Build()

			r := &RunnerSetReconciler{
				Client:   fakec,
				Scheme:   scheme,
				Recorder: new(record.FakeRecorder),
			}

			got, err := r.Reconcile(context.Background(), reconcile.Request{
				NamespacedName: types.NamespacedName{
					Name:      "runnerset-test",
					Namespace: "default",
				},
			})
			if (err != nil) != tt.wantErr {
				t.Errorf("RunnerSetReconciler.Reconcile() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("RunnerSetReconciler.Reconcile() = %v, want %v", got, tt.want)
			}
		})
	}
}
