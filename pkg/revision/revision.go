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
	"bytes"
	"context"
	"sort"

	appsv1 "k8s.io/api/apps/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/util/retry"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type Revisioner interface {
	HashLabelKey() string

	NextRevision(ctx context.Context, c client.Client, obj client.Object, rev int64) (*appsv1.ControllerRevision, error)

	ListRevision(ctx context.Context, c client.Client, obj client.Object) ([]*appsv1.ControllerRevision, error)
}

type revisionableObject struct {
	*unstructured.Unstructured
}

func convertObject(obj client.Object) (*revisionableObject, error) {
	if _, ok := obj.(runtime.Unstructured); ok {
		obj = obj.DeepCopyObject().(client.Object)
	}

	u, err := runtime.DefaultUnstructuredConverter.ToUnstructured(obj)
	if err != nil {
		return nil, err
	}

	return &revisionableObject{Unstructured: &unstructured.Unstructured{Object: u}}, nil
}

// MakeHistory make a revision history for given owner and returns the next revision will be used by owner into rev.
func MakeHistory(ctx context.Context, c client.Client, r Revisioner, owner client.Object, rev *appsv1.ControllerRevision) error {
	log := ctrl.LoggerFrom(ctx)
	var currentRevision, nextRevision *appsv1.ControllerRevision
	revObj, err := convertObject(owner)
	if err != nil {
		return err
	}

	revisions, err := r.ListRevision(ctx, c, owner)
	if err != nil {
		return err
	}

	collision := revObj.GetCollisionCount()
	sort.Stable(SortableRevisions(revisions))
	revisionLen := len(revisions)
	nextRevision, err = r.NextRevision(ctx, c, owner, nextRevisionNumber(revisions))
	if err != nil {
		return err
	}

	equalRevision := FindEqual(revisions, nextRevision, r.HashLabelKey())
	equalRevisionLen := len(equalRevision)
	if equalRevisionLen > 0 && IsEqual(revisions[revisionLen-1], equalRevision[equalRevisionLen-1], r.HashLabelKey()) {
		// if the equivalent revision is immediately prior the next revision has not changed
		nextRevision = revisions[revisionLen-1]
	} else if equalRevisionLen > 0 {
		// if the equivalent revision is not immediately prior we will roll back by incrementing the
		// Revision of the equivalent revision
		newRevision := equalRevision[equalRevisionLen-1]
		if err := retry.RetryOnConflict(retry.DefaultBackoff, func() error {
			if newRevision.Revision == nextRevision.Revision {
				return nil
			}

			// bump revision number to the next revision number
			newRevision.Revision = nextRevision.Revision
			log.V(1).Info("bump revision number", "revision", newRevision.Name)
			if err := c.Update(ctx, newRevision); err != nil {
				return err
			}

			if err := c.Get(ctx, client.ObjectKeyFromObject(newRevision), newRevision); err != nil {
				return err
			}

			return nil
		}); err != nil {
			return err
		}

		// copy updated revision into nextRevision so nextRevision always carry the next revision
		newRevision.DeepCopyInto(nextRevision)
	} else {
		for {
			log.V(1).Info("creating new revision", "revision", nextRevision.Name)
			if err := c.Create(ctx, nextRevision); err != nil {
				if apierrors.IsAlreadyExists(err) {
					existsRevision := &appsv1.ControllerRevision{}
					if err := c.Get(ctx, client.ObjectKeyFromObject(nextRevision), existsRevision); err != nil {
						return err
					}

					if bytes.Equal(existsRevision.Data.Raw, nextRevision.Data.Raw) {
						log.V(1).Info("found equal existing revision", "revision", existsRevision.Name)
						existsRevision.DeepCopyInto(nextRevision)
						break
					}

					log.V(1).Info("found not equal existing revision", "revision", existsRevision.Name)
					collision++
					continue
				}

				return err
			}

			break
		}
	}

	if rev := revObj.GetCurrentRevision(); rev != "" {
		for i := range revisions {
			if revisions[i].Name == rev {
				currentRevision = revisions[i]
			}
		}
	} else {
		currentRevision = nextRevision
	}

	revObj.SetCurrentRevision(currentRevision.Name)
	revObj.SetNextRevision(nextRevision.Name)
	revObj.SetCollisionCount(collision)
	nextRevision.DeepCopyInto(rev)
	return runtime.DefaultUnstructuredConverter.FromUnstructured(revObj.UnstructuredContent(), owner)
}

// TruncateHistory truncate the revisions history according revisionHistoryLimit
func TruncateHistory(ctx context.Context, c client.Client, r Revisioner, owner client.Object, controlled []client.Object) error {
	revisions, err := r.ListRevision(ctx, c, owner)
	if err != nil {
		return err
	}

	revObj, err := convertObject(owner)
	if err != nil {
		return err
	}

	histories := make([]*appsv1.ControllerRevision, 0, len(revisions))
	excluded := map[string]bool{}
	if currentRev := revObj.GetCurrentRevision(); currentRev != "" {
		excluded[currentRev] = true
	}

	if nextRev := revObj.GetNextRevision(); nextRev != "" {
		excluded[nextRev] = true
	}

	for _, c := range controlled {
		if controlledLabel := c.GetLabels(); controlledLabel != nil {
			excluded[controlledLabel[r.HashLabelKey()]] = true
		}
	}

	for _, rev := range revisions {
		if !excluded[rev.Name] {
			histories = append(histories, rev)
		}
	}

	historyLen := len(histories)
	historyLimit := int(revObj.GetRevisionHistoryLimit())
	if historyLen <= historyLimit {
		return nil
	}

	histories = histories[:(historyLen - historyLimit)]
	for _, rev := range histories {
		if err := c.Delete(ctx, rev); err != nil {
			return err
		}
	}

	return nil
}

func (ro *revisionableObject) GetRevisionHistoryLimit() int32 {
	r, _, _ := unstructured.NestedInt64(ro.Object, "spec", "revisionHistoryLimit")
	return int32(r)
}

func (ro *revisionableObject) GetNextRevision() string {
	r, _, _ := unstructured.NestedString(ro.Object, "status", "nextRevision")
	return r
}

func (ro *revisionableObject) GetCurrentRevision() string {
	r, _, _ := unstructured.NestedString(ro.Object, "status", "currentRevision")
	return r
}

func (ro *revisionableObject) GetCollisionCount() int32 {
	r, _, _ := unstructured.NestedInt64(ro.Object, "status", "collisionCount")
	return int32(r)
}

func (ro *revisionableObject) SetNextRevision(rev string) {
	_ = unstructured.SetNestedField(ro.Object, rev, "status", "nextRevision")
}

func (ro *revisionableObject) SetCurrentRevision(rev string) {
	_ = unstructured.SetNestedField(ro.Object, rev, "status", "currentRevision")
}

func (ro *revisionableObject) SetCollisionCount(collision int32) {
	_ = unstructured.SetNestedField(ro.Object, int64(collision), "status", "collisionCount")
}
