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

package utils

import (
	"net"
	"regexp"

	k8stypes "k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/apache/apisix-ingress-controller/internal/types"
)

func NamespacedName(obj client.Object) k8stypes.NamespacedName {
	return k8stypes.NamespacedName{
		Namespace: obj.GetNamespace(),
		Name:      obj.GetName(),
	}
}

func NamespacedNameKind(obj client.Object) types.NamespacedNameKind {
	return types.NamespacedNameKind{
		Namespace: obj.GetNamespace(),
		Name:      obj.GetName(),
		Kind:      obj.GetObjectKind().GroupVersionKind().Kind,
	}
}

func ValidateRemoteAddrs(remoteAddrs []string) error {
	for _, addr := range remoteAddrs {
		if ip := net.ParseIP(addr); ip == nil {
			// addr is not an IP address, try to parse it as a CIDR.
			if _, _, err := net.ParseCIDR(addr); err != nil {
				return err
			}
		}
	}
	return nil
}

var hostDef = "^[a-z0-9]([-a-z0-9]*[a-z0-9])?(\\.[a-z0-9]([-a-z0-9]*[a-z0-9])?)*$"
var hostDefRegex = regexp.MustCompile(hostDef)

// MatchHostDef checks that host matches host's schema
// ref to : https://github.com/apache/apisix/blob/c5fc10d9355a0c177a7532f01c77745ff0639a7f/apisix/schema_def.lua#L40
// ref to : https://github.com/kubernetes/kubernetes/blob/976a940f4a4e84fe814583848f97b9aafcdb083f/staging/src/k8s.io/apimachinery/pkg/util/validation/validation.go#L205
// They define regex differently, but k8s's dns is more accurate
// todo: validate by CRD definition
func MatchHostDef(host string) bool {
	return hostDefRegex.MatchString(host)
}

func IsSubsetOf(a, b map[string]string) bool {
	if len(a) == 0 {
		// Empty labels matches everything.
		return true
	}
	for k, v := range a {
		if vv, ok := b[k]; !ok || vv != v {
			return false
		}
	}
	return true
}
