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
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"os/exec"
	"strings"
)

type WolfServer struct {
	Url string
}

type WolfResp struct {
	OK   bool `json:"ok"`
	Data struct {
		Token string `json:"token,omitempty"`
	} `json:"data"`
	ErrMsg string `json:"errmsg,omitempty"`
	Reason string `json:"reason,omitempty"`
}

func (s *Scaffold) StartWolfServer() (*WolfServer, error) {
	var wolfSvr WolfServer
	err := initServer()
	if err != nil {
		return nil, err
	}
	wolfSvr.Url, err = initURL()
	if err != nil {
		return nil, err
	}
	err = initTestData(wolfSvr.Url)
	if err != nil {
		return nil, err
	}
	return &wolfSvr, nil
}

func (w *WolfServer) Stop() error {
	cmd := exec.Command("sh", "testdata/wolf-rbac/stop.sh")
	err := cmd.Run()
	return err
}

func initServer() error {
	cmd := exec.Command("sh", "testdata/wolf-rbac/start.sh")
	raw, err := cmd.Output()
	if len(raw) > 0 {
		log.Println(string(raw))
	}
	if err != nil {
		return err
	}
	return nil
}

func initURL() (string, error) {
	cmd := exec.Command("sh", "testdata/wolf-rbac/ip.sh")
	ip, err := cmd.Output()
	if err != nil {
		return "", err
	}
	if len(ip) == 0 {
		return "", fmt.Errorf("wolf-server start failed")
	}
	httpsvc := fmt.Sprintf("http://%s:12180", string(ip))
	return httpsvc, nil
}

func wolfRequest(header http.Header, url string, payload string) (*WolfResp, error) {
	req, _ := http.NewRequest("POST", url, strings.NewReader(payload))
	req.Header = header
	resp, err := (&http.Client{}).Do(req)
	if err != nil {
		return nil, fmt.Errorf("request wolf-server path:%s failed, err is %s", url, err.Error())
	}
	var msg WolfResp
	body, _ := ioutil.ReadAll(resp.Body)
	_ = json.Unmarshal(body, &msg)
	if !msg.OK {
		return nil, fmt.Errorf("request wolf-server path:%s failed, reson is %s.", url, msg.Reason)
	}
	return &msg, nil
}

func initTestData(url string) error {
	root := `{ "username": "root", "password": "wolf-123456"}`
	tokenMsg, err := wolfRequest(http.Header{"Content-Type": {"application/json"}}, url+"/wolf/user/login", root)
	if err != nil {
		return err
	}

	app := `{ "id": "test-app", "name": "application for test" }`
	_, err = wolfRequest(http.Header{"Content-Type": {"application/json"}, "x-rbac-token": {tokenMsg.Data.Token}}, url+"/wolf/application", app)
	if err != nil {
		return err
	}

	resource := `{ "appID": "test-app", "matchType": "prefix", "name": "/","action": "GET", "permID": "ALLOW_ALL" }`
	_, err = wolfRequest(http.Header{"Content-Type": {"application/json"}, "x-rbac-token": {tokenMsg.Data.Token}}, url+"/wolf/resource", resource)
	if err != nil {
		return err
	}

	user := `{"username": "test", "nickname": "test", "password": "test-123456", "appIDs": ["test-app"]}`
	_, err = wolfRequest(http.Header{"Content-Type": {"application/json"}, "x-rbac-token": {tokenMsg.Data.Token}}, url+"/wolf/user", user)
	if err != nil {
		return err
	}
	return nil
}
