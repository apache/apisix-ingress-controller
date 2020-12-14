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
package ingress

import (
	"fmt"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	api6Informers "github.com/gxthrj/apisix-ingress-types/pkg/client/informers/externalversions"
	"github.com/spf13/cobra"

	"github.com/api7/ingress-controller/conf"
	"github.com/api7/ingress-controller/pkg/api"
	"github.com/api7/ingress-controller/pkg/config"
	"github.com/api7/ingress-controller/pkg/ingress/controller"
	"github.com/api7/ingress-controller/pkg/log"
)

func dief(template string, args ...interface{}) {
	if !strings.HasSuffix(template, "\n") {
		template += "\n"
	}
	fmt.Fprintf(os.Stderr, template, args...)
	os.Exit(1)
}

func waitForSignal(stopCh chan struct{}) {
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)

	sig := <-sigCh
	log.Infof("signal %d (%s) received", sig, sig.String())
	close(stopCh)
}

// NewIngressCommand creates the ingress sub command for apisix-ingress-controller.
func NewIngressCommand() *cobra.Command {
	var configPath string
	cfg := config.NewDefaultConfig()

	cmd := &cobra.Command{
		Use:   "ingress [flags]",
		Short: "launch the controller",
		Example: `Run apisix-ingress-controller from configuration file:

	apisix-ingress-controller ingress --config-path /path/to/config.json`,
		Run: func(cmd *cobra.Command, args []string) {
			if configPath != "" {
				c, err := config.NewConfigFromFile(configPath)
				if err != nil {
					dief("failed to initialize configuration: %s", err)
				}
				cfg = c
			}

			logger, err := log.NewLogger(
				log.WithLogLevel(cfg.LogLevel),
				log.WithOutputFile(cfg.LogOutput),
			)
			if err != nil {
				dief("failed to initialize logging: %s", err)
			}
			log.DefaultLogger = logger

			kubeClientSet := conf.GetKubeClient()
			apisixClientset := conf.InitApisixClient()
			sharedInformerFactory := api6Informers.NewSharedInformerFactory(apisixClientset, 0)
			stop := make(chan struct{})
			c := &controller.Api6Controller{
				KubeClientSet:             kubeClientSet,
				Api6ClientSet:             apisixClientset,
				SharedInformerFactory:     sharedInformerFactory,
				CoreSharedInformerFactory: conf.CoreSharedInformerFactory,
				Stop:                      stop,
			}
			epInformer := c.CoreSharedInformerFactory.Core().V1().Endpoints()
			conf.EndpointsInformer = epInformer
			// endpoint
			c.Endpoint()
			go c.CoreSharedInformerFactory.Start(stop)

			// ApisixRoute
			c.ApisixRoute()
			// ApisixUpstream
			c.ApisixUpstream()
			// ApisixService
			c.ApisixService()

			go func() {
				time.Sleep(time.Duration(10) * time.Second)
				c.SharedInformerFactory.Start(stop)
			}()

			srv, err := api.NewServer(cfg)
			if err != nil {
				dief("failed to create API Server: %s", err)
			}

			// TODO add sync.WaitGroup
			go func() {
				if err := srv.Run(stop); err != nil {
					dief("failed to launch API Server: %s", err)
				}
			}()

			waitForSignal(stop)
			log.Info("apisix-ingress-controller exited")
		},
	}

	cmd.PersistentFlags().StringVar(&configPath, "config-path", "", "configuration file path for apisix-ingress-controller")
	cmd.PersistentFlags().StringVar(&cfg.LogLevel, "log-level", "warn", "error log level")
	cmd.PersistentFlags().StringVar(&cfg.LogOutput, "log-output", "stderr", "error log output file")
	cmd.PersistentFlags().StringVar(&cfg.HTTPListen, "http-listen", ":8080", "the HTTP Server listen address")
	cmd.PersistentFlags().BoolVar(&cfg.EnableProfiling, "enable-profiling", true, "enable profiling via web interface host:port/debug/pprof")
	cmd.PersistentFlags().StringVar(&cfg.Kubernetes.Kubeconfig, "kubeconfig", "", "Kubernetes configuration file (by default in-cluster configuration will be used)")
	cmd.PersistentFlags().DurationVar(&cfg.Kubernetes.ResyncInterval.Duration, "resync-interval", time.Minute, "the controller resync (with Kubernetes) interval, the minimum resync interval is 30s")
	cmd.PersistentFlags().StringVar(&cfg.APISIX.BaseURL, "apisix-base-url", "", "the base URL for APISIX admin api / manager api")
	cmd.PersistentFlags().StringVar(&cfg.APISIX.AdminKey, "apisix-admin-key", "", "admin key used for the authorization of APISIX admin api / manager api")

	return cmd
}
