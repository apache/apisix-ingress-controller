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
package scaffold

import (
	apisv1 "github.com/gxthrj/apisix-ingress-types/pkg/apis/config/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// ApisixRouteDesc describes the ApisixRoute object.
type ApisixRouteDesc struct {
	Name  string
	Host  string
	Paths []ApisixRoutePath
}

// ApisixRoutePath describes the path of ApisixRoute object.
type ApisixRoutePath struct {
	Path    string
	Backend ApisixRouteBackend
}

// ApisixRouteBackend describes the backend of ApisixRoute object.
type ApisixRouteBackend struct {
	ServiceName string
	ServicePort int64
}

// CreateApisixRoute creates a ApisixRoute object.
func (s *Scaffold) CreateApisixRoute(desc *ApisixRouteDesc) error {
	var paths []apisv1.Path
	for _, path := range desc.Paths {
		paths = append(paths, apisv1.Path{
			Path: path.Path,
			Backend: apisv1.Backend{
				ServiceName: path.Backend.ServiceName,
				ServicePort: path.Backend.ServicePort,
			},
			Plugins: nil,
		})
	}
	route := &apisv1.ApisixRoute{
		ObjectMeta: metav1.ObjectMeta{
			Name:      desc.Name,
			Namespace: s.namespace,
		},
		Spec: &apisv1.ApisixRouteSpec{
			Rules: []apisv1.Rule{
				{
					Host: desc.Host,
					Http: apisv1.Http{
						Paths: paths,
					},
				},
			},
		},
	}

	condFunc := func() (bool, error) {
		_, err := s.apisixClient.ApisixV1().ApisixRoutes(s.namespace).Create(route)
		if err != nil {
			return false, err
		}
		return true, nil
	}
	return waitExponentialBackoff(condFunc)
}
