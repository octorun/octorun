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

package v1alpha2

const (
	// LabelPrefix is well known prefix used to labels the runners.
	// the controller will pass each runners labels with this prefix
	// to the github runners labels.
	//
	// Example:
	//	"octorun.github.io/foo": "bar"
	// will passed to the Github runner label as
	//	`foo=bar`
	LabelPrefix = "octorun.github.io/"

	// LabelRunnerName is used to labels the GitHub runner using `runner: ` prefix.
	// By default, when creating a Runner resource the admission controller will
	// add this label with the value from Runner .metadata.name if created Runner
	// does not have this label.
	//
	// Example:
	// 	apiVersion: octorun.github.io/v1alpha2
	// 	kind: Runner
	// 	metadata:
	// 	name: runner-sample
	// 	labels:
	// 		octorun.github.io/runner: myrunner
	//
	// will passed to the Github runner label as
	//	runner=myrunner
	//
	// NOTE: this defaulting is applicable only if the Runner resource
	// is created directly (not using the runner template. eg: from runnerset).
	LabelRunnerName = LabelPrefix + "runner"

	// LabelRunnerSetName is used to labels the Github runner using `runnerset: ` prefix.
	// By default, when creating a RunnerSet the admission controller will
	// add this label to the Runner controlled by RunnerSet with the value from RunnerSet
	// .metadata.name if created RunnerSet does not have this label.
	//
	// Example:
	// 	apiVersion: octorun.github.io/v1alpha2
	// 	kind: RunnerSet
	// 	metadata:
	//   	name: runnerset-sample
	// 	spec:
	//   	template:
	//     	metadata:
	//       	labels:
	//         	octorun.github.io/runnerset: myrunnerset
	//
	// will passed to Github runner label as
	//	runnerset=myrunnerset
	//
	LabelRunnerSetName = LabelPrefix + "runnerset"

	LabelControllerRevisionHash = LabelPrefix + "revision-hash"
)
