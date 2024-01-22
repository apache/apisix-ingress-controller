// Licensed to the Apache Software Foundation (ASF) under one
// or more contributor license agreements.  See the NOTICE file
// distributed with this work for additional information
// regarding copyright ownership.  The ASF licenses this file
// to you under the Apache License, Version 2.0 (the
// "License"); you may not use this file except in compliance
// with the License.  You may obtain a copy of the License at
//
//	http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing,
// software distributed under the License is distributed on an
// "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY
// KIND, either express or implied.  See the License for the
// specific language governing permissions and limitations
// under the License.
package utils

import (
	"testing"

	"github.com/stretchr/testify/assert"
	networkingv1 "k8s.io/api/networking/v1"
	networkingv1beta1 "k8s.io/api/networking/v1beta1"
)

func TestCompareNetworkingV1LBEqual(t *testing.T) {
	lb1 := []networkingv1.IngressLoadBalancerIngress{
		{
			IP: "0.0.0.0",
		},
	}
	lb2 := []networkingv1.IngressLoadBalancerIngress{
		{
			IP: "0.0.0.0",
		},
	}
	assert.Equal(t, true, CompareNetworkingV1LBEqual(lb1, lb2))

	lb1 = []networkingv1.IngressLoadBalancerIngress{
		{
			IP: "0.0.0.0",
		},
	}
	lb2 = []networkingv1.IngressLoadBalancerIngress{
		{
			IP: "0.0.0.0",
		},
		{
			Hostname: "test.com",
		},
	}
	assert.Equal(t, false, CompareNetworkingV1LBEqual(lb1, lb2))

	lb1 = []networkingv1.IngressLoadBalancerIngress{
		{
			IP: "127.0.0.1",
		},
		{
			IP: "0.0.0.0",
		},
	}
	lb2 = []networkingv1.IngressLoadBalancerIngress{
		{
			IP: "0.0.0.0",
		},
		{
			IP: "127.0.0.1",
		},
	}
	assert.Equal(t, true, CompareNetworkingV1LBEqual(lb1, lb2))

	lb1 = []networkingv1.IngressLoadBalancerIngress{
		{
			IP: "127.0.0.1",
		},
		{
			IP: "1.1.1.1",
		},
	}
	lb2 = []networkingv1.IngressLoadBalancerIngress{
		{
			IP: "127.0.0.1",
		},
		{
			IP: "0.0.0.0",
		},
	}
	assert.Equal(t, false, CompareNetworkingV1LBEqual(lb1, lb2))

	lb1 = []networkingv1.IngressLoadBalancerIngress{
		{
			Hostname: "test.com",
		},
		{
			IP: "127.0.0.1",
		},
		{
			IP: "0.0.0.0",
		},
	}
	lb2 = []networkingv1.IngressLoadBalancerIngress{
		{
			IP: "0.0.0.0",
		},
		{
			IP: "127.0.0.1",
		},
		{
			Hostname: "test.com",
		},
	}
	assert.Equal(t, true, CompareNetworkingV1LBEqual(lb1, lb2))
}

func TestCompareNetworkingV1beta1LBEqual(t *testing.T) {
	lb1 := []networkingv1beta1.IngressLoadBalancerIngress{
		{
			IP: "0.0.0.0",
		},
	}
	lb2 := []networkingv1beta1.IngressLoadBalancerIngress{
		{
			IP: "0.0.0.0",
		},
	}
	assert.Equal(t, true, CompareNetworkingV1beta1LBEqual(lb1, lb2))

	lb1 = []networkingv1beta1.IngressLoadBalancerIngress{
		{
			IP: "0.0.0.0",
		},
	}
	lb2 = []networkingv1beta1.IngressLoadBalancerIngress{
		{
			IP: "0.0.0.0",
		},
		{
			Hostname: "test.com",
		},
	}
	assert.Equal(t, false, CompareNetworkingV1beta1LBEqual(lb1, lb2))

	lb1 = []networkingv1beta1.IngressLoadBalancerIngress{
		{
			IP: "127.0.0.1",
		},
		{
			IP: "1.1.1.1",
		},
	}
	lb2 = []networkingv1beta1.IngressLoadBalancerIngress{
		{
			IP: "127.0.0.1",
		},
		{
			IP: "0.0.0.0",
		},
	}
	assert.Equal(t, false, CompareNetworkingV1beta1LBEqual(lb1, lb2))

	lb1 = []networkingv1beta1.IngressLoadBalancerIngress{
		{
			Hostname: "test.com",
		},
		{
			IP: "127.0.0.1",
		},
		{
			IP: "0.0.0.0",
		},
	}
	lb2 = []networkingv1beta1.IngressLoadBalancerIngress{
		{
			IP: "0.0.0.0",
		},
		{
			IP: "127.0.0.1",
		},
		{
			Hostname: "test.com",
		},
	}
	assert.Equal(t, true, CompareNetworkingV1beta1LBEqual(lb1, lb2))
}
