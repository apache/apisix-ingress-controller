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
package routes

import (
	"strings"

	"github.com/apache/apisix-ingress-controller/cmd/plugin/common"
	"github.com/apache/apisix-ingress-controller/cmd/plugin/kubectl"
	"github.com/apache/apisix-ingress-controller/pkg/log"
	jsoniter "github.com/json-iterator/go"
	"github.com/spf13/cobra"
	"k8s.io/cli-runtime/pkg/genericclioptions"
)

func CreateCommand(flags *genericclioptions.ConfigFlags) *cobra.Command {
	var routeId string
	cmd := &cobra.Command{
		Use:   "routes",
		Short: "Show the apisix routes",
		Run: func(cmd *cobra.Command, args []string) {
			getRoutes(flags, routeId)
			return
		},
	}
	cmd.Flags().StringVar(&routeId, "route-id", "", "apisix route id")
	return cmd
}

// getRoutes get apisix route informations.
func getRoutes(flags *genericclioptions.ConfigFlags, routeId string) {
	pconf := common.NewPluginConfig(flags)
	log.Infof("namespaces: %v \n", pconf.NameSpace)
	pconf.GetConfigMapsData()

	svcName, _ := pconf.GetApisixSvcName()
	pid := kubectl.OpenPortForward(pconf.Ctx, pconf.NameSpace, svcName)
	header := []string{
		"ID",
		"Name",
		"Host",
		"URI",
		"Status",
		"UpstreamId",
		"CreateTime",
		"UpdateTime",
	}

	for {
		if common.CheckPort() {
			log.Info("the k8s port-forward is ready")
			break
		}
		log.Error("the k8s port-forward is not ready")
	}
	var reqApisix common.RequestConf
	reqApisix.AdminKey = pconf.Cfg.APISIX.AdminKey
	reqApisix.URL = "http://127.0.0.1:9180/apisix/admin"
	routes := []string{
		"routes",
		routeId,
	}
	s, err := reqApisix.Get(strings.Join(routes, "/"))
	if err != nil {
		return
	}
	//fmt.Printf("Requests data => %v\n", s)
	if routeId == "" {
		data := fullRoute(s, header)
		common.Show(data, 2)
	} else {
		onlyOneRouteID := specificRoute(s, header)
		common.Show(onlyOneRouteID, 2)
	}
	kubectl.ClosePortForward(pid)
}

// fullRoute Make all route's informations.
func fullRoute(data []byte, header []string) [][]string {
	var printDatas [][]string
	printDatas = append(printDatas, header)
	size := jsoniter.Get(data, "node", "nodes").Size()
	for i := 0; i < size; i++ {
		tmp := jsoniter.Get(data, "node", "nodes", i).Get("value")
		iterms := []string{
			tmp.Get("id").ToString(),
			tmp.Get("name").ToString(),
			tmp.Get("host").ToString(),
			tmp.Get("uri").ToString(),
			tmp.Get("status").ToString(),
			tmp.Get("upstream_id").ToString(),
			tmp.Get("create_time").ToString(),
			tmp.Get("update_time").ToString(),
		}
		printDatas = append(printDatas, iterms)
	}
	return printDatas
}

// specificRoute Make specific route information's.
func specificRoute(data []byte, header []string) [][]string {
	var printData [][]string
	printData = append(printData, header)
	tmp := jsoniter.Get(data, "node", "value")
	iterms := []string{
		tmp.Get("id").ToString(),
		tmp.Get("name").ToString(),
		tmp.Get("host").ToString(),
		tmp.Get("uri").ToString(),
		tmp.Get("status").ToString(),
		tmp.Get("upstream_id").ToString(),
		tmp.Get("create_time").ToString(),
		tmp.Get("update_time").ToString(),
	}
	printData = append(printData, iterms)
	return printData
}
