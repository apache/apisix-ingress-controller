// Licensed to the Apache Software Foundation (ASF) under one
// or more contributor license agreements.  See the NOTICE file
// distributed with this work for additional information
// regarding copyright ownership.  The ASF licenses this file
// to you under the Apache License, Version 2.0 (the
// "License"); you may not use this file except in compliance
// with the License.  You may obtain a copy of the License at
//
//   http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing,
// software distributed under the License is distributed on an
// "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY
// KIND, either express or implied.  See the License for the
// specific language governing permissions and limitations
// under the License.

package server

import (
	"context"
	"net/http"
	"time"

	"github.com/apache/apisix-ingress-controller/internal/provider"
)

type Server struct {
	server *http.Server
	mux    *http.ServeMux
}

func (s *Server) Start(ctx context.Context) error {
	stop := make(chan error, 1)
	go func() {
		if err := s.server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			stop <- err
		}
		close(stop)
	}()
	select {
	case <-ctx.Done():
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		return s.server.Shutdown(shutdownCtx)
	case err := <-stop:
		return err
	}
}

func (s *Server) Register(pathPrefix string, registrant provider.RegisterHandler) {
	subMux := http.NewServeMux()
	registrant.Register(pathPrefix, subMux)
	s.mux.Handle(pathPrefix+"/", http.StripPrefix(pathPrefix, subMux))
	s.mux.HandleFunc(pathPrefix, func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, pathPrefix+"/", http.StatusPermanentRedirect)
	})
}

func NewServer(addr string) *Server {
	mux := http.NewServeMux()
	return &Server{
		server: &http.Server{
			Addr:    addr,
			Handler: mux,
		},
		mux: mux,
	}
}
