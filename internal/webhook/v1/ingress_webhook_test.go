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

package v1

import (
	"context"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	networkingk8siov1 "k8s.io/api/networking/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

var _ = Describe("Ingress Webhook", func() {
	var (
		obj       *networkingk8siov1.Ingress
		oldObj    *networkingk8siov1.Ingress
		validator IngressCustomValidator
	)

	BeforeEach(func() {
		obj = &networkingk8siov1.Ingress{}
		oldObj = &networkingk8siov1.Ingress{}
		validator = IngressCustomValidator{}
		Expect(validator).NotTo(BeNil(), "Expected validator to be initialized")
		Expect(oldObj).NotTo(BeNil(), "Expected oldObj to be initialized")
		Expect(obj).NotTo(BeNil(), "Expected obj to be initialized")
	})

	Context("When creating or updating Ingress under Validating Webhook", func() {
		It("Should return warnings for unsupported annotations on create", func() {
			By("Creating an Ingress with unsupported annotations")
			obj.ObjectMeta = metav1.ObjectMeta{
				Name:      "test-ingress",
				Namespace: "default",
				Annotations: map[string]string{
					"k8s.apisix.apache.org/use-regex":        "true",
					"k8s.apisix.apache.org/enable-websocket": "true",
					"nginx.ingress.kubernetes.io/rewrite":    "/new-path",
				},
			}

			warnings, err := validator.ValidateCreate(context.TODO(), obj)
			Expect(err).ToNot(HaveOccurred())
			Expect(warnings).To(HaveLen(2))
			Expect(warnings[0]).To(ContainSubstring("k8s.apisix.apache.org/use-regex"))
			Expect(warnings[1]).To(ContainSubstring("k8s.apisix.apache.org/enable-websocket"))
		})

		It("Should return no warnings for supported annotations on create", func() {
			By("Creating an Ingress with only supported annotations")
			obj.ObjectMeta = metav1.ObjectMeta{
				Name:      "test-ingress",
				Namespace: "default",
				Annotations: map[string]string{
					"nginx.ingress.kubernetes.io/rewrite": "/new-path",
					"kubernetes.io/ingress.class":         "apisix",
				},
			}

			warnings, err := validator.ValidateCreate(context.TODO(), obj)
			Expect(err).ToNot(HaveOccurred())
			Expect(warnings).To(BeEmpty())
		})

		It("Should return warnings for unsupported annotations on update", func() {
			By("Updating an Ingress with unsupported annotations")
			obj.ObjectMeta = metav1.ObjectMeta{
				Name:      "test-ingress",
				Namespace: "default",
				Annotations: map[string]string{
					"k8s.apisix.apache.org/enable-cors":       "true",
					"k8s.apisix.apache.org/cors-allow-origin": "*",
				},
			}

			warnings, err := validator.ValidateUpdate(context.TODO(), oldObj, obj)
			Expect(err).ToNot(HaveOccurred())
			Expect(warnings).To(HaveLen(2))
			Expect(warnings[0]).To(ContainSubstring("k8s.apisix.apache.org/enable-cors"))
			Expect(warnings[1]).To(ContainSubstring("k8s.apisix.apache.org/cors-allow-origin"))
		})

		It("Should not return warnings for deletion", func() {
			By("Deleting an Ingress with unsupported annotations")
			obj.ObjectMeta = metav1.ObjectMeta{
				Name:      "test-ingress",
				Namespace: "default",
				Annotations: map[string]string{
					"k8s.apisix.apache.org/use-regex": "true",
				},
			}

			warnings, err := validator.ValidateDelete(context.TODO(), obj)
			Expect(err).ToNot(HaveOccurred())
			Expect(warnings).To(BeEmpty())
		})

		It("Should handle Ingress without annotations", func() {
			By("Creating an Ingress without any annotations")
			obj.ObjectMeta = metav1.ObjectMeta{
				Name:      "test-ingress",
				Namespace: "default",
			}

			warnings, err := validator.ValidateCreate(context.TODO(), obj)
			Expect(err).ToNot(HaveOccurred())
			Expect(warnings).To(BeEmpty())
		})
	})

})
