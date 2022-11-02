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

	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/util/validation/field"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/webhook"

	octorunv1 "octorun.github.io/octorun/api/v1alpha2"
	// +kubebuilder:scaffold:imports
)

type RunnerSetWebhook struct {
	Client client.Reader
}

// +kubebuilder:webhook:path=/mutate-octorun-github-io-v1alpha2-runnerset,mutating=true,failurePolicy=fail,sideEffects=None,groups=octorun.github.io,resources=runnersets,verbs=create;update,versions=v1alpha2,name=mrunnerset.octorun.github.io,admissionReviewVersions=v1
// +kubebuilder:webhook:path=/validate-octorun-github-io-v1alpha2-runnerset,mutating=false,failurePolicy=fail,sideEffects=None,groups=octorun.github.io,resources=runnersets,verbs=create;update,versions=v1alpha2,name=vrunnerset.octorun.github.io,admissionReviewVersions=v1

var _ webhook.CustomDefaulter = &RunnerSetWebhook{}
var _ webhook.CustomValidator = &RunnerSetWebhook{}

func (w *RunnerSetWebhook) SetupWithManager(ctx context.Context, mgr ctrl.Manager) error {
	return ctrl.NewWebhookManagedBy(mgr).
		For(&octorunv1.RunnerSet{}).
		WithDefaulter(w).
		WithValidator(w).
		Complete()
}

// Default implements webhook.CustomDefaulter so a webhook will be registered for the type
func (w *RunnerSetWebhook) Default(ctx context.Context, obj runtime.Object) error {
	runnerset, ok := obj.(*octorunv1.RunnerSet)
	if !ok {
		return apierrors.NewBadRequest(fmt.Sprintf("expected a RunnerSet but got a %T", obj))
	}

	if runnerset.Labels == nil {
		runnerset.Labels = make(map[string]string)
	}

	if runnerset.Spec.Template.Labels == nil {
		runnerset.Spec.Template.Labels = make(map[string]string)
	}

	if _, ok := runnerset.Spec.Template.Labels[octorunv1.LabelRunnerSetName]; !ok {
		runnerset.Spec.Template.Labels[octorunv1.LabelRunnerSetName] = runnerset.GetName()
	}

	return nil
}

// ValidateCreate implements webhook.CustomValidator so a webhook will be registered for the type
func (w *RunnerSetWebhook) ValidateCreate(ctx context.Context, obj runtime.Object) error {
	var allErrs field.ErrorList
	runnerset, ok := obj.(*octorunv1.RunnerSet)
	if !ok {
		return apierrors.NewBadRequest(fmt.Sprintf("expected a RunnerSet but got a %T", obj))
	}

	template := runnerset.Spec.Template
	templatePath := field.NewPath("spec", "template")
	if !matchOrgOrRepoURLRegexp.MatchString(template.Spec.URL) {
		allErrs = append(allErrs, field.Invalid(templatePath.Child("spec", "url"), template.Spec.URL, invalidURLMessage))
	}

	if len(allErrs) == 0 {
		return nil
	}

	return apierrors.NewInvalid(runnerset.GetObjectKind().GroupVersionKind().GroupKind(), runnerset.GetName(), allErrs)
}

// ValidateUpdate implements webhook.CustomValidator so a webhook will be registered for the type
func (w *RunnerSetWebhook) ValidateUpdate(ctx context.Context, oldObj, newObj runtime.Object) error {
	var allErrs field.ErrorList
	_, ok := oldObj.(*octorunv1.RunnerSet)
	if !ok {
		return apierrors.NewBadRequest(fmt.Sprintf("expected a RunnerSet but got a %T", oldObj))
	}

	newRunnerSet, ok := newObj.(*octorunv1.RunnerSet)
	if !ok {
		return apierrors.NewBadRequest(fmt.Sprintf("expected a RunnerSet but got a %T", newObj))
	}

	newTemplate := newRunnerSet.Spec.Template
	newTemplatePath := field.NewPath("spec", "template")
	if !matchOrgOrRepoURLRegexp.MatchString(newTemplate.Spec.URL) {
		allErrs = append(allErrs, field.Invalid(newTemplatePath.Child("spec", "url"), newTemplate.Spec.URL, invalidURLMessage))
	}

	if len(allErrs) == 0 {
		return nil
	}

	return apierrors.NewInvalid(newRunnerSet.GetObjectKind().GroupVersionKind().GroupKind(), newRunnerSet.GetName(), allErrs)
}

// ValidateDelete implements webhook.CustomValidator so a webhook will be registered for the type
func (w *RunnerSetWebhook) ValidateDelete(ctx context.Context, obj runtime.Object) error {
	return nil
}
