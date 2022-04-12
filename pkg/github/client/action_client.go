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

package client

import (
	"context"
	"net/url"
	"strings"

	"github.com/google/go-github/v41/github"
)

type ActionClient interface {
	GetRunner(ctx context.Context, runnerURL string, runnerID int64) (Runner, error)
	CreateRunnerToken(ctx context.Context, runnerURL string) (RunnerToken, error)
}

type Runner interface {
	GetName() string
	GetBusy() bool
	GetOS() string
	GetStatus() string
}

type RunnerToken interface {
	GetToken() string
	GetExpiresAt() github.Timestamp
}

type runnerKey struct {
	Owner      string
	Repository string
}

func parseRunnerURL(u string) runnerKey {
	parsedURL, err := url.Parse(u)
	if err != nil {
		return runnerKey{}
	}

	switch paths := strings.Split(strings.TrimPrefix(parsedURL.Path, "/"), "/"); len(paths) {
	case 0:
		return runnerKey{}
	case 1:
		return runnerKey{
			Owner: paths[0],
		}
	default:
		return runnerKey{
			Owner:      paths[0],
			Repository: paths[1],
		}
	}
}

func (gh *Client) GetRunner(ctx context.Context, runnerURL string, runnerID int64) (Runner, error) {
	runnerKey := parseRunnerURL(runnerURL)
	if runnerKey.Repository != "" {
		runner, _, err := gh.Actions.GetRunner(ctx, runnerKey.Owner, runnerKey.Repository, runnerID)
		return runner, err
	}

	runner, _, err := gh.Actions.GetOrganizationRunner(ctx, runnerKey.Owner, runnerID)
	return runner, err
}

func (gh *Client) CreateRunnerToken(ctx context.Context, runnerURL string) (RunnerToken, error) {
	runnerKey := parseRunnerURL(runnerURL)
	if runnerKey.Repository != "" {
		runnerToken, _, err := gh.Actions.CreateRegistrationToken(ctx, runnerKey.Owner, runnerKey.Repository)
		return runnerToken, err
	}

	runnerToken, _, err := gh.Actions.CreateOrganizationRegistrationToken(ctx, runnerKey.Owner)
	return runnerToken, err
}
