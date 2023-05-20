/*
Copyright 2023 The Authors.

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

package revision

import (
	"context"
	"reflect"
	"testing"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/serializer"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	"k8s.io/utils/pointer"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
)

type fakeRevisioner struct {
	revisionFn func(obj client.Object, codec runtime.Codec, rev int64) *appsv1.ControllerRevision
	revisions  []*appsv1.ControllerRevision
}

func fakeRevision(obj client.Object, codec runtime.Codec, rev int64) *appsv1.ControllerRevision {
	rawData, _ := ObjectPatch(obj, codec)
	cr := &appsv1.ControllerRevision{
		Data:     runtime.RawExtension{Raw: rawData},
		Revision: rev,
	}

	hash := Hash(cr, pointer.Int32(0))
	cr.Name = Name(obj.GetName(), hash)
	cr.Namespace = obj.GetNamespace()
	return cr
}

func fakeObj(obj client.Object, revLimit int32, currentRev, nextRev string) client.Object {
	rawMap, _ := runtime.DefaultUnstructuredConverter.ToUnstructured(obj)
	template, _, _ := unstructured.NestedMap(rawMap, "spec", "template")

	u := &unstructured.Unstructured{}
	u.SetName(obj.GetName())
	u.SetNamespace(obj.GetNamespace())
	u.SetKind(obj.GetObjectKind().GroupVersionKind().Kind)
	_ = unstructured.SetNestedMap(u.Object, template, "spec", "template")
	_ = unstructured.SetNestedField(u.Object, int64(revLimit), "spec", "revisionHistoryLimit")
	_ = unstructured.SetNestedField(u.Object, nextRev, "status", "nextRevision")
	_ = unstructured.SetNestedField(u.Object, currentRev, "status", "currentRevision")
	return u
}

func dummyStatefulSet(cmd ...string) *appsv1.StatefulSet {
	return &appsv1.StatefulSet{
		TypeMeta: metav1.TypeMeta{
			Kind: "StatefulSet",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      "dummy",
			Namespace: "dummy",
		},
		Spec: appsv1.StatefulSetSpec{
			Template: corev1.PodTemplateSpec{
				Spec: corev1.PodSpec{
					Containers: []corev1.Container{
						{
							Name:    "dummy",
							Image:   "dummy",
							Command: cmd,
						},
					},
				},
			},
		},
	}
}

func dummyPod(name, rev string) *corev1.Pod {
	podLabels := make(map[string]string)
	podLabels[appsv1.ControllerRevisionHashLabelKey] = rev
	return &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:   name,
			Labels: podLabels,
		},
	}
}

func (r *fakeRevisioner) HashLabelKey() string { return appsv1.ControllerRevisionHashLabelKey }
func (r *fakeRevisioner) NextRevision(ctx context.Context, c client.Client, obj client.Object, rev int64) (*appsv1.ControllerRevision, error) {
	return r.revisionFn(obj, serializer.NewCodecFactory(c.Scheme()).LegacyCodec(appsv1.SchemeGroupVersion), rev), nil
}

func (r *fakeRevisioner) ListRevision(ctx context.Context, c client.Client, obj client.Object) ([]*appsv1.ControllerRevision, error) {
	return r.revisions, nil
}

func TestMakeHistory(t *testing.T) {
	scheme := runtime.NewScheme()
	utilruntime.Must(appsv1.AddToScheme(scheme))
	codec := serializer.NewCodecFactory(scheme).LegacyCodec(appsv1.SchemeGroupVersion)
	tests := []struct {
		name        string
		c           client.Client
		r           Revisioner
		owner       client.Object
		expectedRev *appsv1.ControllerRevision
		wantErr     bool
	}{
		{
			name: "equal_rev_is_immediately_prior",
			c: fake.NewClientBuilder().
				WithScheme(scheme).
				Build(),
			r: &fakeRevisioner{
				revisionFn: fakeRevision,
				revisions: []*appsv1.ControllerRevision{
					fakeRevision(dummyStatefulSet("echo 1"), codec, 1),
					fakeRevision(dummyStatefulSet("whoami"), codec, 2),
				},
			},
			owner:       dummyStatefulSet("whoami"),
			expectedRev: fakeRevision(dummyStatefulSet("whoami"), codec, 2),
		},
		{
			name: "equal_rev_is_not_immediately_prior",
			c: fake.NewClientBuilder().
				WithScheme(scheme).
				WithObjects(fakeRevision(dummyStatefulSet("whoami"), codec, 1)).
				Build(),
			r: &fakeRevisioner{
				revisionFn: fakeRevision,
				revisions: []*appsv1.ControllerRevision{
					fakeRevision(dummyStatefulSet("whoami"), codec, 1),
					fakeRevision(dummyStatefulSet("foo"), codec, 2),
				},
			},
			owner:       fakeObj(dummyStatefulSet("whoami"), 1, fakeRevision(dummyStatefulSet("whoami"), codec, 1).GetName(), ""),
			expectedRev: fakeRevision(dummyStatefulSet("whoami"), codec, 3),
		},
		{
			name: "equal_rev_is_not_immediately_prior_fail_update",
			c: fake.NewClientBuilder().
				WithScheme(scheme).
				WithObjects().
				Build(),
			r: &fakeRevisioner{
				revisionFn: fakeRevision,
				revisions: []*appsv1.ControllerRevision{
					fakeRevision(dummyStatefulSet("whoami"), codec, 1),
					fakeRevision(dummyStatefulSet("foo"), codec, 2),
				},
			},
			owner:   fakeObj(dummyStatefulSet("whoami"), 1, fakeRevision(dummyStatefulSet("whoami"), codec, 1).GetName(), ""),
			wantErr: true,
		},
		{
			name: "create_new_rev_if_there_is_no_one",
			c: fake.NewClientBuilder().
				WithScheme(scheme).
				Build(),
			r: &fakeRevisioner{
				revisionFn: fakeRevision,
				revisions:  []*appsv1.ControllerRevision{},
			},
			owner:       dummyStatefulSet("whoami"),
			expectedRev: fakeRevision(dummyStatefulSet("whoami"), codec, 1),
		},
		{
			name: "rev_already_exists_when_creating_new_rev",
			c: fake.NewClientBuilder().
				WithScheme(scheme).
				WithObjects(fakeRevision(dummyStatefulSet("whoami"), codec, 1)).
				Build(),
			r: &fakeRevisioner{
				revisionFn: fakeRevision,
				revisions:  []*appsv1.ControllerRevision{},
			},
			owner:       dummyStatefulSet("whoami"),
			expectedRev: fakeRevision(dummyStatefulSet("whoami"), codec, 1),
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rev := &appsv1.ControllerRevision{}
			if err := MakeHistory(context.Background(), tt.c, tt.r, tt.owner, rev); (err != nil) != tt.wantErr {
				t.Errorf("MakeHistory() error = %v, wantErr %v", err, tt.wantErr)
			}

			if !tt.wantErr {
				if !reflect.DeepEqual(rev.Name, tt.expectedRev.Name) {
					t.Errorf("got rev name = %v, want %v", rev.Name, tt.expectedRev.Name)
				}

				if !reflect.DeepEqual(rev.Revision, tt.expectedRev.Revision) {
					t.Errorf("got rev number = %v, want %v", rev.Revision, tt.expectedRev.Revision)
				}
			}
		})
	}
}

func TestTruncateHistory(t *testing.T) {
	scheme := runtime.NewScheme()
	utilruntime.Must(appsv1.AddToScheme(scheme))
	tests := []struct {
		name       string
		revisions  []*appsv1.ControllerRevision
		owner      client.Object
		controlled []client.Object
		wantErr    bool
	}{
		{
			name:  "truncate_history",
			owner: fakeObj(dummyStatefulSet(), 3, "rev-5", "rev-5"),
			revisions: []*appsv1.ControllerRevision{
				{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "rev-1",
						Namespace: "test",
					},
				},
				{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "rev-2",
						Namespace: "test",
					},
				},
				{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "rev-3",
						Namespace: "test",
					},
				},
				{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "rev-4",
						Namespace: "test",
					},
				},
				{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "rev-5",
						Namespace: "test",
					},
				},
			},
			controlled: []client.Object{dummyPod("dummy-1", "rev-5")},
		},
		{
			name:  "histories_less_than_limit",
			owner: fakeObj(dummyStatefulSet(), 3, "rev-2", "rev-2"),
			revisions: []*appsv1.ControllerRevision{
				{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "rev-1",
						Namespace: "test",
					},
				},
				{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "rev-2",
						Namespace: "test",
					},
				},
			},
			controlled: []client.Object{dummyPod("dummy-1", "rev-2")},
		},
	}
	for _, tt := range tests {
		revObjFn := func(revisions ...*appsv1.ControllerRevision) []runtime.Object {
			obj := make([]runtime.Object, 0, len(revisions))
			for _, revision := range revisions {
				obj = append(obj, revision)
			}

			return obj
		}

		revisionList := &appsv1.ControllerRevisionList{}
		if err := meta.SetList(revisionList, revObjFn(tt.revisions...)); err != nil {
			t.Fail()
		}

		fakec := fake.NewClientBuilder().
			WithScheme(scheme).
			WithLists(revisionList).
			Build()

		fr := &fakeRevisioner{
			revisions: tt.revisions,
		}
		t.Run(tt.name, func(t *testing.T) {
			if err := TruncateHistory(context.Background(), fakec, fr, tt.owner, tt.controlled); (err != nil) != tt.wantErr {
				t.Errorf("TruncateHistory() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}
