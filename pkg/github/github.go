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

package github

import (
	"octorun.github.io/octorun/pkg/github/client"
	"octorun.github.io/octorun/pkg/github/webhook"
)

type Client interface {
	client.ActionClient
}

type Github struct {
	client *client.Client

	webhookServer *webhook.Server
}

// Opts allows to manipulate Options.
type Opts func(*Options)

// UseFlagOptions configures the Github to use the Options set by parsing option flags from the CLI.
func UseFlagOptions(in *Options) Opts {
	return func(o *Options) {
		*o = *in
	}
}

func New(opts *Options) (*Github, error) {
	c, err := client.New(
		client.WithEndpoint(opts.APIEndpoint),
		client.WithAppID(opts.AppID),
		client.WithAppPrivateKey(opts.AppPrivateKey),
		client.WithInstallationID(opts.AppInstallationID),
		client.WithPersonalAccessToken(opts.AccessToken),
	)

	if err != nil {
		return nil, err
	}

	return &Github{
		client: c,
		webhookServer: &webhook.Server{
			Addr:   opts.WebhookAddress,
			Path:   opts.WebhookPath,
			Secret: opts.WebhookSecret,
		},
	}, nil
}

func (gh *Github) GetClient() *client.Client { return gh.client }

func (gh *Github) GetWebhookServer() *webhook.Server { return gh.webhookServer }
