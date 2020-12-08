package ingress

import (
	"context"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/golang/glog"
	"github.com/spf13/cobra"
	api6Informers "github.com/gxthrj/apisix-ingress-types/pkg/client/informers/externalversions"

	"github.com/api7/ingress-controller/pkg"
	"github.com/api7/ingress-controller/pkg/config"
	"github.com/api7/ingress-controller/log"
	"github.com/api7/ingress-controller/pkg/ingress/controller"
	"github.com/api7/ingress-controller/pkg/utils"
)

// NewIngressCommand creates the ingress sub command for apisix-ingress-controller.
func NewIngressCommand() *cobra.Command {
	var (
		configPath string
	)
	conf := config.NewDefaultConfig()
	cmd := &cobra.Command{
		Use:     "ingress [flags]",
		Short:   "launch the controller",
		Example: `Run apisix-ingress-controller from configuration file:

	apisix-ingress-controller ingress --config-path /path/to/config.json`,
		Run: func(cmd *cobra.Command, args []string) {
			logger := log.GetLogger()
			if configPath != "" {
				fileConf, err := config.NewConfigFromFile(configPath)
				if err != nil {
					utils.Dief("failed to parse configuration file [%s]: %s", configPath, err.Error())
					os.Exit(1)
				}
				conf = fileConf
			}

			if err := config.InitInformer(conf); err != nil {
				utils.Dief("failed to initialize Kubernetes shared index informer: %s", err.Error())
				os.Exit(1)
			}

			defer glog.Flush()
			kubeClientSet := config.GetKubeClient()
			apisixClientset := config.InitApisixClient()
			sharedInformerFactory := api6Informers.NewSharedInformerFactory(apisixClientset, 0)
			stop := make(chan struct{})
			c := &controller.Api6Controller{
				KubeClientSet:             kubeClientSet,
				Api6ClientSet:             apisixClientset,
				SharedInformerFactory:     sharedInformerFactory,
				CoreSharedInformerFactory: config.CoreSharedInformerFactory,
				Stop:                      stop,
			}
			epInformer := c.CoreSharedInformerFactory.Core().V1().Endpoints()
			config.EndpointsInformer = epInformer

			// endpoint
			c.Endpoint()
			go c.CoreSharedInformerFactory.Start(stop)

			// ApisixRoute
			c.ApisixRoute()
			// ApisixUpstream
			c.ApisixUpstream()
			// ApisixService
			c.ApisixService()

			go func(){
				time.Sleep(time.Duration(10)*time.Second)
				c.SharedInformerFactory.Start(stop)
			}()

			router := pkg.Route()
			httpSrv := http.Server{
				Addr: ":8080",
				Handler: router,
			}

			go func() {
				sigCh := make(chan os.Signal, 1)
				signal.Notify(sigCh, syscall.SIGTERM, syscall.SIGINT)
				<-sigCh

				close(stop)

				cancelCtx, cancel := context.WithTimeout(context.TODO(), 5*time.Second)
				defer cancel()
				if err := httpSrv.Shutdown(cancelCtx); err != nil {
					logger.Fatalf("failed to shutdown http server: %s", err.Error())
				}
			}()

			if err := httpSrv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
				logger.Fatalf("ListenAndServe: %s", err)
			}
		},
	}

	cmd.PersistentFlags().StringVar(&configPath, "config-path", "", "file path for the configuration of apisix-ingress-controller")
	cmd.PersistentFlags().StringVar(&conf.Kubeconfig, "kubeconfig", "", "Kubernetes configuration file (by default in-cluster configuration will be used)")
	cmd.PersistentFlags().StringSliceVar(&conf.Etcd.Endpoints, "etcd-endpoints", nil, "etcd endpoints")
	cmd.PersistentFlags().StringVar(&conf.APISIX.BaseURL, "apisix-base-url", "", "the base URL for APISIX instance")
	cmd.PersistentFlags().StringVar(&conf.SyslogServer, "syslog-server", "", "syslog server address")

	return cmd
}
