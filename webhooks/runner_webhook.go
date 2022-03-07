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
	"fmt"
	"reflect"

	"github.com/pkg/errors"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/util/validation/field"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/webhook"

	octorunv1alpha1 "octorun.github.io/octorun/api/v1alpha1"
	// +kubebuilder:scaffold:imports
)

type RunnerWebhook struct {
	Client client.Reader
}

// +kubebuilder:webhook:path=/mutate-octorun-github-io-v1alpha1-runner,mutating=true,failurePolicy=fail,sideEffects=None,groups=octorun.github.io,resources=runners,verbs=create;update,versions=v1alpha1,name=mrunner.octorun.github.io,admissionReviewVersions=v1
// +kubebuilder:webhook:path=/validate-octorun-github-io-v1alpha1-runner,mutating=false,failurePolicy=fail,sideEffects=None,groups=octorun.github.io,resources=runners,verbs=create;update,versions=v1alpha1,name=vrunner.octorun.github.io,admissionReviewVersions=v1

var _ webhook.CustomDefaulter = &RunnerWebhook{}
var _ webhook.CustomValidator = &RunnerWebhook{}

func (w *RunnerWebhook) SetupWithManager(ctx context.Context, mgr ctrl.Manager) error {
	return ctrl.NewWebhookManagedBy(mgr).
		For(&octorunv1alpha1.Runner{}).
		WithDefaulter(w).
		WithValidator(w).
		Complete()
}

// Default implements webhook.CustomDefaulter so a webhook will be registered for the type
func (w *RunnerWebhook) Default(ctx context.Context, obj runtime.Object) error {
	runner, ok := obj.(*octorunv1alpha1.Runner)
	if !ok {
		return apierrors.NewBadRequest(fmt.Sprintf("expected a Runner but got a %T", obj))
	}

	if runner.Labels == nil {
		runner.Labels = make(map[string]string)
	}

	if _, ok := runner.Labels[octorunv1alpha1.LabelRunnerName]; !ok && runner.Name != "" {
		runner.Labels[octorunv1alpha1.LabelRunnerName] = runner.GetName()
	}

	if runner.Spec.Group == "" {
		runner.Spec.Group = "Default"
	}

	if runner.Spec.Workdir == "" {
		runner.Spec.Workdir = "_work"
	}

	return nil
}

// ValidateCreate implements webhook.CustomValidator so a webhook will be registered for the type
func (w *RunnerWebhook) ValidateCreate(ctx context.Context, obj runtime.Object) error {
	return nil
}

// ValidateUpdate implements webhook.CustomValidator so a webhook will be registered for the type
func (w *RunnerWebhook) ValidateUpdate(ctx context.Context, oldObj, newObj runtime.Object) error {
	var allErrs field.ErrorList
	oldRunner, err := runtime.DefaultUnstructuredConverter.ToUnstructured(oldObj)
	if err != nil {
		return apierrors.NewInternalError(errors.Wrap(err, "failed to convert new Runner to unstructured object"))
	}

	newRunner, err := runtime.DefaultUnstructuredConverter.ToUnstructured(newObj)
	if err != nil {
		return apierrors.NewInternalError(errors.Wrap(err, "failed to convert old Runner to unstructured object"))
	}

	newRunnerSpec := newRunner["spec"].(map[string]interface{})
	oldRunnerSpec := oldRunner["spec"].(map[string]interface{})

	// exclude id and os populated by the controller from validation.
	delete(oldRunnerSpec, "id")
	delete(newRunnerSpec, "id")
	delete(oldRunnerSpec, "os")
	delete(newRunnerSpec, "os")

	if !reflect.DeepEqual(oldRunnerSpec, newRunnerSpec) {
		allErrs = append(allErrs, field.Forbidden(field.NewPath("spec"), "spec is immutable"))
	}

	if len(allErrs) == 0 {
		return nil
	}

	return apierrors.NewInvalid(newObj.GetObjectKind().GroupVersionKind().GroupKind(), newObj.(metav1.Object).GetName(), allErrs)
}

// ValidateDelete implements webhook.CustomValidator so a webhook will be registered for the type
func (w *RunnerWebhook) ValidateDelete(ctx context.Context, obj runtime.Object) error {
	return nil
}
