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

package upstreams

import (
	"strconv"
	"strings"

	"github.com/apache/apisix-ingress-controller/cmd/plugin/common"
	"github.com/apache/apisix-ingress-controller/cmd/plugin/kubectl"
	"github.com/apache/apisix-ingress-controller/pkg/log"
	jsoniter "github.com/json-iterator/go"
	"github.com/spf13/cobra"
	"k8s.io/cli-runtime/pkg/genericclioptions"
)

func CreateCommand(flags *genericclioptions.ConfigFlags) *cobra.Command {
	var upstreamId string
	cmd := &cobra.Command{
		Use:   "upstreams",
		Short: "Show the apisix routes",
		Run: func(cmd *cobra.Command, args []string) {
			getUpstreams(flags, upstreamId)
			return
		},
	}
	cmd.Flags().StringVar(&upstreamId, "upstream-id", "", "apisix routes id")
	return cmd
}

// getUpstreams get apisix upstream informations.
func getUpstreams(flags *genericclioptions.ConfigFlags, upstreamId string) {
	pconf := common.NewPluginConfig(flags)

	log.Infof("namespaces: %v \n", pconf.NameSpace)
	pconf.GetConfigMapsData()

	svcName, _ := pconf.GetApisixSvcName()

	pid := kubectl.OpenPortForward(pconf.Ctx, pconf.NameSpace, svcName)
	header := []string{
		"ID",
		"Name",
		"NodeNumbers",
		"Scheme",
		"Type",
		"HashOn",
		"PassHost",
	}

	// wait port-forward ready
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
	upstreams := []string{
		"upstreams",
		upstreamId,
	}
	s, err := reqApisix.Get(strings.Join(upstreams, "/"))
	if err != nil {
		log.Error(err)
		return
	}
	log.Info(string(s))
	var printDatas [][]string
	if upstreamId == "" {
		printDatas = fullUpstream(s, header)
	} else {
		printDatas = specificUpstream(pconf, s)
	}
	if len(printDatas) != 0 {
		common.Show(printDatas, 2)
	}

	kubectl.ClosePortForward(pid)
}

// fullUpstream Make the result data  for all upstream information.
func fullUpstream(data []byte, header []string) [][]string {
	var printDatas [][]string
	printDatas = append(printDatas, header)
	size := jsoniter.Get(data, "node", "nodes").Size()
	for i := 0; i < size; i++ {
		tmp := jsoniter.Get(data, "node", "nodes", i).Get("value")
		nodes := strconv.Itoa(tmp.Get("nodes").Size())
		iterms := []string{
			tmp.Get("id").ToString(),
			tmp.Get("name").ToString(),
			nodes,
			tmp.Get("scheme").ToString(),
			tmp.Get("type").ToString(),
			tmp.Get("hash_on").ToString(),
			tmp.Get("pass_host").ToString(),
			//tmp.Get("update_time").ToString(),
		}
		printDatas = append(printDatas, iterms)
	}
	return printDatas
}

// specificUpstream Only make the upstream-id information
func specificUpstream(pc *common.PluginConf, data []byte) [][]string {
	var printDatas [][]string
	specificHeader := []string{
		"PodName",
		"Host",
		"Port",
		"Weight",
		"Priority",
		"Node",
	}
	printDatas = append(printDatas, specificHeader)
	tmp := jsoniter.Get(data, "node", "value")
	appns := strings.Split(tmp.Get("name").ToString(), "_")
	size := tmp.Get("nodes").Size()
	for i := 0; i < size; i++ {
		podIp := tmp.Get("nodes", i).Get("host").ToString()
		appPodInfo := pc.GetPodInfo(appns[0], podIp)
		iterms := []string{
			appPodInfo.PodName,
			podIp,
			tmp.Get("nodes", i).Get("port").ToString(),
			tmp.Get("nodes", i).Get("weight").ToString(),
			tmp.Get("nodes", i).Get("priority").ToString(),
			appPodInfo.NodeName,
		}

		printDatas = append(printDatas, iterms)
	}
	return printDatas
}

func describeUpstream() {}
