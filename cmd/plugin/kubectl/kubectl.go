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
package kubectl

import (
	"context"
	"fmt"
	"os"
	"os/exec"
)

// openPortForward open a kubectl port-forward for accept apisix admin api. the port equlity  apisix service port
func OpenPortForward(ctx context.Context, namespace, svcName string) int {
	str := fmt.Sprintf("service/%v", svcName)

	cmd := exec.CommandContext(ctx, "kubectl", "port-forward", "-n", namespace, str, "9180:9180")
	cmd.Env = os.Environ()

	if err := cmd.Start(); err != nil {
		fmt.Println("cmd.Start() err :", err)
		return 0
	}
	return cmd.Process.Pid
}

func ClosePortForward(pid int) {
	str := fmt.Sprintf("-s QUIT %v", pid)
	cmd := exec.Command("kill", str)
	if err := cmd.Start(); err != nil {
		return
	}
}
