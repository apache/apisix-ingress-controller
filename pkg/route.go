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
package pkg

import (
	"github.com/julienschmidt/httprouter"
	"net/http"
	"io"
	"encoding/json"
	"io/ioutil"
	"github.com/api7/ingress-controller/log"
)

var logger = log.GetLogger()

func Route() *httprouter.Router{
	router := httprouter.New()
	router.GET("/healthz", Healthz)
	router.GET("/apisix/healthz", Healthz)
	//router.GET("/apisix/sync/upstream/:name", syncPodWithUpstream)
	return router
}

func Healthz(w http.ResponseWriter, req *http.Request, _ httprouter.Params){
	io.WriteString(w, "ok")
}

type CheckResponse struct{
	Ok bool `json:"ok"`
}

type WriteResponse struct{
	Status string `json:"status"`
	Msg string `json:"msg"`
}

func populateMode(w http.ResponseWriter, r *http.Request, params httprouter.Params, model interface{}) error{
	body, err := ioutil.ReadAll(io.LimitReader(r.Body, 1048576))
	if err != nil {
		return err
	}
	if err := r.Body.Close(); err != nil {
		return err
	}
	if err := json.Unmarshal(body, model); err != nil {
		return err
	}
	return nil
}
