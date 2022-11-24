// Licensed to the Apache Software Foundation (ASF) under one or more
// contributor license agreements.  See the NOTICE file distributed with
// this work for additional information regarding copyright ownership.
// The ASF licenses this file to You under the Apache License, Version 2.0
// (the "License"); you may not use this file except in compliance with
// the License.  You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package api

import (
	"context"
	"crypto/tls"
	"net"
	"net/http"
	"net/http/pprof"
	"time"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"

	apirouter "github.com/apache/apisix-ingress-controller/pkg/api/router"
	"github.com/apache/apisix-ingress-controller/pkg/apisix"
	"github.com/apache/apisix-ingress-controller/pkg/config"
	"github.com/apache/apisix-ingress-controller/pkg/log"
	"github.com/apache/apisix-ingress-controller/pkg/metrics"
	"github.com/apache/apisix-ingress-controller/pkg/types"
)

// Server represents the API Server in ingress-apisix-controller.
type Server struct {
	HealthState     *apirouter.HealthState
	httpServer      *gin.Engine
	admissionServer *http.Server
	httpListener    net.Listener
	pprofMu         *http.ServeMux
}

// NewServer initializes the API Server.
func NewServer(cfg *config.Config) (*Server, error) {
	httpListener, err := net.Listen("tcp", cfg.HTTPListen)
	if err != nil {
		return nil, err
	}
	gin.SetMode(gin.ReleaseMode)
	httpServer := gin.New()
	httpServer.Use(log.GinRecovery(log.DefaultLogger, true), log.GinLogger(log.DefaultLogger))
	apirouter.Mount(httpServer)

	srv := &Server{
		HealthState:  new(apirouter.HealthState),
		httpServer:   httpServer,
		httpListener: httpListener,
	}
	apirouter.MountApisixHealthz(httpServer, srv.HealthState)

	if cfg.EnableProfiling {
		srv.pprofMu = new(http.ServeMux)
		srv.pprofMu.HandleFunc("/debug/pprof/cmdline", pprof.Cmdline)
		srv.pprofMu.HandleFunc("/debug/pprof/profile", pprof.Profile)
		srv.pprofMu.HandleFunc("/debug/pprof/symbol", pprof.Symbol)
		srv.pprofMu.HandleFunc("/debug/pprof/trace", pprof.Trace)
		srv.pprofMu.HandleFunc("/debug/pprof/", pprof.Index)
		httpServer.GET("/debug/pprof/*profile", gin.WrapF(srv.pprofMu.ServeHTTP))
	}

	cert, err := tls.LoadX509KeyPair(cfg.CertFilePath, cfg.KeyFilePath)
	if err != nil {
		log.Warnw("failed to load x509 key pair, will not start admission server",
			zap.String("Error", err.Error()),
			zap.String("CertFilePath", cfg.CertFilePath),
			zap.String("KeyFilePath", cfg.KeyFilePath),
		)
	} else {
		admission := gin.New()
		admission.Use(gin.Recovery(), gin.Logger())
		apirouter.MountWebhooks(admission, &apisix.ClusterOptions{
			AdminAPIVersion:  cfg.APISIX.AdminAPIVersion,
			Name:             cfg.APISIX.DefaultClusterName,
			AdminKey:         cfg.APISIX.DefaultClusterAdminKey,
			BaseURL:          cfg.APISIX.DefaultClusterBaseURL,
			MetricsCollector: metrics.NewPrometheusCollector(),
		})

		srv.admissionServer = &http.Server{
			Addr:    cfg.HTTPSListen,
			Handler: admission,
			TLSConfig: &tls.Config{
				Certificates: []tls.Certificate{cert},
			},
		}
	}

	return srv, nil
}

// Run launches the API Server.
func (srv *Server) Run(stopCh <-chan struct{}) error {
	go func() {
		<-stopCh

		closed := make(chan struct{}, 2)
		go srv.closeHttpServer(closed)
		go srv.closeAdmissionServer(closed)

		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		cnt := 2
		for cnt > 0 {
			select {
			case <-ctx.Done():
				log.Errorf("close servers timeout")
				return
			case <-closed:
				cnt--
				log.Debug("close a server")
			}
		}
	}()

	go func() {
		log.Debug("starting http server")
		if err := srv.httpServer.RunListener(srv.httpListener); err != nil && !types.IsUseOfClosedNetConnErr(err) {
			log.Errorf("failed to start http server: %s", err)
		}
	}()

	if srv.admissionServer != nil {
		go func() {
			log.Debug("starting admission server")
			if err := srv.admissionServer.ListenAndServeTLS("", ""); err != nil && !types.IsUseOfClosedNetConnErr(err) {
				log.Errorf("failed to start admission server: %s", err)
			}
		}()
	}

	return nil
}

func (srv *Server) closeHttpServer(closed chan struct{}) {
	if err := srv.httpListener.Close(); err != nil {
		log.Errorf("failed to close http listener: %s", err)
	}
	closed <- struct{}{}
}

func (srv *Server) closeAdmissionServer(closed chan struct{}) {
	if srv.admissionServer != nil {
		if err := srv.admissionServer.Shutdown(context.TODO()); err != nil {
			log.Errorf("failed to shutdown admission server: %s", err)
		}
	}
	closed <- struct{}{}
}
