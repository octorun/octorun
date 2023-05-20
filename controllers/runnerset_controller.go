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

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/serializer"
	kerrors "k8s.io/apimachinery/pkg/util/errors"
	"k8s.io/client-go/tools/record"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"

	octorunv1 "octorun.github.io/octorun/api/v1alpha2"
	"octorun.github.io/octorun/pkg/revision"
	"octorun.github.io/octorun/util/patch"
	"octorun.github.io/octorun/util/sortable"
)

const RunnerSetController = "runnerset.octorun.github.io/controller"

// RunnerSetReconciler reconciles a RunnerSet object
type RunnerSetReconciler struct {
	client.Client
	Scheme     *runtime.Scheme
	Recorder   record.EventRecorder
	Revisioner revision.Revisioner
}

// +kubebuilder:rbac:groups=octorun.github.io,resources=runnersets,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=octorun.github.io,resources=runnersets/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=octorun.github.io,resources=runnersets/finalizers,verbs=update
// +kubebuilder:rbac:groups=octorun.github.io,resources=runners,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=octorun.github.io,resources=runners/status,verbs=get
// +kubebuilder:rbac:groups=core,resources=events,verbs=get;list;watch;create;update;patch
// +kubebuilder:rbac:groups=apps,resources=controllerrevisions,verbs=get;list;watch;create;update;patch

// SetupWithManager sets up the controller with the Manager.
func (r *RunnerSetReconciler) SetupWithManager(ctx context.Context, mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&octorunv1.RunnerSet{}).
		Owns(&octorunv1.Runner{}).
		Complete(r)
}

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
func (r *RunnerSetReconciler) Reconcile(ctx context.Context, req ctrl.Request) (_ ctrl.Result, reterr error) {
	log := ctrl.LoggerFrom(ctx)
	runnerset := &octorunv1.RunnerSet{}
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

	if !runnerset.GetDeletionTimestamp().IsZero() {
		return ctrl.Result{}, nil
	}

	selector, err := metav1.LabelSelectorAsSelector(&runnerset.Spec.Selector)
	if err != nil {
		return ctrl.Result{}, nil
	}

	rev := &appsv1.ControllerRevision{}
	if err := revision.MakeHistory(ctx, r.Client, r.Revisioner, runnerset, rev); err != nil {
		return ctrl.Result{}, err
	}

	runners, err := r.findRunners(ctx, runnerset, rev)
	if err != nil {
		return ctrl.Result{}, err
	}

	if err := r.syncRunners(ctx, runnerset, runners, rev); err != nil {
		return ctrl.Result{}, err
	}

	var updatedRunners int32
	for _, runner := range runners {
		if runner.Labels[r.Revisioner.HashLabelKey()] == rev.Name {
			updatedRunners++
		}
	}

	if runnerset.Status.Runners == updatedRunners {
		runnerset.Status.CurrentRevision = rev.Name
	}

	runnerObj := make([]client.Object, 0, len(runners))
	for _, runner := range runners {
		runnerObj = append(runnerObj, runner)
	}

	runnerset.Status.Selector = selector.String()
	return ctrl.Result{}, revision.TruncateHistory(ctx, r.Client, r.Revisioner, runnerset, runnerObj)
}

// findRunners find Runners managed by given RunnerSet. It will adopt the orphan runner if have matching labels but does not have
// controllerRef. It will also update the several given RunnerSet status field according the Runner phase.
func (r *RunnerSetReconciler) findRunners(ctx context.Context, runnerset *octorunv1.RunnerSet, rev *appsv1.ControllerRevision) ([]*octorunv1.Runner, error) {
	log := ctrl.LoggerFrom(ctx)
	selectorMap, err := metav1.LabelSelectorAsMap(&runnerset.Spec.Selector)
	if err != nil {
		return nil, err
	}

	runnerList := &octorunv1.RunnerList{}
	if err := r.Client.List(ctx, runnerList, client.InNamespace(runnerset.Namespace), client.MatchingLabels(selectorMap)); err != nil {
		return nil, err
	}

	var idleRunners, activeRunners int32
	runners := make([]*octorunv1.Runner, 0, len(runnerList.Items))
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
		case octorunv1.RunnerIdlePhase:
			if runnerset.Spec.UpdateStrategy.Type == octorunv1.RollingUpdateRunnerSetStrategyType &&
				runner.Labels[r.Revisioner.HashLabelKey()] != rev.Name {
				// Do rolling update by deleting runner with missmatch revision label.
				log.V(1).Info("deleting Runner for Rolling Update", "runner", runner)
				if err := r.Delete(ctx, runner); client.IgnoreNotFound(err) != nil {
					log.Error(err, "unable to delete runner", "runner", runner)
				}

				continue
			}

			idleRunners += 1
		case octorunv1.RunnerActivePhase:
			activeRunners += 1
		case octorunv1.RunnerCompletePhase:
			log.V(1).Info("deleting Runner that has Complete phase", "runner", runner)
			if err := r.Delete(ctx, runner); client.IgnoreNotFound(err) != nil {
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
func (r *RunnerSetReconciler) adoptRunner(ctx context.Context, runnerset *octorunv1.RunnerSet, runner *octorunv1.Runner) error {
	log := ctrl.LoggerFrom(ctx)
	if metav1.GetControllerOf(runner) != nil {
		return nil
	}

	runnerPatch := client.MergeFrom(runner.DeepCopy())
	if err := ctrl.SetControllerReference(runnerset, runner, r.Scheme); err != nil {
		return err
	}

	log.Info("adopt orphan Runner", "runner", runner.Name)
	r.Recorder.Eventf(runnerset, corev1.EventTypeNormal, octorunv1.RunnerAdoptedReason, "Successful adopt orphan Runner %s", runner.Name)
	return r.Patch(ctx, runner, runnerPatch)
}

func (r *RunnerSetReconciler) syncRunners(ctx context.Context, runnerset *octorunv1.RunnerSet, runners []*octorunv1.Runner, rev *appsv1.ControllerRevision) error {
	log := ctrl.LoggerFrom(ctx)
	prioritizedRunnersToDelete := func(runners []*octorunv1.Runner, diff int) []*octorunv1.Runner {
		if diff >= len(runners) {
			return runners
		} else if diff <= 0 {
			return []*octorunv1.Runner{}
		}

		sort.Sort(sortable.RunnersToDelete(runners))
		return runners[:diff]
	}

	desiredRunners := int(*(runnerset.Spec.Runners))
	switch diff := len(runners) - desiredRunners; {
	case diff < 0:
		diff *= -1
		log.Info("too few Runner", "runners", len(runners), "desired", desiredRunners, "to be created", diff)
		var errs []error
		for i := 0; i < diff; i++ {
			runnerAnnotation := make(labels.Set)
			for k, v := range runnerset.Spec.Template.Annotations {
				runnerAnnotation[k] = v
			}

			runnerLabels := make(labels.Set)
			for k, v := range runnerset.Spec.Template.Labels {
				runnerLabels[k] = v
			}

			runnerLabels[r.Revisioner.HashLabelKey()] = rev.Name
			runner := &octorunv1.Runner{
				ObjectMeta: metav1.ObjectMeta{
					GenerateName: runnerset.Name + "-",
					Namespace:    runnerset.Namespace,
					Annotations:  runnerAnnotation,
					Labels:       runnerLabels,
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

			log.Info("created runner", "runner", runner.Name)
			r.Recorder.Eventf(runnerset, corev1.EventTypeNormal, octorunv1.RunnerCreatedReason, "Successful create Runner %s", runner.Name)
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
			r.Recorder.Eventf(runnerset, corev1.EventTypeNormal, octorunv1.RunnerDeletedReason, "Successful delete Runner %s", runner.Name)
		}

		return kerrors.NewAggregate(errs)
	}

	log.Info("synced RunnerSet runners", "runners", len(runners), "desired", desiredRunners)
	return nil
}

type RunnerSetRevisioner struct{}

func (r *RunnerSetRevisioner) HashLabelKey() string { return octorunv1.LabelControllerRevisionHash }

func (r *RunnerSetRevisioner) NextRevision(ctx context.Context, c client.Client, obj client.Object, rev int64) (*appsv1.ControllerRevision, error) {
	runnerset := obj.(*octorunv1.RunnerSet)
	templateLabels := runnerset.Spec.Template.GetLabels()
	revisionLabels := make(map[string]string)
	for k, v := range templateLabels {
		revisionLabels[k] = v
	}

	revisionAnnotations := make(map[string]string)
	for k, v := range runnerset.Annotations {
		revisionAnnotations[k] = v
	}

	rawData, err := revision.ObjectPatch(runnerset, serializer.NewCodecFactory(c.Scheme()).LegacyCodec(octorunv1.GroupVersion))
	if err != nil {
		return nil, err
	}

	var controllerRef metav1.OwnerReference
	gvk := runnerset.GroupVersionKind()
	metav1.NewControllerRef(runnerset, octorunv1.GroupVersion.WithKind(gvk.Kind)).DeepCopyInto(&controllerRef)
	cr := &appsv1.ControllerRevision{
		ObjectMeta: metav1.ObjectMeta{
			Labels:          revisionLabels,
			OwnerReferences: []metav1.OwnerReference{controllerRef},
		},
		Data:     runtime.RawExtension{Raw: rawData},
		Revision: rev,
	}

	hash := revision.Hash(cr, runnerset.Status.CollisionCount)
	cr.Name = revision.Name(runnerset.GetName(), hash)
	cr.Namespace = runnerset.GetNamespace()
	cr.Annotations = revisionAnnotations
	cr.Labels[r.HashLabelKey()] = hash
	return cr, nil
}

func (r *RunnerSetRevisioner) ListRevision(ctx context.Context, c client.Client, obj client.Object) ([]*appsv1.ControllerRevision, error) {
	runnerset := obj.(*octorunv1.RunnerSet)
	selector, err := metav1.LabelSelectorAsSelector(&runnerset.Spec.Selector)
	if err != nil {
		return nil, err
	}

	revisionList := &appsv1.ControllerRevisionList{}
	matchLabelsSelector := client.MatchingLabelsSelector{Selector: selector}
	if err := c.List(ctx, revisionList, client.InNamespace(runnerset.Namespace), matchLabelsSelector); err != nil {
		return nil, err
	}

	var revisions []*appsv1.ControllerRevision
	for _, rev := range revisionList.Items {
		revision := rev.DeepCopy()
		ref := metav1.GetControllerOfNoCopy(revision)
		if ref.UID == runnerset.UID {
			revisions = append(revisions, revision)
		}
	}

	return revisions, nil
}
