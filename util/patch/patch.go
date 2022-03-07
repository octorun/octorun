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

package patch

import (
	"context"
	"reflect"

	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	kerrors "k8s.io/apimachinery/pkg/util/errors"

	"sigs.k8s.io/controller-runtime/pkg/client"
)

type Patcher struct {
	client          client.Client
	before          map[string]interface{}
	beforeStatus    interface{}
	beforeHasStatus bool
	patch           client.Patch
	statusPatch     client.Patch
}

func NewPatcher(c client.Client, obj client.Object) (*Patcher, error) {
	// If the object is already unstructured, we need to perform a deepcopy first
	// because the `DefaultUnstructuredConverter.ToUnstructured` function returns
	// the underlying unstructured object map without making a copy.
	if _, ok := obj.(runtime.Unstructured); ok {
		obj = obj.DeepCopyObject().(client.Object)
	}

	// Create a copy of the original object as well as converting that copy to
	// unstructured data.
	before, err := runtime.DefaultUnstructuredConverter.ToUnstructured(obj)
	if err != nil {
		return nil, err
	}

	// Attempt to extract the status from the resource for easier comparison later
	beforeStatus, beforeHasStatus, err := unstructured.NestedFieldCopy(before, "status")
	if err != nil {
		return nil, err
	}

	// If the resource contains a status then remove it from the unstructured
	// copy to avoid unnecessary patching later.
	if beforeHasStatus {
		unstructured.RemoveNestedField(before, "status")
	}

	return &Patcher{
		client:          c,
		before:          before,
		beforeStatus:    beforeStatus,
		beforeHasStatus: beforeHasStatus,
		patch:           client.MergeFrom(obj.DeepCopyObject().(client.Object)),
		statusPatch:     client.MergeFrom(obj.DeepCopyObject().(client.Object)),
	}, nil
}

func (p *Patcher) Patch(ctx context.Context, obj client.Object, opts ...client.PatchOption) error {
	// If the object is already unstructured, we need to perform a deepcopy first
	// because the `DefaultUnstructuredConverter.ToUnstructured` function returns
	// the underlying unstructured object map without making a copy.
	if _, ok := obj.(runtime.Unstructured); ok {
		obj = obj.DeepCopyObject().(client.Object)
	}

	// Convert the resource to unstructured to compare against our before copy.
	after, err := runtime.DefaultUnstructuredConverter.ToUnstructured(obj)
	if err != nil {
		return err
	}

	// Attempt to extract the status from the resource for easier comparison later
	afterStatus, afterHasStatus, err := unstructured.NestedFieldCopy(after, "status")
	if err != nil {
		return err
	}

	// If the resource contains a status then remove it from the unstructured
	// copy to avoid unnecessary patching later.
	if afterHasStatus {
		unstructured.RemoveNestedField(after, "status")
	}

	var errs []error
	if !reflect.DeepEqual(p.before, after) {
		// Only issue a Patch if the before and after resources (minus status) differ
		if err := p.client.Patch(ctx, obj, p.patch, opts...); err != nil {
			errs = append(errs, err)
		}
	}

	if (p.beforeHasStatus || afterHasStatus) && !reflect.DeepEqual(p.beforeStatus, afterStatus) {
		// only issue a Status Patch if the resource has a status and the beforeStatus
		// and afterStatus copies differ
		if err := p.client.Status().Patch(ctx, obj, p.statusPatch, opts...); err != nil {
			errs = append(errs, err)
		}
	}

	return kerrors.NewAggregate(errs)
}
