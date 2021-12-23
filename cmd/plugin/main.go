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
package main

import (
	"fmt"
	"os"

	"github.com/apache/apisix-ingress-controller/cmd/plugin/commands/routes"
	"github.com/apache/apisix-ingress-controller/cmd/plugin/commands/upstreams"
	"github.com/apache/apisix-ingress-controller/pkg/log"

	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
	"k8s.io/cli-runtime/pkg/genericclioptions"
)

var (
	logLevel  string
	logOutput string
)

func main() {
	rootCmd := &cobra.Command{
		Use:   "apisix-ingress-controller",
		Short: "A kubectl plugin for inspecting your apisix-ingress-controller deployments",
	}

	pflag.StringVar(&logLevel, "log-level", "info", "error log level")
	pflag.StringVar(&logOutput, "log-output", "/tmp/apisix-ingress-controller-plugin.log", "error log output file")
	logger, err := log.NewLogger(
		log.WithLogLevel(logLevel),
		log.WithOutputFile(logOutput),
	)
	if err != nil {
		fmt.Println(err)
		return
	}
	log.DefaultLogger = logger
	flags := genericclioptions.NewConfigFlags(true)
	flags.AddFlags(rootCmd.PersistentFlags())

	rootCmd.AddCommand(routes.CreateCommand(flags))
	rootCmd.AddCommand(upstreams.CreateCommand(flags))
	if err := rootCmd.Execute(); err != nil {
		log.Error(err)
		os.Exit(1)
	}

}
