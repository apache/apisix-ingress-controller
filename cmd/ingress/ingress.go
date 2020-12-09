package ingress

import (
	"flag"
	"net/http"
	"time"

	"github.com/golang/glog"
	api6Informers "github.com/gxthrj/apisix-ingress-types/pkg/client/informers/externalversions"
	"github.com/spf13/cobra"

	"github.com/api7/ingress-controller/conf"
	"github.com/api7/ingress-controller/log"
	"github.com/api7/ingress-controller/pkg"
	"github.com/api7/ingress-controller/pkg/ingress/controller"
)

// NewIngressCommand creates the ingress sub command for apisix-ingress-controller.
func NewIngressCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "ingress [flags]",
		Short: "launch the controller",
		Example: `Run apisix-ingress-controller from configuration file:

	apisix-ingress-controller ingress --config-path /path/to/config.json`,
		Run: func(cmd *cobra.Command, args []string) {
			flag.Parse()
			defer glog.Flush()
			var logger = log.GetLogger()
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

			router := pkg.Route()
			err := http.ListenAndServe(":8080", router)
			if err != nil {
				logger.Fatal("ListenAndServe: ", err)
			}
		},
	}

	// TODO: Uncomment these lines.
	// cmd.PersistentFlags().StringVar(&configPath, "config-path", "", "file path for the configuration of apisix-ingress-controller")
	// cmd.PersistentFlags().StringVar(&conf.Kubeconfig, "kubeconfig", "", "Kubernetes configuration file (by default in-cluster configuration will be used)")
	// cmd.PersistentFlags().StringSliceVar(&conf.Etcd.Endpoints, "etcd-endpoints", nil, "etcd endpoints")
	// cmd.PersistentFlags().StringVar(&conf.APISIX.BaseURL, "apisix-base-url", "", "the base URL for APISIX instance")
	// cmd.PersistentFlags().StringVar(&conf.SyslogServer, "syslog-server", "", "syslog server address")

	return cmd
}
