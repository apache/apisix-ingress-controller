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
package config

import (
    "context"
    "fmt"
    "time"

    ginkgo "github.com/onsi/ginkgo/v2"
    "github.com/stretchr/testify/assert"
    v1 "k8s.io/api/core/v1"
    metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

    "github.com/apache/apisix-ingress-controller/test/e2e/scaffold"
)

var _ = ginkgo.Describe("suite-config: deploy ingress controller with config", func() {
    opts := &scaffold.Options{
        Name:                  "default",
        Kubeconfig:            scaffold.GetKubeconfig(),
        APISIXConfigPath:      "testdata/apisix-gw-config.yaml",
        IngressAPISIXReplicas: 1,
        HTTPBinServicePort:    80,
        APISIXRouteVersion:    "apisix.apache.org/v2beta3",
    }
    s := scaffold.NewScaffold(opts)
    ginkgo.It("use configmap with env", func() {
        label := fmt.Sprintf("apisix.ingress.watch=%s", s.Namespace())
        configMap := fmt.Sprintf(_ingressAPISIXConfigMapTemplate, label)
        assert.Nil(ginkgo.GinkgoT(), s.CreateResourceFromString(configMap), "create configmap")

        client := s.GetKubernetesClient()
        deployment, err := client.AppsV1().Deployments(s.Namespace()).Get(context.Background(), "ingress-apisix-controller-deployment-e2e-test", metav1.GetOptions{})
        assert.Nil(ginkgo.GinkgoT(), err, "get apisix ingress controller deployment")

        spec := &deployment.Spec.Template.Spec
        spec.Containers[0].Command = []string{
            "/ingress-apisix/apisix-ingress-controller",
            "ingress",
            "--config-path",
            "/ingress-apisix/conf/config.yaml",
        }
        spec.Volumes = append(spec.Volumes, v1.Volume{
            Name: "apisix-ingress-controller-config",
            VolumeSource: v1.VolumeSource{
                ConfigMap: &v1.ConfigMapVolumeSource{
                    LocalObjectReference: v1.LocalObjectReference{
                        Name: "ingress-apisix-controller-config",
                    },
                },
            },
        })
        spec.Containers[0].VolumeMounts = append(spec.Containers[0].VolumeMounts, v1.VolumeMount{
            Name:      "apisix-ingress-controller-config",
            MountPath: "/ingress-apisix/conf/config.yaml",
            SubPath:   "config.yaml",
        })
        spec.Containers[0].Env = append(spec.Containers[0].Env, v1.EnvVar{
            Name:  "DEFAULT_CLUSTER_BASE_URL",
            Value: "http://apisix-service-e2e-test:9180/apisix/admin",
        }, v1.EnvVar{
            Name:  "DEFAULT_CLUSTER_ADMIN_KEY",
            Value: "edd1c9f034335f136f87ad84b625c8f1",
        })

        _, err = client.AppsV1().Deployments(s.Namespace()).Update(context.Background(), deployment, metav1.UpdateOptions{})
        assert.Nil(ginkgo.GinkgoT(), err, "update apisix ingress controller deployment")

        time.Sleep(10 * time.Second)
        assert.Nil(ginkgo.GinkgoT(), s.WaitAllIngressControllerPodsAvailable(), "wait all ingress controller pod available")
	})
})
