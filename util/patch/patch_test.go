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
	"testing"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"

	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
)

func TestPatcher_Patch(t *testing.T) {
	tests := []struct {
		name    string
		before  client.Object
		after   client.Object
		wantErr bool
	}{
		{
			name: "Add annotation",
			before: &corev1.Pod{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-pod",
					Namespace: "test-namespace",
				},
			},
			after: &corev1.Pod{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-pod",
					Namespace: "test-namespace",
					Annotations: map[string]string{
						"foo": "bar",
					},
				},
			},
		},
		{
			name: "Patch status",
			before: &corev1.Pod{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-pod",
					Namespace: "test-namespace",
				},
			},
			after: &corev1.Pod{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "test-pod",
					Namespace: "test-namespace",
				},
				Status: corev1.PodStatus{
					Phase: corev1.PodRunning,
				},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()
			before := tt.before.DeepCopyObject().(client.Object)
			fclient := fake.NewClientBuilder().WithObjects(before).WithScheme(clientgoscheme.Scheme).Build()
			patcher, err := NewPatcher(fclient, before)
			if err != nil {
				t.Fatalf("Expected no error initializing patcher: %v", err)
			}

			after := tt.after.DeepCopyObject().(client.Object)
			if err := patcher.Patch(ctx, after); (err != nil) != tt.wantErr {
				t.Errorf("Patcher.Patch() error = %v, wantErr %v", err, tt.wantErr)
			}

			patched := tt.before.DeepCopyObject().(client.Object)
			if err := fclient.Get(ctx, client.ObjectKeyFromObject(patched), patched); err != nil {
				t.Fatalf("Unexpected error getting patched object: %v", err)
			}

			if !reflect.DeepEqual(after, patched) {
				t.Errorf("Expected after to be the same after patching\n after: %v\n patched: %v\n", after, patched)
			}
		})
	}
}
