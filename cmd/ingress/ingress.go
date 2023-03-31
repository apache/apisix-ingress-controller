// Licensed to the Apache Software Foundation (ASF) under one or more
// contributor license agreements.  See the NOTICE file distributed with
// this work for additional information regarding copyright ownership.
// The ASF licenses this file to You under the Apache License, Version 2.0
// (the "License"); you may not use this file except in compliance with
// the License.  You may obtain a copy of the License at
//
//	http://www.apache.org/licenses/LICENSE-2.0
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
	"sync"
	"syscall"
	"time"

	"github.com/spf13/cobra"
	"go.uber.org/zap/zapcore"
	"gopkg.in/natefinch/lumberjack.v2"

	"github.com/apache/apisix-ingress-controller/pkg/config"
	"github.com/apache/apisix-ingress-controller/pkg/log"
	controller "github.com/apache/apisix-ingress-controller/pkg/providers"
	"github.com/apache/apisix-ingress-controller/pkg/version"
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

    apisix-ingress-controller ingress --default-apisix-cluster-base-url http://apisix-service:9180/apisix/admin --kubeconfig /path/to/kubeconfig

For Kubernetes cluster version older than v1.19.0, you should always set the --ingress-version option to networking/v1beta1:

    apisix-ingress-controller ingress \
      --default-apisix-cluster-base-url http://apisix-service:9180/apisix/admin \
      --kubeconfig /path/to/kubeconfig \
      --ingress-version networking/v1beta1

If your Kubernetes cluster version is prior to v1.14+, only ingress.extensions/v1beta1 can be used.

    apisix-ingress-controller ingress \
      --default-apisix-cluster-base-url http://apisix-service:9180/apisix/admin \
      --kubeconfig /path/to/kubeconfig \
      --ingress-version extensions/v1beta1

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

			var ws zapcore.WriteSyncer

			options := []log.Option{log.WithLogLevel(cfg.LogLevel), log.WithOutputFile(cfg.LogOutput)}

			if cfg.LogRotateOutputPath != "" {
				ws = zapcore.AddSync(&lumberjack.Logger{
					Filename:   cfg.LogRotateOutputPath,
					MaxSize:    cfg.LogRotationMaxSize,
					MaxBackups: cfg.LogRotationMaxBackups,
					MaxAge:     cfg.LogRotationMaxAge,
				})

				options = append(options, log.WithWriteSyncer(ws))
			}

			logger, err := log.NewLogger(options...)

			if err != nil {
				dief("failed to initialize logging: %s", err)
			}
			log.DefaultLogger = logger
			log.Info("init apisix ingress controller")

			log.Info("version:\n", version.Long())

			// We should make sure that the cfg that's logged out is sanitized.
			cfgCopy := new(config.Config)
			*cfgCopy = *cfg
			cfgCopy.APISIX.DefaultClusterAdminKey = "******"
			data, err := json.MarshalIndent(cfgCopy, "", "  ")
			if err != nil {
				dief("failed to marshal configuration: %s", err)
			}
			log.Info("use configuration\n", string(data))

			stop := make(chan struct{})
			ingress, err := controller.NewController(cfg)
			if err != nil {
				dief("failed to create ingress controller: %s", err)
			}
			wg := sync.WaitGroup{}
			wg.Add(1)
			go func() {
				defer wg.Done()

				log.Info("start ingress controller")

				if err := ingress.Run(stop); err != nil {
					dief("failed to run ingress controller: %s", err)
				}
			}()

			waitForSignal(stop)
			wg.Wait()
			log.Info("apisix ingress controller exited")
		},
	}

	cmd.PersistentFlags().StringVar(&configPath, "config-path", "", "configuration file path for apisix-ingress-controller")
	cmd.PersistentFlags().StringVar(&cfg.LogLevel, "log-level", "info", "error log level")
	cmd.PersistentFlags().StringVar(&cfg.LogOutput, "log-output", "stderr", "error log output file")
	cmd.PersistentFlags().StringVar(&cfg.LogRotateOutputPath, "log-rotate-output-path", "", "rotate log output path")
	cmd.PersistentFlags().IntVar(&cfg.LogRotationMaxSize, "log-rotate-max-size", 100, "rotate log max size")
	cmd.PersistentFlags().IntVar(&cfg.LogRotationMaxAge, "log-rotate-max-age", 0, "old rotate log max age to retain")
	cmd.PersistentFlags().IntVar(&cfg.LogRotationMaxBackups, "log-rotate-max-backups", 0, "old rotate log max numbers to retain")
	cmd.PersistentFlags().StringVar(&cfg.HTTPListen, "http-listen", ":8080", "the HTTP Server listen address")
	cmd.PersistentFlags().StringVar(&cfg.HTTPSListen, "https-listen", ":8443", "the HTTPS Server listen address")
	cmd.PersistentFlags().StringVar(&cfg.IngressPublishService, "ingress-publish-service", "",
		`the controller will use the Endpoint of this Service to update the status information of the Ingress resource.
The format is "namespace/svc-name" to solve the situation that the data plane and the controller are not deployed in the same namespace.`)
	cmd.PersistentFlags().StringSliceVar(&cfg.IngressStatusAddress, "ingress-status-address", []string{},
		`when there is no available information on the Service used for publishing on the data plane,
the static address provided here will be used to update the status information of Ingress.
When ingress-publish-service is specified at the same time, ingress-status-address is preferred.
For example, no available LB exists in the bare metal environment.`)
	cmd.PersistentFlags().BoolVar(&cfg.EnableProfiling, "enable-profiling", true, "enable profiling via web interface host:port/debug/pprof")
	cmd.PersistentFlags().StringVar(&cfg.Kubernetes.Kubeconfig, "kubeconfig", "", "Kubernetes configuration file (by default in-cluster configuration will be used)")
	cmd.PersistentFlags().DurationVar(&cfg.Kubernetes.ResyncInterval.Duration, "resync-interval", time.Minute, "the controller resync (with Kubernetes) interval, the minimum resync interval is 30s")
	cmd.PersistentFlags().StringSliceVar(&cfg.Kubernetes.NamespaceSelector, "namespace-selector", []string{""}, "labels that controller used to select namespaces which will watch for resources")
	cmd.PersistentFlags().StringVar(&cfg.Kubernetes.IngressClass, "ingress-class", config.IngressClass, "the class of an Ingress object is set using the field IngressClassName in Kubernetes clusters version v1.18.0 or higher or the annotation \"kubernetes.io/ingress.class\" (deprecated)")
	cmd.PersistentFlags().StringVar(&cfg.Kubernetes.ElectionID, "election-id", config.IngressAPISIXLeader, "election id used for campaign the controller leader")
	cmd.PersistentFlags().StringVar(&cfg.Kubernetes.IngressVersion, "ingress-version", config.IngressNetworkingV1, "the supported ingress api group version, can be \"networking/v1beta1\", \"networking/v1\" (for Kubernetes version v1.19.0 or higher) and \"extensions/v1beta1\"")
	cmd.PersistentFlags().StringVar(&cfg.Kubernetes.APIVersion, "api-version", config.DefaultAPIVersion, config.APIVersionDescribe)
	cmd.PersistentFlags().BoolVar(&cfg.Kubernetes.WatchEndpointSlices, "watch-endpointslices", false, "whether to watch endpointslices rather than endpoints")
	cmd.PersistentFlags().BoolVar(&cfg.Kubernetes.EnableGatewayAPI, "enable-gateway-api", false, "whether to enable support for Gateway API")
	cmd.PersistentFlags().StringVar(&cfg.APISIX.AdminAPIVersion, "apisix-admin-api-version", "v2", `the APISIX admin API version. can be "v2" or "v3". Default value is v2.`)
	cmd.PersistentFlags().StringVar(&cfg.APISIX.DefaultClusterBaseURL, "default-apisix-cluster-base-url", "", "the base URL of admin api / manager api for the default APISIX cluster")
	cmd.PersistentFlags().StringVar(&cfg.APISIX.DefaultClusterAdminKey, "default-apisix-cluster-admin-key", "", "admin key used for the authorization of admin api / manager api for the default APISIX cluster")
	cmd.PersistentFlags().StringVar(&cfg.APISIX.DefaultClusterName, "default-apisix-cluster-name", "default", "name of the default apisix cluster")
	cmd.PersistentFlags().DurationVar(&cfg.ApisixResourceSyncInterval.Duration, "apisix-resource-sync-interval", 300*time.Second, "interval between syncs in seconds. Default value is 300s.")
	cmd.PersistentFlags().StringVar(&cfg.PluginMetadataConfigMap, "plugin-metadata-cm", "plugin-metadata-config-map", "ConfigMap name of plugin metadata.")

	return cmd
}
