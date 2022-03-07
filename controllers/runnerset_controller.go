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
	"sort"

	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	kerrors "k8s.io/apimachinery/pkg/util/errors"
	"k8s.io/client-go/tools/record"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"

	octorunv1alpha1 "octorun.github.io/octorun/api/v1alpha1"
	"octorun.github.io/octorun/util/patch"
)

const RunnerSetController = "runnerset.octorun.github.io/controller"

// RunnerSetReconciler reconciles a RunnerSet object
type RunnerSetReconciler struct {
	client.Client
	Scheme   *runtime.Scheme
	Recorder record.EventRecorder
}

// +kubebuilder:rbac:groups=octorun.github.io,resources=runnersets,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=octorun.github.io,resources=runnersets/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=octorun.github.io,resources=runnersets/finalizers,verbs=update
// +kubebuilder:rbac:groups=octorun.github.io,resources=runners,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=octorun.github.io,resources=runners/status,verbs=get
// +kubebuilder:rbac:groups=core,resources=events,verbs=get;list;watch;create;update;patch

// SetupWithManager sets up the controller with the Manager.
func (r *RunnerSetReconciler) SetupWithManager(ctx context.Context, mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&octorunv1alpha1.RunnerSet{}).
		Owns(&octorunv1alpha1.Runner{}).
		Complete(r)
}

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
func (r *RunnerSetReconciler) Reconcile(ctx context.Context, req ctrl.Request) (_ ctrl.Result, reterr error) {
	log := ctrl.LoggerFrom(ctx)
	runnerset := &octorunv1alpha1.RunnerSet{}
	if err := r.Get(ctx, req.NamespacedName, runnerset); err != nil {
		if apierrors.IsNotFound(err) {
			// Return early if requested runner set is not found.
			log.V(1).Info("RunnerSet resource not found or already deleted")
			return ctrl.Result{}, nil
		}

		return ctrl.Result{}, err
	}

	patcher, err := patch.NewPatcher(r.Client, runnerset)
	if err != nil {
		return ctrl.Result{}, err
	}

	defer func() {
		if err := patcher.Patch(ctx, runnerset, client.FieldOwner(RunnerSetController)); err != nil {
			reterr = err
		}
	}()

	runners, err := r.findRunners(ctx, runnerset)
	if err != nil {
		return ctrl.Result{}, err
	}

	if !runnerset.GetDeletionTimestamp().IsZero() {
		return ctrl.Result{}, nil
	}

	if err := r.syncRunners(ctx, runnerset, runners); err != nil {
		return ctrl.Result{}, err
	}

	return ctrl.Result{}, nil
}

// findRunners find Runners managed by given RunnerSet. It will adopt the orphan runner if have matching labels but does not have
// controllerRef. It will also update the several given RunnerSet status field according the Runner phase.
func (r *RunnerSetReconciler) findRunners(ctx context.Context, runnerset *octorunv1alpha1.RunnerSet) ([]*octorunv1alpha1.Runner, error) {
	log := ctrl.LoggerFrom(ctx)
	selectorMap, err := metav1.LabelSelectorAsMap(&runnerset.Spec.Selector)
	if err != nil {
		return nil, err
	}

	runnerList := &octorunv1alpha1.RunnerList{}
	if err := r.Client.List(ctx, runnerList, client.InNamespace(runnerset.Namespace), client.MatchingLabels(selectorMap)); err != nil {
		return nil, err
	}

	var idleRunners, activeRunners int32
	runners := make([]*octorunv1alpha1.Runner, 0, len(runnerList.Items))
	for i := range runnerList.Items {
		runner := &runnerList.Items[i]
		// Exclude the runner if not controlled by this runner set
		if metav1.GetControllerOf(runner) != nil && !metav1.IsControlledBy(runner, runnerset) {
			continue
		}

		if err := r.adoptRunner(ctx, runnerset, runner); err != nil {
			log.Error(err, "unable to adopt orphan Runner to the RunnerSet", "runner", runner)
			continue
		}

		switch runner.Status.Phase {
		case octorunv1alpha1.RunnerIdlePhase:
			idleRunners += 1
		case octorunv1alpha1.RunnerActivePhase:
			activeRunners += 1
		case octorunv1alpha1.RunnerCompletePhase:
			log.V(1).Info("deleting Runner that has Complete phase", "runner", runner)
			if err := r.Delete(ctx, runner); err != nil {
				log.Error(err, "unable to delete complete runner", "runner", runner)
			}

			continue
		}

		runners = append(runners, runner)
	}

	runnerset.Status.Runners = int32(len(runners))
	runnerset.Status.IdleRunners = idleRunners
	runnerset.Status.ActiveRunners = activeRunners
	return runners, nil
}

// adoptRunner adopt orphan runner who has not OwnerReference by sets
// given RunnerSet as controller OwnerReference to given Runner.
//
// It will immediately returns if the Runner already owned.
func (r *RunnerSetReconciler) adoptRunner(ctx context.Context, runnerset *octorunv1alpha1.RunnerSet, runner *octorunv1alpha1.Runner) error {
	log := ctrl.LoggerFrom(ctx)
	if metav1.GetControllerOf(runner) != nil {
		return nil
	}

	runnerPatch := client.MergeFrom(runner.DeepCopy())
	if err := ctrl.SetControllerReference(runnerset, runner, r.Scheme); err != nil {
		return err
	}

	log.Info("adopt orphan Runner", "runner", runner.Name)
	return r.Patch(ctx, runner, runnerPatch)
}

func (r *RunnerSetReconciler) syncRunners(ctx context.Context, runnerset *octorunv1alpha1.RunnerSet, runners []*octorunv1alpha1.Runner) error {
	log := ctrl.LoggerFrom(ctx)
	desiredRunners := int(*(runnerset.Spec.Runners))
	switch diff := len(runners) - desiredRunners; {
	case diff < 0:
		diff *= -1
		log.Info("too few Runner", "runners", len(runners), "desired", desiredRunners, "to be created", diff)
		var errs []error
		for i := 0; i < diff; i++ {
			runner := &octorunv1alpha1.Runner{
				ObjectMeta: metav1.ObjectMeta{
					GenerateName: runnerset.Name + "-",
					Namespace:    runnerset.Namespace,
					Labels:       runnerset.Spec.Template.Labels,
					Annotations:  runnerset.Spec.Template.Annotations,
				},
				Spec: runnerset.Spec.Template.Spec,
			}

			if _, err := ctrl.CreateOrUpdate(ctx, r.Client, runner, func() error {
				log.V(1).Info("creating new Runner", "runner", runner.Name)
				return ctrl.SetControllerReference(runnerset, runner, r.Scheme)
			}); err != nil {
				log.Error(err, "unable to create runner", "runner", runner.Name)
				errs = append(errs, err)
			}
		}

		return kerrors.NewAggregate(errs)
	case diff > 0:
		log.Info("too many Runner", "runners", len(runners), "desired", desiredRunners, "to be deleted", diff)
		var errs []error
		for _, runner := range prioritizedRunnersToDelete(runners, diff) {
			log.V(1).Info("deleting runner", "runner", runner.Name)
			if err := r.Delete(ctx, runner); client.IgnoreNotFound(err) != nil {
				log.Error(err, "unable to delete runner", "runner", runner)
				errs = append(errs, err)
			}

			log.Info("deleted runner", "runner", runner.Name)
		}

		return kerrors.NewAggregate(errs)
	}

	log.Info("synced RunnerSet runners", "runners", len(runners), "desired", desiredRunners)
	return nil
}

const (
	mustDelete    float64 = 100.0
	couldDelete   float64 = 50.0
	mustNotDelete float64 = 0.0
)

// runnersToDelete is sortable slice of Runner implement
// sort.Interface.
type runnersToDelete []*octorunv1alpha1.Runner

func (r runnersToDelete) Len() int      { return len(r) }
func (r runnersToDelete) Swap(i, j int) { r[i], r[j] = r[j], r[i] }
func (r runnersToDelete) Less(i, j int) bool {
	priority := func(runner *octorunv1alpha1.Runner) float64 {
		if !runner.GetDeletionTimestamp().IsZero() {
			return mustDelete
		}

		if runner.Status.Phase == octorunv1alpha1.RunnerActivePhase {
			return mustNotDelete
		}

		return couldDelete
	}

	return priority(r[j]) < priority(r[i])
}

func prioritizedRunnersToDelete(runners []*octorunv1alpha1.Runner, diff int) []*octorunv1alpha1.Runner {
	if diff >= len(runners) {
		return runners
	} else if diff <= 0 {
		return []*octorunv1alpha1.Runner{}
	}

	sort.Sort(runnersToDelete(runners))
	return runners[:diff]
}
