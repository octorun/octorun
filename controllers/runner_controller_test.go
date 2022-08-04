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
	"bytes"
	"context"
	"encoding/base64"
	"os"
	"reflect"
	"testing"
	"time"

	"github.com/golang/mock/gomock"
	gogithub "github.com/google/go-github/v41/github"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/meta"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/tools/record"
	"k8s.io/utils/pointer"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	octorunv1alpha1 "octorun.github.io/octorun/api/v1alpha1"
	mghclient "octorun.github.io/octorun/pkg/github/client/mock"
	"octorun.github.io/octorun/util"
	"octorun.github.io/octorun/util/pod"
	"octorun.github.io/octorun/util/remoteexec"
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

		Context("When Runner has eviction-policy annotation", func() {
			BeforeEach(func() {
				runner.SetAnnotations(map[string]string{
					octorunv1alpha1.AnnotationRunnerEvictionPolicy: "IfNotActive",
				})
			})

			It("Should create Pod with annotation `cluster-autoscaler.kubernetes.io/safe-to-evict=true`", func() {
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

				By("Ensuring new runner Pod has `cluster-autoscaler.kubernetes.io/safe-to-evict` annotation")
				Eventually(func() string {
					Expect(crclient.Get(ctx, client.ObjectKeyFromObject(runnerpod), runnerpod)).NotTo(HaveOccurred())
					return runnerpod.GetAnnotations()["cluster-autoscaler.kubernetes.io/safe-to-evict"]
				}, timeout, interval).Should(Equal("true"))
			})
		})
	})
})

func TestRunnerReconciler_Reconcile(t *testing.T) {
	scheme := runtime.NewScheme()
	utilruntime.Must(clientgoscheme.AddToScheme(scheme))
	utilruntime.Must(octorunv1alpha1.AddToScheme(scheme))

	tests := []struct {
		name           string
		runnerFn       func(runner *octorunv1alpha1.Runner) *octorunv1alpha1.Runner
		runnerPodFn    func(runner *octorunv1alpha1.Runner) *corev1.Pod
		runnerSecretFn func(runner *octorunv1alpha1.Runner) *corev1.Secret
		expectFn       func(cmockr *mghclient.MockClientMockRecorder)
		executor       remoteexec.RemoteExecutor
		want           ctrl.Result
		wantErr        bool
	}{
		{
			name:           "runner_not_found",
			runnerFn:       func(runner *octorunv1alpha1.Runner) *octorunv1alpha1.Runner { return &octorunv1alpha1.Runner{} },
			runnerPodFn:    func(runner *octorunv1alpha1.Runner) *corev1.Pod { return &corev1.Pod{} },
			runnerSecretFn: func(runner *octorunv1alpha1.Runner) *corev1.Secret { return &corev1.Secret{} },
			expectFn:       func(cmockr *mghclient.MockClientMockRecorder) {},
			executor:       &remoteexec.FakeRemoteExecutor{},
			want:           reconcile.Result{},
			wantErr:        false,
		},
		{
			name: "runner_has_deletion_timestamp",
			runnerFn: func(runner *octorunv1alpha1.Runner) *octorunv1alpha1.Runner {
				now := metav1.Now()
				runner.SetDeletionTimestamp(&now)
				return runner
			},
			runnerPodFn:    func(runner *octorunv1alpha1.Runner) *corev1.Pod { return &corev1.Pod{} },
			runnerSecretFn: func(runner *octorunv1alpha1.Runner) *corev1.Secret { return &corev1.Secret{} },
			expectFn: func(cmockr *mghclient.MockClientMockRecorder) {
				cmockr.CreateRunnerToken(gomock.Any(), "https://github.com/octorun").Return(&gogithub.RegistrationToken{
					Token: gogithub.String("faketoken"),
					ExpiresAt: &gogithub.Timestamp{
						Time: time.Now().Add(1 * time.Hour),
					},
				}, nil)
			},
			executor: &remoteexec.FakeRemoteExecutor{},
			want:     reconcile.Result{},
			wantErr:  false,
		},
		{
			name: "runner_has_deletion_timestamp_and_has_active_phase",
			runnerFn: func(runner *octorunv1alpha1.Runner) *octorunv1alpha1.Runner {
				now := metav1.Now()
				runner.Spec.ID = pointer.Int64(1)
				runner.DeletionTimestamp = &now
				runner.Status = octorunv1alpha1.RunnerStatus{
					Phase: octorunv1alpha1.RunnerActivePhase,
				}
				return runner
			},
			runnerPodFn:    func(runner *octorunv1alpha1.Runner) *corev1.Pod { return &corev1.Pod{} },
			runnerSecretFn: func(runner *octorunv1alpha1.Runner) *corev1.Secret { return &corev1.Secret{} },
			expectFn: func(cmockr *mghclient.MockClientMockRecorder) {
				cmockr.GetRunner(gomock.Any(), "https://github.com/octorun", int64(1)).Return(&gogithub.Runner{
					ID:     gogithub.Int64(1),
					Status: gogithub.String("online"),
					Busy:   gogithub.Bool(true),
				}, nil)
			},
			executor: &remoteexec.FakeRemoteExecutor{},
			want:     reconcile.Result{RequeueAfter: 60 * time.Second},
			wantErr:  false,
		},
		{
			name: "runner_has_deletion_timestamp_and_has_active_phase_but_already_completed",
			runnerFn: func(runner *octorunv1alpha1.Runner) *octorunv1alpha1.Runner {
				now := metav1.Now()
				runner.Spec.ID = pointer.Int64(1)
				runner.DeletionTimestamp = &now
				runner.Status = octorunv1alpha1.RunnerStatus{
					Phase: octorunv1alpha1.RunnerActivePhase,
				}
				return runner
			},
			runnerPodFn:    func(runner *octorunv1alpha1.Runner) *corev1.Pod { return &corev1.Pod{} },
			runnerSecretFn: func(runner *octorunv1alpha1.Runner) *corev1.Secret { return &corev1.Secret{} },
			expectFn: func(cmockr *mghclient.MockClientMockRecorder) {
				cmockr.GetRunner(gomock.Any(), "https://github.com/octorun", int64(1)).Return(&gogithub.Runner{
					ID:     gogithub.Int64(1),
					Status: gogithub.String("online"),
					Busy:   gogithub.Bool(false),
				}, nil)
			},
			executor: &remoteexec.FakeRemoteExecutor{},
			want:     reconcile.Result{Requeue: true},
			wantErr:  false,
		},
		{
			name:           "runner_just_created",
			runnerFn:       func(runner *octorunv1alpha1.Runner) *octorunv1alpha1.Runner { return runner },
			runnerPodFn:    func(runner *octorunv1alpha1.Runner) *corev1.Pod { return &corev1.Pod{} },
			runnerSecretFn: func(runner *octorunv1alpha1.Runner) *corev1.Secret { return &corev1.Secret{} },
			expectFn: func(cmockr *mghclient.MockClientMockRecorder) {
				cmockr.CreateRunnerToken(gomock.Any(), "https://github.com/octorun").Return(&gogithub.RegistrationToken{
					Token: gogithub.String("faketoken"),
					ExpiresAt: &gogithub.Timestamp{
						Time: time.Now().Add(1 * time.Hour),
					},
				}, nil)
			},
			executor: &remoteexec.FakeRemoteExecutor{},
			want:     reconcile.Result{},
			wantErr:  false,
		},
		{
			name:     "runnerpod_has_pending_phase",
			runnerFn: func(runner *octorunv1alpha1.Runner) *octorunv1alpha1.Runner { return runner },
			runnerPodFn: func(runner *octorunv1alpha1.Runner) *corev1.Pod {
				pod := podForRunner(runner)
				pod.Status.Phase = corev1.PodPending
				return pod
			},
			runnerSecretFn: func(runner *octorunv1alpha1.Runner) *corev1.Secret { return &corev1.Secret{} },
			expectFn: func(cmockr *mghclient.MockClientMockRecorder) {
				cmockr.CreateRunnerToken(gomock.Any(), "https://github.com/octorun").Return(&gogithub.RegistrationToken{
					Token: gogithub.String("faketoken"),
					ExpiresAt: &gogithub.Timestamp{
						Time: time.Now().Add(1 * time.Hour),
					},
				}, nil)
			},
			executor: &remoteexec.FakeRemoteExecutor{},
			want:     reconcile.Result{},
			wantErr:  false,
		},
		{
			name:     "runnerpod_has_running_phase_but_not_yet_ready",
			runnerFn: func(runner *octorunv1alpha1.Runner) *octorunv1alpha1.Runner { return runner },
			runnerPodFn: func(runner *octorunv1alpha1.Runner) *corev1.Pod {
				pod := podForRunner(runner)
				pod.Status.Phase = corev1.PodRunning
				pod.Status.Conditions = []corev1.PodCondition{
					{
						Type:   corev1.PodReady,
						Status: corev1.ConditionFalse,
					},
				}
				return pod
			},
			runnerSecretFn: func(runner *octorunv1alpha1.Runner) *corev1.Secret { return &corev1.Secret{} },
			expectFn: func(cmockr *mghclient.MockClientMockRecorder) {
				cmockr.CreateRunnerToken(gomock.Any(), "https://github.com/octorun").Return(&gogithub.RegistrationToken{
					Token: gogithub.String("faketoken"),
					ExpiresAt: &gogithub.Timestamp{
						Time: time.Now().Add(1 * time.Hour),
					},
				}, nil)
			},
			executor: &remoteexec.FakeRemoteExecutor{},
			want:     reconcile.Result{},
			wantErr:  false,
		},
		{
			name:     "runnerpod_has_running_phase_and_github_runner_online",
			runnerFn: func(runner *octorunv1alpha1.Runner) *octorunv1alpha1.Runner { return runner },
			runnerPodFn: func(runner *octorunv1alpha1.Runner) *corev1.Pod {
				pod := podForRunner(runner)
				pod.Status.Phase = corev1.PodRunning
				pod.Status.Conditions = []corev1.PodCondition{
					{
						Type:   corev1.PodReady,
						Status: corev1.ConditionTrue,
					},
				}
				return pod
			},
			runnerSecretFn: func(runner *octorunv1alpha1.Runner) *corev1.Secret { return &corev1.Secret{} },
			expectFn: func(cmockr *mghclient.MockClientMockRecorder) {
				cmockr.CreateRunnerToken(gomock.Any(), "https://github.com/octorun").Return(&gogithub.RegistrationToken{
					Token: gogithub.String("faketoken"),
					ExpiresAt: &gogithub.Timestamp{
						Time: time.Now().Add(1 * time.Hour),
					},
				}, nil)
				cmockr.GetRunner(gomock.Any(), "https://github.com/octorun", int64(1)).Return(&gogithub.Runner{
					ID:     gogithub.Int64(1),
					Status: gogithub.String("online"),
				}, nil)
			},
			executor: &remoteexec.FakeRemoteExecutor{
				Out:     bytes.NewBufferString("1"),
				Errout:  &bytes.Buffer{},
				Execerr: nil,
			},
			want:    reconcile.Result{},
			wantErr: false,
		},
		{
			name:     "runnerpod_has_running_phase_and_github_runner_offline",
			runnerFn: func(runner *octorunv1alpha1.Runner) *octorunv1alpha1.Runner { return runner },
			runnerPodFn: func(runner *octorunv1alpha1.Runner) *corev1.Pod {
				pod := podForRunner(runner)
				pod.Status.Phase = corev1.PodRunning
				pod.Status.Conditions = []corev1.PodCondition{
					{
						Type:   corev1.PodReady,
						Status: corev1.ConditionTrue,
					},
				}
				return pod
			},
			runnerSecretFn: func(runner *octorunv1alpha1.Runner) *corev1.Secret { return &corev1.Secret{} },
			expectFn: func(cmockr *mghclient.MockClientMockRecorder) {
				cmockr.CreateRunnerToken(gomock.Any(), "https://github.com/octorun").Return(&gogithub.RegistrationToken{
					Token: gogithub.String("faketoken"),
					ExpiresAt: &gogithub.Timestamp{
						Time: time.Now().Add(1 * time.Hour),
					},
				}, nil)
				cmockr.GetRunner(gomock.Any(), "https://github.com/octorun", int64(1)).Return(&gogithub.Runner{
					ID:     gogithub.Int64(1),
					Status: gogithub.String("offline"),
				}, nil)
			},
			executor: &remoteexec.FakeRemoteExecutor{
				Out:     bytes.NewBufferString("1"),
				Errout:  &bytes.Buffer{},
				Execerr: nil,
			},
			want:    reconcile.Result{RequeueAfter: 5 * time.Second},
			wantErr: false,
		},
		{
			name:     "runnerpod_has_running_phase_and_github_runner_busy",
			runnerFn: func(runner *octorunv1alpha1.Runner) *octorunv1alpha1.Runner { return runner },
			runnerPodFn: func(runner *octorunv1alpha1.Runner) *corev1.Pod {
				pod := podForRunner(runner)
				pod.Status.Phase = corev1.PodRunning
				pod.Status.Conditions = []corev1.PodCondition{
					{
						Type:   corev1.PodReady,
						Status: corev1.ConditionTrue,
					},
				}
				return pod
			},
			runnerSecretFn: func(runner *octorunv1alpha1.Runner) *corev1.Secret { return &corev1.Secret{} },
			expectFn: func(cmockr *mghclient.MockClientMockRecorder) {
				cmockr.CreateRunnerToken(gomock.Any(), "https://github.com/octorun").Return(&gogithub.RegistrationToken{
					Token: gogithub.String("faketoken"),
					ExpiresAt: &gogithub.Timestamp{
						Time: time.Now().Add(1 * time.Hour),
					},
				}, nil)
				cmockr.GetRunner(gomock.Any(), "https://github.com/octorun", int64(1)).Return(&gogithub.Runner{
					ID:     gogithub.Int64(1),
					Status: gogithub.String("online"),
					Busy:   gogithub.Bool(true),
				}, nil)
			},
			executor: &remoteexec.FakeRemoteExecutor{
				Out:     bytes.NewBufferString("1"),
				Errout:  &bytes.Buffer{},
				Execerr: nil,
			},
			want:    reconcile.Result{},
			wantErr: false,
		},
		{
			name:     "runnerpod_has_success_phase",
			runnerFn: func(runner *octorunv1alpha1.Runner) *octorunv1alpha1.Runner { return runner },
			runnerPodFn: func(runner *octorunv1alpha1.Runner) *corev1.Pod {
				pod := podForRunner(runner)
				pod.Status.Phase = corev1.PodSucceeded
				pod.Status.StartTime = &metav1.Time{Time: time.Now()}
				return pod
			},
			runnerSecretFn: func(runner *octorunv1alpha1.Runner) *corev1.Secret { return &corev1.Secret{} },
			expectFn: func(cmockr *mghclient.MockClientMockRecorder) {
				cmockr.CreateRunnerToken(gomock.Any(), "https://github.com/octorun").Return(&gogithub.RegistrationToken{
					Token: gogithub.String("anotherfaketoken"),
					ExpiresAt: &gogithub.Timestamp{
						Time: time.Now().Add(1 * time.Hour),
					},
				}, nil)
			},
			executor: &remoteexec.FakeRemoteExecutor{},
			want:     reconcile.Result{},
			wantErr:  false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mctrl := gomock.NewController(t)
			mghc := mghclient.NewMockClient(mctrl)

			runner := tt.runnerFn(&octorunv1alpha1.Runner{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "runner-test",
					Namespace: "default",
					Labels: map[string]string{
						octorunv1alpha1.LabelRunnerName: "runner-test",
					},
				},
				Spec: octorunv1alpha1.RunnerSpec{
					URL: "https://github.com/octorun",
					Image: octorunv1alpha1.RunnerImage{
						Name: "ghcr.io/octorun/runner",
					},
				},
			})

			runnerPod := tt.runnerPodFn(runner)
			runnerSecret := tt.runnerSecretFn(runner)
			fakec := fake.NewClientBuilder().
				WithScheme(scheme).
				WithObjects(
					runner,
					runnerPod,
					runnerSecret,
				).Build()

			tt.expectFn(mghc.EXPECT())
			r := &RunnerReconciler{
				Client:   fakec,
				Github:   mghc,
				Scheme:   scheme,
				Executor: tt.executor,
				Recorder: new(record.FakeRecorder),
			}

			got, err := r.Reconcile(context.Background(), reconcile.Request{
				NamespacedName: types.NamespacedName{
					Namespace: "default",
					Name:      "runner-test",
				},
			})
			if (err != nil) != tt.wantErr {
				t.Errorf("RunnerReconciler.Reconcile() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("RunnerReconciler.Reconcile() = %v, want %v", got, tt.want)
			}
		})
	}
}
