// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package root

import (
	"fmt"
	"os"

	// Import all Kubernetes client auth plugins (e.g. Azure, GCP, OIDC, etc.)
	// to ensure that exec-entrypoint and run can make use of them.

	"go.uber.org/zap/zapcore"
	"gopkg.in/yaml.v2"
	_ "k8s.io/client-go/plugin/pkg/client/auth"

	"github.com/go-logr/zapr"
	"github.com/spf13/cobra"
	"go.uber.org/zap"
	ctrl "sigs.k8s.io/controller-runtime"

	// +kubebuilder:scaffold:imports

	"github.com/apache/apisix-ingress-controller/internal/controller/config"
	"github.com/apache/apisix-ingress-controller/internal/manager"
	"github.com/apache/apisix-ingress-controller/internal/version"
	"github.com/api7/gopkg/pkg/log"
)

type GatewayConfigsFlag struct {
	GatewayConfigs []*config.GatewayConfig
}

func (f *GatewayConfigsFlag) String() string {
	data, _ := yaml.Marshal(f.GatewayConfigs)
	return string(data)
}

func (f *GatewayConfigsFlag) Set(value string) error {
	var gatewayConfigs []*config.GatewayConfig
	if err := yaml.Unmarshal([]byte(value), &gatewayConfigs); err != nil {
		return err
	}
	f.GatewayConfigs = gatewayConfigs
	return nil
}

func (f *GatewayConfigsFlag) Type() string {
	return "gateway_configs"
}

func NewRootCmd() *cobra.Command {
	root := newAPISIXIngressController()
	root.AddCommand(newVersionCmd())
	return root
}

func newVersionCmd() *cobra.Command {
	var long bool
	cmd := &cobra.Command{
		Use:   "version",
		Short: "version for apisix-ingress-controller",
		Run: func(cmd *cobra.Command, _ []string) {
			if long {
				fmt.Print(version.Long())
			} else {
				fmt.Printf("apisix-ingress-controller version %s\n", version.Short())
			}
		},
	}
	cmd.PersistentFlags().BoolVar(&long, "long", false, "show long mode version information")

	return cmd

}

func newAPISIXIngressController() *cobra.Command {
	cfg := config.ControllerConfig
	var configPath string

	cmd := &cobra.Command{
		Use:     "apisix-ingress-controller [command]",
		Long:    "Yet another Ingress controller for Kubernetes using APISIX Gateway as the high performance reverse proxy.",
		Version: version.Short(),
		RunE: func(cmd *cobra.Command, args []string) error {
			if configPath != "" {
				c, err := config.NewConfigFromFile(configPath)
				if err != nil {
					return err
				}
				cfg = c
				config.SetControllerConfig(c)
			}

			if err := cfg.Validate(); err != nil {
				return err
			}

			logLevel, err := zapcore.ParseLevel(cfg.LogLevel)
			if err != nil {
				return err
			}

			l, err := log.NewLogger(
				log.WithOutputFile("stderr"),
				log.WithLogLevel(cfg.LogLevel),
				log.WithSkipFrames(3),
			)
			if err != nil {
				return err
			}
			log.DefaultLogger = l

			// controllers log
			core := zapcore.NewCore(
				zapcore.NewConsoleEncoder(zap.NewDevelopmentEncoderConfig()),
				zapcore.AddSync(zapcore.Lock(os.Stderr)),
				logLevel,
			)
			logger := zapr.NewLogger(zap.New(core, zap.AddCaller()))

			logger.Info("controller start configuration", "config", cfg)
			ctrl.SetLogger(logger.WithName("controller-runtime"))

			ctx := ctrl.LoggerInto(cmd.Context(), logger)
			return manager.Run(ctx, logger)
		},
	}

	cmd.Flags().StringVarP(&configPath,
		"config-path",
		"c",
		"",
		"configuration file path for apisix-ingress-controller",
	)
	cmd.Flags().StringVar(&cfg.MetricsAddr, "metrics-bind-address", "0", "The address the metrics endpoint binds to. "+
		"Use :8443 for HTTPS or :8080 for HTTP, or leave as 0 to disable the metrics service.")
	cmd.Flags().StringVar(&cfg.ProbeAddr, "health-probe-bind-address", ":8081", "The address the probe endpoint binds to.")
	cmd.Flags().StringVar(&cfg.LogLevel, "log-level", config.DefaultLogLevel, "The log level for apisix-ingress-controller")
	cmd.Flags().StringVar(&cfg.ControllerName,
		"controller-name",
		config.DefaultControllerName,
		"The name of the controller",
	)

	return cmd
}
