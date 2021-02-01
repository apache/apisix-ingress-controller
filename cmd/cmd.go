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
package cmd

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/apache/apisix-ingress-controller/cmd/ingress"
	"github.com/apache/apisix-ingress-controller/pkg/version"
)

func newVersionCommand() *cobra.Command {
	var long bool
	cmd := &cobra.Command{
		Use:   "version",
		Short: "version for apisix-ingress-controller",
		Run: func(cmd *cobra.Command, _ []string) {
			if long {
				fmt.Print(version.Long())
			} else {
				fmt.Printf("apisix-ingress-controller version %s\n", version.Short())
			}
		},
	}

	cmd.PersistentFlags().BoolVar(&long, "long", false, "show long mode version information")
	return cmd
}

// NewAPISIXIngressControllerCommand creates the apisix-ingress-controller command.
func NewAPISIXIngressControllerCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "apisix-ingress-controller [command]",
		Long:    "Yet another Ingress controller for Kubernetes using Apache APISIX as the high performance reverse proxy.",
		Version: version.Short(),
	}

	cmd.AddCommand(ingress.NewIngressCommand())
	cmd.AddCommand(newVersionCommand())
	return cmd
}
