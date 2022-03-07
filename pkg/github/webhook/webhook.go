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

package webhook

import (
	"context"
	"fmt"
	"net/http"

	"github.com/google/go-github/v41/github"

	logf "sigs.k8s.io/controller-runtime/pkg/log"
)

var log = logf.Log.WithName("github").WithName("webhook")

func WebhookFor(handler Handler) *Webhook {
	return &Webhook{
		Handler: handler,
	}
}

type Webhook struct {
	Handler Handler

	GetSecretFn func() ([]byte, error)
}

func (wh *Webhook) Handle(ctx context.Context, req Request) {
	wh.Handler.Handle(logf.IntoContext(ctx, log), req)
}

var _ http.Handler = &Webhook{}

func (wh *Webhook) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	secret, err := wh.GetSecretFn()
	if err != nil {
		http.Error(w, fmt.Sprintf("unable to get webhook secret. err: %+v", err), http.StatusInternalServerError)
		return
	}

	payload, err := github.ValidatePayload(r, secret)
	if err != nil {
		http.Error(w, fmt.Sprintf("unable to validate payload. err: %+v", err), http.StatusBadRequest)
		return
	}

	event, err := github.ParseWebHook(github.WebHookType(r), payload)
	if err != nil {
		http.Error(w, fmt.Sprintf("unable to parse webhook. err: %+v", err), http.StatusBadRequest)
		return
	}

	wh.Handle(r.Context(), Request{Event: event})
	w.WriteHeader(http.StatusOK)
}
