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
	"net"
	"net/http"
	"sync"
)

type Server struct {
	Addr string

	Path string

	Secret string

	mux            *http.ServeMux
	defaultingOnce sync.Once
}

func (s *Server) setDefaults() {
	if s.mux == nil {
		s.mux = http.NewServeMux()
	}
}

type HandlerRegistrar interface {
	// Register the given handler to the webhook server.
	WithHandler(h Handler)
}

func (s *Server) WithHandler(h Handler) {
	s.defaultingOnce.Do(s.setDefaults)
	wh := WebhookFor(h)
	wh.GetSecretFn = func() ([]byte, error) {
		return []byte(s.Secret), nil
	}

	s.mux.Handle(s.Path, wh)
}

func (s *Server) Start(ctx context.Context) error {
	s.defaultingOnce.Do(s.setDefaults)
	listener, err := net.Listen("tcp", s.Addr)
	if err != nil {
		return err
	}

	srv := http.Server{
		Handler: s.mux,
	}

	log.Info("serving webhook server", "addr", s.Addr)
	idleConnsClosed := make(chan struct{})
	go func() {
		<-ctx.Done()
		log.Info("shutting down webhook server")
		if err := srv.Shutdown(context.Background()); err != nil {
			log.Error(err, "error shutting down the HTTP server")
		}

		close(idleConnsClosed)
	}()

	if err := srv.Serve(listener); err != nil && err != http.ErrServerClosed {
		return err
	}

	<-idleConnsClosed
	return nil
}
