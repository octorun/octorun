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
	"errors"
	"net/url"

	"github.com/google/go-github/v41/github"
	"golang.org/x/oauth2"
)

type Client struct {
	*github.Client
}

type Opts struct {
	endpoint      string
	personalToken string

	appID          int64
	appKey         string
	installationID string
}

type ClientOption func(o *Opts)

func New(opts ...ClientOption) (*Client, error) {
	var option Opts
	for _, o := range opts {
		o(&option)
	}

	baseURL, err := url.Parse(option.endpoint)
	if err != nil {
		return nil, err
	}

	var ts oauth2.TokenSource
	if option.personalToken != "" {
		ts = newPersonalTokenSource(option.personalToken)
	} else if option.appID != 0 && option.appKey != "" && option.installationID != "" {
		ts, err = newInstallationTokenSource(baseURL.String(), option.appID, option.appKey, option.installationID)
		if err != nil {
			return nil, err
		}
	} else {
		return nil, errors.New("unable to authenticate Github Client.")
	}

	hc := oauth2.NewClient(context.Background(), ts)
	client := github.NewClient(hc)
	client.BaseURL = baseURL
	return &Client{
		Client: client,
	}, nil
}

func WithEndpoint(endpoint string) ClientOption {
	return func(o *Opts) {
		o.endpoint = endpoint
	}
}

func WithAppID(id int64) ClientOption {
	return func(o *Opts) {
		o.appID = id
	}
}

func WithAppPrivateKey(key string) ClientOption {
	return func(o *Opts) {
		o.appKey = key
	}
}

func WithInstallationID(id string) ClientOption {
	return func(o *Opts) {
		o.installationID = id
	}
}

func WithPersonalAccessToken(token string) ClientOption {
	return func(o *Opts) {
		o.personalToken = token
	}
}
