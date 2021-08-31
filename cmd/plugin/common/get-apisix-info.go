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
package common

import (
	"fmt"
	"io/ioutil"
	"net/http"

	"github.com/apache/apisix-ingress-controller/pkg/log"
)

type RequestConf struct {
	URL      string
	AdminKey string
}

func (rc *RequestConf) Get(itermID string) ([]byte, error) {
	newURL := fmt.Sprintf("%v/%v", rc.URL, itermID)
	cli := &http.Client{}
	req, err := http.NewRequest("GET", newURL, nil)
	if err != nil {
		log.Error("http.NewRequest() => ", err)
		return nil, err
	}
	log.Info("new URL => ", newURL)
	req.Header.Set("X-API-KEY", rc.AdminKey)

	resp, err := cli.Do(req)
	defer resp.Body.Close()

	if err != nil {
		log.Error("cli.Do() => ", err)
		return nil, err
	}
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		log.Error("ioutil.ReadAll() => ", err)
		return nil, err
	}

	return body, err
}
