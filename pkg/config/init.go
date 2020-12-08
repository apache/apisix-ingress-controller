package config

import (
	"encoding/json"
	clientSet "github.com/gxthrj/apisix-ingress-types/pkg/client/clientset/versioned"
	"io/ioutil"
	"k8s.io/client-go/informers"
	coreinformers "k8s.io/client-go/informers/core/v1"
	"k8s.io/client-go/kubernetes"
	restclient "k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	"os"
	"runtime"
)

var (
	_hostname    string
	config       *restclient.Config

	// Deprecate: will be removed in the near future without notifications.
	SyslogServer string
)

func init() {
	hostname, err := os.Hostname()
	if err != nil {
		panic(err)
	}
	_hostname = hostname
}

var (
	ENV               string
	basePath          string
	ADMIN_URL         = os.Getenv("APISIX_ADMIN_INTERNAL")
	HOSTNAME          = os.Getenv("HOSTNAME")
	LOCAL_ADMIN_URL   = ""
	podInformer       coreinformers.PodInformer
	svcInformer       coreinformers.ServiceInformer
	nsInformer        coreinformers.NamespaceInformer
	EndpointsInformer coreinformers.EndpointsInformer
	IsLeader          = false
	//etcdClient client.Client
	kubeClient                kubernetes.Interface
	CoreSharedInformerFactory informers.SharedInformerFactory
)

const PROD = "prod"
const HBPROD = "hb-prod"
const BETA = "beta"
const DEV = "dev"
const TEST = "test"
const LOCAL = "local"
const AispeechUpstreamKey = "/apisix/customer/upstream/map"

// Config contains necessary config items for running apisix-ingress-controller.
type Config struct {
	Hostname string `json:"-"`

	SyslogServer string       `json:"syslog_server"`
	Kubeconfig   string       `json:"kubeconfig"`
	Etcd         EtcdConfig   `json:"etcd"`
	APISIX       APISIXConfig `json:"apisix"`
}

// EtcdConfig contains config items about etcd.
type EtcdConfig struct {
	Endpoints []string `json:"endpoints"`
}

// APISIXConfig contains config items about apisix.
type APISIXConfig struct {
	BaseURL string `json:"base_url"`
}

// NewDefaultConfig creates a Config object filled by default value.
func NewDefaultConfig() *Config {
	return &Config{
		Hostname: _hostname,
	}
}

// NewConfigFromFiles creates a Config object and fill it by values in configuration file.
func NewConfigFromFile(configPath string) (*Config, error) {
	data, err := ioutil.ReadFile(configPath)
	if err != nil {
		return nil, err
	}
	conf := NewDefaultConfig()
	if err := json.Unmarshal(data, conf); err != nil {
		return nil, err
	}
	return conf, nil
}

func setEnvironment() {
	if env := os.Getenv("ENV"); env == "" {
		ENV = LOCAL
	} else {
		ENV = env
	}
	_, basePath, _, _ = runtime.Caller(1)
}

//func GetEtcdAPI() client.KeysAPI{
//	return client.NewKeysAPI(etcdClient)
//}


func GetURL() string{
	if ADMIN_URL != "" {
		return ADMIN_URL
	} else {
		return "http://172.16.20.90:30116/apisix/admin"
	}
}

func GetPodInformer() coreinformers.PodInformer{
	return podInformer
}

func GetSvcInformer() coreinformers.ServiceInformer{
	return svcInformer
}

func GetNsInformer() coreinformers.NamespaceInformer{
	return nsInformer
}

func GetKubeClient() kubernetes.Interface{
	return kubeClient
}

func InitKubeClient() (kubernetes.Interface, error) {
	//var err error
	//if ENV == LOCAL {
	//	clientConfig, err := clientcmd.LoadFromFile(K8sAuth.file)
	//	ExceptNilErr(err)
	//
	//	config, err = clientcmd.NewDefaultClientConfig(*clientConfig, &clientcmd.ConfigOverrides{}).ClientConfig()
	//	ExceptNilErr(err)
	//} else {
	//	config, err = restclient.InClusterConfig()
	//	ExceptNilErr(err)
	//}

	k8sClient, err := kubernetes.NewForConfig(config)
	if err != nil {
		return nil, err
	}
	return k8sClient, nil
}

func InitApisixClient() clientSet.Interface {
	apisixRouteClientset, err := clientSet.NewForConfig(config)
	ExceptNilErr(err)
	return apisixRouteClientset
}

// Deprecate: will be removed in the near future without notification.
func SetSyslogServer(srv string) {
	SyslogServer = srv
}

// InitInformer initializes the Kubernetes API objects informers.
// Deprecate: will be refactored in the near future without notifications.
func InitInformer(c *Config) error {
	// 生成一个k8s client
	//var config *restclient.Config
	var err error
	if c.Kubeconfig != "" {
		clientConfig, err := clientcmd.LoadFromFile(c.Kubeconfig)
		if err != nil {
			return err
		}

		config, err = clientcmd.NewDefaultClientConfig(*clientConfig, &clientcmd.ConfigOverrides{}).ClientConfig()
		if err != nil {
			return err
		}
	} else {
		config, err = restclient.InClusterConfig()
		if err != nil {
			return err
		}
	}

	//k8sClient, err := kubernetes.NewForConfig(config)
	kubeClient, err = InitKubeClient()
	if err != nil {
		return err
	}

	// 创建一个informerFactory
	//sharedInformerFactory := informers.NewSharedInformerFactory(k8sClient, 0)
	// 创建一个informerFactory
	CoreSharedInformerFactory = informers.NewSharedInformerFactory(kubeClient, 0)

	// 创建 informers
	podInformer = CoreSharedInformerFactory.Core().V1().Pods()
	svcInformer = CoreSharedInformerFactory.Core().V1().Services()
	nsInformer = CoreSharedInformerFactory.Core().V1().Namespaces()
	//return podInformer, svcInformer, nsInformer
	return nil
}

func ExceptNilErr(err error)  {
	if err != nil {
		panic(err)
	}
}

//func NewEtcdClient() client.Client {
//	cfg := client.Config{
//		Endpoints: EtcdConfig.Addresses,
//		Transport: client.DefaultTransport,
//	}
//	if c, err := client.New(cfg); err != nil {
//		panic(fmt.Sprintf("failed to initialize etcd watcher. %s", err.Error()))
//	} else {
//		return c
//	}
//}

// EtcdWatcher
//type EtcdWatcher struct {
//	client     client.Client
//	etcdKey 	string
//	ctx        context.Context
//	cancels    []context.CancelFunc
//}
//
//
//type BalancerRules struct {
//	RuleSpec *RuleSpec `json:"spec"`
//}
//
//type RuleSpec struct {
//	Ewma []string `json:"ewma"`
//	Sllb []Sllb `json:"sllb"`
//}
//
//type Sllb struct {
//	Name string `json:"name"`
//	Threshold int64 `json:"threshold"`
//	Open string `json:"open"`
//	MakeZero string `json:"makeZero"`
//}
//
//type BalancerLevel struct {
//	LevelSpec *LevelSpec `json:"spec"`
//}
//
//type LevelSpec struct {
//	Pod []string `json:"pod"`
//}

