// Licensed to the Apache Software Foundation (ASF) under one
// or more contributor license agreements.  See the NOTICE file
// distributed with this work for additional information
// regarding copyright ownership.  The ASF licenses this file
// to you under the Apache License, Version 2.0 (the
// "License"); you may not use this file except in compliance
// with the License.  You may obtain a copy of the License at
//
//   http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing,
// software distributed under the License is distributed on an
// "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY
// KIND, either express or implied.  See the License for the
// specific language governing permissions and limitations
// under the License.

package framework

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"time"

	"github.com/api7/gopkg/pkg/log"
	"github.com/gruntwork-io/terratest/modules/k8s"
	. "github.com/onsi/ginkgo/v2" //nolint:staticcheck
	. "github.com/onsi/gomega"    //nolint:staticcheck
	"github.com/stretchr/testify/assert"
	"gopkg.in/yaml.v3"
	"helm.sh/helm/v3/pkg/action"
	"helm.sh/helm/v3/pkg/chart/loader"
	"helm.sh/helm/v3/pkg/cli"
	"helm.sh/helm/v3/pkg/kube"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
)

var (
	API7EELicense string

	dashboardVersion string
)

func (f *Framework) BeforeSuite() {
	// init license and dashboard version
	API7EELicense = os.Getenv("API7_EE_LICENSE")
	if API7EELicense == "" {
		panic("env {API7_EE_LICENSE} is required")
	}

	dashboardVersion = os.Getenv("DASHBOARD_VERSION")
	if dashboardVersion == "" {
		dashboardVersion = "dev"
	}

	_ = k8s.DeleteNamespaceE(GinkgoT(), f.kubectlOpts, _namespace)

	Eventually(func() error {
		_, err := k8s.GetNamespaceE(GinkgoT(), f.kubectlOpts, _namespace)
		if k8serrors.IsNotFound(err) {
			return nil
		}
		return fmt.Errorf("namespace %s still exists", _namespace)
	}, "1m", "2s").Should(Succeed())

	k8s.CreateNamespace(GinkgoT(), f.kubectlOpts, _namespace)

	f.DeployComponents()

	time.Sleep(1 * time.Minute)
	err := f.newDashboardTunnel()
	f.Logf("Dashboard HTTP Tunnel:" + _dashboardHTTPTunnel.Endpoint())
	Expect(err).ShouldNot(HaveOccurred(), "creating dashboard tunnel")

	f.UploadLicense()

	f.setDpManagerEndpoints()
}

func (f *Framework) AfterSuite() {
	f.shutdownDashboardTunnel()
}

// DeployComponents deploy necessary components
func (f *Framework) DeployComponents() {
	f.deploy()
	f.initDashboard()
}

func (f *Framework) UploadLicense() {
	payload := map[string]any{"data": API7EELicense}
	payloadBytes, err := json.Marshal(payload)
	assert.Nil(f.GinkgoT, err)

	respExpect := f.DashboardHTTPClient().PUT("/api/license").
		WithBasicAuth("admin", "admin").
		WithHeader("Content-Type", "application/json").
		WithBytes(payloadBytes).
		Expect()

	body := respExpect.Body().Raw()
	f.Logf("request /api/license, response body: %s", body)

	respExpect.Status(200)
}

func (f *Framework) deploy() {
	debug := func(format string, v ...any) {
		log.Infof(format, v...)
	}

	kubeConfigPath := os.Getenv("KUBECONFIG")
	actionConfig := new(action.Configuration)

	err := actionConfig.Init(
		kube.GetConfig(kubeConfigPath, "", f.kubectlOpts.Namespace),
		f.kubectlOpts.Namespace,
		"memory",
		debug,
	)
	f.GomegaT.Expect(err).ShouldNot(HaveOccurred(), "init helm action config")

	install := action.NewInstall(actionConfig)
	install.Namespace = f.kubectlOpts.Namespace
	install.ReleaseName = "api7ee3"

	chartPath, err := install.LocateChart("api7/api7ee3", cli.New())
	f.GomegaT.Expect(err).ShouldNot(HaveOccurred(), "locate helm chart")

	chart, err := loader.Load(chartPath)
	f.GomegaT.Expect(err).ShouldNot(HaveOccurred(), "load helm chart")

	buf := bytes.NewBuffer(nil)
	_ = valuesTemplate.Execute(buf, map[string]any{
		"DB":  _db,
		"DSN": getDSN(),
		"Tag": dashboardVersion,
	})

	f.Logf("values: %s", buf.String())

	var v map[string]any
	err = yaml.Unmarshal(buf.Bytes(), &v)
	f.GomegaT.Expect(err).ShouldNot(HaveOccurred(), "unmarshal values")
	_, err = install.Run(chart, v)
	if err != nil {
		f.Logf("install dashboard failed, err: %v", err)
	}
	f.GomegaT.Expect(err).ShouldNot(HaveOccurred(), "install dashboard")

	err = f.ensureServiceWithTimeout("api7ee3-dashboard", _namespace, 1, 300)
	f.GomegaT.Expect(err).ShouldNot(HaveOccurred(), "ensuring dashboard service")

	err = f.ensureService("api7-postgresql", _namespace, 1)
	f.GomegaT.Expect(err).ShouldNot(HaveOccurred(), "ensuring postgres service")
}

func (f *Framework) initDashboard() {
	f.deletePods("app.kubernetes.io/name=api7ee3")
	time.Sleep(5 * time.Second)
}

var (
	_dashboardHTTPTunnel  *k8s.Tunnel
	_dashboardHTTPSTunnel *k8s.Tunnel
)

func (f *Framework) newDashboardTunnel() error {
	var (
		httpNodePort  int
		httpsNodePort int
		httpPort      int
		httpsPort     int
	)

	service := k8s.GetService(f.GinkgoT, f.kubectlOpts, "api7ee3-dashboard")

	for _, port := range service.Spec.Ports {
		switch port.Name {
		case "http":
			httpNodePort = int(port.NodePort)
			httpPort = int(port.Port)
		case "https":
			httpsNodePort = int(port.NodePort)
			httpsPort = int(port.Port)
		}
	}

	_dashboardHTTPTunnel = k8s.NewTunnel(f.kubectlOpts, k8s.ResourceTypeService, "api7ee3-dashboard",
		httpNodePort, httpPort)
	_dashboardHTTPSTunnel = k8s.NewTunnel(f.kubectlOpts, k8s.ResourceTypeService, "api7ee3-dashboard",
		httpsNodePort, httpsPort)

	if err := _dashboardHTTPTunnel.ForwardPortE(f.GinkgoT); err != nil {
		return err
	}
	if err := _dashboardHTTPSTunnel.ForwardPortE(f.GinkgoT); err != nil {
		return err
	}

	return nil
}

func (f *Framework) shutdownDashboardTunnel() {
	if _dashboardHTTPTunnel != nil {
		_dashboardHTTPTunnel.Close()
	}
	if _dashboardHTTPSTunnel != nil {
		_dashboardHTTPSTunnel.Close()
	}
}
