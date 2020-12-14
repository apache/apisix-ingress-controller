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
package conf

import "fmt"

type Message struct {
	Code string
	Msg  string
}

var (
	//AA 01 表示ingress-controller本身的错误
	//BB 00 表示系统信息
	SystemError = Message{"010001", "system errno"}

	//BB 01表示更新失败
	UpdateUpstreamNodesError = Message{"010101", "服务%s节点更新失败"}
	AddUpstreamError         = Message{"010102", "增加upstream %s失败"}
	AddUpstreamJsonError     = Message{"010103", "upstream %s json trans error"}
)

func (m Message) ToString(params ...interface{}) string {
	params = append(params, m.Code)
	return fmt.Sprintf(m.Msg+" error_code:%s", params...)
}
