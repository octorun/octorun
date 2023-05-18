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
	"encoding/json"
	"fmt"
	"hash/fnv"
	"strconv"

	"github.com/davecgh/go-spew/spew"

	appsv1 "k8s.io/api/apps/v1"
	apiequality "k8s.io/apimachinery/pkg/api/equality"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/util/rand"
)

// Name returns the Name for a ControllerRevision in the form prefix-hash. If the length
// of prefix is greater than 223 bytes, it is truncated to allow for a name that is no larger than 253 bytes.
func Name(prefix string, hash string) string {
	if len(prefix) > 223 {
		prefix = prefix[:223]
	}

	return fmt.Sprintf("%s-%s", prefix, hash)
}

// Hash hashes the contents of revision's Data using FNV hashing. If probe is not nil, the byte value
// of probe is added written to the hash as well. The returned hash will be a safe encoded string to avoid bad words.
func Hash(revision *appsv1.ControllerRevision, probe *int32) string {
	hf := fnv.New32()
	if len(revision.Data.Raw) > 0 {
		_, _ = hf.Write(revision.Data.Raw)
	}

	if revision.Data.Object != nil {
		hf.Reset()
		printer := spew.ConfigState{
			Indent:         " ",
			SortKeys:       true,
			DisableMethods: true,
			SpewKeys:       true,
		}

		_, _ = printer.Fprintf(hf, "%#v", revision.Data.Object)
	}

	if probe != nil {
		_, _ = hf.Write([]byte(strconv.FormatInt(int64(*probe), 10)))
	}

	return rand.SafeEncodeString(fmt.Sprint(hf.Sum32()))
}

// FindEqual returns all ControllerRevisions in revisions that are equal to needle using IsEqual as the
// equality test. The returned slice preserves the order of revisions.
func FindEqual(revisions []*appsv1.ControllerRevision, needle *appsv1.ControllerRevision, revisionHashLabelKey string) []*appsv1.ControllerRevision {
	var eq []*appsv1.ControllerRevision
	for i := range revisions {
		if IsEqual(revisions[i], needle, revisionHashLabelKey) {
			eq = append(eq, revisions[i])
		}
	}
	return eq
}

// IsEqual returns true if lhs and rhs are either both nil, or both point to non-nil ControllerRevisions that
// contain semantically equivalent data. Otherwise this method returns false.
func IsEqual(lhs *appsv1.ControllerRevision, rhs *appsv1.ControllerRevision, hashLabelKey string) bool {
	var lhsHash, rhsHash *uint32
	if lhs == nil || rhs == nil {
		return lhs == rhs
	}

	if hs, found := lhs.Labels[hashLabelKey]; found {
		hash, err := strconv.ParseInt(hs, 10, 32)
		if err == nil {
			lhsHash = new(uint32)
			*lhsHash = uint32(hash)
		}
	}

	if hs, found := rhs.Labels[hashLabelKey]; found {
		hash, err := strconv.ParseInt(hs, 10, 32)
		if err == nil {
			rhsHash = new(uint32)
			*rhsHash = uint32(hash)
		}
	}

	if lhsHash != nil && rhsHash != nil && *lhsHash != *rhsHash {
		return false
	}

	return bytes.Equal(lhs.Data.Raw, rhs.Data.Raw) && apiequality.Semantic.DeepEqual(lhs.Data.Object, rhs.Data.Object)
}

// ObjectPatch returns a strategic merge patch for given object that will be stored as ControllerRevision data.
func ObjectPatch(obj runtime.Object, codec runtime.Codec) ([]byte, error) {
	data, err := runtime.Encode(codec, obj)
	if err != nil {
		return nil, err
	}

	var raw map[string]interface{}
	if err := json.Unmarshal(data, &raw); err != nil {
		return nil, err
	}

	objCopy := make(map[string]interface{})
	specCopy := make(map[string]interface{})

	spec := raw["spec"].(map[string]interface{})
	template := spec["template"].(map[string]interface{})
	specCopy["template"] = template
	template["$patch"] = "replace"

	objCopy["spec"] = specCopy
	return json.Marshal(objCopy)
}

func nextRevisionNumber(revisions []*appsv1.ControllerRevision) int64 {
	count := len(revisions)
	if count <= 0 {
		return 1
	}

	return revisions[count-1].Revision + 1
}

// SortableRevisions implement sort.Interface is sortable slice of ControllerRevision by its revision.
type SortableRevisions []*appsv1.ControllerRevision

func (r SortableRevisions) Len() int      { return len(r) }
func (r SortableRevisions) Swap(i, j int) { r[i], r[j] = r[j], r[i] }
func (r SortableRevisions) Less(i, j int) bool {
	if r[i].Revision == r[j].Revision {
		if r[j].CreationTimestamp.Equal(&r[i].CreationTimestamp) {
			return r[i].Name < r[j].Name
		}
		return r[j].CreationTimestamp.After(r[i].CreationTimestamp.Time)
	}

	return r[i].Revision < r[j].Revision
}
