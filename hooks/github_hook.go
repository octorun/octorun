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

package hooks

import (
	"context"
	"encoding/base64"
	"fmt"
	"strconv"
	"time"

	"github.com/google/go-github/v41/github"

	"k8s.io/utils/pointer"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/manager"

	octorunv1 "octorun.github.io/octorun/api/v1alpha2"
	"octorun.github.io/octorun/pkg/github/webhook"
)

type GithubHook struct {
	client.Client
}

const (
	// runnerCompositeIndexField is field for controller-runtime cache indexing.
	// it is not a real runner object field.
	runnerCompositeIndexField = "composite_idx"
)

// SetupWithManager sets up the GithubHook with the controller-runtime Manager.
func (gh *GithubHook) SetupWithManager(ctx context.Context, mgr manager.Manager, r manager.Runnable) error {
	// adds an index with a composite index field to be used for querying the Runner using several fields.
	//
	// Using controller-runtime cache for querying the runner here is due to the CRD limitation i.e. not yet supported arbitrary
	// field selectors https://github.com/kubernetes/kubernetes/issues/53459
	// And caching with composite index field here is because controller-runtime cache reader with field selector currently
	// doesn't support non-exact field matches.
	// https://github.com/kubernetes-sigs/controller-runtime/blob/v0.10.0/pkg/cache/internal/cache_reader.go#L116-L126
	if err := mgr.GetCache().IndexField(ctx, &octorunv1.Runner{}, runnerCompositeIndexField, gh.runnerCompositeIndexer); err != nil {
		return err
	}

	if whr, ok := r.(webhook.HandlerRegistrar); ok {
		whr.WithHandler(gh)
	}

	return mgr.Add(r)
}

func (gh *GithubHook) Handle(ctx context.Context, req webhook.Request) {
	switch event := req.Event.(type) {
	case *github.WorkflowJobEvent:
		gh.processWorkflowJobEvent(ctx, event)
	default:
		// ignore the rest event
	}
}

// runnerCompositeIndex returns b64 encoded string of cache field key
// with format "name:{runnerName};id:{runnerID};group:{runnerGroup};url:{runnerURL}"
func (gh *GithubHook) runnerCompositeIndex(runnerName, runnerID, runnerGroup, runnerURL string) string {
	format := "name:%s;id:%s;group:%s;url:%s"
	v := fmt.Sprintf(format, runnerName, runnerID, runnerGroup, runnerURL)
	return base64.StdEncoding.EncodeToString([]byte(v))
}

// runnerCompositeIndexer knowns how to build composite index cache key.
func (gh *GithubHook) runnerCompositeIndexer(o client.Object) []string {
	var v []string
	runner, ok := o.(*octorunv1.Runner)
	if !ok {
		return v
	}

	runnerID := pointer.Int64Deref(runner.Spec.ID, -1)
	if runnerID == -1 {
		return v
	}

	return append(v, gh.runnerCompositeIndex(
		runner.Name,
		strconv.Itoa(int(runnerID)),
		runner.Spec.Group,
		runner.Spec.URL,
	))
}

// Trigger runner reconciler by annotate the runner with runner.octorun.github.io/assigned-job-at annotation.
//
// Just trigger the runner reconciler instead of directly patching the status here is because we expected
// the Runner status.phase field is only managed by the runner-controller (not by other systems, including this hook).
func (gh *GithubHook) triggerRunnerReconciliation(ctx context.Context, runnerKey client.ObjectKey) (reterr error) {
	log := ctrl.LoggerFrom(ctx)
	runner := &octorunv1.Runner{}
	if err := gh.Client.Get(ctx, runnerKey, runner); err != nil {
		return client.IgnoreNotFound(err)
	}

	runnerBase := runner.DeepCopy()
	defer func() {
		log.V(1).Info("triggering Runner reconciler", "runner", runnerKey.String())
		if err := gh.Patch(ctx, runner, client.MergeFrom(runnerBase)); err != nil {
			reterr = err
		}
	}()

	annotation := runner.GetAnnotations()
	if annotation == nil {
		annotation = make(map[string]string)
	}

	annotation[octorunv1.AnnotationRunnerAssignedJobAt] = time.Now().Format(time.RFC3339)
	runner.SetAnnotations(annotation)
	return nil
}

func (gh *GithubHook) processWorkflowJobEvent(ctx context.Context, event *github.WorkflowJobEvent) {
	log := ctrl.LoggerFrom(ctx)
	switch action := event.GetAction(); action {
	case "in_progress":
		log.Info("processing workflowjob event", "action", action)
		runnerID := strconv.Itoa(int(event.WorkflowJob.GetRunnerID()))
		runnerName := event.WorkflowJob.GetRunnerName()
		runnerGroup := event.WorkflowJob.GetRunnerGroupName()
		runnerList := &octorunv1.RunnerList{}
		if event.Repo.Owner.GetType() == "Organization" {
			// If repo owned by Organization try to find runner based on organization url first.
			u := event.Repo.Owner.GetHTMLURL()
			log.V(1).Info("try to find Runner based on organization url", "url", u)
			if err := gh.List(ctx, runnerList,
				client.MatchingFields{runnerCompositeIndexField: gh.runnerCompositeIndex(runnerName, runnerID, runnerGroup, u)},
			); err != nil {
				log.Error(err, "unable to find Runner")
				return
			}
		}

		if len(runnerList.Items) == 0 {
			// runnerList.Items is 0 means runner is not registered using organization URL
			// or repository owner is not an Organization.
			// Find the runner based on repository url then.
			u := event.Repo.GetHTMLURL()
			log.V(1).Info("try to find Runner based on repository url", "url", u)
			if err := gh.List(ctx, runnerList,
				client.MatchingFields{runnerCompositeIndexField: gh.runnerCompositeIndex(runnerName, runnerID, runnerGroup, u)},
			); err != nil {
				log.Error(err, "unable to find Runner")
				return
			}
		}

		switch i := len(runnerList.Items); {
		case i == 0:
			// If the runner is still not found, it means that Github scheduled
			// the WorkflowJob to the runner that is not controlled by octorun.
			log.Info("no Runner found in the cluster", "runner", runnerName, "runner-id", runnerID)
			return
		case i > 1:
			log.Info("unexpected found Runner more than 1", "found Runner", i)
			return
		default:
			log.Info("found Runner", "runner", runnerName, "runner-id", runnerID)
			if err := gh.triggerRunnerReconciliation(ctx, client.ObjectKeyFromObject(&runnerList.Items[0])); err != nil {
				log.Error(err, "failed triggering runner reconciliation")
				return
			}
		}
	default:
		return
	}
}
