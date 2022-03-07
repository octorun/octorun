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
	"flag"

	"octorun.github.io/octorun/pkg/github/client"
	"octorun.github.io/octorun/pkg/github/webhook"
)

type Options struct {
	AccessToken    string
	APIEndpoint    string
	WebhookAddress string
	WebhookPath    string
	WebhookSecret  string
}

func (o *Options) BindFlags(fs *flag.FlagSet) {
	fs.StringVar(&o.AccessToken, "github-access-token", "", "The Github Access Token secret.")
	fs.StringVar(&o.APIEndpoint, "github-api-endpoint", "https://api.github.com/", "The Github API endpoint")
	fs.StringVar(&o.WebhookAddress, "github-webook-address", ":9090", "The Address for Github webhook server")
	fs.StringVar(&o.WebhookPath, "github-webhook-path", "/", "The url path for Github webhook handler")
	fs.StringVar(&o.WebhookSecret, "github-webhook-secret", "", "The Github webhook secret")
}

func (o *Options) GetClient() *client.Client {
	return client.New(
		client.WithAccessToken(o.AccessToken),
		client.WithEndpoint(o.APIEndpoint),
	)
}

func (o *Options) GetWebhookServer() *webhook.Server {
	return &webhook.Server{
		Addr:   o.WebhookAddress,
		Path:   o.WebhookPath,
		Secret: o.WebhookSecret,
	}
}
