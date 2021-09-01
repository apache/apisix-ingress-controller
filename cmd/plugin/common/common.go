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
package common

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"net"
	"os"
	"strings"
	"text/tabwriter"

	"github.com/apache/apisix-ingress-controller/cmd/plugin/pluginutil"
	"github.com/apache/apisix-ingress-controller/pkg/config"
	"github.com/apache/apisix-ingress-controller/pkg/kube"
	"github.com/apache/apisix-ingress-controller/pkg/log"
	"github.com/spf13/viper"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/cli-runtime/pkg/genericclioptions"
)

type PluginConf struct {
	Cfg       config.Config
	NameSpace string
	Ctx       context.Context
	Kubecli   *kube.KubeClient
}

func NewPluginConfig(flags *genericclioptions.ConfigFlags) *PluginConf {
	var pConf PluginConf
	pConf.NameSpace = pluginutil.GetNamespaces(flags)
	pConf.Cfg.Kubernetes.Kubeconfig = pluginutil.GetKubeconfigFile(flags)
	pConf.Ctx = context.Background()
	cli, err := kube.NewKubeClient(&pConf.Cfg)
	if err != nil {
		return &pConf
	}
	pConf.Kubecli = cli
	log.Infof("namespace => %v,kubeconf:%v \n", pConf.NameSpace, *flags.KubeConfig)
	return &pConf
}

func (pc *PluginConf) GetConfigMapsData() {
	var apisixIngressControllerConfig string
	configmaps, err := pc.Kubecli.Client.CoreV1().ConfigMaps(pc.NameSpace).List(pc.Ctx, metav1.ListOptions{})
	if err != nil {
		log.Errorf("Get the %v configmaps %v", pc.NameSpace, err)
		return
	}
	for _, cm := range configmaps.Items {
		label := cm.GetLabels()
		if label["app.kubernetes.io/name"] == "apisix-ingress-controller" {
			log.Infof("ApiSx Ingress Controller Config data: \n %v\n", cm.Data)
			apisixIngressControllerConfig = cm.Data["config.yaml"]
		}
	}
	viper.SetConfigType("yaml")
	viper.ReadConfig(bytes.NewBuffer([]byte(apisixIngressControllerConfig)))
	pc.Cfg.APISIX.AdminKey = viper.Get("apisix.admin_key").(string)
	log.Infof("The apisix admin key is => %v\n", pc.Cfg.APISIX.AdminKey)
}

func (pc *PluginConf) GetApisixSvcName() (string, error) {
	var svcName string
	svcs, err := pc.Kubecli.Client.CoreV1().Services(pc.NameSpace).List(pc.Ctx, metav1.ListOptions{})
	if err != nil {
		errs := fmt.Sprintf("The %s not fond apisix-admin service", pc.NameSpace)
		return svcName, errors.New(errs)
	}
	for _, svc := range svcs.Items {
		label := svc.GetLabels()
		if label["app.kubernetes.io/name"] == "apisix" {
			svcName = svc.GetName()
			break
		}
	}
	return svcName, nil
}

func CheckPort() bool {
	_, err := net.Dial("tcp", "127.0.0.1:9180")
	if err == nil {
		return true
	}
	return false
}

func Show(data [][]string, adjust int) {
	w := new(tabwriter.Writer)
	w.Init(os.Stdout, 0, 8, adjust, '\t', 0)
	for _, v := range data {
		fmt.Fprintf(w, "%v\n", strings.Join(v, "\t"))
	}
	w.Flush()
}
