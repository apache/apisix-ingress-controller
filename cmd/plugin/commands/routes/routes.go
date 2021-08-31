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
	"github.com/apache/apisix-ingress-controller/cmd/plugin/common"
	"github.com/apache/apisix-ingress-controller/cmd/plugin/kubectl"
	"github.com/apache/apisix-ingress-controller/pkg/log"
	jsoniter "github.com/json-iterator/go"
	"github.com/spf13/cobra"
	"k8s.io/cli-runtime/pkg/genericclioptions"
)

type RouteStruct struct {
	ID         string
	URI        string
	Host       string
	Name       string
	UpstreamId string
	Status     int
	UpdateTime int64
	CreateTime int64
	Labels     map[string]string
}

func CreateCommand(flags *genericclioptions.ConfigFlags) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "routes",
		Short: "Show the apisix routes",
		Run: func(cmd *cobra.Command, args []string) {
			getRoutes(flags)
			return
		},
	}

	return cmd
}

func getRoutes(flags *genericclioptions.ConfigFlags) {
	pconf := common.NewPluginConfig(flags)
	log.Infof("namespaces: %v \n", pconf.NameSpace)
	pconf.GetConfigMapsData()

	svcName, _ := pconf.GetApisixSvcName()
	pid := kubectl.OpenPortForward(pconf.Ctx, pconf.NameSpace, svcName)
	for {
		if common.CheckPort() {
			var reqApisix common.RequestConf
			reqApisix.AdminKey = pconf.Cfg.APISIX.AdminKey
			reqApisix.URL = "http://127.0.0.1:9180/apisix/admin/"
			s, err := reqApisix.Get("routes")
			if err != nil {
				return
			}
			//fmt.Printf("Requests data => %v\n", s)
			createPrintData(s)
			break
		}
	}
	kubectl.ClosePortForward(pid)
}

func createPrintData(data []byte) {
	var printData [][]string
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
	printData = append(printData, header)
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
		printData = append(printData, iterms)
	}
	common.Show(printData, 2)
}
