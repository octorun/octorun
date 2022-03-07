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
	"net/http"
	"net/url"

	"github.com/google/go-github/v41/github"
	"golang.org/x/oauth2"
)

type Client struct {
	*github.Client
}

type clientOption struct {
	endpoint string
	token    string
}

type ClientOption func(o *clientOption)

func New(opts ...ClientOption) *Client {
	var (
		httpClient = &http.Client{}
		option     clientOption
	)

	for _, o := range opts {
		o(&option)
	}

	if option.token != "" {
		ts := oauth2.StaticTokenSource(&oauth2.Token{AccessToken: option.token})
		httpClient = oauth2.NewClient(context.Background(), ts)
	}

	client := github.NewClient(httpClient)
	client.BaseURL, _ = url.Parse(option.endpoint)
	return &Client{
		Client: client,
	}
}

func WithAccessToken(token string) ClientOption {
	return func(o *clientOption) {
		o.token = token
	}
}

func WithEndpoint(endpoint string) ClientOption {
	return func(o *clientOption) {
		o.endpoint = endpoint
	}
}
