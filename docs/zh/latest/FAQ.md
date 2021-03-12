---
title: FAQ
---

<!--
#
# Licensed to the Apache Software Foundation (ASF) under one or more
# contributor license agreements.  See the NOTICE file distributed with
# this work for additional information regarding copyright ownership.
# The ASF licenses this file to You under the Apache License, Version 2.0
# (the "License"); you may not use this file except in compliance with
# the License.  You may obtain a copy of the License at
#
#     http://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.
#
-->

1. 如何绑定 Service 和 Upstream?

所有资源对象都由 namespace / name / port 组成的的联合Id 来唯一确定。如果联合Id相同，`service` 和 `upstream` 将被视为绑定关系。

2. 当修改一个 CRD, 其他绑定对象如何感知它 ?

这是一个级联更新问题, 详见 [apisix-ingress-controller Design ideas](./design.md)

3. 我可以混用 mix CRDs 和 admin api 来定义路由规则么?

不可以，目前我们正在实现单向同步，即 CRDs 文件 -> Apache AIPSIX。 如果通过 admin api 单独修改配置，它将不会同步到Kubernetes 中的 CRDs。

这是因为 CRDs 一般都是在文件系统中声明的，并且应用到 Kubernetes 的 etcd， 我们遵循 CRD 的定义并同步到 Apache Apisix 的数据面，但是反过来会使情况更加复杂。

4. 出现一些错误的日志，如 "list upstreams failed, err: http get failed, url: blahblahblah, err: status: 401" 是为什么?

到目前为止，apisix-ingress-controller 不支持为 Apache APISIX 设置 admin_key，因此当您部署 APISIX 集群时，admin_key 应该从配置中删除。

注意：由于 APISIX 有两个配置文件，第一个是 config.yaml，包含用户指定的配置，另一个是 config-default.yaml，包含所有默认项，这两个文件中的配置项将被合并。所以这两个文件中的 admin_key 都应该被去掉。您可以自己配制这两个配置文件并将它们加载到 APISIX 中。

5.用 `ApisixRoute` 创建路由失败?

当 `apisix-ingress-controller` 使用 CRD 创建路由时， 它将检查 Kubernetes 中的 `Endpoint` 资源（是否与 namespace_name_port 匹配）。如果找不到相应的端点信息，则不会创建路由并等待下一次重试。

提示：空的上游节点导致的故障是 Apache APISIX 的一个限制，请查看这个 [issue](https://github.com/apache/apisix/issues/3072)

6. `apisix-ingress-controller`的重试规则是什么?

如果在 `apisix-ingress-controller` 解析 CRD 并将配置分发给Apache APISIX 的过程中发生错误，将触发重试。

采用延迟重试方法。第一次失败后，每秒重试一次。触发5次重试后，将启用慢速重试策略，每1分钟重试一次，直到成功。
