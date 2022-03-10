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

package main

import (
	"flag"
	"os"

	// Import all Kubernetes client auth plugins (e.g. Azure, GCP, OIDC, etc.)
	// to ensure that exec-entrypoint and run can make use of them.
	_ "k8s.io/client-go/plugin/pkg/client/auth"

	"k8s.io/apimachinery/pkg/runtime"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	"k8s.io/klog/v2"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/healthz"
	"sigs.k8s.io/controller-runtime/pkg/log/zap"
	crmetrics "sigs.k8s.io/controller-runtime/pkg/metrics"

	octorunv1alpha1 "octorun.github.io/octorun/api/v1alpha1"
	"octorun.github.io/octorun/controllers"
	"octorun.github.io/octorun/hooks"
	"octorun.github.io/octorun/metrics"
	"octorun.github.io/octorun/pkg/github"
	"octorun.github.io/octorun/pkg/statemetrics"
	"octorun.github.io/octorun/util/pod"
	"octorun.github.io/octorun/webhooks"
	// +kubebuilder:scaffold:imports
)

var (
	scheme   = runtime.NewScheme()
	setupLog = ctrl.Log.WithName("setup")
)

func init() {
	utilruntime.Must(clientgoscheme.AddToScheme(scheme))

	utilruntime.Must(octorunv1alpha1.AddToScheme(scheme))
	// +kubebuilder:scaffold:scheme
}

type options struct {
	probeAddr            string
	metricsAddr          string
	enableLeaderElection bool

	Logger zap.Options
	Github github.Options
}

func (o *options) bindFlags(fs *flag.FlagSet) {
	fs.StringVar(&o.probeAddr, "health-probe-bind-address", ":8081", "The address the probe endpoint binds to.")
	fs.StringVar(&o.metricsAddr, "metrics-bind-address", ":8080", "The address the metric endpoint binds to.")
	fs.BoolVar(&o.enableLeaderElection, "leader-elect", false,
		"Enable leader election for controller manager. "+
			"Enabling this will ensure there is only one active controller manager.")

	o.Logger.Development = true
	o.Logger.BindFlags(fs)
	o.Github.BindFlags(fs)
}

func main() {
	var opts options
	opts.bindFlags(flag.CommandLine)
	flag.Parse()

	ctrl.SetLogger(zap.New(zap.UseFlagOptions(&opts.Logger)))
	klog.SetLogger(ctrl.Log)
	ctx := ctrl.SetupSignalHandler()
	mgr, err := ctrl.NewManager(ctrl.GetConfigOrDie(), ctrl.Options{
		Scheme:                 scheme,
		HealthProbeBindAddress: opts.probeAddr,
		MetricsBindAddress:     opts.metricsAddr,
		Port:                   9443,
		LeaderElection:         opts.enableLeaderElection,
		LeaderElectionID:       "octorun.github.io",
	})
	if err != nil {
		setupLog.Error(err, "unable to start manager")
		os.Exit(1)
	}

	if err = (&controllers.RunnerReconciler{
		Client:   mgr.GetClient(),
		Scheme:   mgr.GetScheme(),
		Github:   opts.Github.GetClient(),
		Executor: pod.ExecutorManagedBy(mgr),
		Recorder: mgr.GetEventRecorderFor(controllers.RunnerController),
	}).SetupWithManager(ctx, mgr); err != nil {
		setupLog.Error(err, "unable to create controller", "controller", "Runner")
		os.Exit(1)
	}
	if err = (&controllers.RunnerSetReconciler{
		Client:   mgr.GetClient(),
		Scheme:   mgr.GetScheme(),
		Recorder: mgr.GetEventRecorderFor(controllers.RunnerSetController),
	}).SetupWithManager(ctx, mgr); err != nil {
		setupLog.Error(err, "unable to create controller", "controller", "RunnerSet")
		os.Exit(1)
	}

	if err = (&webhooks.RunnerWebhook{
		Client: mgr.GetAPIReader(),
	}).SetupWithManager(ctx, mgr); err != nil {
		setupLog.Error(err, "unable to create webhook", "webhook", "Runner")
		os.Exit(1)
	}
	if err = (&webhooks.RunnerSetWebhook{
		Client: mgr.GetAPIReader(),
	}).SetupWithManager(ctx, mgr); err != nil {
		setupLog.Error(err, "unable to create webhook", "webhook", "RunnerSet")
		os.Exit(1)
	}
	// +kubebuilder:scaffold:builder

	if err := (&hooks.GithubHook{
		Client: mgr.GetClient(),
	}).SetupWithManager(ctx, mgr, opts.Github.GetWebhookServer()); err != nil {
		setupLog.Error(err, "unable to set up github webhook")
		os.Exit(1)
	}

	if err := mgr.AddHealthzCheck("healthz", healthz.Ping); err != nil {
		setupLog.Error(err, "unable to set up health check")
		os.Exit(1)
	}
	if err := mgr.AddReadyzCheck("readyz", healthz.Ping); err != nil {
		setupLog.Error(err, "unable to set up ready check")
		os.Exit(1)
	}

	if err := crmetrics.Registry.Register(statemetrics.NewCollector(mgr,
		&metrics.RunnerProvider{},
		&metrics.RunnerSetProvider{},
	)); err != nil {
		setupLog.Error(err, "unable to register statemetrics collector")
		os.Exit(1)
	}

	setupLog.Info("starting manager")
	if err := mgr.Start(ctx); err != nil {
		setupLog.Error(err, "problem running manager")
		os.Exit(1)
	}
}
