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
//
package scaffold

import (
	"fmt"
	"os/exec"
)

func (s *Scaffold) WolfRbacStartedHttpSvr() {
	cmd := exec.Command("sh", "testdata/wolf-rbac/start.sh")
	_ = cmd.Run()
}

func (s *Scaffold) WolfRbacIPAddress() (string, error) {
	cmd := exec.Command("sh", "testdata/wolf-rbac/ip.sh")
	s.WolfRbacStartedHttpSvr()
	ip, err := cmd.Output()
	if err != nil {
		return "", err
	}
	if len(ip) == 0 {
		return "", fmt.Errorf("wolf-server start fild")
	}
	httpsvc := fmt.Sprintf("http://%s:12180", string(ip))
	return httpsvc, nil
}

func (s *Scaffold) StopWolfRbac() error {
	cmd := exec.Command("sh", "testdata/wolf-rbac/stop.sh")
	err := cmd.Run()
	return err
}
