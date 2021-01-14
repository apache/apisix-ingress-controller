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
	"encoding/json"
	"fmt"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/spf13/cobra"

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
		Use: "ingress [flags]",
		Long: `launch the ingress controller

You can run apisix-ingress-controller from configuration file or command line options,
if you run it from configuration file, other command line options will be ignored.

Run from configuration file:

    apisix-ingress-controller ingress --config-path /path/to/config.json

Both json and yaml are supported as the configuration file format.

Run from command line options:

    apisix-ingress-controller ingress --apisix-base-url http://apisix-service:9180/apisix/admin --kubeconfig /path/to/kubeconfig

If you run apisix-ingress-controller outside the Kubernetes cluster, --kubeconfig option (or kubeconfig item in configuration file) should be specified explicitly,
or if you run it inside cluster, leave it alone and in-cluster configuration will be discovered and used.

Before you run apisix-ingress-controller, be sure all related resources, like CRDs (ApisixRoute, ApisixUpstream and etc),
the apisix cluster and others are created`,
		Run: func(cmd *cobra.Command, args []string) {
			if configPath != "" {
				c, err := config.NewConfigFromFile(configPath)
				if err != nil {
					dief("failed to initialize configuration: %s", err)
				}
				cfg = c
			}
			if err := cfg.Validate(); err != nil {
				dief("bad configuration: %s", err)
			}

			logger, err := log.NewLogger(
				log.WithLogLevel(cfg.LogLevel),
				log.WithOutputFile(cfg.LogOutput),
			)
			if err != nil {
				dief("failed to initialize logging: %s", err)
			}
			log.DefaultLogger = logger
			log.Info("apisix ingress controller started")

			data, err := json.MarshalIndent(cfg, "", "\t")
			if err != nil {
				dief("failed to show configuration: %s", string(data))
			}
			log.Info("use configuration\n", string(data))

			stop := make(chan struct{})
			ingress, err := controller.NewController(cfg)
			if err != nil {
				dief("failed to create ingress controller: %s", err)
			}
			go func() {
				if err := ingress.Run(stop); err != nil {
					dief("failed to launch ingress controller: %s", err)
				}
			}()

			waitForSignal(stop)
			log.Info("apisix ingress controller exited")
		},
	}

	cmd.PersistentFlags().StringVar(&configPath, "config-path", "", "configuration file path for apisix-ingress-controller")
	cmd.PersistentFlags().StringVar(&cfg.LogLevel, "log-level", "info", "error log level")
	cmd.PersistentFlags().StringVar(&cfg.LogOutput, "log-output", "stderr", "error log output file")
	cmd.PersistentFlags().StringVar(&cfg.HTTPListen, "http-listen", ":8080", "the HTTP Server listen address")
	cmd.PersistentFlags().BoolVar(&cfg.EnableProfiling, "enable-profiling", true, "enable profiling via web interface host:port/debug/pprof")
	cmd.PersistentFlags().StringVar(&cfg.Kubernetes.Kubeconfig, "kubeconfig", "", "Kubernetes configuration file (by default in-cluster configuration will be used)")
	cmd.PersistentFlags().DurationVar(&cfg.Kubernetes.ResyncInterval.Duration, "resync-interval", time.Minute, "the controller resync (with Kubernetes) interval, the minimum resync interval is 30s")
	cmd.PersistentFlags().StringSliceVar(&cfg.Kubernetes.AppNamespaces, "app-namespace", []string{config.NamespaceAll}, "namespaces that controller will watch for resources")
	cmd.PersistentFlags().StringVar(&cfg.Kubernetes.ElectionID, "election-id", config.IngressAPISIXLeader, "election id used for compaign the controller leader")
	cmd.PersistentFlags().StringVar(&cfg.APISIX.BaseURL, "apisix-base-url", "", "the base URL for APISIX admin api / manager api")
	cmd.PersistentFlags().StringVar(&cfg.APISIX.AdminKey, "apisix-admin-key", "", "admin key used for the authorization of APISIX admin api / manager api")

	return cmd
}
