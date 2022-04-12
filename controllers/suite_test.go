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
	"path/filepath"
	"testing"
	"time"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/record"
	"k8s.io/utils/pointer"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/envtest"
	"sigs.k8s.io/controller-runtime/pkg/log/zap"

	octorunv1alpha1 "octorun.github.io/octorun/api/v1alpha1"
	"octorun.github.io/octorun/pkg/github"
	"octorun.github.io/octorun/util"
	"octorun.github.io/octorun/util/pod"
	// +kubebuilder:scaffold:imports
)

// These tests use Ginkgo (BDD-style Go testing framework). Refer to
// http://onsi.github.io/ginkgo/ to learn more about Ginkgo.

var (
	ctx      context.Context
	cancel   context.CancelFunc
	cfg      *rest.Config
	crclient client.Client
	testenv  *envtest.Environment
	testns   *corev1.Namespace
)

func TestControllers(t *testing.T) {
	if testing.Short() {
		t.Skip()
	}

	if _, ok := os.LookupEnv("TEST_GITHUB_URL"); !ok {
		t.Skip("TEST_GITHUB_URL env is not set")
	}

	if _, ok := os.LookupEnv("TEST_GITHUB_ACCESS_TOKEN"); !ok {
		t.Skip("TEST_GITHUB_ACCESS_TOKEN env is not set")
	}

	RegisterFailHandler(Fail)
	RunSpecs(t, "Controller Suite")
}

var _ = BeforeSuite(func() {
	ctrl.SetLogger(zap.New(zap.WriteTo(GinkgoWriter), zap.UseDevMode(true)))
	ctx, cancel = context.WithCancel(context.TODO())
	By("bootstrapping test environment")
	testenv = &envtest.Environment{
		UseExistingCluster:    pointer.Bool(true),
		CRDDirectoryPaths:     []string{filepath.Join("..", "config", "crd", "bases")},
		ErrorIfCRDPathMissing: true,
	}

	scheme := runtime.NewScheme()
	err := clientgoscheme.AddToScheme(scheme)
	Expect(err).NotTo(HaveOccurred())

	cfg, err = testenv.Start()
	Expect(err).NotTo(HaveOccurred())
	Expect(cfg).NotTo(BeNil())

	err = octorunv1alpha1.AddToScheme(scheme)
	Expect(err).NotTo(HaveOccurred())

	crclient, err = client.New(cfg, client.Options{Scheme: scheme})
	Expect(err).NotTo(HaveOccurred())
	Expect(crclient).NotTo(BeNil())

	gh, err := github.New(&github.Options{
		AccessToken: os.Getenv("TEST_GITHUB_ACCESS_TOKEN"),
		APIEndpoint: "https://api.github.com/",
	})
	Expect(err).ToNot(HaveOccurred())

	mgr, err := ctrl.NewManager(cfg, ctrl.Options{
		Scheme: scheme,
	})
	Expect(err).ToNot(HaveOccurred())

	err = (&RunnerReconciler{
		Client:   mgr.GetClient(),
		Scheme:   mgr.GetScheme(),
		Github:   gh.GetClient(),
		Executor: pod.ExecutorManagedBy(mgr),
		Recorder: new(record.FakeRecorder),
	}).SetupWithManager(ctx, mgr)
	Expect(err).ToNot(HaveOccurred())

	err = (&RunnerSetReconciler{
		Client:   mgr.GetClient(),
		Scheme:   mgr.GetScheme(),
		Recorder: new(record.FakeRecorder),
	}).SetupWithManager(ctx, mgr)
	Expect(err).ToNot(HaveOccurred())

	go func() {
		defer GinkgoRecover()
		err = mgr.Start(ctx)
		Expect(err).ToNot(HaveOccurred(), "failed to run manager")
	}()

	testns = &corev1.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			Name: "testing-" + util.RandomString(4),
		},
	}

	By("creating test namespace")
	Expect(crclient.Create(ctx, testns)).To(Succeed())
}, 60)

var _ = AfterSuite(func() {
	By("cleaning up test namespace")
	Expect(client.IgnoreNotFound(crclient.Delete(ctx, testns))).To(Succeed())
	timeout := 10 * time.Minute
	Eventually(func() bool {
		err := crclient.Get(ctx, client.ObjectKeyFromObject(testns), testns)
		return apierrors.IsNotFound(err)
	}, timeout).Should(BeTrue())
	cancel()
	By("tearing down the test environment")
	err := testenv.Stop()
	Expect(err).NotTo(HaveOccurred())
})
