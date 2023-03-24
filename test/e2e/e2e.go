// Licensed to the Apache Software Foundation (ASF) under one or more
// contributor license agreements.  See the NOTICE file distributed with
// this work for additional information regarding copyright ownership.
// The ASF licenses this file to You under the Apache License, Version 2.0
// (the "License"); you may not use this file except in compliance with
// the License.  You may obtain a copy of the License at
//
//	http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.
package e2e

import (
	"os"

	"github.com/apache/apisix-ingress-controller/pkg/log"

	_ "github.com/apache/apisix-ingress-controller/test/e2e/suite-annotations"
	_ "github.com/apache/apisix-ingress-controller/test/e2e/suite-chore"
	_ "github.com/apache/apisix-ingress-controller/test/e2e/suite-cluster"
	_ "github.com/apache/apisix-ingress-controller/test/e2e/suite-features"
	_ "github.com/apache/apisix-ingress-controller/test/e2e/suite-gateway"
	_ "github.com/apache/apisix-ingress-controller/test/e2e/suite-ingress/suite-ingress-features"
	_ "github.com/apache/apisix-ingress-controller/test/e2e/suite-ingress/suite-ingress-resource"
	_ "github.com/apache/apisix-ingress-controller/test/e2e/suite-plugins/suite-plugins-authentication"
	_ "github.com/apache/apisix-ingress-controller/test/e2e/suite-plugins/suite-plugins-general"
	_ "github.com/apache/apisix-ingress-controller/test/e2e/suite-plugins/suite-plugins-other"
	_ "github.com/apache/apisix-ingress-controller/test/e2e/suite-plugins/suite-plugins-security"
	_ "github.com/apache/apisix-ingress-controller/test/e2e/suite-plugins/suite-plugins-traffic"
	_ "github.com/apache/apisix-ingress-controller/test/e2e/suite-plugins/suite-plugins-transformation"
)

func runE2E() {
	if os.Getenv("E2E_ENV") != "ci" {
		var err error
		log.DefaultLogger, err = log.NewLogger(
			log.WithLogLevel("info"),
		)
		if err != nil {
			panic(err)
		}
	}
}
