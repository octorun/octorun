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
	"strings"
	"time"

	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/tools/record"
	"k8s.io/utils/pointer"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"

	octorunv1alpha1 "octorun.github.io/octorun/api/v1alpha1"
	"octorun.github.io/octorun/pkg/github"
	gherrors "octorun.github.io/octorun/pkg/github/errors"
	"octorun.github.io/octorun/util"
	"octorun.github.io/octorun/util/annotations"
	"octorun.github.io/octorun/util/patch"
	"octorun.github.io/octorun/util/pod"
	"octorun.github.io/octorun/util/remoteexec"
)

const RunnerController = "runner.octorun.github.io/controller"

// RunnerReconciler reconciles a Runner object
type RunnerReconciler struct {
	client.Client
	Github   github.Client
	Scheme   *runtime.Scheme
	Executor remoteexec.RemoteExecutor
	Recorder record.EventRecorder
}

// +kubebuilder:rbac:groups=octorun.github.io,resources=runners,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=octorun.github.io,resources=runners/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=octorun.github.io,resources=runners/finalizers,verbs=update
// +kubebuilder:rbac:groups=core,resources=events,verbs=get;list;watch;create;update;patch
// +kubebuilder:rbac:groups=core,resources=secrets,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=core,resources=pods,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=core,resources=pods/status,verbs=get
// +kubebuilder:rbac:groups=core,resources=pods/exec,verbs=create

// SetupWithManager sets up the controller with the Manager.
func (r *RunnerReconciler) SetupWithManager(ctx context.Context, mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&octorunv1alpha1.Runner{}).
		Owns(&corev1.Pod{}).
		Complete(r)
}

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
func (r *RunnerReconciler) Reconcile(ctx context.Context, req ctrl.Request) (_ ctrl.Result, reterr error) {
	log := ctrl.LoggerFrom(ctx)
	runner := &octorunv1alpha1.Runner{}
	if err := r.Get(ctx, req.NamespacedName, runner); err != nil {
		if apierrors.IsNotFound(err) {
			// Return early if requested runner is not found.
			log.V(1).Info("Runner resource not found or already deleted")
			return ctrl.Result{}, nil
		}

		return ctrl.Result{}, err
	}

	patcher, err := patch.NewPatcher(r.Client, runner)
	if err != nil {
		return ctrl.Result{}, err
	}

	defer func() {
		if err := patcher.Patch(ctx, runner, client.FieldOwner(RunnerController)); err != nil {
			reterr = err
		}
	}()

	runnerPod := podForRunner(runner)
	runnerSecret := secretForRunner(runner)
	if !runner.GetDeletionTimestamp().IsZero() {
		// Handle deletion if we have non zero deletion timestamp
		// by cleaning up owned resources.
		if runner.Status.Phase == octorunv1alpha1.RunnerActivePhase {
			// If the runner is in the active phase wait until finish its job.
			log.Info("Runner is in active phase. wait until runner job to be completed before deleting")

			// Check runner status from Github.
			runnerid := pointer.Int64Deref(runner.Spec.ID, -1)
			if runnerid == -1 {
				// This is unexpected condition when runner has Active phase
				// but don't have an ID.
				runner.Status.Phase = octorunv1alpha1.RunnerCompletePhase
				return ctrl.Result{Requeue: true}, nil
			}

			ghrunner, err := r.Github.GetRunner(ctx, runner.Spec.URL, runnerid)
			if err != nil && !(gherrors.IsForbidden(err) || gherrors.IsNotFound(err)) {
				log.Error(err, "unable to retrieve Runner information from Github")
				return ctrl.Result{}, err
			}

			if !ghrunner.GetBusy() {
				runner.Status.Phase = octorunv1alpha1.RunnerCompletePhase
				return ctrl.Result{Requeue: true}, nil
			}

			return ctrl.Result{RequeueAfter: 60 * time.Second}, nil
		}

		log.Info("deleting Runner resources")
		if _, err := ctrl.CreateOrUpdate(ctx, r.Client, runnerSecret, func() error {
			if annotations.IsTokenExpired(runnerSecret) {
				log.V(1).Info("registration token has expired. Refresh before deleting", "secret", runnerSecret.Name)
				rt, err := r.Github.CreateRunnerToken(ctx, runner.Spec.URL)
				if err != nil && !(gherrors.IsForbidden(err) || gherrors.IsNotFound(err)) {
					return err
				}

				annotations.AnnotateTokenExpires(runnerSecret, rt.GetExpiresAt().UTC().Format(time.RFC3339))
				if runnerSecret.Data == nil {
					runnerSecret.Data = make(map[string][]byte)
				}

				runnerSecret.Data["token"] = []byte(rt.GetToken())
			}

			return ctrl.SetControllerReference(runner, runnerSecret, r.Scheme)
		}); err != nil {
			return ctrl.Result{}, err
		}

		log.V(1).Info("deleting Runner pod", "pod", runnerPod.Name)
		if err := r.Delete(ctx, runnerPod); client.IgnoreNotFound(err) != nil {
			return ctrl.Result{}, err
		}

		log.V(1).Info("deleting Runner registration token secret", "secret", runnerSecret.Name)
		if err := r.Delete(ctx, runnerSecret); client.IgnoreNotFound(err) != nil {
			return ctrl.Result{}, err
		}

		controllerutil.RemoveFinalizer(runner, RunnerController)
		log.Info("deleted Runner resources")
		return ctrl.Result{}, nil
	}

	log.Info("reconciling Runner resources")
	controllerutil.AddFinalizer(runner, RunnerController)

	// Create a runner secret if it doesn't exist or update it if the token has expired.
	if op, err := ctrl.CreateOrUpdate(ctx, r.Client, runnerSecret, func() error {
		log.V(1).Info("reconciling Runner registration token secret", "secret", runnerSecret.Name)
		if runnerSecret.CreationTimestamp.IsZero() || annotations.IsTokenExpired(runnerSecret) {
			log.V(1).Info("Runner registration token secret does not exist or already expired", "secret", runnerSecret.Name)
			rt, err := r.Github.CreateRunnerToken(ctx, runner.Spec.URL)
			if err != nil {
				return err
			}

			annotations.AnnotateTokenExpires(runnerSecret, rt.GetExpiresAt().UTC().Format(time.RFC3339))
			runnerSecret.Data["token"] = []byte(rt.GetToken())
		}

		return ctrl.SetControllerReference(runner, runnerSecret, r.Scheme)
	}); err != nil {
		if gherrors.IsForbidden(err) || gherrors.IsNotFound(err) {
			// If we got forbidden or not found error from Github here just record an Warning event and return error nil (terminal failure).
			// returning Requeue false with nil error here is to prevent the controller keep reconciling and trying to create the registration token.
			// Users must have to recreate the runner with the proper spec.
			//
			// It's ok to just record an Warning event here since users still can gather information on why
			// the runner keeps in Pending status eg: using `kubectl describe` command
			//
			// TODO(prksu): consider introducing terminal failure based on Runner status/condition?
			log.Error(err, "Unable to create Runner registration token")
			r.Recorder.Eventf(runner, corev1.EventTypeWarning, octorunv1alpha1.RunnerSecretFailedReason, "Unable to create Runner registration token: %v", err)
			return ctrl.Result{Requeue: false}, nil
		}

		log.Error(err, "failed reconciling Runner registration token secret", "secret", runnerSecret.Name)
		return ctrl.Result{}, err
	} else {
		log.V(1).Info("reconciled Runner registration token secret", "secret", runnerSecret.Name, "op", op)
	}

	// Create a runner pod if it doesn't exist. actually, it's never updating the runner pod and we won't.
	if op, err := ctrl.CreateOrUpdate(ctx, r.Client, runnerPod, func() error {
		log.V(1).Info("reconciling Runner pod", "pod", runnerPod.Name)
		return ctrl.SetControllerReference(runner, runnerPod, r.Scheme)
	}); err != nil {
		log.Error(err, "failed reconciling Runner pod", "pod", runnerPod.Name)
		return ctrl.Result{}, err
	} else {
		log.V(1).Info("reconciled Runner pod", "pod", runnerPod.Name, "op", op)
	}

	// All resources already reconciled. Set runner phase to "Pending" for now it will overwritten
	// based on runner pod phase, conditions and runner status from Github.
	runner.Status.Phase = octorunv1alpha1.RunnerPendingPhase
	return r.reconcileStatus(ctx, runner, runnerPod)
}

func (r *RunnerReconciler) reconcileStatus(ctx context.Context, runner *octorunv1alpha1.Runner, runnerPod *corev1.Pod) (ctrl.Result, error) {
	log := ctrl.LoggerFrom(ctx)
	switch runnerPod.Status.Phase {
	case corev1.PodPending:
		// Returns early if Runner Pod is in Pending phase. It will automatically
		// reconciling again once Runner Pod phase has changed
		r.Recorder.Event(runner, corev1.EventTypeNormal, octorunv1alpha1.RunnerPodPendingReason, "Waiting for Pod to be Running.")
		log.V(1).Info("Runner pod is Pending. Waiting for Runner pod to be Running", "pod", runnerPod.Name)
		return ctrl.Result{}, nil
	case corev1.PodRunning:
		// Once pod is in Running phase check if this pod condition is ready.
		if !pod.PodConditionIsReady(runnerPod) {
			// Returns early if Runner Pod is not Ready. It will automatically
			// reconciling again once Runner Pod condition has changed
			log.V(1).Info("Runner pod is not Ready. Waiting for Runner pod Readiness", "pod", runnerPod.Name)
			return ctrl.Result{}, nil
		}

		log.V(1).Info("find Runner id from Pod", "pod", runnerPod.Name)
		runnerid, err := util.FindRunnerIDFromPod(runnerPod, r.Executor)
		if err != nil {
			log.Error(err, "unable to retrieve Runner id from Pod", "pod", runnerPod.Name)
			return ctrl.Result{}, err
		}

		runner.Spec.ID = pointer.Int64(runnerid)
		ghrunner, err := r.Github.GetRunner(ctx, runner.Spec.URL, runnerid)
		if err != nil {
			log.Error(err, "unable to retrieve Runner information from Github")
			return ctrl.Result{}, err
		}

		runner.Spec.OS = ghrunner.GetOS()
		if ghrunner.GetStatus() == "offline" {
			// Sometimes github runner doesn't go online instantly after registered.
			// Since no one can retrigger the reconcilication we need to requeue here.
			log.V(1).Info("Runner is offline", "runner", ghrunner.GetName())
			meta.SetStatusCondition(&runner.Status.Conditions, metav1.Condition{
				Type:    octorunv1alpha1.RunnerConditionOnline,
				Status:  metav1.ConditionFalse,
				Reason:  octorunv1alpha1.RunnerOfflineReason,
				Message: "Github Runner has Offline status",
			})
			return ctrl.Result{RequeueAfter: 5 * time.Second}, nil
		}

		log.V(1).Info("Runner is online. wait for a job!", "runner", ghrunner.GetName())
		runner.Status.Phase = octorunv1alpha1.RunnerIdlePhase
		r.Recorder.Event(runner, corev1.EventTypeNormal, octorunv1alpha1.RunnerOnlineReason, "Runner wait for a job.")
		meta.SetStatusCondition(&runner.Status.Conditions, metav1.Condition{
			Type:    octorunv1alpha1.RunnerConditionOnline,
			Status:  metav1.ConditionTrue,
			Reason:  octorunv1alpha1.RunnerOnlineReason,
			Message: "Github Runner has Online status",
		})

		if ghrunner.GetBusy() {
			// Mark this runner phase into Active once the runner is busy.
			log.V(1).Info("Runner is busy", "runner", ghrunner.GetName())
			r.Recorder.Event(runner, corev1.EventTypeNormal, octorunv1alpha1.RunnerBusyReason, "Runner got a job.")
			runner.Status.Phase = octorunv1alpha1.RunnerActivePhase
		}

		return ctrl.Result{}, nil
	case corev1.PodSucceeded:
		r.Recorder.Event(runner, corev1.EventTypeNormal, octorunv1alpha1.RunnerPodSucceededReason, "Runner complete his job.")
		runner.Status.Phase = octorunv1alpha1.RunnerCompletePhase
		meta.SetStatusCondition(&runner.Status.Conditions, metav1.Condition{
			Type:    octorunv1alpha1.RunnerConditionOnline,
			Status:  metav1.ConditionFalse,
			Reason:  octorunv1alpha1.RunnerPodSucceededReason,
			Message: "Runner Pod has Succeeded phase",
		})
		return ctrl.Result{}, nil
	default:
		return ctrl.Result{}, nil
	}
}

func secretForRunner(runner *octorunv1alpha1.Runner) *corev1.Secret {
	return &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      runner.Name + "-registration-token",
			Namespace: runner.Namespace,
		},
		Data:      make(map[string][]byte),
		Immutable: pointer.BoolPtr(false),
	}
}

func podForRunner(runner *octorunv1alpha1.Runner) *corev1.Pod {
	runnerLabels := make([]string, 0)
	for k, v := range runner.ObjectMeta.Labels {
		if !strings.HasPrefix(k, octorunv1alpha1.LabelPrefix) {
			continue
		}

		runnerLabel := strings.TrimPrefix(k, octorunv1alpha1.LabelPrefix) + "=" + v
		runnerLabels = append(runnerLabels, runnerLabel)
	}

	return &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:      runner.Name,
			Namespace: runner.Namespace,
			Labels:    runner.Labels,
		},
		Spec: corev1.PodSpec{
			RestartPolicy:    corev1.RestartPolicyOnFailure,
			ImagePullSecrets: runner.Spec.Image.PullSecrets,
			Containers: []corev1.Container{
				{
					Name:            "runner",
					Image:           runner.Spec.Image.Name,
					ImagePullPolicy: runner.Spec.Image.PullPolicy,
					Env: []corev1.EnvVar{
						{
							Name:  "URL",
							Value: runner.Spec.URL,
						},
						{
							Name: "RUNNER_NAME",
							ValueFrom: &corev1.EnvVarSource{
								FieldRef: &corev1.ObjectFieldSelector{
									FieldPath: "metadata.name",
								},
							},
						},
						{
							Name:  "RUNNER_LABELS",
							Value: strings.Join(runnerLabels, ","),
						},
						{
							Name:  "RUNNER_GROUP",
							Value: runner.Spec.Group,
						},
						{
							Name:  "RUNNER_WORKDIR",
							Value: runner.Spec.Workdir,
						},
						{
							Name:  "RUNNER_TOKEN_FILE",
							Value: "/var/run/secrets/runner.octorun.github.io/registration-token/token",
						},
					},
					StartupProbe: &corev1.Probe{
						ProbeHandler: corev1.ProbeHandler{
							Exec: &corev1.ExecAction{
								Command: []string{"cat", ".runner"},
							},
						},
						InitialDelaySeconds: 10,
						PeriodSeconds:       3,
					},
					VolumeMounts: append([]corev1.VolumeMount{
						{
							Name:      "registration-token",
							ReadOnly:  true,
							MountPath: "/var/run/secrets/runner.octorun.github.io/registration-token",
						},
					}, runner.Spec.VolumeMounts...),
					Resources:       runner.Spec.Resources,
					SecurityContext: runner.Spec.SecurityContext,
				},
			},
			Volumes: append([]corev1.Volume{
				{
					Name: "registration-token",
					VolumeSource: corev1.VolumeSource{
						Secret: &corev1.SecretVolumeSource{
							SecretName: runner.Name + "-registration-token",
						},
					},
				},
			}, runner.Spec.Volumes...),
			ServiceAccountName: runner.Spec.ServiceAccountName,
			SecurityContext: &corev1.PodSecurityContext{
				RunAsUser:    pointer.Int64(1000),
				RunAsGroup:   pointer.Int64(1000),
				RunAsNonRoot: pointer.Bool(true),
			},
			NodeSelector:     runner.Spec.Placement.NodeSelector,
			Affinity:         runner.Spec.Placement.Affinity,
			Tolerations:      runner.Spec.Placement.Tolerations,
			RuntimeClassName: runner.Spec.RuntimeClassName,
		},
	}
}
