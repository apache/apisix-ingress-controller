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

package framework

import (
	"context"
	_ "embed"
	"encoding/base64"

	"github.com/gruntwork-io/terratest/modules/k8s"
	"github.com/gruntwork-io/terratest/modules/logger"
	. "github.com/onsi/ginkgo/v2" //nolint:staticcheck
	. "github.com/onsi/gomega"    //nolint:staticcheck
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

var (
	// TODO: set namespace from env
	_namespace = "api7-ee-e2e"
	_framework *Framework
)

type DataPlanePod struct {
	Selector string
	PodName  string
}

type DataPlaneContext struct {
	Context    context.Context
	CancelFunc context.CancelFunc
}

type Framework struct {
	Context context.Context
	GinkgoT GinkgoTInterface
	GomegaT *GomegaWithT

	Logger logger.TestLogger

	kubectlOpts *k8s.KubectlOptions
	clientset   *kubernetes.Clientset
	restConfig  *rest.Config
	K8sClient   client.Client
}

// NewFramework create a global framework with special settings.
func NewFramework() *Framework {
	f := &Framework{
		GinkgoT: GinkgoT(),
		GomegaT: NewWithT(GinkgoT(4)),
		Logger:  logger.Terratest,
	}

	// FIXME if we need some precise control on the context
	f.Context = context.TODO()

	f.kubectlOpts = k8s.NewKubectlOptions("", "", _namespace)
	restCfg, err := buildRestConfig("")
	f.GomegaT.Expect(err).ShouldNot(HaveOccurred(), "building API Server rest config")
	f.restConfig = restCfg

	clientset, err := kubernetes.NewForConfig(restCfg)
	f.GomegaT.Expect(err).ShouldNot(HaveOccurred(), "creating Kubernetes clientset")
	f.clientset = clientset

	k8sClient, err := client.New(restCfg, client.Options{})
	f.GomegaT.Expect(err).ShouldNot(HaveOccurred(), "creating controller-runtime client")
	f.K8sClient = k8sClient

	_framework = f

	return f
}

func GetFramework() *Framework {
	return _framework
}

func (f *Framework) Base64Encode(src string) string {
	return base64.StdEncoding.EncodeToString([]byte(src))
}

func (f *Framework) Logf(format string, v ...any) {
	f.Logger.Logf(f.GinkgoT, format, v...)
}
