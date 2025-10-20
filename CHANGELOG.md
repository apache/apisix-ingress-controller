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

# Table of Contents

- [2.0.0-rc5](#200-rc5)
- [2.0.0-rc4](#200-rc4)
- [2.0.0-rc3](#200-rc3)
- [2.0.0-rc2](#200-rc2)
- [2.0.0-rc1](#200-rc1)
- [1.8.0](#180)
- [1.7.0](#170)
- [1.6.0](#160)
- [1.6.0-rc1](#160-rc1)
- [1.5.1](#151)
- [1.5.0](#150)
- [1.5.0-rc1](#150-rc1)
- [1.4.1](#141)
- [1.4.0](#140)
- [1.3.0](#130)
- [1.2.0](#120)
- [1.1.0](#110)
- [1.0.0](#100)
- [0.6.0](#060)
- [0.5.0](#050)
- [0.4.0](#040)
- [0.3.0](#030)
- [0.2.0](#020)
- [0.1.0](#010)

# 2.0.0-rc5

apisix-ingress-controller 2.0.0-rc5

Welcome to the v2.0.0-rc5 release of apisix-ingress-controller!  
*This is a pre-release of apisix-ingress-controller*

## Highlights

- **Gateway API Support**: Added support for **TCPRoute**, **UDPRoute**, **GRPCRoute**, and **TLSRoute**, achieving comprehensive Gateway API coverage.  
- **Enhanced Webhook Validation**:  
  - Introduced **certificate conflict detection** and **Secret/Service resource checking** to improve configuration consistency and security.  
  - Added admission webhooks for **IngressClass** and **Gateway** resources.  
- **GatewayProxy Improvements**: Added **conflict detection** and **stricter provider validation** to prevent misconfiguration between instances.  
- **APISIX Upstream Enhancements**: Added support for **health checks**, **service discovery**, and **port-level settings**.  
- **Improved Inter-Container Communication**: Added **Unix socket** support for faster internal communication.  
- **Conformance and Stability Fixes**: Fixed multiple **Gateway API conformance tests** (PathRewrite, QueryParamMatching, RewriteHost, etc.) to ensure better compatibility.  
- **Resilience Improvements**: Introduced **retry mechanism** on sync failures.  
- **Debugging and Observability**: Added a **unified API server** for debugging and enhanced logging consistency.  

---

## Features

* feat: add certificate conflict detection to admission webhooks [#2603](https://github.com/apache/apisix-ingress-controller/pull/2603)  
* feat: support resolve `svc.ports[].appProtocol` [#2601](https://github.com/apache/apisix-ingress-controller/pull/2601)  
* feat: add conflict detection for gateway proxy [#2600](https://github.com/apache/apisix-ingress-controller/pull/2600)  
* feat(gateway-api): support TLSRoute [#2594](https://github.com/apache/apisix-ingress-controller/pull/2594)  
* feat: support UDPRoute webhook [#2588](https://github.com/apache/apisix-ingress-controller/pull/2588)  
* feat: add Unix socket support for inter-container communication [#2587](https://github.com/apache/apisix-ingress-controller/pull/2587)  
* feat(apisixupstream): support portLevelSettings [#2582](https://github.com/apache/apisix-ingress-controller/pull/2582)  
* feat: add secret/service resource checker for webhook [#2583](https://github.com/apache/apisix-ingress-controller/pull/2583)  
* feat(gateway-api): add support for UDPRoute [#2578](https://github.com/apache/apisix-ingress-controller/pull/2578)  
* feat(apisixupstream): support discovery [#2577](https://github.com/apache/apisix-ingress-controller/pull/2577)  
* feat(apisixupstream): support healthcheck [#2574](https://github.com/apache/apisix-ingress-controller/pull/2574)  
* feat: add webhook for ingressclass and gateway [#2572](https://github.com/apache/apisix-ingress-controller/pull/2572)  
* feat(gateway-api): support GRPCRoute [#2570](https://github.com/apache/apisix-ingress-controller/pull/2570)  
* feat: add webhook server [#2566](https://github.com/apache/apisix-ingress-controller/pull/2566)  
* feat: add unified API server with debugging capabilities [#2550](https://github.com/apache/apisix-ingress-controller/pull/2550)  
* feat: add support for named servicePort in ApisixRoute backend [#2553](https://github.com/apache/apisix-ingress-controller/pull/2553)  
* feat: support stream_route for ApisixRoute [#2551](https://github.com/apache/apisix-ingress-controller/pull/2551)  
* feat: add support for CORS httproutefilter [#2548](https://github.com/apache/apisix-ingress-controller/pull/2548)  
* feat: add support for TCPRoute [#2564](https://github.com/apache/apisix-ingress-controller/pull/2564)  
* feat: support retry in case of sync failure [#2534](https://github.com/apache/apisix-ingress-controller/pull/2534)  

---

## Bug Fixes

* fix(ingress): port.name matching failure for ExternalName Services [#2604](https://github.com/apache/apisix-ingress-controller/pull/2604)  
* fix(gatewayproxy): add stricter validation rules for provider [#2602](https://github.com/apache/apisix-ingress-controller/pull/2602)  
* fix: generate unique SSL IDs to prevent certificate conflicts across different hosts [#2592](https://github.com/apache/apisix-ingress-controller/pull/2592)  
* fix(conformance-test): HTTPRoutePathRewrite [#2597](https://github.com/apache/apisix-ingress-controller/pull/2597)  
* fix(conformance-test): HTTPRouteQueryParamMatching [#2598](https://github.com/apache/apisix-ingress-controller/pull/2598)  
* fix(conformance-test): HTTPRouteRewriteHost [#2596](https://github.com/apache/apisix-ingress-controller/pull/2596)  
* fix: residual data issue when updating ingressClassName [#2543](https://github.com/apache/apisix-ingress-controller/pull/2543)  
* fix: responseHeaderModifier fails to synchronize [#2544](https://github.com/apache/apisix-ingress-controller/pull/2544)  
* fix: handle httproute multi backend refs [#2540](https://github.com/apache/apisix-ingress-controller/pull/2540)  
* fix: sync exception caused by ingress endpoint 0 [#2538](https://github.com/apache/apisix-ingress-controller/pull/2538)  
* fix: use upstream id instead of inline upstream in traffic-split plugin [#2546](https://github.com/apache/apisix-ingress-controller/pull/2546)  
* fix: hmac-auth plugin spec compatibility with latest apisix [#2528](https://github.com/apache/apisix-ingress-controller/pull/2528)

Please try out the release binaries and report any issues at
https://github.com/apache/apisix-ingress-controller/issues.

### Contributors

* AlinsRan
* Ashing Zheng
* Ashish Tiwari
* Traky Deng
* ArthurMorgan

### Changes
<details><summary>53 commits</summary>
<p>

  * [`351d20a5`](https://github.com/apache/apisix-ingress-controller/commit/351d20a51af496012b822bae225839cac6832750) feat: add certificate conflict detection to admission webhooks (#2603)
  * [`18882b92`](https://github.com/apache/apisix-ingress-controller/commit/18882b928a4679a85ba18e242f90e6c48151961c) fix(ingress): port.name matching failure for ExternalName Services (#2604)
  * [`f28b3429`](https://github.com/apache/apisix-ingress-controller/commit/f28b34292be45587184d9faa8cb5bfc477d5d1b8) feat: support resolve svc.ports[].appProtocol (#2601)
  * [`3c808e22`](https://github.com/apache/apisix-ingress-controller/commit/3c808e226a33c4355f5e254c10fe61b85c331f2f) fix(gatewayproxy): add stricter validation rules for provider (#2602)
  * [`ace684d1`](https://github.com/apache/apisix-ingress-controller/commit/ace684d18fa0001f4cff2a9ce291a80f8e9897a7) feat: add conflict detection for gateway proxy (#2600)
  * [`5f0d1af1`](https://github.com/apache/apisix-ingress-controller/commit/5f0d1af1dea737f8c509ee6f52b34f55bccd5b2d) fix: generate unique SSL IDs to prevent certificate conflicts across different hosts (#2592)
  * [`15132023`](https://github.com/apache/apisix-ingress-controller/commit/15132023371821df66489c9ea1d4006789c27e71) fix(conformance-test): HTTPRoutePathRewrite (#2597)
  * [`1afb9ace`](https://github.com/apache/apisix-ingress-controller/commit/1afb9acea8f55270055e7274e18647731d6150f1) feat(gateway-api): support TLSRoute (#2594)
  * [`a368e287`](https://github.com/apache/apisix-ingress-controller/commit/a368e2876cdfa80e5ffda3565e28cda53e28f370) fix(conformance-test): HTTPRouteQueryParamMatching (#2598)
  * [`f6db4561`](https://github.com/apache/apisix-ingress-controller/commit/f6db45615cf9fdb3928d26cce5307243973c7b83) fix(conformance-test): HTTPRouteRewriteHost (#2596)
  * [`0f29ac7b`](https://github.com/apache/apisix-ingress-controller/commit/0f29ac7b65a8a6561c46a376a4a930fffd270cbe) docs: clarify support for Ingress Annotations (#2575)
  * [`d9550d88`](https://github.com/apache/apisix-ingress-controller/commit/d9550d8887012b63e5803cb281f48b32010158a2) chore: unify the logging component (#2584)
  * [`63c7d111`](https://github.com/apache/apisix-ingress-controller/commit/63c7d111a4fc4e557e570f34b2a452c9d4941bb2) feat: support udproute webhook (#2588)
  * [`dc8b6621`](https://github.com/apache/apisix-ingress-controller/commit/dc8b66214663ba534575aa1be8167786d01df613) feat: add Unix socket support for inter-container communication (#2587)
  * [`501b4e89`](https://github.com/apache/apisix-ingress-controller/commit/501b4e89f2567a8166660131d36511824cf13375) test: add e2e test case for webhook (#2585)
  * [`fe5c1357`](https://github.com/apache/apisix-ingress-controller/commit/fe5c1357ba76b1f73ecd49a0cc68583277271411) feat(apisixupstream): support portLevelSettings (#2582)
  * [`ec819175`](https://github.com/apache/apisix-ingress-controller/commit/ec819175cff42354a55525efc9cd28f1a9e52c18) feat: add secret/service resource checker for webhook (#2583)
  * [`68664908`](https://github.com/apache/apisix-ingress-controller/commit/686649083da6a2cd7f063485ccbe6190a735be1c) feat(gateway-api): add support for UDPRoute (#2578)
  * [`5bb2afd6`](https://github.com/apache/apisix-ingress-controller/commit/5bb2afd62cdf0d23878c2ca3cade5defd75a25cf) feat: add secret/service resource checker for webhook (#2580)
  * [`be91920e`](https://github.com/apache/apisix-ingress-controller/commit/be91920e3a6a4c2d0cace198284debddab4299ea) feat(apisixupstream): support discovery (#2577)
  * [`b4276e3b`](https://github.com/apache/apisix-ingress-controller/commit/b4276e3bf175da83043bc227240e19c168ed04e4) chore: add stream_route test for standalone (#2565)
  * [`0fd8e9d7`](https://github.com/apache/apisix-ingress-controller/commit/0fd8e9d70179e7aaf74504a87ca34e0e0031532d) feat: add support for TCPRoute (#2564)
  * [`7510e5c3`](https://github.com/apache/apisix-ingress-controller/commit/7510e5c377a3ca95d2fbcad19814cfc590685117) feat(apisixupstream): support healthcheck (#2574)
  * [`3b3bb2ca`](https://github.com/apache/apisix-ingress-controller/commit/3b3bb2ca6f678a0cbe9e3c2dc16b3f152a0a2fbd) feat: add webhook for ingressclass and gateway (#2572)
  * [`fa9f775e`](https://github.com/apache/apisix-ingress-controller/commit/fa9f775ead5242819022fbb3afb4aaf2d1750e9b) docs: add ingressClassName to explicitly specify which ingress class should handle each resource (#2573)
  * [`51077751`](https://github.com/apache/apisix-ingress-controller/commit/51077751aab206f8719031dab7bc6dd75b959941) feat(gateway-api): support GRPCRoute (#2570)
  * [`992bead3`](https://github.com/apache/apisix-ingress-controller/commit/992bead3770d9c1ade182090bfa0b264d056c4d6) remove repetitive namespace (#2568)
  * [`734849a1`](https://github.com/apache/apisix-ingress-controller/commit/734849a16987bd111c6b73671b52f4eda45fe2a1) feat: add webhook server (#2566)
  * [`7399778b`](https://github.com/apache/apisix-ingress-controller/commit/7399778b0b261353662d1dff5de012818d7d17c7) chore: backport ldap auth test (#2569)
  * [`72626839`](https://github.com/apache/apisix-ingress-controller/commit/726268396c4639f0389462acd291fa2ce917dcb5) chore: remove redundant debug logging for metrics response (#2567)
  * [`a47c8c6e`](https://github.com/apache/apisix-ingress-controller/commit/a47c8c6e2c26abe943a8b8970b181fea1d64e6d9) docs: add config.yaml reference doc; explain a few parameters and gateway port being ignored (#2552)
  * [`bdbb3b9a`](https://github.com/apache/apisix-ingress-controller/commit/bdbb3b9a862ae9fb9e0ffe9fe8c960384e5dc174) docs: add FAQ page (#2561)
  * [`b47ed044`](https://github.com/apache/apisix-ingress-controller/commit/b47ed044d6b28dd183a25f6eb79d3aaf88ffe8fe) feat: add unified API server with debugging capabilities (#2550)
  * [`7fee6bd3`](https://github.com/apache/apisix-ingress-controller/commit/7fee6bd3315db3c790880080cd59cba015f3e950) chore: use constant variable instead of hard code (#2560)
  * [`4e1bd6eb`](https://github.com/apache/apisix-ingress-controller/commit/4e1bd6eb43591ae28c541bd43567955a02f6f829) chore(ci): remove next branch trigger condition (#2563)
  * [`0154b13a`](https://github.com/apache/apisix-ingress-controller/commit/0154b13ad2e12927db1c8676cd0dcb9904c59984) test: unified gatewayproxy yaml acquisition (#2562)
  * [`3d4d833b`](https://github.com/apache/apisix-ingress-controller/commit/3d4d833b7e893c39dd973f030ccb33fe0528b7c3) docs: update gateway api docs (#2558)
  * [`8f85e7f9`](https://github.com/apache/apisix-ingress-controller/commit/8f85e7f9df14cbe682413b50a12eb8b034b4f4d0) chore: add more conformance-test report for gateway-api (#2557)
  * [`76c695c8`](https://github.com/apache/apisix-ingress-controller/commit/76c695c8b12d368598ffda5794a45920e1a097c0) feat: add support for named servicePort in ApisixRoute backend (#2553)
  * [`476783aa`](https://github.com/apache/apisix-ingress-controller/commit/476783aab6090dbfb1051fc2ad021de5f490aada) chore: add skip_mtls_uril_regex test for ApisixTLS (#2555)
  * [`40ae032b`](https://github.com/apache/apisix-ingress-controller/commit/40ae032be36510b5d8469cc08d42ca4c3224c9c8) chore: migrate e2e test for secretRef in ApisixRoute.plugins (#2556)
  * [`53267ff8`](https://github.com/apache/apisix-ingress-controller/commit/53267ff86218f48d45bd5725a1ba1938238d69cb) feat: support stream_route for ApisixRoute (#2551)
  * [`ebaed224`](https://github.com/apache/apisix-ingress-controller/commit/ebaed2241210bbfdc88bdb878e8239167de98c71) docs: specify namespace in metadata explicitly (#2549)
  * [`7f6cff4a`](https://github.com/apache/apisix-ingress-controller/commit/7f6cff4a0ecbd4085bc73641994f4d39f083b612) fix: use upstream id instead of inline upstream in traffic-split plugin (#2546)
  * [`cb69f53c`](https://github.com/apache/apisix-ingress-controller/commit/cb69f53cf1e6a79805047a1deadcfa93be67264b) feat: add support for CORS httproutefilter (#2548)
  * [`2d57b6a2`](https://github.com/apache/apisix-ingress-controller/commit/2d57b6a25d22ed0f14e38b89d51011b475534436) upgrade: gateway-api version to v1.3.0 (#2547)
  * [`0a7b0402`](https://github.com/apache/apisix-ingress-controller/commit/0a7b0402528091e7435674a2ef42e2742e2a297e) fix: residual data issue when updating ingressClassName (#2543)
  * [`f1724106`](https://github.com/apache/apisix-ingress-controller/commit/f17241061700893ba0d196964780c3f2b8f137d1) fix: responseHeaderModifier fails to synchronize (#2544)
  * [`ec8624b1`](https://github.com/apache/apisix-ingress-controller/commit/ec8624b11d5c71d6904c6f9b292b260794db3e5f) fix: handle httproute multi backend refs (#2540)
  * [`c831c161`](https://github.com/apache/apisix-ingress-controller/commit/c831c161e2d66e75ceb0819871dd394d0b0409f1) fix: sync exception caused by ingress endpoint 0 (#2538)
  * [`efc29bff`](https://github.com/apache/apisix-ingress-controller/commit/efc29bff519783f9598b1f60055bfbec8ee304eb) docs: correct description for externalNodes (#2541)
  * [`7c926e66`](https://github.com/apache/apisix-ingress-controller/commit/7c926e66b70256c5e132863ab0b3e2ec7fb4043d) feat: support retry in case of sync failure (#2534)
  * [`06981b18`](https://github.com/apache/apisix-ingress-controller/commit/06981b18d3f089b14e4ead306d22a6f34d958fea) fix: hmac-auth plugin spec compatibility with latest apisix (#2528)
</p>
</details>

### Dependency Changes

* **cel.dev/expr**                                             v0.19.1 **_new_**
* **github.com/cpuguy83/go-md2man/v2**                         v2.0.5 -> v2.0.6
* **github.com/eclipse/paho.mqtt.golang**                      v1.5.0 **_new_**
* **github.com/evanphx/json-patch/v5**                         v5.9.0 -> v5.9.11
* **github.com/fatih/color**                                   v1.17.0 -> v1.18.0
* **github.com/google/btree**                                  v1.1.3 **_new_**
* **github.com/google/cel-go**                                 v0.20.1 -> v0.22.0
* **github.com/google/go-cmp**                                 v0.6.0 -> v0.7.0
* **github.com/google/pprof**                                  813a5fbdbec8 -> d1b30febd7db
* **github.com/gorilla/websocket**                             v1.5.1 -> v1.5.3
* **github.com/miekg/dns**                                     v1.1.62 -> v1.1.65
* **github.com/moby/spdystream**                               v0.4.0 -> v0.5.0
* **github.com/onsi/ginkgo/v2**                                v2.20.0 -> v2.22.0
* **github.com/onsi/gomega**                                   v1.34.1 -> v1.36.1
* **github.com/spf13/cobra**                                   v1.8.1 -> v1.9.1
* **github.com/spf13/pflag**                                   v1.0.5 -> v1.0.6
* **github.com/stoewer/go-strcase**                            v1.2.0 -> v1.3.0
* **go.opentelemetry.io/auto/sdk**                             v1.1.0 **_new_**
* **go.opentelemetry.io/otel**                                 v1.29.0 -> v1.34.0
* **go.opentelemetry.io/otel/metric**                          v1.29.0 -> v1.34.0
* **go.opentelemetry.io/otel/sdk**                             v1.29.0 -> v1.34.0
* **go.opentelemetry.io/otel/trace**                           v1.29.0 -> v1.34.0
* **golang.org/x/crypto**                                      v0.36.0 -> v0.37.0
* **golang.org/x/mod**                                         v0.20.0 -> v0.23.0
* **golang.org/x/net**                                         v0.38.0 -> v0.39.0
* **golang.org/x/sync**                                        v0.12.0 -> v0.13.0
* **golang.org/x/sys**                                         v0.31.0 -> v0.32.0
* **golang.org/x/term**                                        v0.30.0 -> v0.31.0
* **golang.org/x/text**                                        v0.23.0 -> v0.24.0
* **golang.org/x/tools**                                       v0.24.0 -> v0.30.0
* **google.golang.org/genproto/googleapis/api**                dd2ea8efbc28 -> 5f5ef82da422
* **google.golang.org/genproto/googleapis/rpc**                dd2ea8efbc28 -> 1a7da9e5054f
* **google.golang.org/grpc**                                   v1.67.1 -> v1.71.1
* **google.golang.org/protobuf**                               v1.35.1 -> v1.36.6
* **gopkg.in/evanphx/json-patch.v4**                           v4.12.0 **_new_**
* **k8s.io/api**                                               v0.31.1 -> v0.32.3
* **k8s.io/apiextensions-apiserver**                           v0.31.1 -> v0.32.3
* **k8s.io/apimachinery**                                      v0.31.1 -> v0.32.3
* **k8s.io/apiserver**                                         v0.31.1 -> v0.32.3
* **k8s.io/client-go**                                         v0.31.1 -> v0.32.3
* **k8s.io/component-base**                                    v0.31.1 -> v0.32.3
* **k8s.io/kube-openapi**                                      f0e62f92d13f -> 32ad38e42d3f
* **k8s.io/utils**                                             18e509b52bc8 -> 3ea5e8cea738
* **sigs.k8s.io/apiserver-network-proxy/konnectivity-client**  v0.30.3 -> v0.31.0
* **sigs.k8s.io/controller-runtime**                           v0.19.0 -> v0.20.4
* **sigs.k8s.io/gateway-api**                                  v1.2.0 -> v1.3.0
* **sigs.k8s.io/json**                                         bc3834ca7abd -> 9aa6b5e7a4b3
* **sigs.k8s.io/structured-merge-diff/v4**                     v4.4.1 -> v4.7.0

Previous release can be found at [2.0.0-rc4](https://github.com/apache/apisix-ingress-controller/releases/tag/2.0.0-rc4)

# 2.0.0-rc4

apisix-ingress-controller 2.0.0-rc4

Welcome to the v2.0.0-rc4 release of apisix-ingress-controller!

*This is a pre-release of apisix-ingress-controller*

This is a release candidate (RC) version.

## Highlights

In APISIX Standalone mode, launching an ADC process for each endpoint causes high startup overhead that grows with the number of endpoints. ADC Server mode addresses this by running as a persistent service, reducing CPU cost and improving synchronization efficiency.

### Features

* feat: support adc server mode [#2520](https://github.com/apache/apisix-ingress-controller/pull/2520)

### Bugfixes

* fix: set websocket when passed true and add websocket e2e test [#2497](https://github.com/apache/apisix-ingress-controller/pull/2497)  
* fix: deadlock occurs when updating configuration fails [#2531](https://github.com/apache/apisix-ingress-controller/pull/2531)  
* fix: traffic-split weight distribution and add e2e tests [#2495](https://github.com/apache/apisix-ingress-controller/pull/2495)  
* fix: list is missing index parameter [#2513](https://github.com/apache/apisix-ingress-controller/pull/2513)  
* fix: status should not be recorded when ingressclass does not match [#2519](https://github.com/apache/apisix-ingress-controller/pull/2519)  
* fix: support tlsSecret from http.backends in ApisixRoute [#2518](https://github.com/apache/apisix-ingress-controller/pull/2518)

Please try out the release binaries and report any issues at
https://github.com/apache/apisix-ingress-controller/issues.

### Contributors

* Ashish Tiwari
* AlinsRan
* Traky Deng
* iliya
* 琚致远 / Zhiyuan Ju

### Changes
<details><summary>23 commits</summary>
<p>

  * [`07672cce`](https://github.com/apache/apisix-ingress-controller/commit/07672ccea3d4082fc6371ffea0691359178f6b83) fix: deadlock occurs when updating configuration fails (#2531)
  * [`ba39a7ac`](https://github.com/apache/apisix-ingress-controller/commit/ba39a7ac7e804519ada390c88af6c615c5ef5809) chore: upgrade adc to 0.21.0 (#2532)
  * [`8c5f0dcb`](https://github.com/apache/apisix-ingress-controller/commit/8c5f0dcb2e8b2a249fb6402c7671b672a31d13fd) chore: migrate redirect plugin e2e tests (#2529)
  * [`69db98c4`](https://github.com/apache/apisix-ingress-controller/commit/69db98c4f8db674ddb15c746438963b684e91fcd) chore: remove adc binary from dockerfile (#2530)
  * [`75d068aa`](https://github.com/apache/apisix-ingress-controller/commit/75d068aaeae516da5a19055566be1af8672b2cf2) feat: support adc server mode (#2520)
  * [`1faf2ae4`](https://github.com/apache/apisix-ingress-controller/commit/1faf2ae472ff16b102654a609bb131aa876f5c70) chore: migrate e2e tests for httproute basic (#2525)
  * [`2a798d13`](https://github.com/apache/apisix-ingress-controller/commit/2a798d13fc5f185c289c0e1327676a7e2efc29c9) fix(test): Unstable controllername assertion (#2523)
  * [`e5d831e4`](https://github.com/apache/apisix-ingress-controller/commit/e5d831e41731d5525fbc12db319465d4283eacd7) chore: remove redundant backend traffic policy attachment (#2524)
  * [`eb7c06a6`](https://github.com/apache/apisix-ingress-controller/commit/eb7c06a6f527df1b756ea50ece2bc0fa7dbc6c4e) chore: migrate retries/timeout tests for apisixupstream (#2517)
  * [`404d1508`](https://github.com/apache/apisix-ingress-controller/commit/404d15087f425cf56400aa732ab66494c98d85c6) docs: mention stream is currently not supported in the CRD docs (#2522)
  * [`6bc3731a`](https://github.com/apache/apisix-ingress-controller/commit/6bc3731a5c7fee65fe7e1f141484272843ab2bfa) fix: support tlsSecret from http.backends in ApisixRoute (#2518)
  * [`227062d2`](https://github.com/apache/apisix-ingress-controller/commit/227062d2a8862560bc3c1fa33e99ab119bfec5ca) fix: status should not be recorded when ingressclass does not match (#2519)
  * [`5775f23a`](https://github.com/apache/apisix-ingress-controller/commit/5775f23a8385aa1d109f1b3e8e51d2727b26c172) fix: list is missing index parameter (#2513)
  * [`95787e6e`](https://github.com/apache/apisix-ingress-controller/commit/95787e6e68309f136cda80b546e4cbe8b5bccffc) chore: refactor provider (#2507)
  * [`c9ead0ee`](https://github.com/apache/apisix-ingress-controller/commit/c9ead0eef4248107626edbd9fa27dd35e421423b) fix indentation (#2512)
  * [`ce0c5f4c`](https://github.com/apache/apisix-ingress-controller/commit/ce0c5f4c2d6c5e04c3c069696f6c726003a51216) refactor: E2E tests to support parallel tests (#2501)
  * [`77b8210c`](https://github.com/apache/apisix-ingress-controller/commit/77b8210c8084ba05cac00016d60a79c8327e268e) docs: update load balancing Gateway API doc for RC3 fix (#2506)
  * [`7a435c97`](https://github.com/apache/apisix-ingress-controller/commit/7a435c978e5eec0a51e5a3f78bef84bfb80578d3) chore(deps): bump golang.org/x/oauth2 from 0.24.0 to 0.27.0 (#2485)
  * [`40712363`](https://github.com/apache/apisix-ingress-controller/commit/40712363d01ee9394eeafc9b2ae8fe0d7e2caa44) chore(test): Refactor loop to use range over integer in test (#2494)
  * [`ac5e56dd`](https://github.com/apache/apisix-ingress-controller/commit/ac5e56dd5e44142370ee54b72876faa20c098b5f) chore: add test cases for external service (#2500)
  * [`a2bea453`](https://github.com/apache/apisix-ingress-controller/commit/a2bea453adf4ea36f2bde5f59e18cb79e7aebf85) docs: fix links (#2502)
  * [`49ef9d40`](https://github.com/apache/apisix-ingress-controller/commit/49ef9d4028362ce6e7bd202093440ca9ac381fa2) fix: set websocket when passed true and add websocket e2e test (#2497)
  * [`a35cad5e`](https://github.com/apache/apisix-ingress-controller/commit/a35cad5e78e7f15ffc5a9a86b9da246184af34cf) fix: traffic-split weight distribution and add e2e tests (#2495)
</p>
</details>

### Dependency Changes

* **golang.org/x/oauth2**  v0.24.0 -> v0.27.0

Previous release can be found at [2.0.0-rc3](https://github.com/apache/apisix-ingress-controller/releases/tag/2.0.0-rc3)

# 2.0.0-rc3

apisix-ingress-controller 2.0.0-rc3

Welcome to the 2.0.0-rc3 release of apisix-ingress-controller!

This is a release candidate (RC) version.

## Highlights

### Features

* feat: support custom metrics [#2480](https://github.com/apache/apisix-ingress-controller/pull/2480)  
* feat: support event triggered synchronization [#2478](https://github.com/apache/apisix-ingress-controller/pull/2478)

### Bugfixes

* fix: route names with the same prefix were mistakenly deleted [#2472](https://github.com/apache/apisix-ingress-controller/pull/2472)  
* fix: should not return when service type is ExternalName [#2468](https://github.com/apache/apisix-ingress-controller/pull/2468)  
* fix: remove duplicate sync func [#2476](https://github.com/apache/apisix-ingress-controller/pull/2476)  
* fix: full sync during restart results in loss of dataplane traffic [#2489](https://github.com/apache/apisix-ingress-controller/pull/2489)

Please try out the release binaries and report any issues at
https://github.com/apache/apisix-ingress-controller/issues.

### Contributors

* AlinsRan
* Ashing Zheng
* Traky Deng

### Changes
<details><summary>14 commits</summary>
<p>

  * [`66c2b0a`](https://github.com/apache/apisix-ingress-controller/commit/66c2b0acf14fc13f461e5f262753bdc0598f5d1a) fix: full sync during restart results in loss of dataplane traffic (#2489)
  * [`f6196ff`](https://github.com/apache/apisix-ingress-controller/commit/f6196ff50c20d0564b79179f668673e9b5582c7d) chore: differentiate the API versions for CRD testing (#2492)
  * [`7a6151c`](https://github.com/apache/apisix-ingress-controller/commit/7a6151ce2f175717637ac95418a050191a58ad61) feat: support event triggered synchronization (#2478)
  * [`38023b2`](https://github.com/apache/apisix-ingress-controller/commit/38023b272c45a142cb46c95733f3585ce4495636) fix: doc broken links (#2490)
  * [`7ede0e3`](https://github.com/apache/apisix-ingress-controller/commit/7ede0e3aa3c10e8541b5df5b0e2a95d3fe93e16d) docs: update getting started docs (RC2) (#2481)
  * [`eb7a65b`](https://github.com/apache/apisix-ingress-controller/commit/eb7a65b31036629eb97ebc73001982296ad10f6b) chore: update status only when changes occur (#2473)
  * [`f02e350`](https://github.com/apache/apisix-ingress-controller/commit/f02e35086f1cf647dff04b5492b2511c0aab1af0) docs: fix description error in upgrade doc (#2440)
  * [`94fcceb`](https://github.com/apache/apisix-ingress-controller/commit/94fcceb78ad816b0e07a5d8a0f18097318262b8e) feat: support custom metrics (#2480)
  * [`1156414`](https://github.com/apache/apisix-ingress-controller/commit/1156414fc1b8de6b29009059ecd58749454548ee) fix: remove duplicate sync func (#2476)
  * [`f536c26`](https://github.com/apache/apisix-ingress-controller/commit/f536c26c918e42b4de416eda1c4ffa95e9937a55) chore: refactor e2e-test (#2467)
  * [`4745958`](https://github.com/apache/apisix-ingress-controller/commit/4745958edf3d7a62155d51acabbb80f288f3982b) fix: route names with the same prefix were mistakenly deleted (#2472)
  * [`2b9b787`](https://github.com/apache/apisix-ingress-controller/commit/2b9b787a9397ed9348ccfa9eb8d568acc671f6fd) fix: should not return when service type is ExternalName (#2468)
  * [`d91a3ba`](https://github.com/apache/apisix-ingress-controller/commit/d91a3ba9946bd3206d78588d3e29e1a22d5a66fd) doc: recommended to use apisix-standalone mode for installation. (#2470)
  * [`0a4e05c`](https://github.com/apache/apisix-ingress-controller/commit/0a4e05c009e4786e36c896666c3a5ffb40112aca) chore(ci): remove add-pr-comment step (#2463)
</p>
</details>

### Dependency Changes

* **filippo.io/edwards25519**                                           v1.1.0 **_new_**
* **github.com/aws/aws-sdk-go-v2**                                      v1.32.5 **_new_**
* **github.com/aws/aws-sdk-go-v2/aws/protocol/eventstream**             v1.6.7 **_new_**
* **github.com/aws/aws-sdk-go-v2/config**                               v1.28.5 **_new_**
* **github.com/aws/aws-sdk-go-v2/credentials**                          v1.17.46 **_new_**
* **github.com/aws/aws-sdk-go-v2/feature/ec2/imds**                     v1.16.20 **_new_**
* **github.com/aws/aws-sdk-go-v2/feature/s3/manager**                   v1.17.41 **_new_**
* **github.com/aws/aws-sdk-go-v2/internal/configsources**               v1.3.24 **_new_**
* **github.com/aws/aws-sdk-go-v2/internal/endpoints/v2**                v2.6.24 **_new_**
* **github.com/aws/aws-sdk-go-v2/internal/ini**                         v1.8.1 **_new_**
* **github.com/aws/aws-sdk-go-v2/internal/v4a**                         v1.3.24 **_new_**
* **github.com/aws/aws-sdk-go-v2/service/acm**                          v1.30.6 **_new_**
* **github.com/aws/aws-sdk-go-v2/service/autoscaling**                  v1.51.0 **_new_**
* **github.com/aws/aws-sdk-go-v2/service/cloudwatchlogs**               v1.44.0 **_new_**
* **github.com/aws/aws-sdk-go-v2/service/dynamodb**                     v1.37.1 **_new_**
* **github.com/aws/aws-sdk-go-v2/service/ec2**                          v1.193.0 **_new_**
* **github.com/aws/aws-sdk-go-v2/service/ecr**                          v1.36.6 **_new_**
* **github.com/aws/aws-sdk-go-v2/service/ecs**                          v1.52.0 **_new_**
* **github.com/aws/aws-sdk-go-v2/service/iam**                          v1.38.1 **_new_**
* **github.com/aws/aws-sdk-go-v2/service/internal/accept-encoding**     v1.12.1 **_new_**
* **github.com/aws/aws-sdk-go-v2/service/internal/checksum**            v1.4.5 **_new_**
* **github.com/aws/aws-sdk-go-v2/service/internal/endpoint-discovery**  v1.10.5 **_new_**
* **github.com/aws/aws-sdk-go-v2/service/internal/presigned-url**       v1.12.5 **_new_**
* **github.com/aws/aws-sdk-go-v2/service/internal/s3shared**            v1.18.5 **_new_**
* **github.com/aws/aws-sdk-go-v2/service/kms**                          v1.37.6 **_new_**
* **github.com/aws/aws-sdk-go-v2/service/lambda**                       v1.69.0 **_new_**
* **github.com/aws/aws-sdk-go-v2/service/rds**                          v1.91.0 **_new_**
* **github.com/aws/aws-sdk-go-v2/service/route53**                      v1.46.2 **_new_**
* **github.com/aws/aws-sdk-go-v2/service/s3**                           v1.69.0 **_new_**
* **github.com/aws/aws-sdk-go-v2/service/secretsmanager**               v1.34.6 **_new_**
* **github.com/aws/aws-sdk-go-v2/service/sns**                          v1.33.6 **_new_**
* **github.com/aws/aws-sdk-go-v2/service/sqs**                          v1.37.1 **_new_**
* **github.com/aws/aws-sdk-go-v2/service/ssm**                          v1.56.0 **_new_**
* **github.com/aws/aws-sdk-go-v2/service/sso**                          v1.24.6 **_new_**
* **github.com/aws/aws-sdk-go-v2/service/ssooidc**                      v1.28.5 **_new_**
* **github.com/aws/aws-sdk-go-v2/service/sts**                          v1.33.1 **_new_**
* **github.com/aws/smithy-go**                                          v1.22.1 **_new_**
* **github.com/cpuguy83/go-md2man/v2**                                  v2.0.4 -> v2.0.5
* **github.com/go-sql-driver/mysql**                                    v1.7.1 -> v1.8.1
* **github.com/gruntwork-io/terratest**                                 v0.47.0 -> v0.50.0
* **github.com/jackc/pgpassfile**                                       v1.0.0 **_new_**
* **github.com/jackc/pgservicefile**                                    5a60cdf6a761 **_new_**
* **github.com/jackc/pgx/v5**                                           v5.7.1 **_new_**
* **github.com/jackc/puddle/v2**                                        v2.2.2 **_new_**
* **github.com/pquerna/otp**                                            v1.2.0 -> v1.4.0
* **github.com/stretchr/testify**                                       v1.9.0 -> v1.10.0
* **github.com/urfave/cli**                                             v1.22.14 -> v1.22.16
* **go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp**     v0.53.0 -> v0.54.0
* **go.opentelemetry.io/otel**                                          v1.28.0 -> v1.29.0
* **go.opentelemetry.io/otel/metric**                                   v1.28.0 -> v1.29.0
* **go.opentelemetry.io/otel/sdk**                                      v1.28.0 -> v1.29.0
* **go.opentelemetry.io/otel/trace**                                    v1.28.0 -> v1.29.0
* **golang.org/x/oauth2**                                               v0.21.0 -> v0.24.0
* **golang.org/x/time**                                                 v0.5.0 -> v0.8.0
* **google.golang.org/genproto/googleapis/api**                         ef581f913117 -> dd2ea8efbc28
* **google.golang.org/genproto/googleapis/rpc**                         f6361c86f094 -> dd2ea8efbc28
* **google.golang.org/grpc**                                            v1.66.2 -> v1.67.1
* **google.golang.org/protobuf**                                        v1.34.2 -> v1.35.1

Previous release can be found at [2.0.0-rc2](https://github.com/apache/apisix-ingress-controller/releases/tag/2.0.0-rc2)

# 2.0.0-rc2

apisix-ingress-controller 2.0.0-rc2

Welcome to the 2.0.0-rc2 release of apisix-ingress-controller!

This is a Patch version release.

## Highlights

### Features

* feat: support gatewayproxy controller to discovery of dataplane endpoints [#2444](https://github.com/apache/apisix-ingress-controller/pull/2444)
* feat: add synchronization status to CRD [#2460](https://github.com/apache/apisix-ingress-controller/pull/2460)

### Bugfix

* fix: should not contain plaintext token in log message [#2462](https://github.com/apache/apisix-ingress-controller/pull/2462)
* fix: add more event filter across controllers [#2449](https://github.com/apache/apisix-ingress-controller/pull/2449)
* fix: a failing endpoint shouldn't affect others [#2452](https://github.com/apache/apisix-ingress-controller/pull/2452)
* fix: Add provider endpoints to translate context [#2442](https://github.com/apache/apisix-ingress-controller/pull/2442)
* fix: config not provided should not be retried [#2454](https://github.com/apache/apisix-ingress-controller/pull/2454)
* fix: apisixroute backend service reference to apisixupstream [#2453](https://github.com/apache/apisix-ingress-controller/pull/2453)
* fix: adc backend server on different mode [#2455](https://github.com/apache/apisix-ingress-controller/pull/2455)
* fix: support filter endpoint when translate backend ref [#2451](https://github.com/apache/apisix-ingress-controller/pull/2451)
* fix: reduce the complexity of calculating route priority [#2459](https://github.com/apache/apisix-ingress-controller/pull/2459)

Please try out the release binaries and report any issues at
https://github.com/apache/apisix-ingress-controller/issues.

### Contributors

* AlinsRan
* Ashing Zheng
* 悟空

### Changes
<details><summary>20 commits</summary>
<p>

  * [`6c5847c`](https://github.com/apache/apisix-ingress-controller/commit/6c5847c05d35c75ac691e587838061dc74089295) fix: should not contain plaintext token in log message. (#2462)
  * [`cdc6b38`](https://github.com/apache/apisix-ingress-controller/commit/cdc6b38615a41570bd1a13feff453155a5dc90c5) chore: unified logger print (#2456)
  * [`9d7e018`](https://github.com/apache/apisix-ingress-controller/commit/9d7e01831f5b1b1e97bdaa522ff8a5161ba19290) feat: add synchronization status to CRD (#2460)
  * [`df2362b`](https://github.com/apache/apisix-ingress-controller/commit/df2362b0c535fe1455cf09f3216dd92bd7a2ac21) doc: add getting-started doc (#2450)
  * [`d317e5e`](https://github.com/apache/apisix-ingress-controller/commit/d317e5e29ef24eca7cde6c4c73f2fb0138887f31) fix: reduce the complexity of calculating route priority (#2459)
  * [`3a017c7`](https://github.com/apache/apisix-ingress-controller/commit/3a017c79c9848a9e9e7060fc325b605df7aad05f) fix: adc backend server on different mode (#2455)
  * [`43bbe76`](https://github.com/apache/apisix-ingress-controller/commit/43bbe76801dd0b1243221277b91fc713bac1d157) fix: apisixroute backend service reference to apisixupstream (#2453)
  * [`4f22fb6`](https://github.com/apache/apisix-ingress-controller/commit/4f22fb6a4cf195a77cfd488603bf2929d3f24f44) fix: config not provided should not be retried (#2454)
  * [`bce4c69`](https://github.com/apache/apisix-ingress-controller/commit/bce4c6963c54d4c97ad3eef6f09d2d85e475dbce) fix: a failing endpoint shouldn't affect others (#2452)
  * [`d8be46e`](https://github.com/apache/apisix-ingress-controller/commit/d8be46e71934beec746356e105f1fad2e5a3a580) fix: support filter endpoint when translate backend ref. (#2451)
  * [`18f03ea`](https://github.com/apache/apisix-ingress-controller/commit/18f03ea661205b2da69170d8a40470fe791a67c9) fix: add more event filter across controllers (#2449)
  * [`dc03c31`](https://github.com/apache/apisix-ingress-controller/commit/dc03c314bff2bfeec1d10ecb1923ea48c4ae5a01) chore: Update artifact and report names with provider type (#2447)
  * [`634bc52`](https://github.com/apache/apisix-ingress-controller/commit/634bc5224c059c2958038f9a49530d65f779d029) feat(ci): support build dev image (#2448)
  * [`6352263`](https://github.com/apache/apisix-ingress-controller/commit/635226396969b939e91bd3a05f5825252c1a5686) feat: gatewayproxy controller (#2444)
  * [`40a2d2c`](https://github.com/apache/apisix-ingress-controller/commit/40a2d2c72398168ac0d863e9f2f9539cb265b3d7) doc: add config.json (#2446)
  * [`5d20cec`](https://github.com/apache/apisix-ingress-controller/commit/5d20cec8d329a039edc0ecc93ea0cdfe4ac7d80b) docs: add install and developer-guide doc (#2439)
  * [`66e87fc`](https://github.com/apache/apisix-ingress-controller/commit/66e87fcfcfac422457661959d71f97a8f783f2a9) docs: remove unless commit in changelog 200-rc1 (#2441)
  * [`fdcc436`](https://github.com/apache/apisix-ingress-controller/commit/fdcc4360f0fc51852f4dc0df22fca171416eb381) chore: move generate-crd to assets (#2445)
  * [`dfc76d6`](https://github.com/apache/apisix-ingress-controller/commit/dfc76d6fc8a83c24e327026e8abb0d125fbe15fb) fix: Add provider endpoints to translate context (#2442)
  * [`409a474`](https://github.com/apache/apisix-ingress-controller/commit/409a474ffec94d7db3f2b5187d42124b205fce40) chore: move doc to en/latest directory (#2443)
</p>
</details>

### Dependency Changes

This release has no dependency changes

Previous release can be found at [2.0.0-rc1](https://github.com/apache/apisix-ingress-controller/releases/tag/2.0.0-rc1)

# 2.0.0-rc1

apisix-ingress-controller 2.0.0-rc1

Welcome to the 2.0.0-rc1 release of apisix-ingress-controller!

This is a feature release.

## Highlights

**Add Gateway API Extensions `apisix.apache.org/v1alpha1`**

Enable additional features not included in the standard Kubernetes Gateway API, developed and maintained by Gateway API implementers to extend functionality securely and reliably.

* GatewayProxy: Defines connection settings between the APISIX Ingress Controller and APISIX, including auth, endpoints, and global plugins. Referenced via parametersRef in Gateway, GatewayClass, or IngressClass

* BackendTrafficPolicy: Defines traffic management settings for backend services, including load balancing, timeouts, retries, and host header handling in the APISIX Ingress Controller.

* Consumer: Defines API consumers and their credentials, enabling authentication and plugin configuration for controlling access to API endpoints.

* PluginConfig: Defines reusable plugin configurations that can be referenced by other resources like HTTPRoute, enabling separation of routing logic and plugin settings for better reusability and manageability.

* HTTPRoutePolicy: Configures advanced traffic management and routing policies for HTTPRoute or Ingress resources, enhancing functionality without modifying the original resources.

**Support APISIX Standalone API-Driven Mode (Experimental)**

This new implementation addresses the issue of ETCD instability in Kubernetes, removing the need for ETCD support. Routing rules are now stored entirely in memory and can be updated through the API. This change allows you to run Ingress Controllers more reliably in a stateless mode.

You can enable this mode in APISIX Ingress Controller configuration file by specifying:

```yaml
provider:
  type: "apisix-standalone"
```

**For major changes introduced in this release, refer to the [upgrade guide](https://github.com/apache/apisix-ingress-controller/blob/0db882d66d5b9dfb7dc9dd9d2045d4709b1c6ed2/docs/upgrade-guide.md#upgrading-from-1xx-to-200-key-changes-and-considerations).**

If you encounter any problems while using the implementation, please [submit an issue](https://github.com/apache/apisix-ingress-controller/issues) along with the reproduction steps. The APISIX Team will review and resolve it.

### Contributors

* AlinsRan
* Ashing Zheng

### Changes
<details><summary>12 commits</summary>
<p>

  * [`c1533c9`](https://github.com/apache/apisix-ingress-controller/commit/c1533c9ddf4b1ed6db999d1535370824b8c150e1) fix: the sync_period of the provider should not be 0s (#2438)
  * [`0db882d`](https://github.com/apache/apisix-ingress-controller/commit/0db882d66d5b9dfb7dc9dd9d2045d4709b1c6ed2) chore: remove useless example files in dockerfile (#2434)
  * [`11ecb35`](https://github.com/apache/apisix-ingress-controller/commit/11ecb353d074b7392046d08e52bc824a3eeb6ee7) fix: set default provider type (#2436)
  * [`16f9d60`](https://github.com/apache/apisix-ingress-controller/commit/16f9d609ad63a9ff1d11aa1d1dfceaf89a603a60) fix(crd): missing shortname and printcolumn (#2435)
  * [`e6fa3b8`](https://github.com/apache/apisix-ingress-controller/commit/e6fa3b845ed30a077d2f2235790701d9653e0403) chore: upgrade adc to 0.20.0 (#2432)
  * [`03877e0`](https://github.com/apache/apisix-ingress-controller/commit/03877e06abbdf8fda712c65a9f0f6613bdbf5f59) fix(ci): run e2e group by resource api group (#2431)
  * [`b21d429`](https://github.com/apache/apisix-ingress-controller/commit/b21d429a5efea0571bb0e9f4b5a1633e578d0ce9) chore: revert release-src cmd in makefile (#2433)
  * [`5588c00`](https://github.com/apache/apisix-ingress-controller/commit/5588c00f116d86daea268a43b3adfc1023ad6a03) docs: update resources and overview (#2430)
  * [`67ad69a`](https://github.com/apache/apisix-ingress-controller/commit/67ad69ab2fe84cc439b0a95dd20132108b596a60) chore: remove charts folder (#2428)
  * [`c7d7732`](https://github.com/apache/apisix-ingress-controller/commit/c7d77325a46d9c158e21a0562db7164c7fa34bd9) chore: remove useless provider (#2429)
  * [`cfa8fd5`](https://github.com/apache/apisix-ingress-controller/commit/cfa8fd5159ef8c899dfc7d311365e26c6f2392e1) feat: support apisix provider type and add ingress docs (#2427)
  * [`756ed51`](https://github.com/apache/apisix-ingress-controller/commit/756ed51df778d44b61df7e5c3b78bd2dd9c8afbe) refactor: new apisix ingress controller (#2421)
</p>
</details>

### Dependency Changes

* **github.com/Masterminds/goutils**                                   v1.1.1 **_new_**
* **github.com/Masterminds/semver/v3**                                 v3.2.1 **_new_**
* **github.com/Masterminds/sprig/v3**                                  v3.2.3 **_new_**
* **github.com/TylerBrock/colorjson**                                  8a50f05110d2 **_new_**
* **github.com/ajg/form**                                              v1.5.1 **_new_**
* **github.com/andybalholm/brotli**                                    v1.0.4 **_new_**
* **github.com/antlr4-go/antlr/v4**                                    v4.13.0 **_new_**
* **github.com/api7/gopkg**                                            v0.2.0 -> 0f3730f9b57a
* **github.com/asaskevich/govalidator**                                a9d515a09cc2 **_new_**
* **github.com/aws/aws-sdk-go**                                        v1.44.245 **_new_**
* **github.com/blang/semver/v4**                                       v4.0.0 **_new_**
* **github.com/boombuler/barcode**                                     6c824513bacc **_new_**
* **github.com/cenkalti/backoff/v4**                                   v4.3.0 **_new_**
* **github.com/cespare/xxhash/v2**                                     v2.2.0 -> v2.3.0
* **github.com/cpuguy83/go-md2man/v2**                                 v2.0.4 **_new_**
* **github.com/davecgh/go-spew**                                       v1.1.1 -> d8f796af33cc
* **github.com/emicklei/go-restful/v3**                                v3.10.2 -> v3.12.0
* **github.com/evanphx/json-patch**                                    v5.6.0 -> v5.9.0
* **github.com/evanphx/json-patch/v5**                                 v5.6.0 -> v5.9.0
* **github.com/fatih/color**                                           v1.17.0 **_new_**
* **github.com/fatih/structs**                                         v1.1.0 **_new_**
* **github.com/felixge/httpsnoop**                                     v1.0.4 **_new_**
* **github.com/fsnotify/fsnotify**                                     v1.7.0 **_new_**
* **github.com/fxamacker/cbor/v2**                                     v2.7.0 **_new_**
* **github.com/gavv/httpexpect/v2**                                    v2.16.0 **_new_**
* **github.com/go-errors/errors**                                      v1.4.2 **_new_**
* **github.com/go-logr/logr**                                          v1.2.4 -> v1.4.2
* **github.com/go-logr/stdr**                                          v1.2.2 **_new_**
* **github.com/go-logr/zapr**                                          v1.3.0 **_new_**
* **github.com/go-openapi/jsonpointer**                                v0.20.0 -> v0.21.0
* **github.com/go-openapi/jsonreference**                              v0.20.2 -> v0.21.0
* **github.com/go-openapi/swag**                                       v0.22.4 -> v0.23.0
* **github.com/go-sql-driver/mysql**                                   v1.7.1 **_new_**
* **github.com/go-task/slim-sprig/v3**                                 v3.0.0 **_new_**
* **github.com/gobwas/glob**                                           v0.2.3 **_new_**
* **github.com/golang/protobuf**                                       v1.5.3 -> v1.5.4
* **github.com/google/cel-go**                                         v0.20.1 **_new_**
* **github.com/google/go-cmp**                                         v0.5.9 -> v0.6.0
* **github.com/google/go-querystring**                                 v1.1.0 **_new_**
* **github.com/google/pprof**                                          813a5fbdbec8 **_new_**
* **github.com/google/uuid**                                           v1.3.0 -> v1.6.0
* **github.com/gorilla/websocket**                                     v1.5.0 -> v1.5.1
* **github.com/grpc-ecosystem/grpc-gateway/v2**                        v2.20.0 **_new_**
* **github.com/gruntwork-io/go-commons**                               v0.8.0 **_new_**
* **github.com/gruntwork-io/terratest**                                v0.47.0 **_new_**
* **github.com/hashicorp/errwrap**                                     v1.0.0 -> v1.1.0
* **github.com/hashicorp/go-uuid**                                     v1.0.1 **_new_**
* **github.com/hashicorp/golang-lru**                                  v0.5.4 -> v1.0.2
* **github.com/hpcloud/tail**                                          v1.0.0 **_new_**
* **github.com/huandu/xstrings**                                       v1.4.0 **_new_**
* **github.com/imdario/mergo**                                         v0.3.15 -> v0.3.16
* **github.com/imkira/go-interpol**                                    v1.1.0 **_new_**
* **github.com/jmespath/go-jmespath**                                  v0.4.0 **_new_**
* **github.com/klauspost/compress**                                    v1.17.4 **_new_**
* **github.com/mattn/go-colorable**                                    v0.1.13 **_new_**
* **github.com/mattn/go-isatty**                                       v0.0.19 -> v0.0.20
* **github.com/mattn/go-zglob**                                        e3c945676326 **_new_**
* **github.com/miekg/dns**                                             v1.1.62 **_new_**
* **github.com/mitchellh/copystructure**                               v1.2.0 **_new_**
* **github.com/mitchellh/go-homedir**                                  v1.1.0 **_new_**
* **github.com/mitchellh/go-wordwrap**                                 v1.0.1 **_new_**
* **github.com/mitchellh/reflectwalk**                                 v1.0.2 **_new_**
* **github.com/moby/spdystream**                                       v0.4.0 **_new_**
* **github.com/mxk/go-flowrate**                                       cca7078d478f **_new_**
* **github.com/onsi/ginkgo/v2**                                        v2.20.0 **_new_**
* **github.com/onsi/gomega**                                           v1.34.1 **_new_**
* **github.com/pmezard/go-difflib**                                    v1.0.0 -> 5d4384ee4fb2
* **github.com/pquerna/otp**                                           v1.2.0 **_new_**
* **github.com/prometheus/client_golang**                              v1.16.0 -> v1.19.1
* **github.com/prometheus/client_model**                               v0.4.0 -> v0.6.1
* **github.com/prometheus/common**                                     v0.44.0 -> v0.55.0
* **github.com/prometheus/procfs**                                     v0.11.1 -> v0.15.1
* **github.com/russross/blackfriday/v2**                               v2.1.0 **_new_**
* **github.com/samber/lo**                                             v1.47.0 **_new_**
* **github.com/sanity-io/litter**                                      v1.5.5 **_new_**
* **github.com/sergi/go-diff**                                         v1.3.1 **_new_**
* **github.com/shopspring/decimal**                                    v1.3.1 **_new_**
* **github.com/spf13/cast**                                            v1.6.0 **_new_**
* **github.com/spf13/cobra**                                           v1.8.0 -> v1.8.1
* **github.com/stoewer/go-strcase**                                    v1.2.0 **_new_**
* **github.com/stretchr/testify**                                      v1.8.4 -> v1.9.0
* **github.com/urfave/cli**                                            v1.22.14 **_new_**
* **github.com/valyala/bytebufferpool**                                v1.0.0 **_new_**
* **github.com/valyala/fasthttp**                                      v1.34.0 **_new_**
* **github.com/x448/float16**                                          v0.8.4 **_new_**
* **github.com/xeipuuv/gojsonpointer**                                 4e3ac2762d5f -> 02993c407bfb
* **github.com/yalp/jsonpath**                                         5cc68e5049a0 **_new_**
* **github.com/yudai/gojsondiff**                                      v1.0.0 **_new_**
* **github.com/yudai/golcs**                                           ecda9a501e82 **_new_**
* **go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp**    v0.53.0 **_new_**
* **go.opentelemetry.io/otel**                                         v1.28.0 **_new_**
* **go.opentelemetry.io/otel/exporters/otlp/otlptrace**                v1.28.0 **_new_**
* **go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracegrpc**  v1.27.0 **_new_**
* **go.opentelemetry.io/otel/metric**                                  v1.28.0 **_new_**
* **go.opentelemetry.io/otel/sdk**                                     v1.28.0 **_new_**
* **go.opentelemetry.io/otel/trace**                                   v1.28.0 **_new_**
* **go.opentelemetry.io/proto/otlp**                                   v1.3.1 **_new_**
* **go.uber.org/zap**                                                  v1.26.0 -> v1.27.0
* **golang.org/x/arch**                                                v0.3.0 -> v0.6.0
* **golang.org/x/crypto**                                              v0.14.0 -> v0.36.0
* **golang.org/x/exp**                                                 8a7402abbf56 **_new_**
* **golang.org/x/mod**                                                 v0.12.0 -> v0.20.0
* **golang.org/x/net**                                                 v0.17.0 -> v0.38.0
* **golang.org/x/oauth2**                                              v0.8.0 -> v0.21.0
* **golang.org/x/sync**                                                v0.12.0 **_new_**
* **golang.org/x/sys**                                                 v0.13.0 -> v0.31.0
* **golang.org/x/term**                                                v0.13.0 -> v0.30.0
* **golang.org/x/text**                                                v0.13.0 -> v0.23.0
* **golang.org/x/time**                                                v0.3.0 -> v0.5.0
* **golang.org/x/tools**                                               v0.12.0 -> v0.24.0
* **gomodules.xyz/jsonpatch/v2**                                       v2.4.0 **_new_**
* **google.golang.org/genproto/googleapis/api**                        6bfd019c3878 -> ef581f913117
* **google.golang.org/genproto/googleapis/rpc**                        6bfd019c3878 -> f6361c86f094
* **google.golang.org/grpc**                                           v1.57.0 -> v1.66.2
* **google.golang.org/protobuf**                                       v1.31.0 -> v1.34.2
* **gopkg.in/fsnotify.v1**                                             v1.4.7 **_new_**
* **gopkg.in/tomb.v1**                                                 dd632973f1e7 **_new_**
* **k8s.io/api**                                                       v0.28.4 -> v0.31.1
* **k8s.io/apiextensions-apiserver**                                   v0.31.1 **_new_**
* **k8s.io/apimachinery**                                              v0.28.4 -> v0.31.1
* **k8s.io/apiserver**                                                 v0.31.1 **_new_**
* **k8s.io/client-go**                                                 v0.28.2 -> v0.31.1
* **k8s.io/component-base**                                            v0.31.1 **_new_**
* **k8s.io/klog/v2**                                                   v2.100.1 -> v2.130.1
* **k8s.io/kube-openapi**                                              2695361300d9 -> f0e62f92d13f
* **k8s.io/kubectl**                                                   v0.30.3 **_new_**
* **k8s.io/utils**                                                     d93618cff8a2 -> 18e509b52bc8
* **moul.io/http2curl/v2**                                             v2.3.0 **_new_**
* **sigs.k8s.io/apiserver-network-proxy/konnectivity-client**          v0.30.3 **_new_**
* **sigs.k8s.io/controller-runtime**                                   v0.16.2 -> v0.19.0
* **sigs.k8s.io/gateway-api**                                          v0.8.0 -> v1.2.0
* **sigs.k8s.io/structured-merge-diff/v4**                             v4.3.0 -> v4.4.1
* **sigs.k8s.io/yaml**                                                 v1.3.0 -> v1.4.0

Previous release can be found at [v1.8.4](https://github.com/apache/apisix-ingress-controller/releases/tag/v1.8.4)

# 1.8.0

## What's New

- docs: update keys based helm chart version @Revolyssup (#2085)
- feat: add `skip_mtls_uri_regex` support for ApisixTls @aynp (#1915)
- feat: add support for multiple labels with same key @Revolyssup (#2099)
- feat: Allow merging nested values in plugin config secretRef @Revolyssup (#2096)
- feat: allow configuring timeout and retries for upstream with ingress @Revolyssup (#1876)
- ci: add workflow to push docker image @Revolyssup (#2081)
- fix: upgrade etcd-adapter @Revolyssup (#2078)
- docs: Add doc for ApisixConsumer @Revolyssup (#2074)
- fix: create unique TLS object for each item in Ingress tls @Revolyssup (#1989)
- feat: add release-drafter @Revolyssup (#2068)
- fix: replace string comparison with 64 bit int @Revolyssup (#2062)
- chore: add Revolyssup in reviewers @Revolyssup (#2059)
- chore(deps): bump google.golang.org/grpc from 1.42.0 to 1.56.3 in /test/e2e/testbackend @dependabot (#2026)
- docs: add best practice docs to avoid race condition bw kubelet and apisix @Revolyssup (#2045)
- chore: correct Makefile comments @jiangfucheng (#2038)
- Renamed field in examples according to CRD @nayavu (#2032)

## 🐛 Bug Fixes

- fix: Some CRDs missing status sub-resource @Chever-John (#1809)

## 🧰 Maintenance

- chore(deps): bump github.com/onsi/ginkgo/v2 from 2.13.1 to 2.13.2 in /test/e2e @dependabot (#2079)
- chore(deps): bump golang.org/x/crypto from 0.14.0 to 0.17.0 in /test/e2e @dependabot (#2107)
- chore(deps): bump k8s.io/client-go from 0.28.4 to 0.29.0 in /test/e2e @dependabot (#2105)
- chore(deps): bump k8s.io/api from 0.28.2 to 0.28.4 @dependabot (#2056)
- chore(deps): bump k8s.io/client-go from 0.28.3 to 0.28.4 in /test/e2e @dependabot (#2052)
- chore(deps): bump github.com/onsi/ginkgo/v2 from 2.13.0 to 2.13.1 in /test/e2e @dependabot (#2041)
- chore(deps): bump google.golang.org/grpc from 1.57.0 to 1.57.1 in /test/e2e @dependabot (#2024)
- chore(deps): bump k8s.io/apimachinery from 0.28.3 to 0.28.4 in /test/e2e @dependabot (#2054)
- chore(deps): bump github.com/gorilla/websocket from 1.5.0 to 1.5.1 in /test/e2e @dependabot (#2035)
- chore(deps): bump k8s.io/client-go from 0.28.2 to 0.28.3 in /test/e2e @dependabot (#2016)
- chore(deps): bump k8s.io/api from 0.28.2 to 0.28.3 in /test/e2e @dependabot (#2018)

## 👨🏽‍💻 Contributors

Thank you to our contributors for making this release possible:
@Chever-John, @Revolyssup, @aynp, @dependabot, @dependabot[bot], @jiangfucheng and @nayavu

# 1.7.0

Welcome to the 1.7.0 release of apisix-ingress-controller!

This is a feature release.

## Highlights

The API version of all custom resources has been upgraded to v2 in v1.5 release. In 1.7 we removed the v2beta3 API. Please go to [#707](https://github.com/apache/apisix-ingress-controller/issues/707) for detail.

In this version we added more Gateway API support and add IngressClass support for all CRDs.

From this version, we try to add a new architecture, then user can reduce etcd of APISIX. (This feature is experimental.)

Please try out the release binaries and report any issues at
https://github.com/apache/apisix-ingress-controller/issues.

### Contributors

* dependabot[bot]
* Jintao Zhang
* Xin Rong
* Navendu Pottekkat
* Sarasa Kisaragi
* Abhishek Choudhary
* Ashish Tiwari
* Aryan
* Gallardot
* Priyansh Singh
* chengzw
* ermao
* ikatlinsky
* lsy
* Abhith Rajan
* Anh Le (Andy)
* Ashing Zheng
* AuruTus
* Basuotian
* Carson Yang
* Chever John
* Deepjyoti Barman
* Eng Zer Jun
* Fatpa
* German Lashevich
* Joanthan Chen
* John Chever
* Rishav Raj
* Traky Deng
* Tristan
* basefas
* fabriceli
* fengxsong
* harvies
* machinly
* oliver
* sakulali
* tanzhe
* tyltr

### Changes
<details><summary>171 commits</summary>
<p>

* [`7ecd088`](https://github.com/apache/apisix-ingress-controller/commit/7ecd08884591a1f376c2eaec1c75c8ae4ac753f2) chore(deps): bump sigs.k8s.io/gateway-api from 0.6.2 to 0.8.0 (#1945)
* [`2641c32`](https://github.com/apache/apisix-ingress-controller/commit/2641c3218306ca6be3db83504c50a473d7b33fe2) chore(deps): bump k8s.io/code-generator from 0.28.0 to 0.28.1 (#1949)
* [`9f54d9c`](https://github.com/apache/apisix-ingress-controller/commit/9f54d9ceee714ebf050023b8d50b24d69ec5a01f) chore(deps): bump sigs.k8s.io/controller-runtime from 0.14.6 to 0.16.1 (#1947)
* [`519fd5c`](https://github.com/apache/apisix-ingress-controller/commit/519fd5c2e6b2ae3d0815ba27cfeb9212aeb4364c) chore(deps): bump k8s.io/client-go from 0.27.4 to 0.28.1 (#1940)
* [`1466e89`](https://github.com/apache/apisix-ingress-controller/commit/1466e892e58af42ddfbb35ee2ab42e5570444dca) feat: use HOSTNAME as controller name and add default value. (#1946)
* [`0bbdc4f`](https://github.com/apache/apisix-ingress-controller/commit/0bbdc4f9f0350451686c85386e1a82cb8a00c7f8) feat: support controller as etcd server (#1803)
* [`cf88af9`](https://github.com/apache/apisix-ingress-controller/commit/cf88af9ae2ad845bbda79b6f71d36c07e8e686de) chore: add Gallardot for deps reviewer (#1942)
* [`aae52d5`](https://github.com/apache/apisix-ingress-controller/commit/aae52d5c840385acf699cd083368b179eb270200) chore(deps): bump github.com/eclipse/paho.mqtt.golang in /test/e2e (#1891)
* [`1bea14f`](https://github.com/apache/apisix-ingress-controller/commit/1bea14f93bfc36723bb3af4a6eabe41b271607af) chore(deps): bump google.golang.org/grpc in /test/e2e (#1886)
* [`e0a2b17`](https://github.com/apache/apisix-ingress-controller/commit/e0a2b17bc6d6a5010f3ad2c36a2cae77f4f38d3a) chore(deps): bump golang.org/x/net from 0.12.0 to 0.14.0 (#1920)
* [`c1f241b`](https://github.com/apache/apisix-ingress-controller/commit/c1f241b35246058cdf9b935af32ace034a13c765) chore(deps): bump k8s.io/client-go from 0.27.4 to 0.28.1 in /test/e2e (#1938)
* [`c3174d4`](https://github.com/apache/apisix-ingress-controller/commit/c3174d46fb2dfabd1472326688df873509d72ce2) chore(deps): bump go.uber.org/zap from 1.24.0 to 1.25.0 (#1922)
* [`28d7c90`](https://github.com/apache/apisix-ingress-controller/commit/28d7c9025114c2576cffb2aec0d860eeb515b198) chore(deps): bump github.com/onsi/ginkgo/v2 in /test/e2e (#1933)
* [`7b3deb5`](https://github.com/apache/apisix-ingress-controller/commit/7b3deb5e84d257f46ce5c513d72b22a24af1be67) feat: add support for host pass in upstream crd (#1889)
* [`14e3c61`](https://github.com/apache/apisix-ingress-controller/commit/14e3c61f44c4c2fbe621026303b60ad07da9e57a) chore(deps): bump go.uber.org/zap from 1.24.0 to 1.25.0 in /test/e2e (#1921)
* [`fa07c66`](https://github.com/apache/apisix-ingress-controller/commit/fa07c662e6cb18bffea89423efb0e716fd72ccdd) fix(ci): udp forward failed and missing pigz (#1929)
* [`c3dff87`](https://github.com/apache/apisix-ingress-controller/commit/c3dff8718910d8f5d6e11ccadc897c452793154d) dep: downgraded k8s.io/kube-openapi (#1919)
* [`b7329b0`](https://github.com/apache/apisix-ingress-controller/commit/b7329b04719a74948cfa6e35aafa509822c7a1b5) chore: clean up apisix v1 (#1916)
* [`f2ae01a`](https://github.com/apache/apisix-ingress-controller/commit/f2ae01a718f5819f8c34d7591fb2ba98f211a8bb) chore(deps): bump k8s.io/client-go from 0.27.1 to 0.27.4 (#1917)
* [`37e9201`](https://github.com/apache/apisix-ingress-controller/commit/37e9201cf0f4423be450bc38d6e19789eed06eb4) chore: Upgrade Go tool chain version 1.19 to version 1.20 (#1788)
* [`3fa789d`](https://github.com/apache/apisix-ingress-controller/commit/3fa789df1fdf996b7dcac806633b076432581ce4) chore: remove support for Ingress in the extensions/v1beta1 API version (#1840)
* [`3f45ca9`](https://github.com/apache/apisix-ingress-controller/commit/3f45ca99db6d354e381679c39f776064ebd69c55) ci: auto certs and upgrade APISIX to 3.4.1 version (#1911)
* [`8e3104b`](https://github.com/apache/apisix-ingress-controller/commit/8e3104b3fde9a863a96044e02c34ad63c217f1ae) docs: Add QA about exposing gateway as loadbalancer (#1907)
* [`c40b664`](https://github.com/apache/apisix-ingress-controller/commit/c40b664dd22e46bf8ce498aa797d7e8e41b23073) ci: cron ci must use the logical AND condition (#1850)
* [`e809cfb`](https://github.com/apache/apisix-ingress-controller/commit/e809cfbaaad6f6c3cbd23f5ec8e93bc1d87f543d) feat: Allow response header rewrite via Ingress annotations (#1861)
* [`3efd796`](https://github.com/apache/apisix-ingress-controller/commit/3efd7963a9ed74742639a21e58ce2a14ffe6bb8a) chore(deps): bump github.com/gin-gonic/gin from 1.9.0 to 1.9.1 (#1852)
* [`a79c140`](https://github.com/apache/apisix-ingress-controller/commit/a79c14045b04f123695a093ef54196f6e6e71698) chore(deps): bump k8s.io/client-go from 0.27.1 to 0.27.3 in /test/e2e (#1866)
* [`8e86331`](https://github.com/apache/apisix-ingress-controller/commit/8e86331cd1cdb28830ede553d1f67f54b7d1c069) docs: update docs links (#1873)
* [`32c0751`](https://github.com/apache/apisix-ingress-controller/commit/32c07516b6bf2fe8ff86ecdf65ee04a7afc0c535) Update issuer.yaml (#1856)
* [`7540872`](https://github.com/apache/apisix-ingress-controller/commit/7540872b4158884dae2734af5ceaffbb23ca3b15) chore(deps): bump github.com/gin-gonic/gin in /test/e2e (#1851)
* [`e5db08a`](https://github.com/apache/apisix-ingress-controller/commit/e5db08a3fd16d0a56d218b5d054efbe53152c5b6) chore(deps): bump github.com/onsi/ginkgo/v2 in /test/e2e (#1837)
* [`373839a`](https://github.com/apache/apisix-ingress-controller/commit/373839a9552bc86a652b3b9f5122f64913e9e8e1) chore(deps): bump github.com/stretchr/testify in /test/e2e (#1842)
* [`ff43aee`](https://github.com/apache/apisix-ingress-controller/commit/ff43aeeea1d4a01c376c5ab64e49a819ecb97a1a) docs: Update powered-by.md (#1841)
* [`e91dbf5`](https://github.com/apache/apisix-ingress-controller/commit/e91dbf5303c2ec8a775eabc8a7d9955e053b20f1) chore(deps): Update dependencies (#1833)
* [`113defc`](https://github.com/apache/apisix-ingress-controller/commit/113defc7329febff963f06bb675f51d2c8379784) chore: rename all v2beta3 to v2 in e2e templates (#1832)
* [`7b81a8b`](https://github.com/apache/apisix-ingress-controller/commit/7b81a8b0dc785217adab3377db428564f6e64726) chore: StringToByte without mem-allocation supports v1.20 (#1750)
* [`050d201`](https://github.com/apache/apisix-ingress-controller/commit/050d2014f157c683eaac7a693712b2c9f1afbd74) chore: remove v2beta3 (#1817)
* [`c6a13b3`](https://github.com/apache/apisix-ingress-controller/commit/c6a13b3b8619b4ac6ba0cd9c921da4e13550a945) chore(deps): bump github.com/onsi/ginkgo/v2 in /test/e2e (#1827)
* [`31891ba`](https://github.com/apache/apisix-ingress-controller/commit/31891bac85f02bc3487069af47a22ad403a863e5) fix: Referencing empty objects during tcproute and httproute updates (#1825)
* [`2182a48`](https://github.com/apache/apisix-ingress-controller/commit/2182a48cbca785373eca745a13d8cf2b7d9ab6c8) CI: add regression testing with apisix:dev (#1721)
* [`2641b78`](https://github.com/apache/apisix-ingress-controller/commit/2641b782b7c46c53f865f6d7e4b1e702e9141d4e) chore: add docker compose and docker-compose compatible (#1808)
* [`abfacd6`](https://github.com/apache/apisix-ingress-controller/commit/abfacd6ab7ff8129898ef9a1c5e880b92fd52313) fix: Keep health checker running when health check failed. Make healthcheck function pure (#1779)
* [`a414df7`](https://github.com/apache/apisix-ingress-controller/commit/a414df7531f904481afccce44f5436f3d45d2c86) fix: secret reference update Ingress (#1780)
* [`2061824`](https://github.com/apache/apisix-ingress-controller/commit/2061824543bca34704f434f3040c08c35764524e) chore: upgrade ginkgo 1.9.0 to 1.9.2 (#1800)
* [`4b1ad1b`](https://github.com/apache/apisix-ingress-controller/commit/4b1ad1bb94845bcaf7f40119ccf256185bb62d59) feat: sync consumer crd labels to apisix (#1540)
* [`98ff8e5`](https://github.com/apache/apisix-ingress-controller/commit/98ff8e525365b3f1d25a404d63b5da2195f6fe54) fix: error message typo (#1790)
* [`3a8fdf6`](https://github.com/apache/apisix-ingress-controller/commit/3a8fdf641cde6d0095ecee6917c35cec7193624d) refactor: update status (#1618)
* [`5ef48f9`](https://github.com/apache/apisix-ingress-controller/commit/5ef48f98e973af8f72bcdb69b40c72e8587ac66b) chore(deps): bump github.com/gavv/httpexpect/v2 in /test/e2e (#1774)
* [`8e43700`](https://github.com/apache/apisix-ingress-controller/commit/8e437006dc227c1bccf3fec93d70cdac40dc31a3) dep: Updated some dependencies. (#1782)
* [`2f9a4c8`](https://github.com/apache/apisix-ingress-controller/commit/2f9a4c85a6f32d43b95e30059fba352a15771dbc) chore: use docker buildkit to cache go build cache (#1778)
* [`b4d1eed`](https://github.com/apache/apisix-ingress-controller/commit/b4d1eed6fd99104b55a95bccb5d279696a6bc2f8) feat: CRDs ingressClassName field cannot be modified (#1728)
* [`adf9757`](https://github.com/apache/apisix-ingress-controller/commit/adf97572e0fc0c6da94514029c3e7be1d4341287) chore(deps): bump github.com/spf13/cobra from 1.6.1 to 1.7.0 (#1773)
* [`aad3ef6`](https://github.com/apache/apisix-ingress-controller/commit/aad3ef6fdd7866c096b96eaa3d19fed9f8fc3335) e2e: ingress annotations does not need to use v2beta3 (#1503)
* [`e6dbaa7`](https://github.com/apache/apisix-ingress-controller/commit/e6dbaa7acb6624d9a7ebca723784380badbeb092) fix: malformed URL created in schemaClient (#1772)
* [`97f9ef9`](https://github.com/apache/apisix-ingress-controller/commit/97f9ef904b3e1e0d66e6fddae59712e7ec849fca) feat: support webhook validate plugin (#1355)
* [`eb01907`](https://github.com/apache/apisix-ingress-controller/commit/eb019076c27a2ebbca378ae4fd509bdb2a7cfe1c) docs: describe how to generate secret from cert file (#1769)
* [`bacb8f8`](https://github.com/apache/apisix-ingress-controller/commit/bacb8f8111656b3a765dc71352f43ea244de06f1) feat: sync apisix upstream labels (#1553)
* [`38710e7`](https://github.com/apache/apisix-ingress-controller/commit/38710e71d9dc11a9962198c8927912c44c92a54c) chore(deps): bump k8s.io/code-generator from 0.26.2 to 0.26.3 (#1764)
* [`b316705`](https://github.com/apache/apisix-ingress-controller/commit/b316705b373da02835e095feeaa5fa5f5053920a) docs: add ApisixPluginConfig and update examples (#1752)
* [`045f5e7`](https://github.com/apache/apisix-ingress-controller/commit/045f5e70d283353b4382ba64e6ae6677751711ee) docs: Add lost entries of `discovery` in Upstream's reference doc. (#1766)
* [`2cb99b8`](https://github.com/apache/apisix-ingress-controller/commit/2cb99b89b99d5e8ce1dd277a0868c177630b4f43) feat: support comparison in resource sync (#1742)
* [`0602314`](https://github.com/apache/apisix-ingress-controller/commit/0602314d81e06c0e571b009f24eacb28784e46b4) docs: add traffic-split plugin usage (#1696)
* [`99b6634`](https://github.com/apache/apisix-ingress-controller/commit/99b66340fada0920d635f559e651f3d3ecf80c1f) docs: Deploy to OpenShift (#1761)
* [`e0f4cc2`](https://github.com/apache/apisix-ingress-controller/commit/e0f4cc238120d3750c72c78fd52bc157c08398ee) docs: added Docker to prerequisite of Installation with Kind (#1751)
* [`7ccf531`](https://github.com/apache/apisix-ingress-controller/commit/7ccf5317821b0996b389af87b1144958e16dedec) fix: missing upstream name in gateway-api routes (#1754)
* [`405b6fb`](https://github.com/apache/apisix-ingress-controller/commit/405b6fb2f9740d23cd75a19c831cd17c35cd6ebb) chore(deps): bump github.com/onsi/ginkgo/v2 in /test/e2e (#1746)
* [`379e92e`](https://github.com/apache/apisix-ingress-controller/commit/379e92ee2c6c55dad0fae7f31203281bebc89152) chore(deps): bump golang.org/x/net from 0.7.0 to 0.8.0 (#1725)
* [`0ad8eaa`](https://github.com/apache/apisix-ingress-controller/commit/0ad8eaadf32085c922b53053c40e3749467171b8) docs: add tutorial on using custom Plugins (#1745)
* [`c5b2ae8`](https://github.com/apache/apisix-ingress-controller/commit/c5b2ae841070f687dface28814bc21349f8d9952) chore(deps): bump k8s.io/client-go from 0.26.2 to 0.26.3 (#1734)
* [`8730f88`](https://github.com/apache/apisix-ingress-controller/commit/8730f883cf28f9d1b7f77b00b67fd2071a967079) chore(deps): bump k8s.io/client-go from 0.26.2 to 0.26.3 in /test/e2e (#1730)
* [`c6dd810`](https://github.com/apache/apisix-ingress-controller/commit/c6dd810a5c40f727a13b925d0345f68697d39a71) feat: make multiple controllers handle different ApisixRoute CRDs (#593)
* [`6e22838`](https://github.com/apache/apisix-ingress-controller/commit/6e22838d6f67785f69c5f8e2eb66ce02ee45a7a3) feat: support ingressClass for ApisixGlobalRule (#1718)
* [`8b9726d`](https://github.com/apache/apisix-ingress-controller/commit/8b9726d40b1d2b21fabed084589230540782fe76) chore(deps): bump github.com/onsi/ginkgo/v2 in /test/e2e (#1723)
* [`2cf5963`](https://github.com/apache/apisix-ingress-controller/commit/2cf59631d1dfdfca77f704b0836bf4fa3ed95ad5) ci: Upgrade Ginkgo to latest version (#1727)
* [`277669e`](https://github.com/apache/apisix-ingress-controller/commit/277669e7d970c3fe147d509479ff02aa75a23fd9) e2e: more stability (#1739)
* [`a431dd0`](https://github.com/apache/apisix-ingress-controller/commit/a431dd0b3c85b9630cea7bebb39b20aeaa39d451) feat: Support GatewayAPI route attachment restriction (#1440)
* [`d871a2c`](https://github.com/apache/apisix-ingress-controller/commit/d871a2c3253fc04856ad42d645df06c8f3c70124) fix: when secret created later than apisixtls it should be updated (#1715)
* [`f1395f1`](https://github.com/apache/apisix-ingress-controller/commit/f1395f11af09fc8ecffea68500bb1e13518e78e8) ci: regression test on apisix-and-all and apisix (#1726)
* [`271d89f`](https://github.com/apache/apisix-ingress-controller/commit/271d89feedc73619ccf8c2780bb124aba9b019d7) feat: ApisixClusterConfig support IngressClass (#1720)
* [`07c7d9d`](https://github.com/apache/apisix-ingress-controller/commit/07c7d9d66414e2b3215cdd61c02c1b248c7cbb05) feat: ApisixConsumer support ingressClass (#1717)
* [`3abe8af`](https://github.com/apache/apisix-ingress-controller/commit/3abe8af8a7db8f12fa4b3016ab794716da08fe8a) feat: ApisixTls suuport ingressClass (#1714)
* [`cfaa246`](https://github.com/apache/apisix-ingress-controller/commit/cfaa246da04d9cc698431795eeadde1c03969ff6) chore(deps): bump golang.org/x/net in /test/e2e/testbackend (#1702)
* [`23d10a3`](https://github.com/apache/apisix-ingress-controller/commit/23d10a3bdecfeb2f80798e4bb4d55ebd02ef4de8) chore(deps): bump k8s.io/client-go from 0.26.1 to 0.26.2 (#1709)
* [`93c795b`](https://github.com/apache/apisix-ingress-controller/commit/93c795b327fc4e9185e26a8eacc8de01774dd6e4) feat: support ingressClass for ApisixPluginConfig (#1716)
* [`97f6aed`](https://github.com/apache/apisix-ingress-controller/commit/97f6aed4c0b8279744bb480ddcc5d8c776d04258) chore(deps): bump github.com/onsi/ginkgo/v2 in /test/e2e (#1703)
* [`d8f7353`](https://github.com/apache/apisix-ingress-controller/commit/d8f73534220019a9461e6263de8a748cef9b656f) chore(deps): bump k8s.io/client-go from 0.26.1 to 0.26.2 in /test/e2e (#1705)
* [`5ec21c1`](https://github.com/apache/apisix-ingress-controller/commit/5ec21c1e008d8062d7c1eecff0a0f8039b4e0292) chore(deps): bump golang.org/x/text in /test/e2e/testbackend (#1684)
* [`879b433`](https://github.com/apache/apisix-ingress-controller/commit/879b43380f2ced78b6722bacbb2650732a794813) chore(deps): bump github.com/stretchr/testify from 1.8.1 to 1.8.2 (#1689)
* [`13d2b5d`](https://github.com/apache/apisix-ingress-controller/commit/13d2b5d1ea186bed0265d51900bdf063b15afbc5) docs: monitoring apisix with helm chart (#1683)
* [`7d62b7e`](https://github.com/apache/apisix-ingress-controller/commit/7d62b7e45463b6f70ad5585ba2cf745a2fe07824) chore(deps): bump github.com/gavv/httpexpect/v2 in /test/e2e (#1706)
* [`b616db4`](https://github.com/apache/apisix-ingress-controller/commit/b616db488ba31ee40b246799d339a6559ce4d6ff) chore(deps): bump k8s.io/api from 0.26.1 to 0.26.2 in /test/e2e (#1704)
* [`d702809`](https://github.com/apache/apisix-ingress-controller/commit/d7028098cdb9d6c48d9fc382a9b15bd78aeecc47) chore(deps): bump k8s.io/code-generator from 0.26.1 to 0.26.2 (#1708)
* [`fad7955`](https://github.com/apache/apisix-ingress-controller/commit/fad7955d82be0b64ff38826da538b8f768071efd) docs: using APISIX Ingress as Istio egress gateway (#1667)
* [`acf3e36`](https://github.com/apache/apisix-ingress-controller/commit/acf3e3694c04ed85640cfb50f3884a75fcf428b4) chore(deps): bump golang.org/x/sys in /test/e2e/testbackend (#1687)
* [`7862e28`](https://github.com/apache/apisix-ingress-controller/commit/7862e28fe81f1b71f9e2724915d9d2d00f50e136) chore(deps): bump github.com/gin-gonic/gin from 1.8.2 to 1.9.0 (#1701)
* [`5fcd3d0`](https://github.com/apache/apisix-ingress-controller/commit/5fcd3d0ed720b994447ca190127d9d137494b24b) feat: ApisixUpstream support IngressClass (#1674)
* [`4cd8ad5`](https://github.com/apache/apisix-ingress-controller/commit/4cd8ad52dcc18dcd874d663fac5e31eebe417720) feat: sync plugin-config labels to apisix (#1538)
* [`ec09d4f`](https://github.com/apache/apisix-ingress-controller/commit/ec09d4f5e19e3e1461d4bb4d6107dbe8a6abdaf3) docs: Update the-hard-way.md (#1700)
* [`db4dc71`](https://github.com/apache/apisix-ingress-controller/commit/db4dc71a9a88d92315551fcb0eed30552643fb87) docs: fix typo in aks deployment guide (#1695)
* [`9df7af6`](https://github.com/apache/apisix-ingress-controller/commit/9df7af65e95bcca361033d713da5a8ac617ea733) ci: add yamllint rules (#1666)
* [`4091ea0`](https://github.com/apache/apisix-ingress-controller/commit/4091ea00312d6273940be03dac32d5ed6e3bc43d) chore(deps): bump github.com/stretchr/testify in /test/e2e (#1691)
* [`51d0ecd`](https://github.com/apache/apisix-ingress-controller/commit/51d0ecdbf11c2899ff2a6fcfc94b21083d34c2eb) fix: set the health check log level by gin to debug (#1580)
* [`3f76ae4`](https://github.com/apache/apisix-ingress-controller/commit/3f76ae4685b92bb1339f8b02e7e359fa1e216746) feat: Add prefer_name into ApisixClusterConfig (#1519)
* [`de1928e`](https://github.com/apache/apisix-ingress-controller/commit/de1928e3887ce9e5bc5ceb243fa76aad4af8077f) docs: update grpc proxy (#1698)
* [`f6b3349`](https://github.com/apache/apisix-ingress-controller/commit/f6b334979e6e5eff3732e977c38dc76a82117e3e) chore(deps): bump github.com/gavv/httpexpect/v2 in /test/e2e (#1690)
* [`69fbdb2`](https://github.com/apache/apisix-ingress-controller/commit/69fbdb20fd0218bcd951c443e3b1838f27c227ca) feat: support disable resource periodically sync (#1685)
* [`7a87083`](https://github.com/apache/apisix-ingress-controller/commit/7a87083617fd52a2434b03ed065fd48901fccaf4) bump golang.org/x/net from 0.5.0 to 0.7.0 (#1678)
* [`872f291`](https://github.com/apache/apisix-ingress-controller/commit/872f2912999564e783d8d8caf6ef503358471882) chore(deps): bump github.com/onsi/ginkgo/v2 in /test/e2e (#1679)
* [`86c14c7`](https://github.com/apache/apisix-ingress-controller/commit/86c14c742fb63675d695f04ff0fcd24588159201) docs: fix jwtAuth configuration error in documents (#1680)
* [`0ff7aca`](https://github.com/apache/apisix-ingress-controller/commit/0ff7aca50d463196a4de302aa3265617180305f4) chore(deps): bump golang.org/x/net from 0.5.0 to 0.7.0 in /test/e2e (#1677)
* [`caf2639`](https://github.com/apache/apisix-ingress-controller/commit/caf2639673fe1877549d3f341489f6ce8e0997a9) chore(deps): bump golang.org/x/net from 0.5.0 to 0.6.0 (#1668)
* [`5beb519`](https://github.com/apache/apisix-ingress-controller/commit/5beb519f24b0a8dc534e97285bb442721eeb13a4) docs: small adjustments to Check CRD status tutorial (#1670)
* [`1b66a8e`](https://github.com/apache/apisix-ingress-controller/commit/1b66a8ef576d58cba03fb88858d4be8555146d4e) docs: update the apisix image version and ingress image version (#1633)
* [`4241b67`](https://github.com/apache/apisix-ingress-controller/commit/4241b673b3fd385e82d5ca3a39fed8acafa5bd64) fix: panic at empty http spec (#1660)
* [`bcf44c6`](https://github.com/apache/apisix-ingress-controller/commit/bcf44c6cff1ca8b3d9686a7bb3b591e7cf71a457) ci: update license-checker (#1652)
* [`199dcff`](https://github.com/apache/apisix-ingress-controller/commit/199dcffe174d68ffe9984ab471565084b95788eb) feat: support disable status (#1595)
* [`88d04f2`](https://github.com/apache/apisix-ingress-controller/commit/88d04f2916082e3a068cef79e716ced63abf1c30) chore(deps): bump github.com/gavv/httpexpect/v2 in /test/e2e (#1662)
* [`65701d8`](https://github.com/apache/apisix-ingress-controller/commit/65701d82a4b361706d4c56b0bc56e8d81767e474) chore(deps): bump gopkg.in/natefinch/lumberjack.v2 from 2.0.0 to 2.2.1 (#1664)
* [`a0a50fe`](https://github.com/apache/apisix-ingress-controller/commit/a0a50fe643fe42aa2d3c83408e0aa5b720238270) fix: Ingress delete events can be handler after svc be deleted (#1576)
* [`e232a07`](https://github.com/apache/apisix-ingress-controller/commit/e232a0782cd9169143df0e7acff63f5482838c83) docs: update Prometheus tutorial (#1635)
* [`3bc0587`](https://github.com/apache/apisix-ingress-controller/commit/3bc0587fa044ef8c21e3b396d88bfbd96db6979e) chore: Add more types in the pull request template (#1644)
* [`03b635a`](https://github.com/apache/apisix-ingress-controller/commit/03b635aef8e4b621ce01123505896e364e66763f) chore(deps): bump github.com/gavv/httpexpect/v2 in /test/e2e (#1651)
* [`84d11a1`](https://github.com/apache/apisix-ingress-controller/commit/84d11a182589b4f4728fcd59786f938cbc7f454f) chore: update issue templates (#1590)
* [`3db5dc2`](https://github.com/apache/apisix-ingress-controller/commit/3db5dc2023f16d4e8b6bfd1f171f54611f6868c2) chore(deps): bump github.com/onsi/ginkgo/v2 in /test/e2e (#1632)
* [`acee3f6`](https://github.com/apache/apisix-ingress-controller/commit/acee3f6c28bb9e0fd23eab9fdc478e2ce103bd84) docs: fix type of .spec.plugins (#1645)
* [`7503056`](https://github.com/apache/apisix-ingress-controller/commit/75030566f324af508febd1cfaa65320565c938ec) fix: Add ApisixUpstream CRD status property (#1641)
* [`eb86829`](https://github.com/apache/apisix-ingress-controller/commit/eb868296016499e556f68feb73c10f93aecc7523) docs: Update NOTICE (#1636)
* [`aa7967d`](https://github.com/apache/apisix-ingress-controller/commit/aa7967deacf8bc3a25f8958ae9273155d3fe20a4) docs: rename references file to skip lint (#1638)
* [`5e0f89f`](https://github.com/apache/apisix-ingress-controller/commit/5e0f89f0e85b42a3b7867168d7901115ce85d654) test(e2e): add stream tcp proxy with SNI test (#1533)
* [`33d42c3`](https://github.com/apache/apisix-ingress-controller/commit/33d42c31dbc70e43b08d9d4c95ca7ecd8a572633) feat: add ldap-auth authorization method (#1588)
* [`ccdd6a2`](https://github.com/apache/apisix-ingress-controller/commit/ccdd6a2ab3fbfc14d14daf67c27204eb9ff43d38) docs: add Gateway API installation instructions (#1616)
* [`905e1c5`](https://github.com/apache/apisix-ingress-controller/commit/905e1c557fa141155ab574344ea43a13bc0017bf) chore: upgrade gateway-api v0.5.1 to v0.6.0 (#1623)
* [`fe4e2af`](https://github.com/apache/apisix-ingress-controller/commit/fe4e2afcc56a4999c0cec800deaca91d33da2646) chore: add AlinsRan to dependabot reviewer (#1631)
* [`0518f01`](https://github.com/apache/apisix-ingress-controller/commit/0518f01d9f229cc93ee6c490b8619efbf9466f1b) chore(deps): bump golang.org/x/net from 0.4.0 to 0.5.0 (#1621)
* [`e4d8ac9`](https://github.com/apache/apisix-ingress-controller/commit/e4d8ac90f55b2fa49665934494fb9d572983c693) docs: using tool auto generate references (#1630)
* [`fa57ff5`](https://github.com/apache/apisix-ingress-controller/commit/fa57ff5e1c9b231e62087b963385a2b049bb4e5e) feat: add new ApisixGlobalRule resource to support global rules (#1586)
* [`4c0535b`](https://github.com/apache/apisix-ingress-controller/commit/4c0535ba929c64ed9f9ac603300156879fe79e26) doc: update 1.6 upgrade guide (#1592)
* [`1f4ade7`](https://github.com/apache/apisix-ingress-controller/commit/1f4ade7b30fb00ab68ce0b943adb17c0defe891a) chore(deps): bump github.com/onsi/ginkgo/v2 in /test/e2e (#1620)
* [`26a202d`](https://github.com/apache/apisix-ingress-controller/commit/26a202d7e1abaf00f539d80037b348a673fb94b1) chore(deps): bump k8s.io/code-generator from 0.26.0 to 0.26.1 (#1622)
* [`a16b3dd`](https://github.com/apache/apisix-ingress-controller/commit/a16b3dd5ee36c1c139f0dbcc3632cc846ef2c5b0) doc: add svc-namespace description to the annotations (#1605)
* [`123d080`](https://github.com/apache/apisix-ingress-controller/commit/123d080d4e92e7becb95da0835e7775365a71d2a) feat: add support for filter_func for ApisixRoute (#1545)
* [`9476e13`](https://github.com/apache/apisix-ingress-controller/commit/9476e136612bebd4fbe93bc13b08c6852b0e60bc) chore(deps): bump github.com/gavv/httpexpect/v2 in /test/e2e (#1617)
* [`72577c1`](https://github.com/apache/apisix-ingress-controller/commit/72577c18866836db46695ffefb6f489191b37ffc) chore(deps): bump k8s.io/client-go from 0.26.0 to 0.26.1 (#1614)
* [`00b3442`](https://github.com/apache/apisix-ingress-controller/commit/00b3442eff469bf13f7e527be18c945f0c1f6dc1) docs: update prowered-by.md (#1604)
* [`9aae0e3`](https://github.com/apache/apisix-ingress-controller/commit/9aae0e39ed7aecd8a40098ca630344e5ed558e8f) ci: add goimports-reviser (#1606)
* [`4006ea8`](https://github.com/apache/apisix-ingress-controller/commit/4006ea8baee31adca64b30ba7609aff553adf636) chore(deps): bump github.com/gavv/httpexpect/v2 in /test/e2e (#1611)
* [`2ee88ad`](https://github.com/apache/apisix-ingress-controller/commit/2ee88ad83bcafdfe7c7124c9a1cceddd7f0427b5) chore(deps): bump some dependencies (#1603)
* [`8e31a9b`](https://github.com/apache/apisix-ingress-controller/commit/8e31a9bec67ce93292604853610173bcac607c81) chore(deps): bump k8s.io/client-go from 0.26.0 to 0.26.1 in /test/e2e (#1613)
* [`afa9403`](https://github.com/apache/apisix-ingress-controller/commit/afa940362cce48b172a3e7f9645473e2bdfc8d68) docs: add tutorial for Gateway API (#1615)
* [`7c809c6`](https://github.com/apache/apisix-ingress-controller/commit/7c809c6881fe48a4675399880333e2f3d7cc762a) docs: add Gateway API example to the "Getting started" guide (#1607)
* [`31714eb`](https://github.com/apache/apisix-ingress-controller/commit/31714eb4ae13c25ea0de1b2b7af0469f01938738) chore(deps): bump github.com/onsi/ginkgo/v2 in /test/e2e (#1599)
* [`1acb058`](https://github.com/apache/apisix-ingress-controller/commit/1acb058e2025d93a57283dec7b499c78ae568688) chore(deps): bump dependencies from 0.25.4 to 0.26.0 (#1520)
* [`eb0bd81`](https://github.com/apache/apisix-ingress-controller/commit/eb0bd817f89dfa604a1a20aa519f1eb593c5aa90) docs: update compatibility with APISIX (#1598)
* [`d3f2359`](https://github.com/apache/apisix-ingress-controller/commit/d3f23598846c81d1758d7b540fb1edadd2f56439) docs: update controller to use adminAPIVersion=v3 (#1593)
* [`2024a09`](https://github.com/apache/apisix-ingress-controller/commit/2024a09da38cd138576bd2d436efa5c29021922d) docs: add note about enabling the Plugin (#1596)
* [`32561d0`](https://github.com/apache/apisix-ingress-controller/commit/32561d01e7a99bae01a392f46f95cdd388073f5b) fix: allow passing plugin config name for route with no backends (#1578)
* [`84390d4`](https://github.com/apache/apisix-ingress-controller/commit/84390d4f372c2fb8388c5c6a20c41082a53e8e34) docs: add CHANGELOG for v1.6.0 (#1585)
* [`d701fef`](https://github.com/apache/apisix-ingress-controller/commit/d701fefb0051ce3dfe8b3b51f5488f9606c8fce3) docs: add example link. (#1582) (#1583)
* [`78272a5`](https://github.com/apache/apisix-ingress-controller/commit/78272a54928f5f43bdb0efb2ce672b1f91d4b4da) docs: Update the-hard-way.md (#1581) (#1584)
* [`486b46a`](https://github.com/apache/apisix-ingress-controller/commit/486b46abd1c9665005e83036433af213aaa28f21) chore: rename TranslateXXNotStrictly to GenerateXXDeleteMark (#1490)
* [`b62be90`](https://github.com/apache/apisix-ingress-controller/commit/b62be90c8246177843e4fc2256e31cafd69b269c) chore(deps): bump github.com/onsi/ginkgo/v2 in /test/e2e (#1541)
* [`634b43f`](https://github.com/apache/apisix-ingress-controller/commit/634b43f508dfba604b150afa42c6a43394f5a1ac) feat: sync route crd labels to apisix (#1526)
* [`695a5e1`](https://github.com/apache/apisix-ingress-controller/commit/695a5e1fb9b58ae83e8bf8221a0a2e3c14213b0c) docs: add instructions to use Network LoadBalancer (#1557)
* [`949c1da`](https://github.com/apache/apisix-ingress-controller/commit/949c1da56d8de67fc490bc4cf52607958119e7de) chore: admin-api default version v2 (#1558)
* [`5f98bc1`](https://github.com/apache/apisix-ingress-controller/commit/5f98bc17468a6a86b55672f4b5a025ba5abb48d6) docs: add match stream route with SNI tutorial (#1543)
* [`b5e89cf`](https://github.com/apache/apisix-ingress-controller/commit/b5e89cf6661077dbb40e7d5deef840180f32cedb) chore: set v1.6 as protect branch (#1556)
* [`1c42993`](https://github.com/apache/apisix-ingress-controller/commit/1c429936f5d6c1851c34b7bffdeaa38c6f4bc834) fix: bad configuration item: apisix-admin-api-version (#1551)
* [`e734b2d`](https://github.com/apache/apisix-ingress-controller/commit/e734b2d702a7356ca1e4cc4b7770771d93bf03e0) chore: extra annotations logs (#1549)
* [`60061d0`](https://github.com/apache/apisix-ingress-controller/commit/60061d05148c979a85ea6db4e596fa8172726173) docs: update tutorial on installing APISIX in Kubernetes (#1550)
* [`39cffdc`](https://github.com/apache/apisix-ingress-controller/commit/39cffdc8d82f6840b8b4a69890cb77aa77e14282) docs: update synchronization status check docs (#1548)
* [`9208f58`](https://github.com/apache/apisix-ingress-controller/commit/9208f582b936d60da26cb5c977fcd978344c663c) docs: update APISIX CRD tutorial (#1544)
</p>
</details>

### Dependency Changes

* **github.com/api7/etcd-adapter**                   v0.2.2 **_new_**
* **github.com/api7/gopkg**                          v0.2.0 **_new_**
* **github.com/bytedance/sonic**                     v1.9.1 **_new_**
* **github.com/chenzhuoyu/base64x**                  fe3a3abad311 **_new_**
* **github.com/evanphx/json-patch**                  v4.12.0 -> v5.6.0
* **github.com/evanphx/json-patch/v5**               v5.6.0 **_new_**
* **github.com/gabriel-vasile/mimetype**             v1.4.2 **_new_**
* **github.com/gin-gonic/gin**                       v1.8.1 -> v1.9.1
* **github.com/go-logr/logr**                        v1.2.3 -> v1.2.4
* **github.com/go-openapi/jsonpointer**              v0.19.5 -> v0.20.0
* **github.com/go-openapi/jsonreference**            v0.20.0 -> v0.20.2
* **github.com/go-openapi/swag**                     v0.22.3 -> v0.22.4
* **github.com/go-playground/locales**               v0.14.0 -> v0.14.1
* **github.com/go-playground/universal-translator**  v0.18.0 -> v0.18.1
* **github.com/goccy/go-json**                       v0.9.10 -> v0.10.2
* **github.com/golang/protobuf**                     v1.5.2 -> v1.5.3
* **github.com/google/btree**                        v1.1.2 **_new_**
* **github.com/google/gnostic-models**               v0.6.8 **_new_**
* **github.com/google/go-cmp**                       v0.5.8 -> v0.5.9
* **github.com/google/gofuzz**                       v1.1.0 -> v1.2.0
* **github.com/google/uuid**                         v1.3.0 **_new_**
* **github.com/gorilla/websocket**                   v1.5.0 **_new_**
* **github.com/grpc-ecosystem/grpc-gateway**         v1.16.0 **_new_**
* **github.com/imdario/mergo**                       v0.3.13 -> v0.3.15
* **github.com/inconshreveable/mousetrap**           v1.0.1 -> v1.1.0
* **github.com/k3s-io/kine**                         v0.10.2 **_new_**
* **github.com/klauspost/cpuid/v2**                  v2.2.4 **_new_**
* **github.com/leodido/go-urn**                      v1.2.1 -> v1.2.4
* **github.com/mattn/go-isatty**                     v0.0.14 -> v0.0.19
* **github.com/prometheus/client_golang**            v1.14.0 -> v1.16.0
* **github.com/prometheus/client_model**             v0.3.0 -> v0.4.0
* **github.com/prometheus/common**                   v0.37.0 -> v0.44.0
* **github.com/prometheus/procfs**                   v0.8.0 -> v0.11.1
* **github.com/sirupsen/logrus**                     v1.9.3 **_new_**
* **github.com/soheilhy/cmux**                       v0.1.5 **_new_**
* **github.com/spf13/cobra**                         v1.6.1 -> v1.7.0
* **github.com/stretchr/testify**                    v1.8.1 -> v1.8.4
* **github.com/tmc/grpc-websocket-proxy**            673ab2c3ae75 **_new_**
* **github.com/twitchyliquid64/golang-asm**          v0.15.1 **_new_**
* **go.etcd.io/etcd/api/v3**                         v3.5.9 **_new_**
* **go.uber.org/multierr**                           v1.8.0 -> v1.11.0
* **go.uber.org/zap**                                v1.24.0 -> v1.25.0
* **golang.org/x/arch**                              v0.3.0 **_new_**
* **golang.org/x/crypto**                            630584e8d5aa -> v0.12.0
* **golang.org/x/mod**                               86c51ed26bb4 -> v0.12.0
* **golang.org/x/net**                               46097bf591d3 -> v0.14.0
* **golang.org/x/oauth2**                            ee480838109b -> v0.8.0
* **golang.org/x/sys**                               fb04ddd9f9c8 -> v0.11.0
* **golang.org/x/term**                              03fcf44c2211 -> v0.11.0
* **golang.org/x/text**                              v0.3.7 -> v0.12.0
* **golang.org/x/time**                              90d013bbcef8 -> v0.3.0
* **golang.org/x/tools**                             v0.1.12 -> v0.12.0
* **google.golang.org/genproto**                     6bfd019c3878 **_new_**
* **google.golang.org/genproto/googleapis/api**      6bfd019c3878 **_new_**
* **google.golang.org/genproto/googleapis/rpc**      6bfd019c3878 **_new_**
* **google.golang.org/grpc**                         v1.57.0 **_new_**
* **google.golang.org/protobuf**                     v1.28.1 -> v1.31.0
* **gopkg.in/go-playground/assert.v1**               v1.2.1 **_new_**
* **gopkg.in/go-playground/pool.v3**                 v3.1.1 **_new_**
* **gopkg.in/natefinch/lumberjack.v2**               v2.0.0 -> v2.2.1
* **k8s.io/api**                                     v0.25.4 -> v0.28.1
* **k8s.io/apimachinery**                            v0.25.4 -> v0.28.1
* **k8s.io/client-go**                               v0.25.4 -> v0.28.1
* **k8s.io/code-generator**                          v0.25.4 -> v0.28.1
* **k8s.io/gengo**                                   391367153a38 -> ab3349d207d4
* **k8s.io/kube-openapi**                            a70c9af30aea -> 2695361300d9
* **k8s.io/utils**                                   ee6ede2d64ed -> d93618cff8a2
* **sigs.k8s.io/controller-runtime**                 v0.16.1 **_new_**
* **sigs.k8s.io/gateway-api**                        v0.5.1 -> v0.8.0
* **sigs.k8s.io/json**                               f223a00ba0e2 -> bc3834ca7abd

Previous release can be found at [1.6.0](https://github.com/apache/apisix-ingress-controller/releases/tag/1.6.0)

# 1.6.0

Welcome to the 1.6.0 release of apisix-ingress-controller!

This is a feature release.

## Highlights

The API version of all custom resources has been upgraded to v2 in v1.5 release. In 1.6 we removed the v2beta2 API. Please go to [#707](https://github.com/apache/apisix-ingress-controller/issues/707) for detail.

In this version we added more Gateway API support. e.g. TCPRoute/UDPRoute/HTTPRouteFilter etc.

From this version, we can proxy external services and external name services. And integrated with the service discovery component.

Please try out the release binaries and report any issues at
https://github.com/apache/apisix-ingress-controller/issues.

### Contributors

* dependabot[bot]
* Jintao Zhang
* Xin Rong
* Navendu Pottekkat
* Xin Rong
* Sarasa Kisaragi
* Ashing Zheng
* xiangtianyu
* 林靖
* Floyd
* Navendu Pottekkat
* dongjunduo
* lsy
* seven dickens
* Baoyuan
* Gallardot
* Jayson Reis
* LinkMaq
* Marco Aurelio Caldas Miranda
* Nicolas Frankel
* Qi Guo
* StevenBrown008
* Young
* Yousri
* YuanYingdong
* cmssczy
* incubator4
* mango
* redtacs
* soulbird
* thomas
* xianshun163
* 失眠是真滴难受

### Changes
<details><summary>129 commits</summary>
<p>

* [`88b1d45`](https://github.com/apache/apisix-ingress-controller/commit/88b1d45f1f851b96652424db45170135513f68ab) chore: admin-api default version v2 (#1558) (#1559)
* [`3b99ebf`](https://github.com/apache/apisix-ingress-controller/commit/3b99ebf6e84687904adb346314cd523fc7a5351d) fix: bad configuration item: apisix-admin-api-version (#1551) (#1555)
* [`b76074f`](https://github.com/apache/apisix-ingress-controller/commit/b76074f92fae038dcdcd8db25a866a29405ef943) chore: extra annotations logs (#1549) (#1554)
* [`15d881e`](https://github.com/apache/apisix-ingress-controller/commit/15d881eebb2c6cc1a29fa87c81b1cb1db57f498e) chore: 1.6.0-rc1 release (#1537)
* [`67d60fe`](https://github.com/apache/apisix-ingress-controller/commit/67d60fe9858f89f0e4ad575e4e0f5ed540fe5ef5) docs: add external service discovery tutorial (#1535)
* [`f162f71`](https://github.com/apache/apisix-ingress-controller/commit/f162f7119abd76b5a71c285fbfae68ed2faf88fb) feat: support for specifying port in external services (#1500)
* [`4208ca7`](https://github.com/apache/apisix-ingress-controller/commit/4208ca7cef4e54e22544050deed45bd768ad5ffa) refactor: unified factory and informer (#1530)
* [`a118727`](https://github.com/apache/apisix-ingress-controller/commit/a118727200524150b9062ba915bf50d361b2a9e1) docs: update Ingress controller httpbin tutorial (#1524)
* [`c0cb74d`](https://github.com/apache/apisix-ingress-controller/commit/c0cb74dd66c1c040339160905bf5e9fad0d6fe1a) docs: add external service tutorial (#1527)
* [`d22a6fc`](https://github.com/apache/apisix-ingress-controller/commit/d22a6fc820f7699af411b8ecaa971307cfc82dbd) chore(deps): bump github.com/onsi/ginkgo/v2 in /test/e2e (#1506)
* [`c4cedad`](https://github.com/apache/apisix-ingress-controller/commit/c4cedad549215c90b95ba389553c94370fe07a12) chore(deps): bump go.uber.org/zap from 1.23.0 to 1.24.0 (#1510)
* [`c6c2742`](https://github.com/apache/apisix-ingress-controller/commit/c6c2742fe9fe60efd40f2c0ecc5c2fc7f2166a2a) chore(deps): bump github.com/gavv/httpexpect/v2 in /test/e2e (#1509)
* [`b4255f1`](https://github.com/apache/apisix-ingress-controller/commit/b4255f1dd9cc69e7a3a44deeeeb8d4f17aea9e50) ci: using ubuntu-20.04 by default (#1504)
* [`03cfcb8`](https://github.com/apache/apisix-ingress-controller/commit/03cfcb840690759e80eaffd2eff22a43e8aa07b6) chore(deps): bump k8s.io/client-go from 0.25.4 to 0.26.0 in /test/e2e (#1505)
* [`0009b5d`](https://github.com/apache/apisix-ingress-controller/commit/0009b5d6951c89d2b67e0a440d4a75952fb3154c) feat: support secret plugin config (#1486)
* [`8cf79c2`](https://github.com/apache/apisix-ingress-controller/commit/8cf79c2e8f7278b52bf83f6cab6e85ab73d7266f) fix: ingress.tls secret not found (#1394)
* [`7e8f076`](https://github.com/apache/apisix-ingress-controller/commit/7e8f0763a595d566804ec397cfcb03214f2477df) fix: many namespace lead to provider stuck (#1386)
* [`2ce1ed3`](https://github.com/apache/apisix-ingress-controller/commit/2ce1ed3ebfb9c5041d28445f8747e1665d4207b4) chore: use httptest.NewRequest instead of http.Request as incoming server request in test case (#1498)
* [`768a35f`](https://github.com/apache/apisix-ingress-controller/commit/768a35f66c879bc35ef71e1f9ec4caa1ec94d3b9) feat: add Ingress annotation to support response-rewrite (#1487)
* [`051fc48`](https://github.com/apache/apisix-ingress-controller/commit/051fc48de133699cfd5d12e28358226913550871) e2e: support docker hub as registry (#1489)
* [`cc48ae9`](https://github.com/apache/apisix-ingress-controller/commit/cc48ae9bc34299f56507eb9539c1c57602aa9638) chore: use field cluster.name as value but not "default" (#1495)
* [`931ab06`](https://github.com/apache/apisix-ingress-controller/commit/931ab0699ff1d5791484928492f101101365aee9) feat: ingress annotations supports the specified upstream schema (#1451)
* [`de9f84f`](https://github.com/apache/apisix-ingress-controller/commit/de9f84fce92e9ad26d487ccafa4bb7e800176776) doc: update upgrade guide (#1479)
* [`7511166`](https://github.com/apache/apisix-ingress-controller/commit/7511166a7c55f8cd0578c106515b15fdf3f058c6) chore(deps): bump go.uber.org/zap from 1.23.0 to 1.24.0 in /test/e2e (#1488)
* [`afbc4f7`](https://github.com/apache/apisix-ingress-controller/commit/afbc4f7369481615b8d55488392200f3b29b123a) docs: fix typo (#1491)
* [`1097792`](https://github.com/apache/apisix-ingress-controller/commit/109779232f607f13307de52f2f667f8d060ec25c) chore: replace io/ioutil package (#1485)
* [`ed92690`](https://github.com/apache/apisix-ingress-controller/commit/ed92690f5aabb4ece4b92d860d72d85bdfa23db0) fix:sanitize log output when exposing sensitive values (#1480)
* [`8e39e71`](https://github.com/apache/apisix-ingress-controller/commit/8e39e71002e44d303ecbad274317561aeb62db3d) feat: add control http method using kubernetes ingress by annotations (#1471)
* [`bccf762`](https://github.com/apache/apisix-ingress-controller/commit/bccf762ac1b6386e4bd8911180ea13ac5a14bdfe) chore: bump actions. (#1484)
* [`adf7d27`](https://github.com/apache/apisix-ingress-controller/commit/adf7d27033ef103ef483486f4ba155d0e9aee471) docs: add more user cases (#1482)
* [`0f7b3f3`](https://github.com/apache/apisix-ingress-controller/commit/0f7b3f375f1fbf35ebb374005c7c7690ed2df337) Revert "chore: update actions and add more user cases. (#1478)" (#1481)
* [`14353d3`](https://github.com/apache/apisix-ingress-controller/commit/14353d30bee1263e16e0b8cd96f6616d5bdae95a) chore(deps): bump github.com/onsi/ginkgo/v2 in /test/e2e (#1477)
* [`ee99eba`](https://github.com/apache/apisix-ingress-controller/commit/ee99ebaaabf9a3f51eb798a904f88483cac907a3) chore(deps): bump github.com/gavv/httpexpect/v2 in /test/e2e (#1476)
* [`7471e64`](https://github.com/apache/apisix-ingress-controller/commit/7471e6493729227964b17acc31bcd6a36d15cdca) chore: update actions and add more user cases. (#1478)
* [`136d40d`](https://github.com/apache/apisix-ingress-controller/commit/136d40d7ed1c8ff7d871f183e34417f6b9e7c50c) chore(deps): some deps bump (#1475)
* [`c2cea69`](https://github.com/apache/apisix-ingress-controller/commit/c2cea69db6b158714c98b7b8710716fd24a74b27) docs: change ingress class annotation to spec.ingressClassName (#1425)
* [`c8d3bd5`](https://github.com/apache/apisix-ingress-controller/commit/c8d3bd52fd8820475be763cd52b51b981382f285) feat: add support for integrate with DP service discovery (#1465)
* [`ec88b49`](https://github.com/apache/apisix-ingress-controller/commit/ec88b49cdc3164c44cef990482fc29ca7490aa3c) chore(deps): bump k8s.io/code-generator from 0.25.3 to 0.25.4 (#1456)
* [`803fdeb`](https://github.com/apache/apisix-ingress-controller/commit/803fdeb1f3ebcdb49d31d5b1702b172407c21c66) chore(deps): bump k8s.io/client-go from 0.25.3 to 0.25.4 (#1458)
* [`e6eb3bf`](https://github.com/apache/apisix-ingress-controller/commit/e6eb3bf2aed4c39e91271e4f74b61ff6fe97ad16) feat: support variable in ApisixRoute exprs scope (#1466)
* [`be7edf6`](https://github.com/apache/apisix-ingress-controller/commit/be7edf6b880e4904553c98f57dd78766632e1520) feat: support apisix v3 admin api (#1449)
* [`d95ae08`](https://github.com/apache/apisix-ingress-controller/commit/d95ae083057d561522d68376d9ad9c4c22ae51cd) docs: Add more descriptions and examples in the prometheus doc (#1467)
* [`d610041`](https://github.com/apache/apisix-ingress-controller/commit/d610041dbd8cfb76c424250b8e9e8fa7b1a1b22f) chore(deps): bump k8s.io/client-go from 0.25.3 to 0.25.4 in /test/e2e (#1453)
* [`632d5c1`](https://github.com/apache/apisix-ingress-controller/commit/632d5c1291bc0775d6280a58536ca9902676541f) feat: support HTTPRequestMirrorFilter in HTTPRoute (#1461)
* [`47906a5`](https://github.com/apache/apisix-ingress-controller/commit/47906a533369dc76c3d2f7fcba10ce8030bd4ce6) docs: update ApisixRoute/v2 reference (#1423)
* [`6ec804f`](https://github.com/apache/apisix-ingress-controller/commit/6ec804f454f56b4f6d2a2a6c6adabfcd05404aef) feat(makefile): allow to custom registry port for `make kind-up` (#1417)
* [`a318f49`](https://github.com/apache/apisix-ingress-controller/commit/a318f4990640d3a743045b5945440ef7900096ff) docs: update ApisixUpstream reference (#1450)
* [`51c0745`](https://github.com/apache/apisix-ingress-controller/commit/51c074539125ee3240bb8d6326395197c588587e) docs: update API references (#1459)
* [`8c3515d`](https://github.com/apache/apisix-ingress-controller/commit/8c3515dca1cd62b30829b2b6e1dac03ff5734007) chore(deps): bump github.com/onsi/ginkgo/v2 in /test/e2e (#1454)
* [`5570d28`](https://github.com/apache/apisix-ingress-controller/commit/5570d28ff80f22d9657ca1b5ddb0b71f63b4596b) chore: gateway-api v0.5.1 (#1445)
* [`32863b4`](https://github.com/apache/apisix-ingress-controller/commit/32863b4d27f97dbbd4926f5e4a2c0d45b3744fee) docs: update "Development" guide (#1443)
* [`e18f157`](https://github.com/apache/apisix-ingress-controller/commit/e18f1578e2a0bf816326283084a4f8f882b4c441) docs: update "FAQ" page (#1437)
* [`febeab4`](https://github.com/apache/apisix-ingress-controller/commit/febeab4cac49dd97a6b257324ba9ba3b5396c514) modify Dockerfile go version from 1.18 to 1.19 ,consist with go mod (#1446)
* [`d43fd97`](https://github.com/apache/apisix-ingress-controller/commit/d43fd9716bc40278c17d37afd3c202a322773d99) fix: ApisixPluginConfig shouldn't be deleted when ar or au be deleted. (#1439)
* [`6f83da5`](https://github.com/apache/apisix-ingress-controller/commit/6f83da5ca55105d48c8d502b56ccf0ab2190f29e) feat: support redirect and requestHeaderModifier in HTTPRoute filter (#1426)
* [`bfd058d`](https://github.com/apache/apisix-ingress-controller/commit/bfd058d87724a41861358a10d9a8ad62ae5977f9) fix: cluster.metricsCollector invoked before assign when MountWebhooks (#1428)
* [`38b12fb`](https://github.com/apache/apisix-ingress-controller/commit/38b12fb4a5a2169eb3585e5b7e2f78c8ce447862) feat: support sni based tls route (#1051)
* [`53f26c1`](https://github.com/apache/apisix-ingress-controller/commit/53f26c1b5c078b39b448f8adb7db27e662f5bd51) feat: delete "app_namespaces" param (#1429)
* [`1cfe95a`](https://github.com/apache/apisix-ingress-controller/commit/1cfe95afd536e4f5c180d66f07b9dbbed45b20a4) doc: fix server-secret.yaml in mtls.md (#1432)
* [`b128bff`](https://github.com/apache/apisix-ingress-controller/commit/b128bff8e39deb8e25734995f3e01f0623fbe0b4) chore: remove v2beta2 API Version (#1431)
* [`6b38e80`](https://github.com/apache/apisix-ingress-controller/commit/6b38e806b0862b98e7565934b47143230d23bad8) chore(deps): bump github.com/spf13/cobra from 1.6.0 to 1.6.1 (#1411)
* [`6879c81`](https://github.com/apache/apisix-ingress-controller/commit/6879c81fa5768db13e566d86b85f1c1fe9cf4073) feat: support ingress and backend service in different namespace (#1377)
* [`00855fa`](https://github.com/apache/apisix-ingress-controller/commit/00855fa4aa9b895f87d347e2bceabdbf576e0bb3) feat: support plugin_metadata of apisix (#1369)
* [`4e0749e`](https://github.com/apache/apisix-ingress-controller/commit/4e0749e6cb8bde21eb4ad8d4407bd0e9455230a8) docs: update "ApisixTls" and "ApisixClusterConfig" (#1414)
* [`c38ae66`](https://github.com/apache/apisix-ingress-controller/commit/c38ae6670c018a50a3bbda328c82cfbc3d8b8f59) chore(deps): bump github.com/eclipse/paho.mqtt.golang in /test/e2e (#1410)
* [`eff8ce1`](https://github.com/apache/apisix-ingress-controller/commit/eff8ce1b70513980d96f26e7ac526b6180e14856) chore(deps): bump github.com/stretchr/testify from 1.8.0 to 1.8.1 (#1404)
* [`f9f36d3`](https://github.com/apache/apisix-ingress-controller/commit/f9f36d3f30f65dd99f5fe72c7d419c6fb01a1274) docs: update ApisixUpstream docs (#1407)
* [`812ae50`](https://github.com/apache/apisix-ingress-controller/commit/812ae50d776f11abe24ed4f1496989007d33b8ba) chore(deps): bump github.com/stretchr/testify in /test/e2e (#1401)
* [`b734af3`](https://github.com/apache/apisix-ingress-controller/commit/b734af30604ccd642448c754c65e682bdb36da63) chore(deps): bump github.com/slok/kubewebhook/v2 from 2.3.0 to 2.5.0 (#1403)
* [`b52d357`](https://github.com/apache/apisix-ingress-controller/commit/b52d3577efed636d52906f1337f84f7f2635c036) chore(deps): bump github.com/spf13/cobra from 1.5.0 to 1.6.0 (#1402)
* [`cc33365`](https://github.com/apache/apisix-ingress-controller/commit/cc3336557de27673f8772b67c936d1de3f914771) chore(deps): bump github.com/onsi/ginkgo/v2 in /test/e2e (#1400)
* [`10c3d6e`](https://github.com/apache/apisix-ingress-controller/commit/10c3d6eb77274bea525580b63ed38f22d6c56d58) docs: update annotations page (#1399)
* [`3182dbc`](https://github.com/apache/apisix-ingress-controller/commit/3182dbc1ed6418cc573e9f0b313c69f25fb09de8) chore(deps): bump k8s.io/client-go from 0.25.2 to 0.25.3 (#1397)
* [`dcd57bb`](https://github.com/apache/apisix-ingress-controller/commit/dcd57bb86edd5e47993e871f6a1d659a66c485f6) feat: ingress extensions/v1beta1 support tls (#1392)
* [`5b8ae78`](https://github.com/apache/apisix-ingress-controller/commit/5b8ae78f37e42ef1e401f4bdd517318c4d75e07a) docs: fix typo (#1398)
* [`048b9a9`](https://github.com/apache/apisix-ingress-controller/commit/048b9a9f5a79cb7beb6d4285fdfbd97fc93d7105) docs: ensure parity in examples (#1396)
* [`f564cb7`](https://github.com/apache/apisix-ingress-controller/commit/f564cb78d7d511a4d1dcc807ab2ae9d0c3ac6fa5) docs: update ApisixRoute docs (#1391)
* [`5c79821`](https://github.com/apache/apisix-ingress-controller/commit/5c798213da804493d3664ae4bc39dfceb9686f0d) feat: support external service (#1306)
* [`7a89a0a`](https://github.com/apache/apisix-ingress-controller/commit/7a89a0a9792691167dad0b5556c95966d18bc455) chore(deps): bump k8s.io/code-generator from 0.25.1 to 0.25.3 (#1384)
* [`5f2c398`](https://github.com/apache/apisix-ingress-controller/commit/5f2c39815f483b30f4ae3305556564244563a589) chore(deps): bump k8s.io/client-go from 0.25.1 to 0.25.2 in /test/e2e (#1361)
* [`8a17eea`](https://github.com/apache/apisix-ingress-controller/commit/8a17eea26e96570dd1054f258e484bf7814627eb) feat: add Gateway UDPRoute (#1278)
* [`40f1372`](https://github.com/apache/apisix-ingress-controller/commit/40f1372d7502a5044a782200b3a339d2a7024400) chore: release v1.5.0 (#1360) (#1373)
* [`7a6dcfb`](https://github.com/apache/apisix-ingress-controller/commit/7a6dcfbb9a67f1f1a3c473e0495ea6c1ed3d6a4b) docs: update golang version to 1.19 (#1370)
* [`3877ee8`](https://github.com/apache/apisix-ingress-controller/commit/3877ee843b67bf72b95bfdbd8b24d7fe292a3d1a) feat: support Gateway API TCPRoute (#1217)
* [`f71b376`](https://github.com/apache/apisix-ingress-controller/commit/f71b376291c5c8e6ca7681bf047b7e2e7c363068) e2e: remove debug log (#1358)
* [`3619b74`](https://github.com/apache/apisix-ingress-controller/commit/3619b741fd8a2779c92b0b3ae0a95bed3bc3cd4b) chore(deps): bump k8s.io/xxx from 0.24.4 to 0.25.1 and Go 1.19 (#1290)
* [`1f3983e`](https://github.com/apache/apisix-ingress-controller/commit/1f3983e67f389ceaeaa1dfe85de9c40e70b25c08) modify powered-by.md (#1350)
* [`e51a2c7`](https://github.com/apache/apisix-ingress-controller/commit/e51a2c70f42e670d470469277cf962c45e1d3d51) feat: update secret referenced by ingress (#1243)
* [`7bd6a03`](https://github.com/apache/apisix-ingress-controller/commit/7bd6a037fa1c34b812b9f3d12229d83063b57674) docs: update user cases (#1337)
* [`654aaec`](https://github.com/apache/apisix-ingress-controller/commit/654aaecde48f972cadd29be884735fc931a90b57) docs: add slack invitation badge (#1333)
* [`f296f11`](https://github.com/apache/apisix-ingress-controller/commit/f296f118542f93b28b9673197dcc81be181d2685) fix: Using different protocols at the same time in ApisixUpstream (#1331)
* [`4fa3b56`](https://github.com/apache/apisix-ingress-controller/commit/4fa3b56fd6a962c7d389ab1d7903e8015888c363) fix: crd resource status is not updated (#1335)
* [`3fd6112`](https://github.com/apache/apisix-ingress-controller/commit/3fd6112ccc6303231c55cd018af024ac4eca1ef7) docs: Add KubeGems to powered-by.md (#1334)
* [`85bcfbc`](https://github.com/apache/apisix-ingress-controller/commit/85bcfbc9f5e697f367c33382a7410b446dc39cbb) chore(deps): bump github.com/gruntwork-io/terratest in /test/e2e (#1327)
* [`9d663ab`](https://github.com/apache/apisix-ingress-controller/commit/9d663abec02f09ece4af9aa99713de3e878146f6) fix: support resolveGranularity of ApisixRoute (#1251)
* [`5c0ea2b`](https://github.com/apache/apisix-ingress-controller/commit/5c0ea2b42138b0d0df59d85936d2b72feaa669c5) feat: support update and delete of HTTPRoute (#1315)
* [`94dbbed`](https://github.com/apache/apisix-ingress-controller/commit/94dbbed486897ca5ab3790808b779c9a394a1b46) chore(deps): bump github.com/gruntwork-io/terratest in /test/e2e (#1319)
* [`866d40f`](https://github.com/apache/apisix-ingress-controller/commit/866d40f60eb5909ae9a4fb910f79f21bcd3ca7dd) chore(deps): bump github.com/onsi/ginkgo/v2 in /test/e2e (#1318)
* [`848a78f`](https://github.com/apache/apisix-ingress-controller/commit/848a78f494cf42b42858e1a42a3b89f1f2ae158b) fix: ingress class not effect in resource sync logic (#1311)
* [`d6f1352`](https://github.com/apache/apisix-ingress-controller/commit/d6f13521af3e89ed9f0408781953adfaeb537378) fix: type assertion failed (#1314)
* [`0b999ec`](https://github.com/apache/apisix-ingress-controller/commit/0b999ec1d3087f7bf0037acbe2d398eaa0efab2a) chore(refactor): annotations handle (#1245)
* [`9dafeb8`](https://github.com/apache/apisix-ingress-controller/commit/9dafeb88c94aa57526e3eb4e33ae439030d34c4f) chore: protect v1.5.0 and enable CI for it (#1294)
* [`cb6c696`](https://github.com/apache/apisix-ingress-controller/commit/cb6c6963816aa41a6b843bff351bcdf4dc4e5fa5) docs: add powered-by.md (#1293)
* [`6b86d2a`](https://github.com/apache/apisix-ingress-controller/commit/6b86d2a15f9d1e245ec06d2c8fb1d4c65b7c96b2) e2e: delete duplicate log data on failure (#1297)
* [`31b3ef8`](https://github.com/apache/apisix-ingress-controller/commit/31b3ef84be4e96dc663de883ef5730a03bfb962b) docs: add ApisixUpstream healthCheck explanation to resolveGranularity (#1279)
* [`f1bd4c0`](https://github.com/apache/apisix-ingress-controller/commit/f1bd4c026f1652318afc7b2aabe9590c2882247b) fix config missing default_cluster_name yaml (#1277)
* [`1087941`](https://github.com/apache/apisix-ingress-controller/commit/1087941d827dbf859b534328368ff8c65635db2a) fix: namespace_selector invalid when restarting (#1238)
* [`c4b04b3`](https://github.com/apache/apisix-ingress-controller/commit/c4b04b3a177e64705124bdc0b1462ac3d3f31e3f) chore(deps): bump github.com/eclipse/paho.mqtt.golang in /test/e2e (#1255)
* [`5e844e4`](https://github.com/apache/apisix-ingress-controller/commit/5e844e44b4f58458574f9e570b2244945ee92d3a) docs: update installation guide (#1272)
* [`ef07421`](https://github.com/apache/apisix-ingress-controller/commit/ef07421700cff966a24d6b3b391feaed543be717) ci: set default_branch (#1274)
* [`20eb64e`](https://github.com/apache/apisix-ingress-controller/commit/20eb64ea568ee8acd2e2b1f576cdd154935b6a14) fix: object type should be apisix_upstream and endpointslice and apisix_cluster_config (#1268)
* [`4eede7e`](https://github.com/apache/apisix-ingress-controller/commit/4eede7e96ad60da3388041eeb5a2d4366c4901a3) chore(deps): bump deps from 0.24.3 to 0.24.4 (#1265)
* [`f802271`](https://github.com/apache/apisix-ingress-controller/commit/f802271de1d0d8a39730a0d177a79b83b7af6c11) chore(deps): bump github.com/gruntwork-io/terratest in /test/e2e (#1259)
* [`530ce52`](https://github.com/apache/apisix-ingress-controller/commit/530ce52b278d7e385db3747d33e0d9fe1db0f3d1) feat: add mqtt-proxy plugin in ApisixRoute (#1056)
* [`1a29306`](https://github.com/apache/apisix-ingress-controller/commit/1a2930646ea6127d41fcf12236b3a0e5fb180dda) docs: update "Getting started" guide (#1247)
* [`2fa8a9c`](https://github.com/apache/apisix-ingress-controller/commit/2fa8a9c00e9fd3baa4884bb679879fc659fd218a) fix: log Secret name instead of all data (#1216)
* [`35c9f6b`](https://github.com/apache/apisix-ingress-controller/commit/35c9f6b935454e17384aaefe852a9c25d830b864) fix: nodes convert failed (#1222)
* [`356b220`](https://github.com/apache/apisix-ingress-controller/commit/356b220f4b92c644e8a937164fca2517ed3e6a4f) fix: TestRotateLog (#1246)
* [`e2b68f4`](https://github.com/apache/apisix-ingress-controller/commit/e2b68f455812c872d6e5cb49f2b76355542ba898) chore(deps): bump go.uber.org/zap and github.com/prometheus/client_golang (#1244)
* [`6728776`](https://github.com/apache/apisix-ingress-controller/commit/67287764bebacc867166822807e1e708042f22ea) docs: Fix typo on plugin config name (#1241)
* [`fcfa188`](https://github.com/apache/apisix-ingress-controller/commit/fcfa1882957a1d111c616c1ef646b98a0fb6a70f) feat: add log rotate (#1200)
* [`1c4e7f3`](https://github.com/apache/apisix-ingress-controller/commit/1c4e7f371e6893d9bd3cf96b7ec9ec74ad8e96c7) chore: update contributor over time link for README (#1239)
* [`7115cef`](https://github.com/apache/apisix-ingress-controller/commit/7115cefa6f2a97b1ded5ef1911bca1e7959664a2) chore(deps): bump github.com/gruntwork-io/terratest in /test/e2e (#1228)
* [`537501c`](https://github.com/apache/apisix-ingress-controller/commit/537501cd3415dbd71f45a539f57c757d9a634549) test: add cronjob runs e2e in multiple k8s versions (#1203)
* [`d32c728`](https://github.com/apache/apisix-ingress-controller/commit/d32c728139a706bbad155a5e58514a3e59a841e4) feat: Restruct pkg/ingress (#1204)
* [`dfcbaac`](https://github.com/apache/apisix-ingress-controller/commit/dfcbaac8f2b8c9c5ece12e3454fa57a2a23dba65) docs: add installation on KIND to index (#1220)
* [`e08b2e0`](https://github.com/apache/apisix-ingress-controller/commit/e08b2e0eea3473ccb5c841cfe2ac5daf8395f305) chore: v1.5.0-rc1 release (#1219)
* [`92c1adb`](https://github.com/apache/apisix-ingress-controller/commit/92c1adb382c0f813ddad3ee32700301eb3181228) doc: Refactor the README (#1215)
* [`96df45e`](https://github.com/apache/apisix-ingress-controller/commit/96df45eef553edea5b47664fb0543de48e2c4b6b) docs: fix dead link (#1211)
</p>
</details>

### Dependency Changes

* **github.com/hashicorp/go-immutable-radix**  v1.3.0 -> v1.3.1
* **github.com/hashicorp/go-memdb**            v1.3.3 -> v1.3.4
* **github.com/imdario/mergo**                 v0.3.12 -> v0.3.13
* **github.com/inconshreveable/mousetrap**     v1.0.0 -> v1.0.1
* **github.com/incubator4/go-resty-expr**      v0.1.1 **_new_**
* **github.com/prometheus/client_golang**      v1.12.2 -> v1.14.0
* **github.com/prometheus/client_model**       v0.2.0 -> v0.3.0
* **github.com/prometheus/common**             v0.32.1 -> v0.37.0
* **github.com/prometheus/procfs**             v0.7.3 -> v0.8.0
* **github.com/spf13/cobra**                   v1.5.0 -> v1.6.1
* **github.com/stretchr/testify**              v1.8.0 -> v1.8.1
* **go.uber.org/atomic**                       v1.7.0 -> v1.10.0
* **go.uber.org/zap**                          v1.23.0 -> v1.24.0
* **google.golang.org/protobuf**               v1.28.0 -> v1.28.1
* **gopkg.in/natefinch/lumberjack.v2**         v2.0.0 **_new_**
* **k8s.io/api**                               v0.25.1 -> v0.25.4
* **k8s.io/apimachinery**                      v0.25.1 -> v0.25.4
* **k8s.io/client-go**                         v0.25.1 -> v0.25.4
* **k8s.io/code-generator**                    v0.25.1 -> v0.25.4
* **sigs.k8s.io/gateway-api**                  v0.4.0 -> v0.5.1

Previous release can be found at [1.5.0](https://github.com/apache/apisix-ingress-controller/releases/tag/1.5.0)

# 1.6.0-rc1

Welcome to the 1.6.0-rc1 release of apisix-ingress-controller!

This is a feature release.

## Highlights

The API version of all custom resources has been upgraded to v2 in v1.5 release. In 1.6 we removed the v2beta2 API. Please go to [#707](https://github.com/apache/apisix-ingress-controller/issues/707) for detail.

In this version we added more Gateway API support. e.g. TCPRoute/UDPRoute/HTTPRouteFilter etc.

From this version, we can proxy external services and external name services. And integrated with the service discovery component.

Please try out the release binaries and report any issues at
https://github.com/apache/apisix-ingress-controller/issues.

### Contributors

* dependabot[bot]
* Jintao Zhang
* Xin Rong
* Navendu Pottekkat
* Xin Rong
* Sarasa Kisaragi
* Ashing Zheng
* xiangtianyu
* 林靖
* Floyd
* Navendu Pottekkat
* dongjunduo
* lsy
* seven dickens
* Baoyuan
* Gallardot
* Jayson Reis
* LinkMaq
* Marco Aurelio Caldas Miranda
* Nicolas Frankel
* Qi Guo
* StevenBrown008
* Young
* Yousri
* YuanYingdong
* cmssczy
* incubator4
* mango
* redtacs
* soulbird
* thomas
* xianshun163
* 失眠是真滴难受

### Changes
<details><summary>125 commits</summary>
<p>

* [`67d60fe`](https://github.com/apache/apisix-ingress-controller/commit/67d60fe9858f89f0e4ad575e4e0f5ed540fe5ef5) docs: add external service discovery tutorial (#1535)
* [`f162f71`](https://github.com/apache/apisix-ingress-controller/commit/f162f7119abd76b5a71c285fbfae68ed2faf88fb) feat: support for specifying port in external services (#1500)
* [`4208ca7`](https://github.com/apache/apisix-ingress-controller/commit/4208ca7cef4e54e22544050deed45bd768ad5ffa) refactor: unified factory and informer (#1530)
* [`a118727`](https://github.com/apache/apisix-ingress-controller/commit/a118727200524150b9062ba915bf50d361b2a9e1) docs: update Ingress controller httpbin tutorial (#1524)
* [`c0cb74d`](https://github.com/apache/apisix-ingress-controller/commit/c0cb74dd66c1c040339160905bf5e9fad0d6fe1a) docs: add external service tutorial (#1527)
* [`d22a6fc`](https://github.com/apache/apisix-ingress-controller/commit/d22a6fc820f7699af411b8ecaa971307cfc82dbd) chore(deps): bump github.com/onsi/ginkgo/v2 in /test/e2e (#1506)
* [`c4cedad`](https://github.com/apache/apisix-ingress-controller/commit/c4cedad549215c90b95ba389553c94370fe07a12) chore(deps): bump go.uber.org/zap from 1.23.0 to 1.24.0 (#1510)
* [`c6c2742`](https://github.com/apache/apisix-ingress-controller/commit/c6c2742fe9fe60efd40f2c0ecc5c2fc7f2166a2a) chore(deps): bump github.com/gavv/httpexpect/v2 in /test/e2e (#1509)
* [`b4255f1`](https://github.com/apache/apisix-ingress-controller/commit/b4255f1dd9cc69e7a3a44deeeeb8d4f17aea9e50) ci: using ubuntu-20.04 by default (#1504)
* [`03cfcb8`](https://github.com/apache/apisix-ingress-controller/commit/03cfcb840690759e80eaffd2eff22a43e8aa07b6) chore(deps): bump k8s.io/client-go from 0.25.4 to 0.26.0 in /test/e2e (#1505)
* [`0009b5d`](https://github.com/apache/apisix-ingress-controller/commit/0009b5d6951c89d2b67e0a440d4a75952fb3154c) feat: support secret plugin config (#1486)
* [`8cf79c2`](https://github.com/apache/apisix-ingress-controller/commit/8cf79c2e8f7278b52bf83f6cab6e85ab73d7266f) fix: ingress.tls secret not found (#1394)
* [`7e8f076`](https://github.com/apache/apisix-ingress-controller/commit/7e8f0763a595d566804ec397cfcb03214f2477df) fix: many namespace lead to provider stuck (#1386)
* [`2ce1ed3`](https://github.com/apache/apisix-ingress-controller/commit/2ce1ed3ebfb9c5041d28445f8747e1665d4207b4) chore: use httptest.NewRequest instead of http.Request as incoming server request in test case (#1498)
* [`768a35f`](https://github.com/apache/apisix-ingress-controller/commit/768a35f66c879bc35ef71e1f9ec4caa1ec94d3b9) feat: add Ingress annotation to support response-rewrite (#1487)
* [`051fc48`](https://github.com/apache/apisix-ingress-controller/commit/051fc48de133699cfd5d12e28358226913550871) e2e: support docker hub as registry (#1489)
* [`cc48ae9`](https://github.com/apache/apisix-ingress-controller/commit/cc48ae9bc34299f56507eb9539c1c57602aa9638) chore: use field cluster.name as value but not "default" (#1495)
* [`931ab06`](https://github.com/apache/apisix-ingress-controller/commit/931ab0699ff1d5791484928492f101101365aee9) feat: ingress annotations supports the specified upstream schema (#1451)
* [`de9f84f`](https://github.com/apache/apisix-ingress-controller/commit/de9f84fce92e9ad26d487ccafa4bb7e800176776) doc: update upgrade guide (#1479)
* [`7511166`](https://github.com/apache/apisix-ingress-controller/commit/7511166a7c55f8cd0578c106515b15fdf3f058c6) chore(deps): bump go.uber.org/zap from 1.23.0 to 1.24.0 in /test/e2e (#1488)
* [`afbc4f7`](https://github.com/apache/apisix-ingress-controller/commit/afbc4f7369481615b8d55488392200f3b29b123a) docs: fix typo (#1491)
* [`1097792`](https://github.com/apache/apisix-ingress-controller/commit/109779232f607f13307de52f2f667f8d060ec25c) chore: replace io/ioutil package (#1485)
* [`ed92690`](https://github.com/apache/apisix-ingress-controller/commit/ed92690f5aabb4ece4b92d860d72d85bdfa23db0) fix:sanitize log output when exposing sensitive values (#1480)
* [`8e39e71`](https://github.com/apache/apisix-ingress-controller/commit/8e39e71002e44d303ecbad274317561aeb62db3d) feat: add control http method using kubernetes ingress by annotations (#1471)
* [`bccf762`](https://github.com/apache/apisix-ingress-controller/commit/bccf762ac1b6386e4bd8911180ea13ac5a14bdfe) chore: bump actions. (#1484)
* [`adf7d27`](https://github.com/apache/apisix-ingress-controller/commit/adf7d27033ef103ef483486f4ba155d0e9aee471) docs: add more user cases (#1482)
* [`0f7b3f3`](https://github.com/apache/apisix-ingress-controller/commit/0f7b3f375f1fbf35ebb374005c7c7690ed2df337) Revert "chore: update actions and add more user cases. (#1478)" (#1481)
* [`14353d3`](https://github.com/apache/apisix-ingress-controller/commit/14353d30bee1263e16e0b8cd96f6616d5bdae95a) chore(deps): bump github.com/onsi/ginkgo/v2 in /test/e2e (#1477)
* [`ee99eba`](https://github.com/apache/apisix-ingress-controller/commit/ee99ebaaabf9a3f51eb798a904f88483cac907a3) chore(deps): bump github.com/gavv/httpexpect/v2 in /test/e2e (#1476)
* [`7471e64`](https://github.com/apache/apisix-ingress-controller/commit/7471e6493729227964b17acc31bcd6a36d15cdca) chore: update actions and add more user cases. (#1478)
* [`136d40d`](https://github.com/apache/apisix-ingress-controller/commit/136d40d7ed1c8ff7d871f183e34417f6b9e7c50c) chore(deps): some deps bump (#1475)
* [`c2cea69`](https://github.com/apache/apisix-ingress-controller/commit/c2cea69db6b158714c98b7b8710716fd24a74b27) docs: change ingress class annotation to spec.ingressClassName (#1425)
* [`c8d3bd5`](https://github.com/apache/apisix-ingress-controller/commit/c8d3bd52fd8820475be763cd52b51b981382f285) feat: add support for integrate with DP service discovery (#1465)
* [`ec88b49`](https://github.com/apache/apisix-ingress-controller/commit/ec88b49cdc3164c44cef990482fc29ca7490aa3c) chore(deps): bump k8s.io/code-generator from 0.25.3 to 0.25.4 (#1456)
* [`803fdeb`](https://github.com/apache/apisix-ingress-controller/commit/803fdeb1f3ebcdb49d31d5b1702b172407c21c66) chore(deps): bump k8s.io/client-go from 0.25.3 to 0.25.4 (#1458)
* [`e6eb3bf`](https://github.com/apache/apisix-ingress-controller/commit/e6eb3bf2aed4c39e91271e4f74b61ff6fe97ad16) feat: support variable in ApisixRoute exprs scope (#1466)
* [`be7edf6`](https://github.com/apache/apisix-ingress-controller/commit/be7edf6b880e4904553c98f57dd78766632e1520) feat: support apisix v3 admin api (#1449)
* [`d95ae08`](https://github.com/apache/apisix-ingress-controller/commit/d95ae083057d561522d68376d9ad9c4c22ae51cd) docs: Add more descriptions and examples in the prometheus doc (#1467)
* [`d610041`](https://github.com/apache/apisix-ingress-controller/commit/d610041dbd8cfb76c424250b8e9e8fa7b1a1b22f) chore(deps): bump k8s.io/client-go from 0.25.3 to 0.25.4 in /test/e2e (#1453)
* [`632d5c1`](https://github.com/apache/apisix-ingress-controller/commit/632d5c1291bc0775d6280a58536ca9902676541f) feat: support HTTPRequestMirrorFilter in HTTPRoute (#1461)
* [`47906a5`](https://github.com/apache/apisix-ingress-controller/commit/47906a533369dc76c3d2f7fcba10ce8030bd4ce6) docs: update ApisixRoute/v2 reference (#1423)
* [`6ec804f`](https://github.com/apache/apisix-ingress-controller/commit/6ec804f454f56b4f6d2a2a6c6adabfcd05404aef) feat(makefile): allow to custom registry port for `make kind-up` (#1417)
* [`a318f49`](https://github.com/apache/apisix-ingress-controller/commit/a318f4990640d3a743045b5945440ef7900096ff) docs: update ApisixUpstream reference (#1450)
* [`51c0745`](https://github.com/apache/apisix-ingress-controller/commit/51c074539125ee3240bb8d6326395197c588587e) docs: update API references (#1459)
* [`8c3515d`](https://github.com/apache/apisix-ingress-controller/commit/8c3515dca1cd62b30829b2b6e1dac03ff5734007) chore(deps): bump github.com/onsi/ginkgo/v2 in /test/e2e (#1454)
* [`5570d28`](https://github.com/apache/apisix-ingress-controller/commit/5570d28ff80f22d9657ca1b5ddb0b71f63b4596b) chore: gateway-api v0.5.1 (#1445)
* [`32863b4`](https://github.com/apache/apisix-ingress-controller/commit/32863b4d27f97dbbd4926f5e4a2c0d45b3744fee) docs: update "Development" guide (#1443)
* [`e18f157`](https://github.com/apache/apisix-ingress-controller/commit/e18f1578e2a0bf816326283084a4f8f882b4c441) docs: update "FAQ" page (#1437)
* [`febeab4`](https://github.com/apache/apisix-ingress-controller/commit/febeab4cac49dd97a6b257324ba9ba3b5396c514) modify Dockerfile go version from 1.18 to 1.19 ,consist with go mod (#1446)
* [`d43fd97`](https://github.com/apache/apisix-ingress-controller/commit/d43fd9716bc40278c17d37afd3c202a322773d99) fix: ApisixPluginConfig shouldn't be deleted when ar or au be deleted. (#1439)
* [`6f83da5`](https://github.com/apache/apisix-ingress-controller/commit/6f83da5ca55105d48c8d502b56ccf0ab2190f29e) feat: support redirect and requestHeaderModifier in HTTPRoute filter (#1426)
* [`bfd058d`](https://github.com/apache/apisix-ingress-controller/commit/bfd058d87724a41861358a10d9a8ad62ae5977f9) fix: cluster.metricsCollector invoked before assign when MountWebhooks (#1428)
* [`38b12fb`](https://github.com/apache/apisix-ingress-controller/commit/38b12fb4a5a2169eb3585e5b7e2f78c8ce447862) feat: support sni based tls route (#1051)
* [`53f26c1`](https://github.com/apache/apisix-ingress-controller/commit/53f26c1b5c078b39b448f8adb7db27e662f5bd51) feat: delete "app_namespaces" param (#1429)
* [`1cfe95a`](https://github.com/apache/apisix-ingress-controller/commit/1cfe95afd536e4f5c180d66f07b9dbbed45b20a4) doc: fix server-secret.yaml in mtls.md (#1432)
* [`b128bff`](https://github.com/apache/apisix-ingress-controller/commit/b128bff8e39deb8e25734995f3e01f0623fbe0b4) chore: remove v2beta2 API Version (#1431)
* [`6b38e80`](https://github.com/apache/apisix-ingress-controller/commit/6b38e806b0862b98e7565934b47143230d23bad8) chore(deps): bump github.com/spf13/cobra from 1.6.0 to 1.6.1 (#1411)
* [`6879c81`](https://github.com/apache/apisix-ingress-controller/commit/6879c81fa5768db13e566d86b85f1c1fe9cf4073) feat: support ingress and backend service in different namespace (#1377)
* [`00855fa`](https://github.com/apache/apisix-ingress-controller/commit/00855fa4aa9b895f87d347e2bceabdbf576e0bb3) feat: support plugin_metadata of apisix (#1369)
* [`4e0749e`](https://github.com/apache/apisix-ingress-controller/commit/4e0749e6cb8bde21eb4ad8d4407bd0e9455230a8) docs: update "ApisixTls" and "ApisixClusterConfig" (#1414)
* [`c38ae66`](https://github.com/apache/apisix-ingress-controller/commit/c38ae6670c018a50a3bbda328c82cfbc3d8b8f59) chore(deps): bump github.com/eclipse/paho.mqtt.golang in /test/e2e (#1410)
* [`eff8ce1`](https://github.com/apache/apisix-ingress-controller/commit/eff8ce1b70513980d96f26e7ac526b6180e14856) chore(deps): bump github.com/stretchr/testify from 1.8.0 to 1.8.1 (#1404)
* [`f9f36d3`](https://github.com/apache/apisix-ingress-controller/commit/f9f36d3f30f65dd99f5fe72c7d419c6fb01a1274) docs: update ApisixUpstream docs (#1407)
* [`812ae50`](https://github.com/apache/apisix-ingress-controller/commit/812ae50d776f11abe24ed4f1496989007d33b8ba) chore(deps): bump github.com/stretchr/testify in /test/e2e (#1401)
* [`b734af3`](https://github.com/apache/apisix-ingress-controller/commit/b734af30604ccd642448c754c65e682bdb36da63) chore(deps): bump github.com/slok/kubewebhook/v2 from 2.3.0 to 2.5.0 (#1403)
* [`b52d357`](https://github.com/apache/apisix-ingress-controller/commit/b52d3577efed636d52906f1337f84f7f2635c036) chore(deps): bump github.com/spf13/cobra from 1.5.0 to 1.6.0 (#1402)
* [`cc33365`](https://github.com/apache/apisix-ingress-controller/commit/cc3336557de27673f8772b67c936d1de3f914771) chore(deps): bump github.com/onsi/ginkgo/v2 in /test/e2e (#1400)
* [`10c3d6e`](https://github.com/apache/apisix-ingress-controller/commit/10c3d6eb77274bea525580b63ed38f22d6c56d58) docs: update annotations page (#1399)
* [`3182dbc`](https://github.com/apache/apisix-ingress-controller/commit/3182dbc1ed6418cc573e9f0b313c69f25fb09de8) chore(deps): bump k8s.io/client-go from 0.25.2 to 0.25.3 (#1397)
* [`dcd57bb`](https://github.com/apache/apisix-ingress-controller/commit/dcd57bb86edd5e47993e871f6a1d659a66c485f6) feat: ingress extensions/v1beta1 support tls (#1392)
* [`5b8ae78`](https://github.com/apache/apisix-ingress-controller/commit/5b8ae78f37e42ef1e401f4bdd517318c4d75e07a) docs: fix typo (#1398)
* [`048b9a9`](https://github.com/apache/apisix-ingress-controller/commit/048b9a9f5a79cb7beb6d4285fdfbd97fc93d7105) docs: ensure parity in examples (#1396)
* [`f564cb7`](https://github.com/apache/apisix-ingress-controller/commit/f564cb78d7d511a4d1dcc807ab2ae9d0c3ac6fa5) docs: update ApisixRoute docs (#1391)
* [`5c79821`](https://github.com/apache/apisix-ingress-controller/commit/5c798213da804493d3664ae4bc39dfceb9686f0d) feat: support external service (#1306)
* [`7a89a0a`](https://github.com/apache/apisix-ingress-controller/commit/7a89a0a9792691167dad0b5556c95966d18bc455) chore(deps): bump k8s.io/code-generator from 0.25.1 to 0.25.3 (#1384)
* [`5f2c398`](https://github.com/apache/apisix-ingress-controller/commit/5f2c39815f483b30f4ae3305556564244563a589) chore(deps): bump k8s.io/client-go from 0.25.1 to 0.25.2 in /test/e2e (#1361)
* [`8a17eea`](https://github.com/apache/apisix-ingress-controller/commit/8a17eea26e96570dd1054f258e484bf7814627eb) feat: add Gateway UDPRoute (#1278)
* [`40f1372`](https://github.com/apache/apisix-ingress-controller/commit/40f1372d7502a5044a782200b3a339d2a7024400) chore: release v1.5.0 (#1360) (#1373)
* [`7a6dcfb`](https://github.com/apache/apisix-ingress-controller/commit/7a6dcfbb9a67f1f1a3c473e0495ea6c1ed3d6a4b) docs: update golang version to 1.19 (#1370)
* [`3877ee8`](https://github.com/apache/apisix-ingress-controller/commit/3877ee843b67bf72b95bfdbd8b24d7fe292a3d1a) feat: support Gateway API TCPRoute (#1217)
* [`f71b376`](https://github.com/apache/apisix-ingress-controller/commit/f71b376291c5c8e6ca7681bf047b7e2e7c363068) e2e: remove debug log (#1358)
* [`3619b74`](https://github.com/apache/apisix-ingress-controller/commit/3619b741fd8a2779c92b0b3ae0a95bed3bc3cd4b) chore(deps): bump k8s.io/xxx from 0.24.4 to 0.25.1 and Go 1.19 (#1290)
* [`1f3983e`](https://github.com/apache/apisix-ingress-controller/commit/1f3983e67f389ceaeaa1dfe85de9c40e70b25c08) modify powered-by.md (#1350)
* [`e51a2c7`](https://github.com/apache/apisix-ingress-controller/commit/e51a2c70f42e670d470469277cf962c45e1d3d51) feat: update secret referenced by ingress (#1243)
* [`7bd6a03`](https://github.com/apache/apisix-ingress-controller/commit/7bd6a037fa1c34b812b9f3d12229d83063b57674) docs: update user cases (#1337)
* [`654aaec`](https://github.com/apache/apisix-ingress-controller/commit/654aaecde48f972cadd29be884735fc931a90b57) docs: add slack invitation badge (#1333)
* [`f296f11`](https://github.com/apache/apisix-ingress-controller/commit/f296f118542f93b28b9673197dcc81be181d2685) fix: Using different protocols at the same time in ApisixUpstream (#1331)
* [`4fa3b56`](https://github.com/apache/apisix-ingress-controller/commit/4fa3b56fd6a962c7d389ab1d7903e8015888c363) fix: crd resource status is not updated (#1335)
* [`3fd6112`](https://github.com/apache/apisix-ingress-controller/commit/3fd6112ccc6303231c55cd018af024ac4eca1ef7) docs: Add KubeGems to powered-by.md (#1334)
* [`85bcfbc`](https://github.com/apache/apisix-ingress-controller/commit/85bcfbc9f5e697f367c33382a7410b446dc39cbb) chore(deps): bump github.com/gruntwork-io/terratest in /test/e2e (#1327)
* [`9d663ab`](https://github.com/apache/apisix-ingress-controller/commit/9d663abec02f09ece4af9aa99713de3e878146f6) fix: support resolveGranularity of ApisixRoute (#1251)
* [`5c0ea2b`](https://github.com/apache/apisix-ingress-controller/commit/5c0ea2b42138b0d0df59d85936d2b72feaa669c5) feat: support update and delete of HTTPRoute (#1315)
* [`94dbbed`](https://github.com/apache/apisix-ingress-controller/commit/94dbbed486897ca5ab3790808b779c9a394a1b46) chore(deps): bump github.com/gruntwork-io/terratest in /test/e2e (#1319)
* [`866d40f`](https://github.com/apache/apisix-ingress-controller/commit/866d40f60eb5909ae9a4fb910f79f21bcd3ca7dd) chore(deps): bump github.com/onsi/ginkgo/v2 in /test/e2e (#1318)
* [`848a78f`](https://github.com/apache/apisix-ingress-controller/commit/848a78f494cf42b42858e1a42a3b89f1f2ae158b) fix: ingress class not effect in resource sync logic (#1311)
* [`d6f1352`](https://github.com/apache/apisix-ingress-controller/commit/d6f13521af3e89ed9f0408781953adfaeb537378) fix: type assertion failed (#1314)
* [`0b999ec`](https://github.com/apache/apisix-ingress-controller/commit/0b999ec1d3087f7bf0037acbe2d398eaa0efab2a) chore(refactor): annotations handle (#1245)
* [`9dafeb8`](https://github.com/apache/apisix-ingress-controller/commit/9dafeb88c94aa57526e3eb4e33ae439030d34c4f) chore: protect v1.5.0 and enable CI for it (#1294)
* [`cb6c696`](https://github.com/apache/apisix-ingress-controller/commit/cb6c6963816aa41a6b843bff351bcdf4dc4e5fa5) docs: add powered-by.md (#1293)
* [`6b86d2a`](https://github.com/apache/apisix-ingress-controller/commit/6b86d2a15f9d1e245ec06d2c8fb1d4c65b7c96b2) e2e: delete duplicate log data on failure (#1297)
* [`31b3ef8`](https://github.com/apache/apisix-ingress-controller/commit/31b3ef84be4e96dc663de883ef5730a03bfb962b) docs: add ApisixUpstream healthCheck explanation to resolveGranularity (#1279)
* [`f1bd4c0`](https://github.com/apache/apisix-ingress-controller/commit/f1bd4c026f1652318afc7b2aabe9590c2882247b) fix config missing default_cluster_name yaml (#1277)
* [`1087941`](https://github.com/apache/apisix-ingress-controller/commit/1087941d827dbf859b534328368ff8c65635db2a) fix: namespace_selector invalid when restarting (#1238)
* [`c4b04b3`](https://github.com/apache/apisix-ingress-controller/commit/c4b04b3a177e64705124bdc0b1462ac3d3f31e3f) chore(deps): bump github.com/eclipse/paho.mqtt.golang in /test/e2e (#1255)
* [`5e844e4`](https://github.com/apache/apisix-ingress-controller/commit/5e844e44b4f58458574f9e570b2244945ee92d3a) docs: update installation guide (#1272)
* [`ef07421`](https://github.com/apache/apisix-ingress-controller/commit/ef07421700cff966a24d6b3b391feaed543be717) ci: set default_branch (#1274)
* [`20eb64e`](https://github.com/apache/apisix-ingress-controller/commit/20eb64ea568ee8acd2e2b1f576cdd154935b6a14) fix: object type should be apisix_upstream and endpointslice and apisix_cluster_config (#1268)
* [`4eede7e`](https://github.com/apache/apisix-ingress-controller/commit/4eede7e96ad60da3388041eeb5a2d4366c4901a3) chore(deps): bump deps from 0.24.3 to 0.24.4 (#1265)
* [`f802271`](https://github.com/apache/apisix-ingress-controller/commit/f802271de1d0d8a39730a0d177a79b83b7af6c11) chore(deps): bump github.com/gruntwork-io/terratest in /test/e2e (#1259)
* [`530ce52`](https://github.com/apache/apisix-ingress-controller/commit/530ce52b278d7e385db3747d33e0d9fe1db0f3d1) feat: add mqtt-proxy plugin in ApisixRoute (#1056)
* [`1a29306`](https://github.com/apache/apisix-ingress-controller/commit/1a2930646ea6127d41fcf12236b3a0e5fb180dda) docs: update "Getting started" guide (#1247)
* [`2fa8a9c`](https://github.com/apache/apisix-ingress-controller/commit/2fa8a9c00e9fd3baa4884bb679879fc659fd218a) fix: log Secret name instead of all data (#1216)
* [`35c9f6b`](https://github.com/apache/apisix-ingress-controller/commit/35c9f6b935454e17384aaefe852a9c25d830b864) fix: nodes convert failed (#1222)
* [`356b220`](https://github.com/apache/apisix-ingress-controller/commit/356b220f4b92c644e8a937164fca2517ed3e6a4f) fix: TestRotateLog (#1246)
* [`e2b68f4`](https://github.com/apache/apisix-ingress-controller/commit/e2b68f455812c872d6e5cb49f2b76355542ba898) chore(deps): bump go.uber.org/zap and github.com/prometheus/client_golang (#1244)
* [`6728776`](https://github.com/apache/apisix-ingress-controller/commit/67287764bebacc867166822807e1e708042f22ea) docs: Fix typo on plugin config name (#1241)
* [`fcfa188`](https://github.com/apache/apisix-ingress-controller/commit/fcfa1882957a1d111c616c1ef646b98a0fb6a70f) feat: add log rotate (#1200)
* [`1c4e7f3`](https://github.com/apache/apisix-ingress-controller/commit/1c4e7f371e6893d9bd3cf96b7ec9ec74ad8e96c7) chore: update contributor over time link for README (#1239)
* [`7115cef`](https://github.com/apache/apisix-ingress-controller/commit/7115cefa6f2a97b1ded5ef1911bca1e7959664a2) chore(deps): bump github.com/gruntwork-io/terratest in /test/e2e (#1228)
* [`537501c`](https://github.com/apache/apisix-ingress-controller/commit/537501cd3415dbd71f45a539f57c757d9a634549) test: add cronjob runs e2e in multiple k8s versions (#1203)
* [`d32c728`](https://github.com/apache/apisix-ingress-controller/commit/d32c728139a706bbad155a5e58514a3e59a841e4) feat: Restruct pkg/ingress (#1204)
* [`dfcbaac`](https://github.com/apache/apisix-ingress-controller/commit/dfcbaac8f2b8c9c5ece12e3454fa57a2a23dba65) docs: add installation on KIND to index (#1220)
* [`e08b2e0`](https://github.com/apache/apisix-ingress-controller/commit/e08b2e0eea3473ccb5c841cfe2ac5daf8395f305) chore: v1.5.0-rc1 release (#1219)
* [`92c1adb`](https://github.com/apache/apisix-ingress-controller/commit/92c1adb382c0f813ddad3ee32700301eb3181228) doc: Refactor the README (#1215)
* [`96df45e`](https://github.com/apache/apisix-ingress-controller/commit/96df45eef553edea5b47664fb0543de48e2c4b6b) docs: fix dead link (#1211)
</p>
</details>

### Dependency Changes

* **github.com/hashicorp/go-immutable-radix**  v1.3.0 -> v1.3.1
* **github.com/hashicorp/go-memdb**            v1.3.3 -> v1.3.4
* **github.com/imdario/mergo**                 v0.3.12 -> v0.3.13
* **github.com/inconshreveable/mousetrap**     v1.0.0 -> v1.0.1
* **github.com/incubator4/go-resty-expr**      v0.1.1 **_new_**
* **github.com/prometheus/client_golang**      v1.12.2 -> v1.14.0
* **github.com/prometheus/client_model**       v0.2.0 -> v0.3.0
* **github.com/prometheus/common**             v0.32.1 -> v0.37.0
* **github.com/prometheus/procfs**             v0.7.3 -> v0.8.0
* **github.com/spf13/cobra**                   v1.5.0 -> v1.6.1
* **github.com/stretchr/testify**              v1.8.0 -> v1.8.1
* **go.uber.org/atomic**                       v1.7.0 -> v1.10.0
* **go.uber.org/zap**                          v1.23.0 -> v1.24.0
* **google.golang.org/protobuf**               v1.28.0 -> v1.28.1
* **gopkg.in/natefinch/lumberjack.v2**         v2.0.0 **_new_**
* **k8s.io/api**                               v0.25.1 -> v0.25.4
* **k8s.io/apimachinery**                      v0.25.1 -> v0.25.4
* **k8s.io/client-go**                         v0.25.1 -> v0.25.4
* **k8s.io/code-generator**                    v0.25.1 -> v0.25.4
* **sigs.k8s.io/gateway-api**                  v0.4.0 -> v0.5.1

Previous release can be found at [1.5.0](https://github.com/apache/apisix-ingress-controller/releases/tag/1.5.0)

# 1.5.1

Welcome to the 1.5.1 release of apisix-ingress-controller!

This is a Patch version release.

Please try out the release binaries and report any issues at
https://github.com/apache/apisix-ingress-controller/issues.

### Contributors

* Jintao Zhang
* Young

### Changes
<details><summary>5 commits</summary>
<p>

* [`93930b7`](https://github.com/apache/apisix-ingress-controller/commit/93930b7b2e2c3cb465d303194abb40c66405ed6a) bug: failed to reflect pluginConfig delete to cache(#1439) (#1470)
* [`97e417b`](https://github.com/apache/apisix-ingress-controller/commit/97e417b8d9c66df655c0e9a6d0c7f9ebbce63757) fix: cluster.metricsCollector invoked before assign when MountWebhooks (#1428) (#1469)
* [`a288408`](https://github.com/apache/apisix-ingress-controller/commit/a288408ef71b7e7c456ba7e178013eefba8ee21c) cherry-pick #1331: fix: Using different protocols at the same time in ApisixUpstream (#1464)
* [`dd5acd3`](https://github.com/apache/apisix-ingress-controller/commit/dd5acd3b94321a54552e6f50c80cd61e6a97960d) docs: fix `server-secret.yaml` link in `mtls.md` (#1434)
* [`21f39e9`](https://github.com/apache/apisix-ingress-controller/commit/21f39e966dedb0765a9848302f8cb713aa461cfe) fix: handle v2 ApisixPluginConfig status (#1409)
</p>
</details>

### Dependency Changes

This release has no dependency changes

Previous release can be found at [1.5.0](https://github.com/apache/apisix-ingress-controller/releases/tag/1.5.0)

# 1.5.0

Welcome to the 1.5.0 release of apisix-ingress-controller!

This is a feature release.

## Highlights

The API version of all custom resources has been upgraded to v2 in this release and mark v2beta3 as deprecated. We plan to remove the v2beta2 API version in the next release. Please go to [#707](https://github.com/apache/apisix-ingress-controller/issues/707) for detail.

We have added partial support for Gateway API, which is not enabled by default, you can set `enable_gateway_api=true` to enable it.

Ingress resources can now use all APISIX plugin configurations by setting the annotation `k8s.apisix.apache.org/plugin-config-name=xxx`.

Please try out the release binaries and report any issues at
https://github.com/apache/apisix-ingress-controller/issues.

### Contributors

* Jintao Zhang
* Sarasa Kisaragi
* Xin Rong
* John Chever
* cmssczy
* dependabot[bot]
* nevercase
* Gallardot
* Nic
* lsy
* mango
* Fatpa
* Hoshea Jiang
* JasonZhu
* Yu.Bozhong
* seven dickens
* FesonX
* GhangZh
* JasonZhu
* Kowsz
* LetsGO
* Sindweller
* SkyeYoung
* Zack Sun
* bin-ya
* champly
* chen zhuo
* fengxsong
* greenhandatsjtu
* hahayyun
* hf400159
* wangyunpeng
* 罗泽轩

### Changes
<details><summary>154 commits</summary>
<p>

* [`f0a61d2`](https://github.com/apache/apisix-ingress-controller/commit/f0a61d2430ab22aefde174f5f628cfede9704664) [v1.5 cherry-pick]chore(deps): bump k8s.io/xxx from 0.24.4 to 0.25.1 and Go 1.19 (#1290) (#1357)
* [`1531680`](https://github.com/apache/apisix-ingress-controller/commit/1531680e668913e6007ec6ac1ea6622ec237d149) fix: crd resource status is not updated (#1335) (#1336)
* [`5a3ce7c`](https://github.com/apache/apisix-ingress-controller/commit/5a3ce7ca526652055b65d48e7448102657022f4f) fix: ingress class not effect in resource sync logic (#1311) (#1330)
* [`d43fda7`](https://github.com/apache/apisix-ingress-controller/commit/d43fda7f73d597e73ab59ccc4f728377e150ef9c) feat: support update and delete of HTTPRoute (#1315) (#1329)
* [`4abed4d`](https://github.com/apache/apisix-ingress-controller/commit/4abed4d7759b9175233cd1f3049d7782f1f7f2bc) [v1.5] fix: type assertion failed (#1303)
* [`8241b15`](https://github.com/apache/apisix-ingress-controller/commit/8241b153fb8220f141f0da77449442bcb3f53172) fix: namespace_selector invalid when restarting (#1238) (#1291)
* [`3b56ee3`](https://github.com/apache/apisix-ingress-controller/commit/3b56ee3cd9b9e40073e556fd799a62e573c3dcb8) chore: protect v1.5.0 and enable CI for it (#1294) (#1299)
* [`ca063cd`](https://github.com/apache/apisix-ingress-controller/commit/ca063cdfd400f6bd54421dd3168ebb4d42266fd8) fix: nodes convert failed (#1222) (#1250)
* [`0701f95`](https://github.com/apache/apisix-ingress-controller/commit/0701f9507e9ea66b163447ee7b64580d60beeaf9) chore: bump version
* [`652a79e`](https://github.com/apache/apisix-ingress-controller/commit/652a79e246fbd0f4aa62ef21e24be7ecaf37370d) chore: changelog for v1.5.0-rc1
* [`cccad72`](https://github.com/apache/apisix-ingress-controller/commit/cccad72a1e0ef60525c69371b4b27c4598c587c1) chore: mark v2beta3 deprecated (#1198)
* [`698ab6d`](https://github.com/apache/apisix-ingress-controller/commit/698ab6d52d4eef0d4404fa3bec43b20039ce9370) chore: Using APISIX 2.15.0 for CI (#1197)
* [`8094868`](https://github.com/apache/apisix-ingress-controller/commit/80948682d73cb7813d06f85e548392f19dc465f4) e2e: add sync case (#1196)
* [`37a8e5c`](https://github.com/apache/apisix-ingress-controller/commit/37a8e5c837cd2a8a459bec054c3ed73bac9b52bb) fix: translate error of old Ingress (#1193)
* [`339531f`](https://github.com/apache/apisix-ingress-controller/commit/339531f3eceb2ce24f07cc6b255fde94f201879a) doc: add a notice about the compatibility of Ingress and Dashboard (#1195)
* [`2cc586b`](https://github.com/apache/apisix-ingress-controller/commit/2cc586bbe06d3dd4c1c904fd487a9eaa13b5b3a2) fix: apisix_upstream sync panic (#1192)
* [`6cc718b`](https://github.com/apache/apisix-ingress-controller/commit/6cc718bb85a3d61f3702ba0514e5408d48a873da) helm: update deploy cluster role (#1131)
* [`3d720c0`](https://github.com/apache/apisix-ingress-controller/commit/3d720c0f9ba41fb2067d3a552a45d357c7a420f7) fix: translate error of old ApisixRoute (#1191)
* [`8b51c6e`](https://github.com/apache/apisix-ingress-controller/commit/8b51c6e173db12592f6fcfa8d59aaa6fd50e7922) docs: update all api-version to v2 (#1189)
* [`5f45b63`](https://github.com/apache/apisix-ingress-controller/commit/5f45b63b3d3716dd5de9f0288a6dd5c28228c40d) fix: ScopeQuery should be case sensitive (#1168) (#1188)
* [`af03e7a`](https://github.com/apache/apisix-ingress-controller/commit/af03e7a95785d0905f33a3458ce2cda02db41853) chore: update APISIX v2.14.1 (#1145)
* [`516e677`](https://github.com/apache/apisix-ingress-controller/commit/516e6771ff9dd94347177c83b290a2292f4f350c) chore(deps): bump github.com/gin-gonic/gin from 1.7.7 to 1.8.1 (#1184)
* [`a0bc739`](https://github.com/apache/apisix-ingress-controller/commit/a0bc73990c9d6e2aee4b063232e09de01de8012e) fix: trigger ApisixRoute event when service is created (#1152)
* [`1765ec9`](https://github.com/apache/apisix-ingress-controller/commit/1765ec9ffe683a323fd3a851e0ea069173950481) test: keep namespace when test failed in dev mod (#1158)
* [`b1add53`](https://github.com/apache/apisix-ingress-controller/commit/b1add53cbc915af476d61a2798db82eb6bef80b1) Chore dep update (#1180)
* [`8c2cfbc`](https://github.com/apache/apisix-ingress-controller/commit/8c2cfbc24db8a0ee76a681dec92b85920abd47b8) chore(deps): bump github.com/stretchr/testify from 1.7.0 to 1.8.0 (#1175)
* [`1a5f2c1`](https://github.com/apache/apisix-ingress-controller/commit/1a5f2c16f96034ec6d66c65a1eaec603fe1c85db) chore(deps): bump github.com/spf13/cobra from 1.2.1 to 1.5.0 (#1176)
* [`3299260`](https://github.com/apache/apisix-ingress-controller/commit/3299260a958b200c2e3f51789a569971fe8d2157) chore(deps): bump k8s.io/client-go and go-memdb etc. (#1172)
* [`628abb9`](https://github.com/apache/apisix-ingress-controller/commit/628abb945f7bc515bd5fcccf1479443305236478) ci: upgrade e2e-test-ci (#1149)
* [`9a6bd92`](https://github.com/apache/apisix-ingress-controller/commit/9a6bd925a3bfb93b2734098d6e29860105b791b1) chore(deps): bump github.com/gruntwork-io/terratest in /test/e2e (#1156)
* [`edb19cd`](https://github.com/apache/apisix-ingress-controller/commit/edb19cdf34e2f3b3558f0bdbbfa458e6093b51a3) chore(deps): bump github.com/gorilla/websocket in /test/e2e (#1114)
* [`f198f33`](https://github.com/apache/apisix-ingress-controller/commit/f198f33c74fd2fc3361d01a60f37f37368f6a155) chore(deps): some dependency updates (#1160)
* [`e75f7e9`](https://github.com/apache/apisix-ingress-controller/commit/e75f7e9b1a3c1b3a9bee3a764d379ca661bd7bb9) chore: change description and labels for this project (#1150)
* [`35ca03c`](https://github.com/apache/apisix-ingress-controller/commit/35ca03cbb4a4779a131ed52f5da2515624c9f00d) test: add e2e tests and CRDs for ApisixUpstream v2 (#1147)
* [`6cf8bb7`](https://github.com/apache/apisix-ingress-controller/commit/6cf8bb7c491f5ee9fbc887eef724308ba96e2242) feat: Add annotations to combine ApisixPluginConfig with k8s ingress resource (#1139)
* [`73498bd`](https://github.com/apache/apisix-ingress-controller/commit/73498bd761e661989b7397eeb12bf18c57683abe) docs: update crd version (#1134)
* [`a649751`](https://github.com/apache/apisix-ingress-controller/commit/a649751dbdd42cea64ad574dda6fe1eeba6a4a5a) feat: ApisixUpstream v2 (#1141)
* [`a73b52d`](https://github.com/apache/apisix-ingress-controller/commit/a73b52d3946dcc08473888445a0c66152e659120) feat: support endpointslice, and improve test/e2e/endpoints.go tests. (#1140)
* [`f0217ae`](https://github.com/apache/apisix-ingress-controller/commit/f0217ae5b022d6086bab2155dd3053567b3fc3aa) ci: pin skywalking-eyes to release (#1143)
* [`62e0ea2`](https://github.com/apache/apisix-ingress-controller/commit/62e0ea20031ebfa664af5b0d5ab2e76336bac107) chore: add log for syncManifest delete upstream (#1132)
* [`374f865`](https://github.com/apache/apisix-ingress-controller/commit/374f86565ac3c84b1a19e0450c2251b1dfce58f0) make api version const consistent (#1133)
* [`93c10e6`](https://github.com/apache/apisix-ingress-controller/commit/93c10e6023ab10aa70e8a61992372ce0869139a9) fix: verify through the cache first, then delete (#1135)
* [`a642b14`](https://github.com/apache/apisix-ingress-controller/commit/a642b1492dc9e00f350aaacb06fea6025bc4e809) doc: fix enable-authentication-and-restriction.md link failed (#1137)
* [`e25abdb`](https://github.com/apache/apisix-ingress-controller/commit/e25abdb2be54865be03a7d486960e9c77eb1ba86) fix: ns should unwatch after unlabeling it (#1130)
* [`398f816`](https://github.com/apache/apisix-ingress-controller/commit/398f816a57e496d0966c6103ab8ab1cae28cf852) e2e-test: Optimize the runtime of ingress/features, and support more default value in NewScaffold (#1128)
* [`4d172a0`](https://github.com/apache/apisix-ingress-controller/commit/4d172a0306eed6216fd8bf057fd1506053a4a303) chore(deps): bump github.com/stretchr/testify in /test/e2e (#1113)
* [`aae2105`](https://github.com/apache/apisix-ingress-controller/commit/aae2105e123008a0170b68a0133432695ee230c9) feat: ingress annotations support enable websocket (#1101)
* [`70c0870`](https://github.com/apache/apisix-ingress-controller/commit/70c08706ae7cae73bc9b08d2ec2d29b98e5b4f89) chore(deps): bump github.com/gruntwork-io/terratest from 0.32.8 to 0.40.17 in /test/e2e (#1112)
* [`4bc9f0c`](https://github.com/apache/apisix-ingress-controller/commit/4bc9f0cac1205ed71bf355817ec7db09690398f8) fix: update Makefile verify-mdlint (#1126)
* [`4aa2ca5`](https://github.com/apache/apisix-ingress-controller/commit/4aa2ca5a80149e2cd5bddf8b1fb51ef2f64cfbc3) test: support ApisixRoute v2 and split suit-plugins (#1103)
* [`810f1a1`](https://github.com/apache/apisix-ingress-controller/commit/810f1a1c5b232808e77f8f153d3ec90c3008e3c0) docs: rename practices to tutorials and add index (#1123)
* [`b1dc75e`](https://github.com/apache/apisix-ingress-controller/commit/b1dc75e198dbacc4d66e9d2bfbb259cfea9c67a3) chore: enable dependabot for security (#1111)
* [`0e1f8d4`](https://github.com/apache/apisix-ingress-controller/commit/0e1f8d4afdf4d90743ef238872d1e383cdfa93a4) fix : The ingress backend is modified several times, resulting in residual update events (#1040)
* [`b33d70c`](https://github.com/apache/apisix-ingress-controller/commit/b33d70c429d32db6d2a065463d37a73ff59ab4ad) feat: support gateway TLSRoute (#1087)
* [`49991e2`](https://github.com/apache/apisix-ingress-controller/commit/49991e2e74b3a9a40d2ee9bb645408a338458921) feat: sync CRD and ingress resource to apisix mechanism. (#1102)
* [`f453e80`](https://github.com/apache/apisix-ingress-controller/commit/f453e8071fd7d320b7847235d6b61596c8ef3402) chore: enable stale GitHub action (#1107)
* [`9e0c658`](https://github.com/apache/apisix-ingress-controller/commit/9e0c658cd2387165b5e0b5fc8d91f6464e3ad406) fix: upstream nodes filed IP occupation. (#1064)
* [`d46b8e0`](https://github.com/apache/apisix-ingress-controller/commit/d46b8e0f79172ca38b3e8ce93552b7ba1225aca4) feat: support v2 in resource compare (#1093)
* [`0f95dbe`](https://github.com/apache/apisix-ingress-controller/commit/0f95dbe989921d6b76156d62a3ec93804d21ac50) doc: add v2 CRD reference (#1068)
* [`5756273`](https://github.com/apache/apisix-ingress-controller/commit/57562739ee06ae080bdbabc35ed3b2e2cd397e42) infra: update golang 1.18 (#1095)
* [`a69a55a`](https://github.com/apache/apisix-ingress-controller/commit/a69a55aff459ff6aa842c4ab7914b022883a8d46) Revert "feat: sync CRD and ingress resource to APISIX mechanism. (#1022)" (#1099)
* [`6394cdd`](https://github.com/apache/apisix-ingress-controller/commit/6394cdd11e96f205756bb8c19eaa047bdc114774) feat: sync CRD and ingress resource to APISIX mechanism. (#1022)
* [`50d6026`](https://github.com/apache/apisix-ingress-controller/commit/50d6026df3953f9005bdd6ce4055909d0b2e3310) fix: make ApisixRouteHTTPBackend support serivce name (#1096)
* [`3214c69`](https://github.com/apache/apisix-ingress-controller/commit/3214c698f9dd2dc7797fce837e16d15d60f64050) fix: e2e robustness. (#1078)
* [`c48a62a`](https://github.com/apache/apisix-ingress-controller/commit/c48a62abfbd0b546c1d89bb7931caf1ed11abbb3) feat: support GatewayClass, refactor gateway modules (#1079)
* [`a0b88d1`](https://github.com/apache/apisix-ingress-controller/commit/a0b88d11c4e906722d7928e7bdd985435e9cbe1c) fix: tag for keyAuth field (#1080)
* [`f0d64b6`](https://github.com/apache/apisix-ingress-controller/commit/f0d64b6d8ce4f4f1b5f15d03938f3a87b71a84a8) docs: correct typo & link (#1073)
* [`3520830`](https://github.com/apache/apisix-ingress-controller/commit/3520830ad446ffc8087d48c82e284d1ae08db64c) e2e: gateway api httproute (#1060)
* [`2af39c9`](https://github.com/apache/apisix-ingress-controller/commit/2af39c949883dc831aa64be075aa7cc0edd2dd8a) docs: add how to change Admin API key for APISIX (#1031)
* [`96dd07f`](https://github.com/apache/apisix-ingress-controller/commit/96dd07f847bff9d425b223e0bcac0eea2efe6316) e2e-test: add e2e tests for ApisixPluginConfig v2 (#1067)
* [`d3a823f`](https://github.com/apache/apisix-ingress-controller/commit/d3a823f590843468a555ccc68502acc568d9a9f3) doc: update enable-authentication-and-restriction, jwt-auth and wolf-rbac examples. (#1018)
* [`deb0440`](https://github.com/apache/apisix-ingress-controller/commit/deb044039e889e88310523f369833bb137c0e145) docs: add "how to use go plugin runner with APISIX Ingress" (#994)
* [`8d76428`](https://github.com/apache/apisix-ingress-controller/commit/8d764286b018c22b20b62591b7bb68b93de0a93b) e2e-test: upgrade to ginkgo v2 (#1046)
* [`408eb0d`](https://github.com/apache/apisix-ingress-controller/commit/408eb0d1c9b05517e530688e2e6e82c721699548) feat: support ApisixPluginConfig v2 (#984)
* [`e1d496d`](https://github.com/apache/apisix-ingress-controller/commit/e1d496dd074eb19982b9e198370440ebd0d16f87) e2e-test: add e2e tests and CRDs for ApisixConsumer v2 (#1044)
* [`6c7452f`](https://github.com/apache/apisix-ingress-controller/commit/6c7452ffab358c47e25882d802e4cc891e95eb37) feat: support gateway API HTTPRoute (#1037)
* [`5477fb0`](https://github.com/apache/apisix-ingress-controller/commit/5477fb0e5f62f1c9b1716824bcf0fb7f038b02a7) test: fix wolf-rbac and mTLS test cases (#1055)
* [`df7a724`](https://github.com/apache/apisix-ingress-controller/commit/df7a724ce11d23ad441209cf2592426f251f597c) e2e-test: add e2e tests and CRDs for ApisixClusterConfig v2 (#1016)
* [`25daa6e`](https://github.com/apache/apisix-ingress-controller/commit/25daa6e2f02ceb459f605ca2bb5a3aaa8973c624) feat: add csrf plugin annotation in ingress resource (#1023)
* [`59ba41a`](https://github.com/apache/apisix-ingress-controller/commit/59ba41a8c5780b9273610e944aabda700078a3db) feat: add hmac-auth authorization method (#1035)
* [`49dd015`](https://github.com/apache/apisix-ingress-controller/commit/49dd015085a68ce4c26ba60076ef281ed71af2aa) doc: update contribute.md doc (#1036)
* [`f6f0a3b`](https://github.com/apache/apisix-ingress-controller/commit/f6f0a3b5552ba8fda556e0edb9296c1c0a4c3e31) feat: support ApisixConsumer v2 (#989)
* [`bef2010`](https://github.com/apache/apisix-ingress-controller/commit/bef2010bbc33667bbefd3e718997fe4d20e2f5f8) doc: paraphrasing some descriptions (#1028)
* [`9bd4b71`](https://github.com/apache/apisix-ingress-controller/commit/9bd4b714ceb8fd427b325757e10090f056803cd3) chore: Changelog for 1.4.1 (#1029)
* [`537b947`](https://github.com/apache/apisix-ingress-controller/commit/537b947ab708c12c50e33b37c94caf73589bb65d) doc: add apisix_pluginconfig document (#1025)
* [`bb5104e`](https://github.com/apache/apisix-ingress-controller/commit/bb5104e46911d187b2d624378b0f4c3ed5e7e38a) feat: add wolf-rbac authorization method. (#1011)
* [`3cccd56`](https://github.com/apache/apisix-ingress-controller/commit/3cccd5666e098f374c262eb443de194d69d6a55e) feat: add jwt-auth authorization method (#1009)
* [`cd5063f`](https://github.com/apache/apisix-ingress-controller/commit/cd5063f04881248338595cb85100ef1009e75a80) e2e-test: add e2e tests and CRDs for ApisixTls v2 (#1014)
* [`bac9813`](https://github.com/apache/apisix-ingress-controller/commit/bac9813e4c90a56dadba89808da2faa3d0834b79) feat: support ApisixClusterConfig v2 (#977)
* [`e2f19b5`](https://github.com/apache/apisix-ingress-controller/commit/e2f19b563e672f21f9bc72c3890ed74908ddc9cc) feat: support ApisixTls v2 (#967)
* [`75a4166`](https://github.com/apache/apisix-ingress-controller/commit/75a4166bf52fa24da1a9a0f92c0fd7dfc99d5480) docs: added "how to access Apache APISIX Prometheus Metrics on k8s" (#973)
* [`670d671`](https://github.com/apache/apisix-ingress-controller/commit/670d671d436701ec8083248c6b23713a50ec0c4c) feat:add authorization-annotation the ingress resource (#985)
* [`78efb00`](https://github.com/apache/apisix-ingress-controller/commit/78efb006a4285a9c558cb50524478f944f849906) feat: update an redirect annotation for ingress resource (#975)
* [`3a175e5`](https://github.com/apache/apisix-ingress-controller/commit/3a175e5b9a2c221641c74271eca94079d41501a3) chore: modify metrics name apisix_bad_status_codes to apisix_status_codes (#1012)
* [`f63a29f`](https://github.com/apache/apisix-ingress-controller/commit/f63a29f71e700b381503c9485f41e5225fbe1d9c) doc: add 'enable authentication and restriction' document (#972)
* [`1899d90`](https://github.com/apache/apisix-ingress-controller/commit/1899d9018ba05eeb34c171d279c445815f248bf8) feat: improve the e2e test of referer-restriction plugin (#976)
* [`795be22`](https://github.com/apache/apisix-ingress-controller/commit/795be227d9c73bccc079ca263f098cf54d745f62) docs: fix link in certificate management docs (#1007)
* [`92b89b3`](https://github.com/apache/apisix-ingress-controller/commit/92b89b37d2f63eab6f5c6deb6e3d2a5ccae975aa) chore: update apisix to 2.13.1 (#996)
* [`eefeec8`](https://github.com/apache/apisix-ingress-controller/commit/eefeec87c53bcf674f9f1241f47b11a163c6ee48) docs: update apisix_upstream.md (#983)
* [`4a0fc0c`](https://github.com/apache/apisix-ingress-controller/commit/4a0fc0c4b91650bf073047f64c76e82080f3578f) chore: Fix some code formats (#968)
* [`0f4391a`](https://github.com/apache/apisix-ingress-controller/commit/0f4391a87a38ae9a2c61a0cea717cb831fc55506) refactor: encapsulate functions to reuse code (#971)
* [`64e2768`](https://github.com/apache/apisix-ingress-controller/commit/64e276813eb539baed44e574c20d2336e119b101) ci: add 3 plugin test cases for e2e (#965)
* [`f081121`](https://github.com/apache/apisix-ingress-controller/commit/f0811211a718193f8c8a44b7ad12b163ed641f19) feat: add e2e test for serverless plugin (#964)
* [`eb02429`](https://github.com/apache/apisix-ingress-controller/commit/eb02429de6b92fb7fc6e322f052f33098bd4654f) feat: support forward-auth plugin (#937)
* [`77ab065`](https://github.com/apache/apisix-ingress-controller/commit/77ab065500cdcf2a95b9126d222f3ae82efac8f5) ci: add dependency-review (#963)
* [`fe628f6`](https://github.com/apache/apisix-ingress-controller/commit/fe628f68ab8ed36547e0d224694a09cba06453b1) docs: fix subset field typo (#961)
* [`aee6e78`](https://github.com/apache/apisix-ingress-controller/commit/aee6e7893a731b623fd3bf4f0c1f7bd8d35efae1) fix ApisixConsumerBasicAuthValue password-yaml field error (#960)
* [`0790458`](https://github.com/apache/apisix-ingress-controller/commit/079045836392432e6836f67f9076f63c85dd6243) ci: fix server-info e2e test case(#959)
* [`22cfb5e`](https://github.com/apache/apisix-ingress-controller/commit/22cfb5ec7482e1bca6d293091ce8c7aa5342260b) Add a pre-check for E2E tests (#957)
* [`4bdc947`](https://github.com/apache/apisix-ingress-controller/commit/4bdc9471172a8b1574c61ba5ad5b8fbc4160fc22) Split e2e test cases (#949)
* [`de33d05`](https://github.com/apache/apisix-ingress-controller/commit/de33d05aeff9ec5fdd1a2431c01535566437ee12) feat(e2e): add e2e test for prometheus (#942)
* [`7e4ec36`](https://github.com/apache/apisix-ingress-controller/commit/7e4ec36e36c9114d99156e72b19142d2026533fc) fix: ingress update event handler not filter by watching namespaces (#947)
* [`b5ea236`](https://github.com/apache/apisix-ingress-controller/commit/b5ea23679136b4fc6181d838c8bd36c03ef687b4) docs: update the hard way. (#946)
* [`f58f3d5`](https://github.com/apache/apisix-ingress-controller/commit/f58f3d51889fe72609ca0b837b373e62306f9a3a) feat: change ApisixRoute to v2 api version (#943)
* [`3b99353`](https://github.com/apache/apisix-ingress-controller/commit/3b993533d66b25c186f4096727d373e5ed4131c6) feat: introduce v2 apiversion (#939)
* [`cb45119`](https://github.com/apache/apisix-ingress-controller/commit/cb45119b4c4ec7e9814487b4f29789b4778075e5) doc: add doc about installing apisix ingress with kind (#933)
* [`4da91b7`](https://github.com/apache/apisix-ingress-controller/commit/4da91b7971d3defd137f35de84f107612e6f96bd) chore: drop v2beta1 api version (#928)
* [`81831d5`](https://github.com/apache/apisix-ingress-controller/commit/81831d51b39a1df2db4beeb7d9d7af429e0425fc) docs: remove ApisixRoute v2beta1 & v2alphq1 (#930)
* [`0a66151`](https://github.com/apache/apisix-ingress-controller/commit/0a66151853b2e0ee1d9c43af62f144f9dc63a688) fix: watch all namespaces by default (#919)
* [`2178857`](https://github.com/apache/apisix-ingress-controller/commit/2178857fbe2608cec1dc8ce5b1e0ec232568b375) fix: ApisixRouteEvent type assertion (#925)
* [`c9e0c96`](https://github.com/apache/apisix-ingress-controller/commit/c9e0c965cd2226e2c44aa4e692a8bf57d1586aa3) docs: remove development from sidebar config (#923)
* [`f31f520`](https://github.com/apache/apisix-ingress-controller/commit/f31f5201100169a61cd6b8220683453f2f81379d) docs: merge contribute.md and development.md (#909)
* [`11bd92b`](https://github.com/apache/apisix-ingress-controller/commit/11bd92beb7e45035a12ec2cd27f4295276fa1af2) docs: upgrade apiVersion from v2beta1 to v2beta3 (#916)
* [`75098d1`](https://github.com/apache/apisix-ingress-controller/commit/75098d1e4b26136de3164a3aabd6ed018ffdcd6b) chore: clean up useless code (#902)
* [`4025151`](https://github.com/apache/apisix-ingress-controller/commit/4025151e44b9f64aa0881248faa3088487b53ec6) feat: format gin logger (#904)
* [`48c924c`](https://github.com/apache/apisix-ingress-controller/commit/48c924c3397f2a433482204934a37aac4c4726b8) docs: add pre-commit todo in the development guide (#907)
* [`1159522`](https://github.com/apache/apisix-ingress-controller/commit/1159522bf22181b67c2f075055894f488e9bc648) fix: controller err handler should ignore not found error (#893)
* [`f84a083`](https://github.com/apache/apisix-ingress-controller/commit/f84a083719b5de74c68dd3bf206b1a88f2a540c4) feat: support custom registry for e2e test (#896)
* [`1ddbfa6`](https://github.com/apache/apisix-ingress-controller/commit/1ddbfa68bb05becda7eb301f3ce6740c02b4d0a1) fix: fix ep resourceVersion comparison and clean up (#901)
* [`8348d01`](https://github.com/apache/apisix-ingress-controller/commit/8348d010507f679a17f6126e779780c76358446c) chore: shorten the route name for Ingress transformations (#898)
* [`b5448c3`](https://github.com/apache/apisix-ingress-controller/commit/b5448c37e7b043b4d2a5b5d35b115429a20760a0) fetching newest Endpoint before sync (#821)
* [`bbaba6f`](https://github.com/apache/apisix-ingress-controller/commit/bbaba6f9c22196e5d3142ad7e1a2224474a1c553) fix: filter useless pod update event (#894)
* [`5f6a7c1`](https://github.com/apache/apisix-ingress-controller/commit/5f6a7c10b97c5eb575d23140db433bc8dfe60a2c) fix: avoid create pluginconfig in the tranlsation of route (#845)
* [`035c60e`](https://github.com/apache/apisix-ingress-controller/commit/035c60e456abf064af23634822f8453606fc5cc5) fix: check if stream_routes is disabled (#868)
* [`8d25525`](https://github.com/apache/apisix-ingress-controller/commit/8d255252a206311bde756eb21424c7b6743d6187) docs: fix #887 (#890)
* [`cc9b6be`](https://github.com/apache/apisix-ingress-controller/commit/cc9b6be606b4d41267064c45e7ef1e9f5a6d8e47) fix json unmarshal error when list plguins (#888)
* [`e2475cf`](https://github.com/apache/apisix-ingress-controller/commit/e2475cf8573f9f1efd6956c0e4e6eab2142b8061) feat: add format tool (#885)
* [`d30bef5`](https://github.com/apache/apisix-ingress-controller/commit/d30bef56d5502724901c7eae42b319ad1b92c76c) fix: endless retry if namespace doesn't exist (#882)
* [`27f0cfa`](https://github.com/apache/apisix-ingress-controller/commit/27f0cfa8879712070736a91b4b67c1ea0cdd4835) update the-hard-way.md (#875)
* [`77383b8`](https://github.com/apache/apisix-ingress-controller/commit/77383b88d9c664a64a28374c7f48cdb1d4d5f823) fix ingress delete panic (#872)
* [`26a1f43`](https://github.com/apache/apisix-ingress-controller/commit/26a1f43d5c2f360ce280f0aa81c8bbcc12036cbd) feat: add update command to Makefile (#881)
* [`545d22d`](https://github.com/apache/apisix-ingress-controller/commit/545d22d47414e7635c8074aaa28ad6105ae116e3) chore: clean up v1 version related code (#867)
* [`32a096c`](https://github.com/apache/apisix-ingress-controller/commit/32a096c24dd330e537fd5fed7fcccb63fd8dd813) fix: objects get from lister must be treated as read-only (#829)
* [`6b0c139`](https://github.com/apache/apisix-ingress-controller/commit/6b0c139ce2fbf56c683d86b8f93c7b4ef854fbc6) fix: ApisixClusterConfig e2e test case (#859)
* [`df8316a`](https://github.com/apache/apisix-ingress-controller/commit/df8316aaceec40ae86857b1ef972a812c55b1252) rename command line options and update doc. (#848)
* [`de52243`](https://github.com/apache/apisix-ingress-controller/commit/de522437bd3db3dc3e8d8f33c92f322de67531b8) feat: ensure that the lease can be actively released before program shutdown to reduce the time required for failover (#827)
* [`81d59b5`](https://github.com/apache/apisix-ingress-controller/commit/81d59b52882779b7b10c8e725c8b4d80ee01f8e0) chore: update ingress/comapre.go watchingNamespac from v2beta1 to v2beta3 (#832)
* [`3040cf5`](https://github.com/apache/apisix-ingress-controller/commit/3040cf54c5dc95a444b9c3132f5b911495ad5f80) fix: add v2beta3 register resources (#833)
* [`56d866b`](https://github.com/apache/apisix-ingress-controller/commit/56d866bcbb1f25060a6da31b088e22bee80500db) chore: Update NOTICE to 2022 (#834)
* [`e40cc31`](https://github.com/apache/apisix-ingress-controller/commit/e40cc3152e4b80e9b95aeca01c1b737fbe183f8f) refactor: remove BaseURL and AdminKey in config (#826)
* [`2edc4da`](https://github.com/apache/apisix-ingress-controller/commit/2edc4da9433af58db8ae2fd317f489b287405b2c) chore: fix typo in ApidixRoute CRD (#830)
* [`8364821`](https://github.com/apache/apisix-ingress-controller/commit/83648210fe194fa42f5582e8773969be0a01fa57) fix: consumer name contain "-" (#828)
* [`ae69cd3`](https://github.com/apache/apisix-ingress-controller/commit/ae69cd3ee42245100b1c559df4518a418f66f6bd) docs: Grafana Dashboard Configuration (#731)
* [`990971d`](https://github.com/apache/apisix-ingress-controller/commit/990971d4e892b47a6758edb40042c3decde92846) chore: v1.4 release (#819)
</p>
</details>

### Dependency Changes

* **github.com/beorn7/perks**                           v1.0.1 **_new_**
* **github.com/cespare/xxhash/v2**                      v2.1.2 **_new_**
* **github.com/davecgh/go-spew**                        v1.1.1 **_new_**
* **github.com/emicklei/go-restful/v3**                 v3.9.0 **_new_**
* **github.com/evanphx/json-patch**                     v4.12.0 **_new_**
* **github.com/gin-contrib/sse**                        v0.1.0 **_new_**
* **github.com/gin-gonic/gin**                          v1.7.7 -> v1.8.1
* **github.com/go-logr/logr**                           v1.2.3 **_new_**
* **github.com/go-openapi/jsonpointer**                 v0.19.5 **_new_**
* **github.com/go-openapi/jsonreference**               v0.20.0 **_new_**
* **github.com/go-openapi/swag**                        v0.22.3 **_new_**
* **github.com/go-playground/locales**                  v0.14.0 **_new_**
* **github.com/go-playground/universal-translator**     v0.18.0 **_new_**
* **github.com/go-playground/validator/v10**            v10.11.0 **_new_**
* **github.com/goccy/go-json**                          v0.9.10 **_new_**
* **github.com/gogo/protobuf**                          v1.3.2 **_new_**
* **github.com/golang/groupcache**                      41bb18bfe9da **_new_**
* **github.com/golang/protobuf**                        v1.5.2 **_new_**
* **github.com/google/gnostic**                         v0.6.9 **_new_**
* **github.com/google/go-cmp**                          v0.5.8 **_new_**
* **github.com/google/gofuzz**                          v1.1.0 **_new_**
* **github.com/hashicorp/errwrap**                      v1.0.0 **_new_**
* **github.com/hashicorp/go-immutable-radix**           v1.3.0 **_new_**
* **github.com/hashicorp/go-memdb**                     v1.3.2 -> v1.3.3
* **github.com/hashicorp/golang-lru**                   v0.5.4 **_new_**
* **github.com/imdario/mergo**                          v0.3.12 **_new_**
* **github.com/inconshreveable/mousetrap**              v1.0.0 **_new_**
* **github.com/josharian/intern**                       v1.0.0 **_new_**
* **github.com/json-iterator/go**                       v1.1.12 **_new_**
* **github.com/leodido/go-urn**                         v1.2.1 **_new_**
* **github.com/mailru/easyjson**                        v0.7.7 **_new_**
* **github.com/mattn/go-isatty**                        v0.0.14 **_new_**
* **github.com/matttproud/golang_protobuf_extensions**  c182affec369 **_new_**
* **github.com/modern-go/concurrent**                   bacd9c7ef1dd **_new_**
* **github.com/modern-go/reflect2**                     v1.0.2 **_new_**
* **github.com/munnerz/goautoneg**                      a7dc8b61c822 **_new_**
* **github.com/pelletier/go-toml/v2**                   v2.0.2 **_new_**
* **github.com/pkg/errors**                             v0.9.1 **_new_**
* **github.com/pmezard/go-difflib**                     v1.0.0 **_new_**
* **github.com/prometheus/client_golang**               v1.11.0 -> v1.12.2
* **github.com/prometheus/common**                      v0.32.1 **_new_**
* **github.com/prometheus/procfs**                      v0.7.3 **_new_**
* **github.com/spf13/cobra**                            v1.2.1 -> v1.5.0
* **github.com/spf13/pflag**                            v1.0.5 **_new_**
* **github.com/stretchr/testify**                       v1.7.0 -> v1.8.0
* **github.com/ugorji/go/codec**                        v1.2.7 **_new_**
* **github.com/xeipuuv/gojsonpointer**                  4e3ac2762d5f **_new_**
* **github.com/xeipuuv/gojsonreference**                bd5ef7bd5415 **_new_**
* **go.uber.org/atomic**                                v1.7.0 **_new_**
* **go.uber.org/multierr**                              v1.7.0 -> v1.8.0
* **go.uber.org/zap**                                   v1.19.1 -> v1.23.0
* **golang.org/x/crypto**                               630584e8d5aa **_new_**
* **golang.org/x/mod**                                  86c51ed26bb4 **_new_**
* **golang.org/x/net**                                  fe4d6282115f -> 46097bf591d3
* **golang.org/x/oauth2**                               ee480838109b **_new_**
* **golang.org/x/sys**                                  bce67f096156 -> fb04ddd9f9c8
* **golang.org/x/term**                                 03fcf44c2211 **_new_**
* **golang.org/x/text**                                 v0.3.7 **_new_**
* **golang.org/x/time**                                 90d013bbcef8 **_new_**
* **golang.org/x/tools**                                v0.1.12 **_new_**
* **google.golang.org/appengine**                       v1.6.7 **_new_**
* **google.golang.org/protobuf**                        v1.28.0 **_new_**
* **gopkg.in/inf.v0**                                   v0.9.1 **_new_**
* **gopkg.in/yaml.v3**                                  v3.0.1 **_new_**
* **k8s.io/api**                                        v0.22.4 -> v0.25.1
* **k8s.io/apimachinery**                               v0.22.4 -> v0.25.1
* **k8s.io/client-go**                                  v0.22.4 -> v0.25.1
* **k8s.io/code-generator**                             v0.22.1 -> v0.25.1
* **k8s.io/gengo**                                      391367153a38 **_new_**
* **k8s.io/klog/v2**                                    v2.80.1 **_new_**
* **k8s.io/kube-openapi**                               a70c9af30aea **_new_**
* **k8s.io/utils**                                      ee6ede2d64ed **_new_**
* **sigs.k8s.io/json**                                  f223a00ba0e2 **_new_**
* **sigs.k8s.io/structured-merge-diff/v4**              v4.2.3 **_new_**
* **sigs.k8s.io/yaml**                                  v1.3.0 **_new_**

Previous release can be found at [1.4.0](https://github.com/apache/apisix-ingress-controller/releases/tag/1.4.0)

# 1.5.0-rc1

Welcome to the 1.5.0 release of apisix-ingress-controller!

This is a feature release.

## Highlights

The API version of all custom resources has been upgraded to v2 in this release and mark v2beta3 as deprecated. We plan to remove the v2beta2 API version in the next release. Please go to [#707](https://github.com/apache/apisix-ingress-controller/issues/707) for detail.

We have added partial support for Gateway API, which is not enabled by default, you can set `enable_gateway_api=true` to enable it.

Ingress resources can now use all APISIX plugin configurations by setting the annotation `k8s.apisix.apache.org/plugin-config-name=xxx`.

Please try out the release binaries and report any issues at
https://github.com/apache/apisix-ingress-controller/issues.

### Contributors

* Jintao Zhang
* Sarasa Kisaragi
* Xin Rong
* John Chever
* cmssczy
* dependabot[bot]
* nevercase
* Gallardot
* Nic
* lsy
* mango
* Fatpa
* Hoshea Jiang
* JasonZhu
* Xin Rong
* Yu.Bozhong
* seven dickens
* FesonX
* GhangZh
* JasonZhu
* Kowsz
* LetsGO
* Sindweller
* SkyeYoung
* Zack Sun
* bin-ya
* champly
* chen zhuo
* fengxsong
* greenhandatsjtu
* hahayyun
* hf400159
* wangyunpeng
* 罗泽轩

### Changes
<details><summary>144 commits</summary>
<p>

* [`cccad72`](https://github.com/apache/apisix-ingress-controller/commit/cccad72a1e0ef60525c69371b4b27c4598c587c1) chore: mark v2beta3 deprecated (#1198)
* [`698ab6d`](https://github.com/apache/apisix-ingress-controller/commit/698ab6d52d4eef0d4404fa3bec43b20039ce9370) chore: Using APISIX 2.15.0 for CI (#1197)
* [`8094868`](https://github.com/apache/apisix-ingress-controller/commit/80948682d73cb7813d06f85e548392f19dc465f4) e2e: add sync case (#1196)
* [`37a8e5c`](https://github.com/apache/apisix-ingress-controller/commit/37a8e5c837cd2a8a459bec054c3ed73bac9b52bb) fix: translate error of old Ingress (#1193)
* [`339531f`](https://github.com/apache/apisix-ingress-controller/commit/339531f3eceb2ce24f07cc6b255fde94f201879a) doc: add a notice about the compatibility of Ingress and Dashboard (#1195)
* [`2cc586b`](https://github.com/apache/apisix-ingress-controller/commit/2cc586bbe06d3dd4c1c904fd487a9eaa13b5b3a2) fix: apisix_upstream sync panic (#1192)
* [`6cc718b`](https://github.com/apache/apisix-ingress-controller/commit/6cc718bb85a3d61f3702ba0514e5408d48a873da) helm: update deploy cluster role (#1131)
* [`3d720c0`](https://github.com/apache/apisix-ingress-controller/commit/3d720c0f9ba41fb2067d3a552a45d357c7a420f7) fix: translate error of old ApisixRoute (#1191)
* [`8b51c6e`](https://github.com/apache/apisix-ingress-controller/commit/8b51c6e173db12592f6fcfa8d59aaa6fd50e7922) docs: update all api-version to v2 (#1189)
* [`5f45b63`](https://github.com/apache/apisix-ingress-controller/commit/5f45b63b3d3716dd5de9f0288a6dd5c28228c40d) fix: ScopeQuery should be case sensitive (#1168) (#1188)
* [`af03e7a`](https://github.com/apache/apisix-ingress-controller/commit/af03e7a95785d0905f33a3458ce2cda02db41853) chore: update APISIX v2.14.1 (#1145)
* [`516e677`](https://github.com/apache/apisix-ingress-controller/commit/516e6771ff9dd94347177c83b290a2292f4f350c) chore(deps): bump github.com/gin-gonic/gin from 1.7.7 to 1.8.1 (#1184)
* [`a0bc739`](https://github.com/apache/apisix-ingress-controller/commit/a0bc73990c9d6e2aee4b063232e09de01de8012e) fix: trigger ApisixRoute event when service is created (#1152)
* [`1765ec9`](https://github.com/apache/apisix-ingress-controller/commit/1765ec9ffe683a323fd3a851e0ea069173950481) test: keep namespace when test failed in dev mod (#1158)
* [`b1add53`](https://github.com/apache/apisix-ingress-controller/commit/b1add53cbc915af476d61a2798db82eb6bef80b1) Chore dep update (#1180)
* [`8c2cfbc`](https://github.com/apache/apisix-ingress-controller/commit/8c2cfbc24db8a0ee76a681dec92b85920abd47b8) chore(deps): bump github.com/stretchr/testify from 1.7.0 to 1.8.0 (#1175)
* [`1a5f2c1`](https://github.com/apache/apisix-ingress-controller/commit/1a5f2c16f96034ec6d66c65a1eaec603fe1c85db) chore(deps): bump github.com/spf13/cobra from 1.2.1 to 1.5.0 (#1176)
* [`3299260`](https://github.com/apache/apisix-ingress-controller/commit/3299260a958b200c2e3f51789a569971fe8d2157) chore(deps): bump k8s.io/client-go and go-memdb etc. (#1172)
* [`628abb9`](https://github.com/apache/apisix-ingress-controller/commit/628abb945f7bc515bd5fcccf1479443305236478) ci: upgrade e2e-test-ci (#1149)
* [`9a6bd92`](https://github.com/apache/apisix-ingress-controller/commit/9a6bd925a3bfb93b2734098d6e29860105b791b1) chore(deps): bump github.com/gruntwork-io/terratest in /test/e2e (#1156)
* [`edb19cd`](https://github.com/apache/apisix-ingress-controller/commit/edb19cdf34e2f3b3558f0bdbbfa458e6093b51a3) chore(deps): bump github.com/gorilla/websocket in /test/e2e (#1114)
* [`f198f33`](https://github.com/apache/apisix-ingress-controller/commit/f198f33c74fd2fc3361d01a60f37f37368f6a155) chore(deps): some dependency updates (#1160)
* [`e75f7e9`](https://github.com/apache/apisix-ingress-controller/commit/e75f7e9b1a3c1b3a9bee3a764d379ca661bd7bb9) chore: change description and labels for this project (#1150)
* [`35ca03c`](https://github.com/apache/apisix-ingress-controller/commit/35ca03cbb4a4779a131ed52f5da2515624c9f00d) test: add e2e tests and CRDs for ApisixUpstream v2 (#1147)
* [`6cf8bb7`](https://github.com/apache/apisix-ingress-controller/commit/6cf8bb7c491f5ee9fbc887eef724308ba96e2242) feat: Add annotations to combine ApisixPluginConfig with k8s ingress resource (#1139)
* [`73498bd`](https://github.com/apache/apisix-ingress-controller/commit/73498bd761e661989b7397eeb12bf18c57683abe) docs: update crd version (#1134)
* [`a649751`](https://github.com/apache/apisix-ingress-controller/commit/a649751dbdd42cea64ad574dda6fe1eeba6a4a5a) feat: ApisixUpstream v2 (#1141)
* [`a73b52d`](https://github.com/apache/apisix-ingress-controller/commit/a73b52d3946dcc08473888445a0c66152e659120) feat: support endpointslice, and improve test/e2e/endpoints.go tests. (#1140)
* [`f0217ae`](https://github.com/apache/apisix-ingress-controller/commit/f0217ae5b022d6086bab2155dd3053567b3fc3aa) ci: pin skywalking-eyes to release (#1143)
* [`62e0ea2`](https://github.com/apache/apisix-ingress-controller/commit/62e0ea20031ebfa664af5b0d5ab2e76336bac107) chore: add log for syncManifest delete upstream (#1132)
* [`374f865`](https://github.com/apache/apisix-ingress-controller/commit/374f86565ac3c84b1a19e0450c2251b1dfce58f0) make api version const consistent (#1133)
* [`93c10e6`](https://github.com/apache/apisix-ingress-controller/commit/93c10e6023ab10aa70e8a61992372ce0869139a9) fix: verify through the cache first, then delete (#1135)
* [`a642b14`](https://github.com/apache/apisix-ingress-controller/commit/a642b1492dc9e00f350aaacb06fea6025bc4e809) doc: fix enable-authentication-and-restriction.md link failed (#1137)
* [`e25abdb`](https://github.com/apache/apisix-ingress-controller/commit/e25abdb2be54865be03a7d486960e9c77eb1ba86) fix: ns should unwatch after unlabeling it (#1130)
* [`398f816`](https://github.com/apache/apisix-ingress-controller/commit/398f816a57e496d0966c6103ab8ab1cae28cf852) e2e-test: Optimize the runtime of ingress/features, and support more default value in NewScaffold (#1128)
* [`4d172a0`](https://github.com/apache/apisix-ingress-controller/commit/4d172a0306eed6216fd8bf057fd1506053a4a303) chore(deps): bump github.com/stretchr/testify in /test/e2e (#1113)
* [`aae2105`](https://github.com/apache/apisix-ingress-controller/commit/aae2105e123008a0170b68a0133432695ee230c9) feat: ingress annotations support enable websocket (#1101)
* [`70c0870`](https://github.com/apache/apisix-ingress-controller/commit/70c08706ae7cae73bc9b08d2ec2d29b98e5b4f89) chore(deps): bump github.com/gruntwork-io/terratest from 0.32.8 to 0.40.17 in /test/e2e (#1112)
* [`4bc9f0c`](https://github.com/apache/apisix-ingress-controller/commit/4bc9f0cac1205ed71bf355817ec7db09690398f8) fix: update Makefile verify-mdlint (#1126)
* [`4aa2ca5`](https://github.com/apache/apisix-ingress-controller/commit/4aa2ca5a80149e2cd5bddf8b1fb51ef2f64cfbc3) test: support ApisixRoute v2 and split suit-plugins (#1103)
* [`810f1a1`](https://github.com/apache/apisix-ingress-controller/commit/810f1a1c5b232808e77f8f153d3ec90c3008e3c0) docs: rename practices to tutorials and add index (#1123)
* [`b1dc75e`](https://github.com/apache/apisix-ingress-controller/commit/b1dc75e198dbacc4d66e9d2bfbb259cfea9c67a3) chore: enable dependabot for security (#1111)
* [`0e1f8d4`](https://github.com/apache/apisix-ingress-controller/commit/0e1f8d4afdf4d90743ef238872d1e383cdfa93a4) fix : The ingress backend is modified several times, resulting in residual update events (#1040)
* [`b33d70c`](https://github.com/apache/apisix-ingress-controller/commit/b33d70c429d32db6d2a065463d37a73ff59ab4ad) feat: support gateway TLSRoute (#1087)
* [`49991e2`](https://github.com/apache/apisix-ingress-controller/commit/49991e2e74b3a9a40d2ee9bb645408a338458921) feat: sync CRD and ingress resource to apisix mechanism. (#1102)
* [`f453e80`](https://github.com/apache/apisix-ingress-controller/commit/f453e8071fd7d320b7847235d6b61596c8ef3402) chore: enable stale GitHub action (#1107)
* [`9e0c658`](https://github.com/apache/apisix-ingress-controller/commit/9e0c658cd2387165b5e0b5fc8d91f6464e3ad406) fix: upstream nodes filed IP occupation. (#1064)
* [`d46b8e0`](https://github.com/apache/apisix-ingress-controller/commit/d46b8e0f79172ca38b3e8ce93552b7ba1225aca4) feat: support v2 in resource compare (#1093)
* [`0f95dbe`](https://github.com/apache/apisix-ingress-controller/commit/0f95dbe989921d6b76156d62a3ec93804d21ac50) doc: add v2 CRD reference (#1068)
* [`5756273`](https://github.com/apache/apisix-ingress-controller/commit/57562739ee06ae080bdbabc35ed3b2e2cd397e42) infra: update golang 1.18 (#1095)
* [`a69a55a`](https://github.com/apache/apisix-ingress-controller/commit/a69a55aff459ff6aa842c4ab7914b022883a8d46) Revert "feat: sync CRD and ingress resource to APISIX mechanism. (#1022)" (#1099)
* [`6394cdd`](https://github.com/apache/apisix-ingress-controller/commit/6394cdd11e96f205756bb8c19eaa047bdc114774) feat: sync CRD and ingress resource to APISIX mechanism. (#1022)
* [`50d6026`](https://github.com/apache/apisix-ingress-controller/commit/50d6026df3953f9005bdd6ce4055909d0b2e3310) fix: make ApisixRouteHTTPBackend support serivce name (#1096)
* [`3214c69`](https://github.com/apache/apisix-ingress-controller/commit/3214c698f9dd2dc7797fce837e16d15d60f64050) fix: e2e robustness. (#1078)
* [`c48a62a`](https://github.com/apache/apisix-ingress-controller/commit/c48a62abfbd0b546c1d89bb7931caf1ed11abbb3) feat: support GatewayClass, refactor gateway modules (#1079)
* [`a0b88d1`](https://github.com/apache/apisix-ingress-controller/commit/a0b88d11c4e906722d7928e7bdd985435e9cbe1c) fix: tag for keyAuth field (#1080)
* [`f0d64b6`](https://github.com/apache/apisix-ingress-controller/commit/f0d64b6d8ce4f4f1b5f15d03938f3a87b71a84a8) docs: correct typo & link (#1073)
* [`3520830`](https://github.com/apache/apisix-ingress-controller/commit/3520830ad446ffc8087d48c82e284d1ae08db64c) e2e: gateway api httproute (#1060)
* [`2af39c9`](https://github.com/apache/apisix-ingress-controller/commit/2af39c949883dc831aa64be075aa7cc0edd2dd8a) docs: add how to change Admin API key for APISIX (#1031)
* [`96dd07f`](https://github.com/apache/apisix-ingress-controller/commit/96dd07f847bff9d425b223e0bcac0eea2efe6316) e2e-test: add e2e tests for ApisixPluginConfig v2 (#1067)
* [`d3a823f`](https://github.com/apache/apisix-ingress-controller/commit/d3a823f590843468a555ccc68502acc568d9a9f3) doc: update enable-authentication-and-restriction, jwt-auth and wolf-rbac examples. (#1018)
* [`deb0440`](https://github.com/apache/apisix-ingress-controller/commit/deb044039e889e88310523f369833bb137c0e145) docs: add "how to use go plugin runner with APISIX Ingress" (#994)
* [`8d76428`](https://github.com/apache/apisix-ingress-controller/commit/8d764286b018c22b20b62591b7bb68b93de0a93b) e2e-test: upgrade to ginkgo v2 (#1046)
* [`408eb0d`](https://github.com/apache/apisix-ingress-controller/commit/408eb0d1c9b05517e530688e2e6e82c721699548) feat: support ApisixPluginConfig v2 (#984)
* [`e1d496d`](https://github.com/apache/apisix-ingress-controller/commit/e1d496dd074eb19982b9e198370440ebd0d16f87) e2e-test: add e2e tests and CRDs for ApisixConsumer v2 (#1044)
* [`6c7452f`](https://github.com/apache/apisix-ingress-controller/commit/6c7452ffab358c47e25882d802e4cc891e95eb37) feat: support gateway API HTTPRoute (#1037)
* [`5477fb0`](https://github.com/apache/apisix-ingress-controller/commit/5477fb0e5f62f1c9b1716824bcf0fb7f038b02a7) test: fix wolf-rbac and mTLS test cases (#1055)
* [`df7a724`](https://github.com/apache/apisix-ingress-controller/commit/df7a724ce11d23ad441209cf2592426f251f597c) e2e-test: add e2e tests and CRDs for ApisixClusterConfig v2 (#1016)
* [`25daa6e`](https://github.com/apache/apisix-ingress-controller/commit/25daa6e2f02ceb459f605ca2bb5a3aaa8973c624) feat: add csrf plugin annotation in ingress resource (#1023)
* [`59ba41a`](https://github.com/apache/apisix-ingress-controller/commit/59ba41a8c5780b9273610e944aabda700078a3db) feat: add hmac-auth authorization method (#1035)
* [`49dd015`](https://github.com/apache/apisix-ingress-controller/commit/49dd015085a68ce4c26ba60076ef281ed71af2aa) doc: update contribute.md doc (#1036)
* [`f6f0a3b`](https://github.com/apache/apisix-ingress-controller/commit/f6f0a3b5552ba8fda556e0edb9296c1c0a4c3e31) feat: support ApisixConsumer v2 (#989)
* [`bef2010`](https://github.com/apache/apisix-ingress-controller/commit/bef2010bbc33667bbefd3e718997fe4d20e2f5f8) doc: paraphrasing some descriptions (#1028)
* [`9bd4b71`](https://github.com/apache/apisix-ingress-controller/commit/9bd4b714ceb8fd427b325757e10090f056803cd3) chore: Changelog for 1.4.1 (#1029)
* [`537b947`](https://github.com/apache/apisix-ingress-controller/commit/537b947ab708c12c50e33b37c94caf73589bb65d) doc: add apisix_pluginconfig document (#1025)
* [`bb5104e`](https://github.com/apache/apisix-ingress-controller/commit/bb5104e46911d187b2d624378b0f4c3ed5e7e38a) feat: add wolf-rbac authorization method. (#1011)
* [`3cccd56`](https://github.com/apache/apisix-ingress-controller/commit/3cccd5666e098f374c262eb443de194d69d6a55e) feat: add jwt-auth authorization method (#1009)
* [`cd5063f`](https://github.com/apache/apisix-ingress-controller/commit/cd5063f04881248338595cb85100ef1009e75a80) e2e-test: add e2e tests and CRDs for ApisixTls v2 (#1014)
* [`bac9813`](https://github.com/apache/apisix-ingress-controller/commit/bac9813e4c90a56dadba89808da2faa3d0834b79) feat: support ApisixClusterConfig v2 (#977)
* [`e2f19b5`](https://github.com/apache/apisix-ingress-controller/commit/e2f19b563e672f21f9bc72c3890ed74908ddc9cc) feat: support ApisixTls v2 (#967)
* [`75a4166`](https://github.com/apache/apisix-ingress-controller/commit/75a4166bf52fa24da1a9a0f92c0fd7dfc99d5480) docs: added "how to access Apache APISIX Prometheus Metrics on k8s" (#973)
* [`670d671`](https://github.com/apache/apisix-ingress-controller/commit/670d671d436701ec8083248c6b23713a50ec0c4c) feat:add authorization-annotation the ingress resource (#985)
* [`78efb00`](https://github.com/apache/apisix-ingress-controller/commit/78efb006a4285a9c558cb50524478f944f849906) feat: update an redirect annotation for ingress resource (#975)
* [`3a175e5`](https://github.com/apache/apisix-ingress-controller/commit/3a175e5b9a2c221641c74271eca94079d41501a3) chore: modify metrics name apisix_bad_status_codes to apisix_status_codes (#1012)
* [`f63a29f`](https://github.com/apache/apisix-ingress-controller/commit/f63a29f71e700b381503c9485f41e5225fbe1d9c) doc: add 'enable authentication and restriction' document (#972)
* [`1899d90`](https://github.com/apache/apisix-ingress-controller/commit/1899d9018ba05eeb34c171d279c445815f248bf8) feat: improve the e2e test of referer-restriction plugin (#976)
* [`795be22`](https://github.com/apache/apisix-ingress-controller/commit/795be227d9c73bccc079ca263f098cf54d745f62) docs: fix link in certificate management docs (#1007)
* [`92b89b3`](https://github.com/apache/apisix-ingress-controller/commit/92b89b37d2f63eab6f5c6deb6e3d2a5ccae975aa) chore: update apisix to 2.13.1 (#996)
* [`eefeec8`](https://github.com/apache/apisix-ingress-controller/commit/eefeec87c53bcf674f9f1241f47b11a163c6ee48) docs: update apisix_upstream.md (#983)
* [`4a0fc0c`](https://github.com/apache/apisix-ingress-controller/commit/4a0fc0c4b91650bf073047f64c76e82080f3578f) chore: Fix some code formats (#968)
* [`0f4391a`](https://github.com/apache/apisix-ingress-controller/commit/0f4391a87a38ae9a2c61a0cea717cb831fc55506) refactor: encapsulate functions to reuse code (#971)
* [`64e2768`](https://github.com/apache/apisix-ingress-controller/commit/64e276813eb539baed44e574c20d2336e119b101) ci: add 3 plugin test cases for e2e (#965)
* [`f081121`](https://github.com/apache/apisix-ingress-controller/commit/f0811211a718193f8c8a44b7ad12b163ed641f19) feat: add e2e test for serverless plugin (#964)
* [`eb02429`](https://github.com/apache/apisix-ingress-controller/commit/eb02429de6b92fb7fc6e322f052f33098bd4654f) feat: support forward-auth plugin (#937)
* [`77ab065`](https://github.com/apache/apisix-ingress-controller/commit/77ab065500cdcf2a95b9126d222f3ae82efac8f5) ci: add dependency-review (#963)
* [`fe628f6`](https://github.com/apache/apisix-ingress-controller/commit/fe628f68ab8ed36547e0d224694a09cba06453b1) docs: fix subset field typo (#961)
* [`aee6e78`](https://github.com/apache/apisix-ingress-controller/commit/aee6e7893a731b623fd3bf4f0c1f7bd8d35efae1) fix ApisixConsumerBasicAuthValue password-yaml field error (#960)
* [`0790458`](https://github.com/apache/apisix-ingress-controller/commit/079045836392432e6836f67f9076f63c85dd6243) ci: fix server-info e2e test case(#959)
* [`22cfb5e`](https://github.com/apache/apisix-ingress-controller/commit/22cfb5ec7482e1bca6d293091ce8c7aa5342260b) Add a pre-check for E2E tests (#957)
* [`4bdc947`](https://github.com/apache/apisix-ingress-controller/commit/4bdc9471172a8b1574c61ba5ad5b8fbc4160fc22) Split e2e test cases (#949)
* [`de33d05`](https://github.com/apache/apisix-ingress-controller/commit/de33d05aeff9ec5fdd1a2431c01535566437ee12) feat(e2e): add e2e test for prometheus (#942)
* [`7e4ec36`](https://github.com/apache/apisix-ingress-controller/commit/7e4ec36e36c9114d99156e72b19142d2026533fc) fix: ingress update event handler not filter by watching namespaces (#947)
* [`b5ea236`](https://github.com/apache/apisix-ingress-controller/commit/b5ea23679136b4fc6181d838c8bd36c03ef687b4) docs: update the hard way. (#946)
* [`f58f3d5`](https://github.com/apache/apisix-ingress-controller/commit/f58f3d51889fe72609ca0b837b373e62306f9a3a) feat: change ApisixRoute to v2 api version (#943)
* [`3b99353`](https://github.com/apache/apisix-ingress-controller/commit/3b993533d66b25c186f4096727d373e5ed4131c6) feat: introduce v2 apiversion (#939)
* [`cb45119`](https://github.com/apache/apisix-ingress-controller/commit/cb45119b4c4ec7e9814487b4f29789b4778075e5) doc: add doc about installing apisix ingress with kind (#933)
* [`4da91b7`](https://github.com/apache/apisix-ingress-controller/commit/4da91b7971d3defd137f35de84f107612e6f96bd) chore: drop v2beta1 api version (#928)
* [`81831d5`](https://github.com/apache/apisix-ingress-controller/commit/81831d51b39a1df2db4beeb7d9d7af429e0425fc) docs: remove ApisixRoute v2beta1 & v2alphq1 (#930)
* [`0a66151`](https://github.com/apache/apisix-ingress-controller/commit/0a66151853b2e0ee1d9c43af62f144f9dc63a688) fix: watch all namespaces by default (#919)
* [`2178857`](https://github.com/apache/apisix-ingress-controller/commit/2178857fbe2608cec1dc8ce5b1e0ec232568b375) fix: ApisixRouteEvent type assertion (#925)
* [`c9e0c96`](https://github.com/apache/apisix-ingress-controller/commit/c9e0c965cd2226e2c44aa4e692a8bf57d1586aa3) docs: remove development from sidebar config (#923)
* [`f31f520`](https://github.com/apache/apisix-ingress-controller/commit/f31f5201100169a61cd6b8220683453f2f81379d) docs: merge contribute.md and development.md (#909)
* [`11bd92b`](https://github.com/apache/apisix-ingress-controller/commit/11bd92beb7e45035a12ec2cd27f4295276fa1af2) docs: upgrade apiVersion from v2beta1 to v2beta3 (#916)
* [`75098d1`](https://github.com/apache/apisix-ingress-controller/commit/75098d1e4b26136de3164a3aabd6ed018ffdcd6b) chore: clean up useless code (#902)
* [`4025151`](https://github.com/apache/apisix-ingress-controller/commit/4025151e44b9f64aa0881248faa3088487b53ec6) feat: format gin logger (#904)
* [`48c924c`](https://github.com/apache/apisix-ingress-controller/commit/48c924c3397f2a433482204934a37aac4c4726b8) docs: add pre-commit todo in the development guide (#907)
* [`1159522`](https://github.com/apache/apisix-ingress-controller/commit/1159522bf22181b67c2f075055894f488e9bc648) fix: controller err handler should ignore not found error (#893)
* [`f84a083`](https://github.com/apache/apisix-ingress-controller/commit/f84a083719b5de74c68dd3bf206b1a88f2a540c4) feat: support custom registry for e2e test (#896)
* [`1ddbfa6`](https://github.com/apache/apisix-ingress-controller/commit/1ddbfa68bb05becda7eb301f3ce6740c02b4d0a1) fix: fix ep resourceVersion comparison and clean up (#901)
* [`8348d01`](https://github.com/apache/apisix-ingress-controller/commit/8348d010507f679a17f6126e779780c76358446c) chore: shorten the route name for Ingress transformations (#898)
* [`b5448c3`](https://github.com/apache/apisix-ingress-controller/commit/b5448c37e7b043b4d2a5b5d35b115429a20760a0) fetching newest Endpoint before sync (#821)
* [`bbaba6f`](https://github.com/apache/apisix-ingress-controller/commit/bbaba6f9c22196e5d3142ad7e1a2224474a1c553) fix: filter useless pod update event (#894)
* [`5f6a7c1`](https://github.com/apache/apisix-ingress-controller/commit/5f6a7c10b97c5eb575d23140db433bc8dfe60a2c) fix: avoid create pluginconfig in the tranlsation of route (#845)
* [`035c60e`](https://github.com/apache/apisix-ingress-controller/commit/035c60e456abf064af23634822f8453606fc5cc5) fix: check if stream_routes is disabled (#868)
* [`8d25525`](https://github.com/apache/apisix-ingress-controller/commit/8d255252a206311bde756eb21424c7b6743d6187) docs: fix #887 (#890)
* [`cc9b6be`](https://github.com/apache/apisix-ingress-controller/commit/cc9b6be606b4d41267064c45e7ef1e9f5a6d8e47) fix json unmarshal error when list plguins (#888)
* [`e2475cf`](https://github.com/apache/apisix-ingress-controller/commit/e2475cf8573f9f1efd6956c0e4e6eab2142b8061) feat: add format tool (#885)
* [`d30bef5`](https://github.com/apache/apisix-ingress-controller/commit/d30bef56d5502724901c7eae42b319ad1b92c76c) fix: endless retry if namespace doesn't exist (#882)
* [`27f0cfa`](https://github.com/apache/apisix-ingress-controller/commit/27f0cfa8879712070736a91b4b67c1ea0cdd4835) update the-hard-way.md (#875)
* [`77383b8`](https://github.com/apache/apisix-ingress-controller/commit/77383b88d9c664a64a28374c7f48cdb1d4d5f823) fix ingress delete panic (#872)
* [`26a1f43`](https://github.com/apache/apisix-ingress-controller/commit/26a1f43d5c2f360ce280f0aa81c8bbcc12036cbd) feat: add update command to Makefile (#881)
* [`545d22d`](https://github.com/apache/apisix-ingress-controller/commit/545d22d47414e7635c8074aaa28ad6105ae116e3) chore: clean up v1 version related code (#867)
* [`32a096c`](https://github.com/apache/apisix-ingress-controller/commit/32a096c24dd330e537fd5fed7fcccb63fd8dd813) fix: objects get from lister must be treated as read-only (#829)
* [`6b0c139`](https://github.com/apache/apisix-ingress-controller/commit/6b0c139ce2fbf56c683d86b8f93c7b4ef854fbc6) fix: ApisixClusterConfig e2e test case (#859)
* [`df8316a`](https://github.com/apache/apisix-ingress-controller/commit/df8316aaceec40ae86857b1ef972a812c55b1252) rename command line options and update doc. (#848)
* [`de52243`](https://github.com/apache/apisix-ingress-controller/commit/de522437bd3db3dc3e8d8f33c92f322de67531b8) feat: ensure that the lease can be actively released before program shutdown to reduce the time required for failover (#827)
* [`81d59b5`](https://github.com/apache/apisix-ingress-controller/commit/81d59b52882779b7b10c8e725c8b4d80ee01f8e0) chore: update ingress/comapre.go watchingNamespac from v2beta1 to v2beta3 (#832)
* [`3040cf5`](https://github.com/apache/apisix-ingress-controller/commit/3040cf54c5dc95a444b9c3132f5b911495ad5f80) fix: add v2beta3 register resources (#833)
* [`56d866b`](https://github.com/apache/apisix-ingress-controller/commit/56d866bcbb1f25060a6da31b088e22bee80500db) chore: Update NOTICE to 2022 (#834)
* [`e40cc31`](https://github.com/apache/apisix-ingress-controller/commit/e40cc3152e4b80e9b95aeca01c1b737fbe183f8f) refactor: remove BaseURL and AdminKey in config (#826)
* [`2edc4da`](https://github.com/apache/apisix-ingress-controller/commit/2edc4da9433af58db8ae2fd317f489b287405b2c) chore: fix typo in ApidixRoute CRD (#830)
* [`8364821`](https://github.com/apache/apisix-ingress-controller/commit/83648210fe194fa42f5582e8773969be0a01fa57) fix: consumer name contain "-" (#828)
* [`ae69cd3`](https://github.com/apache/apisix-ingress-controller/commit/ae69cd3ee42245100b1c559df4518a418f66f6bd) docs: Grafana Dashboard Configuration (#731)
* [`990971d`](https://github.com/apache/apisix-ingress-controller/commit/990971d4e892b47a6758edb40042c3decde92846) chore: v1.4 release (#819)
</p>
</details>

### Dependency Changes

* **github.com/beorn7/perks**                           v1.0.1 **_new_**
* **github.com/cespare/xxhash/v2**                      v2.1.2 **_new_**
* **github.com/davecgh/go-spew**                        v1.1.1 **_new_**
* **github.com/emicklei/go-restful/v3**                 v3.8.0 **_new_**
* **github.com/evanphx/json-patch**                     v4.12.0 **_new_**
* **github.com/gin-contrib/sse**                        v0.1.0 **_new_**
* **github.com/gin-gonic/gin**                          v1.7.7 -> v1.8.1
* **github.com/go-logr/logr**                           v1.2.3 **_new_**
* **github.com/go-openapi/jsonpointer**                 v0.19.5 **_new_**
* **github.com/go-openapi/jsonreference**               v0.20.0 **_new_**
* **github.com/go-openapi/swag**                        v0.21.1 **_new_**
* **github.com/go-playground/locales**                  v0.14.0 **_new_**
* **github.com/go-playground/universal-translator**     v0.18.0 **_new_**
* **github.com/go-playground/validator/v10**            v10.11.0 **_new_**
* **github.com/goccy/go-json**                          v0.9.10 **_new_**
* **github.com/gogo/protobuf**                          v1.3.2 **_new_**
* **github.com/golang/groupcache**                      41bb18bfe9da **_new_**
* **github.com/golang/protobuf**                        v1.5.2 **_new_**
* **github.com/google/gnostic**                         v0.6.9 **_new_**
* **github.com/google/go-cmp**                          v0.5.7 **_new_**
* **github.com/google/gofuzz**                          v1.1.0 **_new_**
* **github.com/hashicorp/errwrap**                      v1.0.0 **_new_**
* **github.com/hashicorp/go-immutable-radix**           v1.3.0 **_new_**
* **github.com/hashicorp/go-memdb**                     v1.3.2 -> v1.3.3
* **github.com/hashicorp/golang-lru**                   v0.5.4 **_new_**
* **github.com/imdario/mergo**                          v0.3.12 **_new_**
* **github.com/inconshreveable/mousetrap**              v1.0.0 **_new_**
* **github.com/josharian/intern**                       v1.0.0 **_new_**
* **github.com/json-iterator/go**                       v1.1.12 **_new_**
* **github.com/leodido/go-urn**                         v1.2.1 **_new_**
* **github.com/mailru/easyjson**                        v0.7.7 **_new_**
* **github.com/mattn/go-isatty**                        v0.0.14 **_new_**
* **github.com/matttproud/golang_protobuf_extensions**  c182affec369 **_new_**
* **github.com/modern-go/concurrent**                   bacd9c7ef1dd **_new_**
* **github.com/modern-go/reflect2**                     v1.0.2 **_new_**
* **github.com/munnerz/goautoneg**                      a7dc8b61c822 **_new_**
* **github.com/pelletier/go-toml/v2**                   v2.0.2 **_new_**
* **github.com/pkg/errors**                             v0.9.1 **_new_**
* **github.com/pmezard/go-difflib**                     v1.0.0 **_new_**
* **github.com/prometheus/client_golang**               v1.11.0 -> v1.12.2
* **github.com/prometheus/common**                      v0.32.1 **_new_**
* **github.com/prometheus/procfs**                      v0.7.3 **_new_**
* **github.com/spf13/cobra**                            v1.2.1 -> v1.5.0
* **github.com/spf13/pflag**                            v1.0.5 **_new_**
* **github.com/stretchr/testify**                       v1.7.0 -> v1.8.0
* **github.com/ugorji/go/codec**                        v1.2.7 **_new_**
* **github.com/xeipuuv/gojsonpointer**                  4e3ac2762d5f **_new_**
* **github.com/xeipuuv/gojsonreference**                bd5ef7bd5415 **_new_**
* **go.uber.org/atomic**                                v1.7.0 **_new_**
* **go.uber.org/multierr**                              v1.7.0 -> v1.8.0
* **go.uber.org/zap**                                   v1.19.1 -> v1.21.0
* **golang.org/x/crypto**                               630584e8d5aa **_new_**
* **golang.org/x/mod**                                  86c51ed26bb4 **_new_**
* **golang.org/x/net**                                  fe4d6282115f -> 46097bf591d3
* **golang.org/x/oauth2**                               d3ed0bb246c8 **_new_**
* **golang.org/x/sys**                                  bce67f096156 -> 8c9f86f7a55f
* **golang.org/x/term**                                 03fcf44c2211 **_new_**
* **golang.org/x/text**                                 v0.3.7 **_new_**
* **golang.org/x/time**                                 90d013bbcef8 **_new_**
* **golang.org/x/tools**                                v0.1.11 **_new_**
* **golang.org/x/xerrors**                              65e65417b02f **_new_**
* **google.golang.org/appengine**                       v1.6.7 **_new_**
* **google.golang.org/protobuf**                        v1.28.0 **_new_**
* **gopkg.in/inf.v0**                                   v0.9.1 **_new_**
* **gopkg.in/yaml.v3**                                  v3.0.1 **_new_**
* **k8s.io/api**                                        v0.22.4 -> v0.24.3
* **k8s.io/apimachinery**                               v0.22.4 -> v0.24.3
* **k8s.io/client-go**                                  v0.22.4 -> v0.24.3
* **k8s.io/code-generator**                             v0.22.1 -> v0.24.3
* **k8s.io/gengo**                                      397b4ae3bce7 **_new_**
* **k8s.io/klog/v2**                                    v2.70.1 **_new_**
* **k8s.io/kube-openapi**                               011e075b9cb8 **_new_**
* **k8s.io/utils**                                      3a6ce19ff2f9 **_new_**
* **sigs.k8s.io/json**                                  9f7c6b3444d2 **_new_**
* **sigs.k8s.io/structured-merge-diff/v4**              v4.2.1 **_new_**
* **sigs.k8s.io/yaml**                                  v1.3.0 **_new_**

Previous release can be found at [1.4.0](https://github.com/apache/apisix-ingress-controller/releases/tag/1.4.0)

# 1.4.1

Welcome to the 1.4.1 release of apisix-ingress-controller!

This is a Patch version release.

## Highlights

### Roadmap

In next release(v1.5), custom resource's API version v2 will be GA released. Please go to [#707](https://github.com/apache/apisix-ingress-controller/issues/707) for detail.

### Bug fixes

* fix: consumer name contain "-" [#828](https://github.com/apache/apisix-ingress-controller/pull/828)
* fix: fix typo in ApidixRoute CRD [#830](https://github.com/apache/apisix-ingress-controller/pull/830)
* fix: add v2beta3 register resources [#833](https://github.com/apache/apisix-ingress-controller/pull/833)
* fix: ApisixClusterConfig e2e test case [#859](https://github.com/apache/apisix-ingress-controller/pull/859)
* fix: objects get from lister must be treated as read-only [#829](https://github.com/apache/apisix-ingress-controller/pull/829)
* fix ingress delete panic [#872](https://github.com/apache/apisix-ingress-controller/pull/872)
* fix json unmarshal error when list plguins [#888](https://github.com/apache/apisix-ingress-controller/pull/888)
* fix: check if stream_routes is disabled [#868](https://github.com/apache/apisix-ingress-controller/pull/868)
* fix: avoid create pluginconfig in the tranlsation of route [#845](https://github.com/apache/apisix-ingress-controller/pull/845)
* fix: filter useless pod update event [#894](https://github.com/apache/apisix-ingress-controller/pull/894)
* fix: fix ep resourceVersion comparison and clean up [#901](https://github.com/apache/apisix-ingress-controller/pull/901)
* fix: ingress update event handler not filter by watching namespaces [#947](https://github.com/apache/apisix-ingress-controller/pull/)

Please try out the release binaries and report any issues at
https://github.com/apache/apisix-ingress-controller/issues.

### Contributors

* Jintao Zhang
* Nic
* cmssczy
* nevercase
* JasonZhu
* Sarasa Kisaragi
* Xin Rong
* Yu.Bozhong
* champly
* chen zhuo

### Changes

<details><summary>24 commits</summary>
<p>

* [`8a257c4`](https://github.com/apache/apisix-ingress-controller/commit/8a257c49ba2b7032c8c440b062f400ce93064219) chore: fix dead links
* [`c90b602`](https://github.com/apache/apisix-ingress-controller/commit/c90b6021cac71efd6ae32583ed0f960592e16c18) ci: trigger v1.4.0 branch jobs
* [`d0bc591`](https://github.com/apache/apisix-ingress-controller/commit/d0bc591867121a5cc3b56d6d1637d8d4fb510bc8) chore: revert isWatchingNamespace to namespaceWatching
* [`e259826`](https://github.com/apache/apisix-ingress-controller/commit/e259826633a3d50930417cfb88da94989256945c) fix ApisixConsumerBasicAuthValue password-yaml field error (#960)
* [`4d087b3`](https://github.com/apache/apisix-ingress-controller/commit/4d087b3d5ccbb6ffca2473be47f0cb7eb1676d09) fix: ingress update event handler not filter by watching namespaces (#947)
* [`46da0e2`](https://github.com/apache/apisix-ingress-controller/commit/46da0e229fef3834dbcc1e52530104ee28b65c85) docs: upgrade apiVersion from v2beta1 to v2beta3 (#916)
* [`9a8c7ce`](https://github.com/apache/apisix-ingress-controller/commit/9a8c7ce0ad511383d324c63b30acf23f2b5120f4) chore: clean up useless code (#902)
* [`eb90123`](https://github.com/apache/apisix-ingress-controller/commit/eb90123412bf002f90f1d178c869f891b7e2e2f1) fix: fix ep resourceVersion comparison and clean up (#901)
* [`db250da`](https://github.com/apache/apisix-ingress-controller/commit/db250daf211fbf9886811f488e4c18729844ec91) chore: shorten the route name for Ingress transformations (#898)
* [`3f14edd`](https://github.com/apache/apisix-ingress-controller/commit/3f14edd814c4976197ccd477b296a3c9ddf13802) fetching newest Endpoint before sync (#821)
* [`8329b7c`](https://github.com/apache/apisix-ingress-controller/commit/8329b7ce1d48ad9ef2bcef8424d842d8bc9dbc09) fix: filter useless pod update event (#894)
* [`cbcae44`](https://github.com/apache/apisix-ingress-controller/commit/cbcae445c5d9ba7dacd990a2df65d07168206775) fix: avoid create pluginconfig in the tranlsation of route (#845)
* [`e0518a4`](https://github.com/apache/apisix-ingress-controller/commit/e0518a4b354c80aaa7b15ab4f6de4f0cf3a13e52) fix: check if stream_routes is disabled (#868)
* [`90dd10e`](https://github.com/apache/apisix-ingress-controller/commit/90dd10e89a9277aebbdab53a683d8231c26b1ac8) fix json unmarshal error when list plguins (#888)
* [`88cc0b3`](https://github.com/apache/apisix-ingress-controller/commit/88cc0b356f763b0bef97e64d4f423e73bde23681) fix ingress delete panic (#872)
* [`64eb176`](https://github.com/apache/apisix-ingress-controller/commit/64eb176ba564d6310b95d481b260ea942cd05bd4) chore: clean up v1 version related code (#867)
* [`bf1d10e`](https://github.com/apache/apisix-ingress-controller/commit/bf1d10e4a213d4bb344b2bbdfd97ed61ecf29fbd) fix: objects get from lister must be treated as read-only (#829)
* [`d1bb4ac`](https://github.com/apache/apisix-ingress-controller/commit/d1bb4ac7709e30767ec6a07830221ffabe9887d0) fix: ApisixClusterConfig e2e test case (#859)
* [`fd76c2a`](https://github.com/apache/apisix-ingress-controller/commit/fd76c2aa0f27046816dc8ddb642a3e6c644ab358) feat: ensure that the lease can be actively released before program shutdown to reduce the time required for failover (#827)
* [`4c94c76`](https://github.com/apache/apisix-ingress-controller/commit/4c94c7660170b35a110707fdd5229912a542314a) chore: update ingress/comapre.go watchingNamespac from v2beta1 to v2beta3 (#832)
* [`f9c60c2`](https://github.com/apache/apisix-ingress-controller/commit/f9c60c2738a13cbf9b221030a5cba7e5caf6495d) fix: add v2beta3 register resources (#833)
* [`4a2ebaf`](https://github.com/apache/apisix-ingress-controller/commit/4a2ebaf43ee4cf42212659b412e7ed8a8ab8ee21) chore: fix typo in ApidixRoute CRD (#830)
* [`46fcf3f`](https://github.com/apache/apisix-ingress-controller/commit/46fcf3f153a00cef868454b06452065172500dd1) fix: consumer name contain "-" (#828)
* [`b7dd90a`](https://github.com/apache/apisix-ingress-controller/commit/b7dd90ad9d6a5102fe67c91c29ab76fc4bad4b1a) chore: v1.4 release

</p>
</details>

### Dependency Changes

This release has no dependency changes

Previous release can be found at [1.4.0](https://github.com/apache/apisix-ingress-controller/releases/tag/1.4.0)

# 1.4.0

Welcome to the 1.4.0 release of apisix-ingress-controller!

This is a **GA** release.

## Highlights

### Roadmap

In next release(v1.5), custom resource's API version v2 will be GA released. Please go to [#707](https://github.com/apache/apisix-ingress-controller/issues/707) for detail.

### Breaking Changes

* In this release(v1.4), all custom resource's API version has been upgraded to `apisix.apache.org/v2beta3`, and we deleted `apisix.apache.org/v2beta3` and `apisix.apache.org/v2alpha1`. Please see [#746](https://github.com/apache/apisix-ingress-controller/pull/746)

### New Features

* We have introduced the **`apisix.apache.org/v2beta3` API version for all custom resources** and deleted `v2alpha1` and `v1` API version  [#746](https://github.com/apache/apisix-ingress-controller/pull/746)
* Initial support for Gateway API [#789](https://github.com/apache/apisix-ingress-controller/pull/789)
* Add a new ApisixPluginConfig CRD for reuse common plugin configurations. [#638](https://github.com/apache/apisix-ingress-controller/issues/638)
* Support regex in Ingress path [#779](https://github.com/apache/apisix-ingress-controller/pull/779)
* We can update the load balancing IP of the Ingress, and it can work in various public cloud environments [#740](https://github.com/apache/apisix-ingress-controller/pull/740)

Please try out the release binaries and report any issues at
<https://github.com/apache/apisix-ingress-controller/issues.>

### Contributors

* Jintao Zhang
* kv
* nevercase
* LXM
* Nic
* chen zhuo
* Mayo Cream
* Nic
* Alex Zhang
* Baoyuan
* Brhetty
* Canh Dinh
* Jintao Zhang
* Sindweller
* Yu.Bozhong
* huzais520
* oliver
* rupipal
* zhang lun hai

### Changes

<details><summary>40 commits</summary>
<p>

* [`a1ef639`](https://github.com/apache/apisix-ingress-controller/commit/a1ef63963c5ee9f4b225e5e663ec060afd4da2f8) feat: add ApisixPluginConfig controller loop and e2e test case (#815)
* [`819b003`](https://github.com/apache/apisix-ingress-controller/commit/819b00318e8cd9b6639913301fb89d2acb168926) fix: delete the cluster object when give up the leadership (#774)
* [`970df2b`](https://github.com/apache/apisix-ingress-controller/commit/970df2bd73b80eabb33b39be86c342b87c511fc4) feat: Initial support for Gateway API (#789)
* [`7b62375`](https://github.com/apache/apisix-ingress-controller/commit/7b6237521b322c3662bece3ab701661f58bed347) fix: some wrong or invalid logs (#804)
* [`52b2e2c`](https://github.com/apache/apisix-ingress-controller/commit/52b2e2c7459b93ff05280849f37662afbeb35b5d) docs(READEME.md): change img size (#805)
* [`eeb7a49`](https://github.com/apache/apisix-ingress-controller/commit/eeb7a49afb7219faaa41e55bef187f3a7ad03f0f) chore: specify the K8S cluster version used for the test (#797)
* [`d9fa775`](https://github.com/apache/apisix-ingress-controller/commit/d9fa77511402976f72a83b719644d9c4b4283128) chore: remove ApisixPluginConfig v2beta2 version (#795)
* [`6110bf5`](https://github.com/apache/apisix-ingress-controller/commit/6110bf54e3185e137fc68cb24771c3170c5c6ce5) feat: implement apisix healthz check (#770)
* [`4a6509c`](https://github.com/apache/apisix-ingress-controller/commit/4a6509c798a28c766c93a11468c205b64d742e98) chore: Issue & PR template (#771)
* [`d4c5b09`](https://github.com/apache/apisix-ingress-controller/commit/d4c5b093e95fca630f9a879111c3394fd1b12ec6) fix: When the spec field of the ApisixUpstream resource is empty, it will panic (#794)
* [`472fbcd`](https://github.com/apache/apisix-ingress-controller/commit/472fbcd62721560ba681883b269cfc72b3c35977) feat: add ApisixPluginConfigs crd to v2beta3 (#792)
* [`413e7ca`](https://github.com/apache/apisix-ingress-controller/commit/413e7ca3f6287551505b6ae6a9ea9a9cb3547c47) feat: implement pluginconfig clients (#638) (#772)
* [`fe4a824`](https://github.com/apache/apisix-ingress-controller/commit/fe4a824f4debe3fd4e4a89584df61d2b6cba8ace) fix: ingress LB status records (#788)
* [`1b2bc34`](https://github.com/apache/apisix-ingress-controller/commit/1b2bc3418bd2c7a2a085e55d7bf937b5c27f1ddb) docs: Optimize installation documentation (#785)
* [`4e84eb8`](https://github.com/apache/apisix-ingress-controller/commit/4e84eb8c88ff922c130dba225ff80a5f52c6b571) feat: support regex in path (#779)
* [`1bbadf0`](https://github.com/apache/apisix-ingress-controller/commit/1bbadf0d8e6aefeb11e55ab0d7230547d3c06135) feat: add v2beta3 (#746)
* [`26d5c5c`](https://github.com/apache/apisix-ingress-controller/commit/26d5c5cf96d7e5ece89aed62dcc557bad8fe61bf) Docs: add more config example (#777)
* [`1141e15`](https://github.com/apache/apisix-ingress-controller/commit/1141e15c2678fc9aa8f9e36ffdf804c6f4c2e441) fix: test case param error (#780)
* [`0c6de2d`](https://github.com/apache/apisix-ingress-controller/commit/0c6de2deb92c72b8d609490283c930a068df7d23) feat: update Ingress LB status (#740)
* [`f470867`](https://github.com/apache/apisix-ingress-controller/commit/f4708675c6304ad019881ad7e0ac7a0affd3e6bd) fix: ingress do not watching any namespace when namespaceSelector is empty (#742)
* [`62d7897`](https://github.com/apache/apisix-ingress-controller/commit/62d78978320e9f757843407cc9424568dd4815f9) fix: If resource synchronization retry occurs, other events of the same resource will be blocked (#760)
* [`b127ff4`](https://github.com/apache/apisix-ingress-controller/commit/b127ff4eb47c95fa4db3b58020d7005f739d9dbd) feat: init ApisixPluginConfig crd #4 (#638) (#694)
* [`703c6b2`](https://github.com/apache/apisix-ingress-controller/commit/703c6b2fdbac5c748d2b3c1e54ac62f94d7de41f) fix: ApisixRoute backendPoint duplicate (#732) (#734)
* [`9fe7298`](https://github.com/apache/apisix-ingress-controller/commit/9fe729889471b0291355a69938a5139ec828cfdf) remove route timeout default value (#733)
* [`81f5ea1`](https://github.com/apache/apisix-ingress-controller/commit/81f5ea1d927989cf65c9502d0091059698552a6b) feat: support https and grpcs as upstream scheme as well as mTLS mode (#755)
* [`9f2cd7f`](https://github.com/apache/apisix-ingress-controller/commit/9f2cd7f856f1879ae2586f2a84f4f39d2654996d) feat: support environment variable in config file (#745)
* [`bdf6721`](https://github.com/apache/apisix-ingress-controller/commit/bdf6721e5b0296aeabe7d89cb888b7c7ce759925) Fix bug typo in yaml (#763)
* [`719c42f`](https://github.com/apache/apisix-ingress-controller/commit/719c42f5390794c2d5ac13fb17b5afa96b71055f) docs: update proxy-the-httpbin-service.md (#757)
* [`580e7d4`](https://github.com/apache/apisix-ingress-controller/commit/580e7d4117f9e3c2a8ed6c313b857beed0e2dd6a) feat: expose more prometheus metrics (#670)
* [`774077a`](https://github.com/apache/apisix-ingress-controller/commit/774077a527e43775bcd6346bebdb2ae0b3f80c22) docs: Customize the namespace used for installation (#747)
* [`4a862e2`](https://github.com/apache/apisix-ingress-controller/commit/4a862e206602ae9c7ac534fdfd9a557748b9ad26) fix: use independent dns service for UDP e2e test (#753)
* [`62b7162`](https://github.com/apache/apisix-ingress-controller/commit/62b71628fb621a6625400e3ebc6847c21000d563) fix: wrong var type in response_rewrite e2e (#754)
* [`da30386`](https://github.com/apache/apisix-ingress-controller/commit/da30386c9a4335a82723b46fe7b1342bf0f42867) fix field tag omitempty (#723)
* [`7063189`](https://github.com/apache/apisix-ingress-controller/commit/706318955efa9cb9cee87ff60a4f036f6e32f6f2) docs: add upgrade guide (#735)
* [`65f7c88`](https://github.com/apache/apisix-ingress-controller/commit/65f7c88193eb6e83b2bb4ca87a981321a99503e5) feat: add label-selector for watching namespace (#715)
* [`dc196ef`](https://github.com/apache/apisix-ingress-controller/commit/dc196ef16f95217213321335c6fc3929578e304a) fix unmarshal apisix/upstream field nodes be null (#724)
* [`2a73216`](https://github.com/apache/apisix-ingress-controller/commit/2a732167c9e1b47a80f4b1fc89b4623bf669332e) fix: verify generation in record status (#706)
* [`97fdc90`](https://github.com/apache/apisix-ingress-controller/commit/97fdc90e313a71436f016f5c2e6a849495399ff9) fix: ignore delete pod cache error msg (#714)
* [`fa27b03`](https://github.com/apache/apisix-ingress-controller/commit/fa27b0318468c0ffab40b8c384a8a6abc056748c) chore: fix spelling error in modules.png (#717)
* [`68125e3`](https://github.com/apache/apisix-ingress-controller/commit/68125e3557428dd0e9424a273c977c85fcffc374) chore: v1.3 release (#716)

</p>
</details>

### Dependency Changes

* **github.com/gin-gonic/gin**             v1.6.3 -> v1.7.7
* **github.com/hashicorp/go-memdb**        v1.0.4 -> v1.3.2
* **github.com/hashicorp/go-multierror**   v1.1.0 -> v1.1.1
* **github.com/prometheus/client_golang**  v1.10.0 -> v1.11.0
* **github.com/spf13/cobra**               v1.1.1 -> v1.2.1
* **go.uber.org/multierr**                 v1.3.0 -> v1.7.0
* **go.uber.org/zap**                      v1.13.0 -> v1.19.1
* **golang.org/x/net**                     4163338589ed -> fe4d6282115f
* **k8s.io/api**                           v0.21.1 -> v0.22.4
* **k8s.io/apimachinery**                  v0.21.1 -> v0.22.4
* **k8s.io/client-go**                     v0.21.1 -> v0.22.4
* **k8s.io/code-generator**                v0.21.1 -> v0.22.1
* **sigs.k8s.io/gateway-api**              v0.4.0 **_new_**

Previous release can be found at [1.3.0](https://github.com/apache/apisix-ingress-controller/releases/tag/1.3.0)

# 1.3.0

Welcome to the 1.3.0 release of apisix-ingress-controller!

This is a **GA** release.

## Highlights

### Roadmap

In next release(v1.4), all custom resource versions will be upgraded to version v2beta3, and version v2 will be GA released in version v1.5. Please go to [#707](https://github.com/apache/apisix-ingress-controller/issues/707) for detail.

### Breaking Changes

* In this release(v1.3), the CRD version has been upgraded to `apiextensions.k8s.io/v1`, which means that **the minimum version of Kubernetes supported by APISIX Ingress is v1.16 and later**.
* The ValidatingWebhookConfiguration version has been upgraded to `admissionregistration.k8s.io/v1`, which means that if you want using the default Dynamic Admission Control, you need ensure that the Kubernetes cluster is at least as new as v1.16.

### New Features

* We have introduced the **v2beta2 version of ApisixRoute** and will drop support for `v2alpha1` ApisixRoute [#698](https://github.com/apache/apisix-ingress-controller/pull/698)
* Add cert-manager support [#685](https://github.com/apache/apisix-ingress-controller/pull/685)
* Add full compare when APISIX Ingress startup [#680](https://github.com/apache/apisix-ingress-controller/pull/680)
* Support TLS for Ingress v1 [#634](https://github.com/apache/apisix-ingress-controller/pull/634)
* Add admission server and a validation webhook for plugins [#573](https://github.com/apache/apisix-ingress-controller/pull/573)
* Add `timeout` field for ApisixRoute CR [#609](https://github.com/apache/apisix-ingress-controller/pull/609)
* Add new metrics `apisix_ingress_controller_check_cluster_health` and `apisix_ingress_controller_sync_success_total` [#627](https://github.com/apache/apisix-ingress-controller/pull/627)

Please try out the release binaries and report any issues at
<https://github.com/apache/apisix-ingress-controller/issues.>

### Contributors

* kv
* Hoshea Jiang
* Jintao Zhang
* Sarasa Kisaragi
* Baoyuan
* SergeBakharev
* Sindweller
* chen zhuo
* liuxiran
* oliver

### Changes

<details><summary>27 commits</summary>
<p>

* [`a290f12`](https://github.com/apache/apisix-ingress-controller/commit/a290f12cac2d7c8bcc51863cf42bc13b59bfe128) docs: correct helm repo (#657)
* [`a01888b`](https://github.com/apache/apisix-ingress-controller/commit/a01888bd195f59ae08a5e1399dd26f2ac438880a) feat: Change field retries to value from pointer. (#647)
* [`6f46ac2`](https://github.com/apache/apisix-ingress-controller/commit/6f46ac29a1bf3e51987169153a10be223fcf414f) Make webhook cover ApisixRoute v2beta2 (#705)
* [`9dd4f40`](https://github.com/apache/apisix-ingress-controller/commit/9dd4f40b9fc74be6c29ba11cf9086ecbbd51f9e2) feat: add webhooks for consumer/tls/upstream (#667)
* [`657a1fd`](https://github.com/apache/apisix-ingress-controller/commit/657a1fd1d06b05015e609c5e50107c7358fc44c0) doc: add grpc proxy (#699)
* [`88be11a`](https://github.com/apache/apisix-ingress-controller/commit/88be11a895d72dfa7d0fef09e2b7d00b3210efe9) fix: CRD v1 preserve unknown fields (#702)
* [`d46b248`](https://github.com/apache/apisix-ingress-controller/commit/d46b2485834e0ab4198a567cc9a8d3d2bcc60e6b) feat: upgrade ApisixRoute v2beta2 apiversion. (#698)
* [`736aba3`](https://github.com/apache/apisix-ingress-controller/commit/736aba38f7de1fef03b6b818aa93e343b1666c95) feat: upgrade admission apiversion to v1 (#697)
* [`0630ac5`](https://github.com/apache/apisix-ingress-controller/commit/0630ac55697eaf01017715fcad87b154cb64d9d4) feat: upgrade CRD version to v1 (#693)
* [`957c315`](https://github.com/apache/apisix-ingress-controller/commit/957c31522e1b1e5f8ef9cab7eb244473a4e0f675) feat: add full compare when ingress startup (#680)
* [`1b71fa3`](https://github.com/apache/apisix-ingress-controller/commit/1b71fa32a45d5b5e8e8fc0ed1b761814d169e51f) feat: support cert-manager (#685)
* [`3e9bdbf`](https://github.com/apache/apisix-ingress-controller/commit/3e9bdbf0cee6d49c8e0db27152d46565df704e8c) fix: the fields in UpstreamPassiveHealthCheckUnhealthy should be timeouts (#687)
* [`5c9cdbe`](https://github.com/apache/apisix-ingress-controller/commit/5c9cdbe7fc2c28f3023d635dbbd9bc833388a2bf) fix: remove the step of deleting httpbinsvc (#677)
* [`7216532`](https://github.com/apache/apisix-ingress-controller/commit/721653216b8fe199c15c23aa726157215b12af3a) Remove volumeMounts when webhook is disabled (#679)
* [`1e1be74`](https://github.com/apache/apisix-ingress-controller/commit/1e1be7401ba3707ba660a7d61df5118fc5725eff) add metric: check_cluster_health and sync_operation_total (#627)
* [`6a8658d`](https://github.com/apache/apisix-ingress-controller/commit/6a8658db1788c687c70c9f235601cc8224e0b38c) fix: add initContainers to verify if apisix is ready (#660)
* [`d4a832c`](https://github.com/apache/apisix-ingress-controller/commit/d4a832cf57eb633e8bc1a3bb1a71ba0ae2360337) feat: route crd add timeout fields (#609)
* [`a9960c2`](https://github.com/apache/apisix-ingress-controller/commit/a9960c2a266686fc438451904c8d66430a7d70ee) Add API for getting schema of route, upstream and consumer (#655)
* [`75a2aaa`](https://github.com/apache/apisix-ingress-controller/commit/75a2aaa979c61aaeab9b5412b937a618d1f56bca) feat: Implement the admission server and a validation webhook for plugins (#573)
* [`270a176`](https://github.com/apache/apisix-ingress-controller/commit/270a176a39d34e1d0b213c9d190368919612db9c) fix: e2e failure due to count returned by APISIX (#640)
* [`c284f38`](https://github.com/apache/apisix-ingress-controller/commit/c284f382576251c7d849a43710f8d09667b05dd1) docs: update practices index for website (#654)
* [`9ab367f`](https://github.com/apache/apisix-ingress-controller/commit/9ab367f9d35e67616f678471d3407566a2b6b126) docs: Supplement FAQ for the error log 'no matches for kind "ApisixRoute" in version "apisix.apache.org/v2beta1"' (#651)
* [`62b7590`](https://github.com/apache/apisix-ingress-controller/commit/62b7590443e037ecd6b41521accea567e09ad340) feat: support TLS for ingress v1 (#634)
* [`68b7d7d`](https://github.com/apache/apisix-ingress-controller/commit/68b7d7d231f548d61455f0e95a4ae157de0f5ff8) chore: release v1.2.0 (#633)
* [`d537ddc`](https://github.com/apache/apisix-ingress-controller/commit/d537ddc62bfabfe383c0bd402833377003a1d8dc) feat: add link check (#635)
* [`d7128a1`](https://github.com/apache/apisix-ingress-controller/commit/d7128a1812053e9341f59f0e9c13c1c513c9db42) chore: skip CodeQL if go files have no changes (#636)
* [`d8854c3`](https://github.com/apache/apisix-ingress-controller/commit/d8854c3bf7fefbc54c0d5b00b5ad669044f791f2) docs: fix config.json (#628)

</p>
</details>

### Dependency Changes

* **github.com/fsnotify/fsnotify**         v1.5.0 **_new_**
* **github.com/prometheus/client_golang**  v1.7.1 -> v1.10.0
* **github.com/slok/kubewebhook/v2**       v2.1.0 **_new_**
* **github.com/stretchr/testify**          v1.6.1 -> v1.7.0
* **github.com/xeipuuv/gojsonschema**      v1.2.0 **_new_**
* **golang.org/x/sys**                     0f9fa26af87c -> bce67f096156

Previous release can be found at [1.2.0](https://github.com/apache/apisix-ingress-controller/releases/tag/1.2.0)

# 1.2.0

Welcome to the 1.2.0 release of apisix-ingress-controller!

This is a **GA** release.

## Highlights

### New Features

* **Support ingress v1beta1 HTTPS** [#596](https://github.com/apache/apisix-ingress-controller/pull/596)
* **Implement schema API** [#601](https://github.com/apache/apisix-ingress-controller/pull/601)

Please try out the release binaries and report any issues at
<https://github.com/apache/apisix-ingress-controller/issues.>

### Contributors

* kv
* Jintao Zhang
* Baoyuan
* Hoshea Jiang
* chen zhuo
* okaybase
* yuanfeng0905
* 天使莫忆

### Changes

<details><summary>20 commits</summary>
<p>

* [`3ab162b`](https://github.com/apache/apisix-ingress-controller/commit/3ab162bfcaf82ecb50e67029990b97c98d0e18eb) chore: bump version v1.2.0
* [`3ad1a1c`](https://github.com/apache/apisix-ingress-controller/commit/3ad1a1cca0bf23da803b703eeed36b7c4931d387) docs: fix install docs (#579)
* [`499962b`](https://github.com/apache/apisix-ingress-controller/commit/499962be2306e341c30a84738c8a530562e00b9f) docs: ApisixRoute v2alpha1 is deprecated (#623)
* [`c1de18f`](https://github.com/apache/apisix-ingress-controller/commit/c1de18fa2e59bdf02398290e15199422cf75ba81) docs: update mTLS support in ApisixTls reference (#624)
* [`3cd6892`](https://github.com/apache/apisix-ingress-controller/commit/3cd689294d236495ffc5ca0071edcd856603a878) fix: ApisixRoute printcolumns (#626)
* [`91d985e`](https://github.com/apache/apisix-ingress-controller/commit/91d985edca67493acc22536c66d4947fe597052f) fix field tag omiteempty (#622)
* [`f78248a`](https://github.com/apache/apisix-ingress-controller/commit/f78248a6aca0ac4754fd4d5d410c28f1cdd3d9c7) fix: sync apisix failed when use v2beta1 ApisixRoute (#620)
* [`00ff017`](https://github.com/apache/apisix-ingress-controller/commit/00ff01768429001e05b87c1a704c12c43c76f012) ci: add ingress log when e2e failed (#616)
* [`e5441a3`](https://github.com/apache/apisix-ingress-controller/commit/e5441a3d0877017f17e96ac44d2a804a509676e7) feat: implement schema API (#601)
* [`5635652`](https://github.com/apache/apisix-ingress-controller/commit/5635652865ef965db58794b08e2fe37ddc9c08e3) fix: timer leak memory (#591)
* [`812e4bd`](https://github.com/apache/apisix-ingress-controller/commit/812e4bd3ca197c7ea227ff7de8b5ee1b6ac9424d) docs: add declarations for the version of APISIX (#595)
* [`915a5d1`](https://github.com/apache/apisix-ingress-controller/commit/915a5d1d99d68f9c08b4d59e9b2e9cb8fa0dde31) test: add assert for test cases (#613)
* [`d12a900`](https://github.com/apache/apisix-ingress-controller/commit/d12a900976e9118fe1b5fd0df137426a294f7a97) fix: add v2beta1 logic (#615)
* [`ac25764`](https://github.com/apache/apisix-ingress-controller/commit/ac25764d46448b4df5772bde236be8334f47a7f7) feat: support ingress v1beta1 https (#596)
* [`866d0bf`](https://github.com/apache/apisix-ingress-controller/commit/866d0bfe38b1e7bfb3340c1f1a317f8d604673f5) docs: modify the format of FAQ.md (#605)
* [`2d12c3f`](https://github.com/apache/apisix-ingress-controller/commit/2d12c3fdd1369e9e263342d0def6473de5c0664f) docs: add v2beta1 description (#602)
* [`7291212`](https://github.com/apache/apisix-ingress-controller/commit/7291212964a7f3505fb1bb624fb79df0c9eb0678) fix: do not need to record status when ApisixUpstream removed (#589)
* [`c78c823`](https://github.com/apache/apisix-ingress-controller/commit/c78c8237b9c114e0e5564bd74a6729bdee04259c) chore: merge from v1.1 (#583)
* [`e649c50`](https://github.com/apache/apisix-ingress-controller/commit/e649c503ca6ba4103a27fef729f8835b90078a93) chore: add udp usage & upgrade the verion of ApisixRoute (#585)
* [`57ec6da`](https://github.com/apache/apisix-ingress-controller/commit/57ec6dafa7b96b8c2fe5c386eedfce74cb581441) fix: misspell in FAQ (#577)

</p>
</details>

### Dependency Changes

* **github.com/google/uuid**  v1.2.0 **_new_**
* **github.com/onsi/ginkgo**  v1.16.4 **_new_**
* **golang.org/x/net**        3d97a244fca7 -> 4163338589ed
* **golang.org/x/sys**        0f9fa26af87c **_new_**
* **golang.org/x/tools**      v0.1.5 **_new_**

Previous release can be found at [1.1.0](https://github.com/apache/apisix-ingress-controller/releases/tag/1.1.0)

# 1.1.0

Welcome to the 1.1.0 release of apisix-ingress-controller!

This is a **GA** release.

- an available Kubernetes cluster (version >= 1.15)
- an available Apache APISIX (version >= 2.7)

## Highlights

### New Features

* **Support EndpointSlices** [#563](https://github.com/apache/apisix-ingress-controller/pull/563) [#574](https://github.com/apache/apisix-ingress-controller/pull/574)
* **Support UDP definition** [#572](https://github.com/apache/apisix-ingress-controller/pull/572) [#576](https://github.com/apache/apisix-ingress-controller/pull/576)

Please try out the release binaries and report any issues at
<https://github.com/apache/apisix-ingress-controller/issues.>

### Contributors

* Alex Zhang
* Fang
* kv
* Jintao Zhang
* Shuyang Wu
* benson211

### Changes

<details><summary>11 commits</summary>
<p>

* [`67f3fd9`](https://github.com/apache/apisix-ingress-controller/commit/67f3fd934b8a8b935440227a5c8ba7923ba91a2a) chore: endpointslice controller (#574)
* [`1c17b41`](https://github.com/apache/apisix-ingress-controller/commit/1c17b41249361444b5b10f4a8897f62484b545b0) feat: add logic for ApisixRoute v2beta1 (#576)
* [`a754f69`](https://github.com/apache/apisix-ingress-controller/commit/a754f69d3f2a7b637db039690ad849976178c148) feat: abstract the endpoints-related logic (#563)
* [`4b16e28`](https://github.com/apache/apisix-ingress-controller/commit/4b16e289073ad88dc6be49ae621294ecf6c92cb4) chore: cleanup apisixservice. (#566)
* [`534fab3`](https://github.com/apache/apisix-ingress-controller/commit/534fab34a6e046a46fb174d2a3620eacb431006f) feat: add v2beta1 structure for ApisixRoute (#572)
* [`c871bdf`](https://github.com/apache/apisix-ingress-controller/commit/c871bdfb76fee37227d7de5a246cd94b867aad1d) test: dump the namespace content when e2e test cases failed (#571)
* [`dbc8133`](https://github.com/apache/apisix-ingress-controller/commit/dbc8133805122947bbad21711d66d0782e66bbb5) doc: update k3s-rke.md (#568)
* [`70d0100`](https://github.com/apache/apisix-ingress-controller/commit/70d010092b626c9c6959bd026e669dbb60153608) chore: update config of installation by Kustomize (#557)
* [`2122d76`](https://github.com/apache/apisix-ingress-controller/commit/2122d76fd28cc7bce54e8f52e2d4c9d04a1e852a) docs: clarify installation by Kustomize (#558)
* [`b4a6889`](https://github.com/apache/apisix-ingress-controller/commit/b4a6889e1564be61de6736af32a2075579c9b51f) Update default version in Makefile (#556)
* [`f5cc76e`](https://github.com/apache/apisix-ingress-controller/commit/f5cc76ec6ac671772063d38a09f508db71ac2e48) chore: remove cancel-workflow.yml since no use (#550)

</p>
</details>

### Dependency Changes

* **github.com/fsnotify/fsnotify**        v1.4.9 **_new_**
* **github.com/gruntwork-io/terratest**   v0.32.8 **_new_**
* **github.com/hashicorp/go-multierror**  v1.0.0 -> v1.1.0
* **k8s.io/api**                          v0.20.2 -> v0.21.1
* **k8s.io/apimachinery**                 v0.20.2 -> v0.21.1
* **k8s.io/client-go**                    v0.20.2 -> v0.21.1

Previous release can be found at [1.0.0](https://github.com/apache/apisix-ingress-controller/releases/tag/1.0.0)

# 1.0.0

Welcome to the 1.0.0 release of apisix-ingress-controller!

This is the first **GA** release.

- an available Kubernetes cluster (version >= 1.14)
- an available Apache APISIX (version >= 2.7)

## Highlights

### New Features

* **Support blocklist-source-range annotation for Ingress source** [#446](https://github.com/apache/apisix-ingress-controller/pull/446)
* **Add ApisixConsumer CRD** [#462](https://github.com/apache/apisix-ingress-controller/pull/462)
* **Support rewrite annotation for Ingress source** [#480](https://github.com/apache/apisix-ingress-controller/pull/480)
* **Support http-to-https redirect annotation for Ingress source** [#484](https://github.com/apache/apisix-ingress-controller/pull/484)
* **Add health check to apisix-admin and make the leader election recyclable** [499](https://github.com/apache/apisix-ingress-controller/pull/499)
* **Support mTLS for ApisixTls** [#492](https://github.com/apache/apisix-ingress-controller/pull/492)
* **Support authentication for ApisixRoute** [#528](https://github.com/apache/apisix-ingress-controller/pull/528)

Please try out the release binaries and report any issues at
<https://github.com/apache/apisix-ingress-controller/issues.>

### Contributors

* Alex Zhang
* Sarasa Kisaragi
* Jintao Zhang
* kv
* Shuyang Wu
* Daming
* Fang
* Ayush das
* Donghui0
* Shivani chauhan
* Yuelin Zheng
* guoqqqi
* 罗泽轩

### Changes

<details><summary>58 commits</summary>
<p>

* [`f3ab30b`](https://github.com/apache/apisix-ingress-controller/commit/f3ab30b41a4e918fe88fb8290e81d136846f2fec) docs: modify readme (#543)
* [`f9df546`](https://github.com/apache/apisix-ingress-controller/commit/f9df5469aa032cd05955f93c59a32883db542c02) ci: do not run workflows for draft PRs (#542)
* [`fca6211`](https://github.com/apache/apisix-ingress-controller/commit/fca62110b81958935263c816f71be96c4500a84e) chore: add authentication for ApisixRoute (#528)
* [`28c584e`](https://github.com/apache/apisix-ingress-controller/commit/28c584ea33824434a7e872f328e8e90a09fb2213) chore: remove echo plugin's auth test case. (#534)
* [`1eee479`](https://github.com/apache/apisix-ingress-controller/commit/1eee479247404893cf5f4ae5ad78c6714a71f63c) fix: nil pointer dereference (#529)
* [`7379d57`](https://github.com/apache/apisix-ingress-controller/commit/7379d57359b82f5521722814d860e49632e717f3) docs: removed navigation title from sidebar to docs dropdown (#531)
* [`2bf4b6b`](https://github.com/apache/apisix-ingress-controller/commit/2bf4b6be29648c1c3e98006edae50252a0555a08) fix: add namespace for subjects of ClusterRoleBinding (#527)
* [`d3ec856`](https://github.com/apache/apisix-ingress-controller/commit/d3ec85657c10a54875cfad05fdebc67b9358cef2) ci: use concurrency to cancel workflow (#525)
* [`5c1aa5e`](https://github.com/apache/apisix-ingress-controller/commit/5c1aa5ef26a6546e5e339bcc6d3cdae31b534da6) docs: add docs about Ingress feature comparison (#526)
* [`d510a8a`](https://github.com/apache/apisix-ingress-controller/commit/d510a8abdc6d4f94d2478eefb53fca16c4b88eb4) doc: update development.md (#524)
* [`f6cb4f9`](https://github.com/apache/apisix-ingress-controller/commit/f6cb4f9a0300b13ee586a9536623962d183d9d6c) feat: consumer controller loop (#516)
* [`3337be7`](https://github.com/apache/apisix-ingress-controller/commit/3337be7c7d5f959301171a243f4c0c0d49360503) feat: subset changes in controllers (#507)
* [`c6ac8a4`](https://github.com/apache/apisix-ingress-controller/commit/c6ac8a40526d3d30b25347dce330630f623c1e00) fix: CI path filter (#522)
* [`fa0d8a6`](https://github.com/apache/apisix-ingress-controller/commit/fa0d8a69b9cc2ceda9b37872841ded56aebc5f8e) ci: remove stale ci/spell-checker configuration (#519)
* [`3d9fd07`](https://github.com/apache/apisix-ingress-controller/commit/3d9fd07cc86318a420a5bf794831d039a7b6d0b8) ci: add changes filter (#520)
* [`38290a2`](https://github.com/apache/apisix-ingress-controller/commit/38290a2893b4bf77869b34648aeb8d55dd298537) feat: ApisixTls support mTLS (#492)
* [`029c0d7`](https://github.com/apache/apisix-ingress-controller/commit/029c0d7a26c0a3cd507f15f5dcdbff0a09799c24) feat: add events and status for ApisixClusterConfig resource (#502)
* [`a89be23`](https://github.com/apache/apisix-ingress-controller/commit/a89be230989ea62d03062181626cc197df655a78) feat: subset translation (#497)
* [`87b7229`](https://github.com/apache/apisix-ingress-controller/commit/87b7229e6db549f4bd65561399d976a91fdd7978) Update license-checker.yml (#510)
* [`495c631`](https://github.com/apache/apisix-ingress-controller/commit/495c6317a618683d2c69c48c489763c4c8285504) chore: add verify scripts and verify-codegen CI (#513)
* [`2f2e6f8`](https://github.com/apache/apisix-ingress-controller/commit/2f2e6f861ca0a27d84ad84763cc0a070e9b6c91d) feat: add permission to events, fix missing subresources in crd. (#514)
* [`880d573`](https://github.com/apache/apisix-ingress-controller/commit/880d5736f089daff6682eae0450eae6c18bfef53) ci: fix cancel workflow not working (#509)
* [`cddcd29`](https://github.com/apache/apisix-ingress-controller/commit/cddcd299459cc1a0ad2aee02e611cc88cda64c8e) feat: add pod controller and pod cache (#490)
* [`23e5ebd`](https://github.com/apache/apisix-ingress-controller/commit/23e5ebdb837cf581db94a613e02e292167d52eae) feat: apisixconsumer translator (#474)
* [`fe2db92`](https://github.com/apache/apisix-ingress-controller/commit/fe2db92740eda6dab6f50cb096b279aec7c0d15b) chore: add docker ignore to avoid unwanted cache miss (#506)
* [`d87f856`](https://github.com/apache/apisix-ingress-controller/commit/d87f856acbaf3f11a0559199ad7090beea7bcc45) ci: fix cancel workflow not working (#508)
* [`a3f58d0`](https://github.com/apache/apisix-ingress-controller/commit/a3f58d07a749b4594e460b4645ef77d8d21598fb) fix: ack.md link fix (#503)
* [`553655b`](https://github.com/apache/apisix-ingress-controller/commit/553655b1148360795a71b27117898ff5642be8a5) chore: add dnsPolicy for sample deployment (#498)
* [`f089ffe`](https://github.com/apache/apisix-ingress-controller/commit/f089ffe9788526b95e43d1c42efc0757b062a8cf) test: remove custom apisix-default.yaml (#494)
* [`b7736db`](https://github.com/apache/apisix-ingress-controller/commit/b7736dbb58f3df91197fda4da9519e90a4de2a1f) ci: cancel duplicate workflow to reduce CI queue time (#505)
* [`582c4b3`](https://github.com/apache/apisix-ingress-controller/commit/582c4b362f26ffa8372bf520c3f774170a56c290) chore: add health check to apisix-admin and make the leader election recyclable (#499)
* [`77a06cc`](https://github.com/apache/apisix-ingress-controller/commit/77a06cc3c6a2762f996b44833e1d802a6007c425) feat: add support for http-to-https redirect annotation (#484)
* [`fa98443`](https://github.com/apache/apisix-ingress-controller/commit/fa98443daaa3b8f4b1be75a4e025eedf06550e51) chore: regenerate codes (#491)
* [`6630aac`](https://github.com/apache/apisix-ingress-controller/commit/6630aaced835265951bfb76453a7a812ad15e7aa) fix: ingress_class configuration invalid(#475) (#477)
* [`e8eddcc`](https://github.com/apache/apisix-ingress-controller/commit/e8eddcc7791d64181a13bf8714ca141a1ca4e7e5) docs: ingress apisix the hard way (#479)
* [`36de069`](https://github.com/apache/apisix-ingress-controller/commit/36de06967bedaaa4296af4a427df920bd7ca63a3) feat: codegen script (#487)
* [`1d7b143`](https://github.com/apache/apisix-ingress-controller/commit/1d7b14343f7d901ac4cc4170fc64d095ad882f72) feat: support rewrite annotation (#480)
* [`5af1fb4`](https://github.com/apache/apisix-ingress-controller/commit/5af1fb49bc8fdb418d3da69c2a283092caaf938b) feat: add essential data structures for service subset selector (#489)
* [`a16e980`](https://github.com/apache/apisix-ingress-controller/commit/a16e980237fb61bdaf9f980660e4cbbf42843c83) fix: fatal error reported when run make build in release src (#485)
* [`1dd5087`](https://github.com/apache/apisix-ingress-controller/commit/1dd5087aea443a0aeddb62a8aa0af90aab2bf48e) chore: consumer data structures (#470)
* [`d6d3796`](https://github.com/apache/apisix-ingress-controller/commit/d6d37960eb5db70577c544e25dbd1f31782270e2) chore: fix e2e ip-restriction plugin text (#488)
* [`92896f1`](https://github.com/apache/apisix-ingress-controller/commit/92896f1c6d0bfd8bd0e31c7293b9d6b9befdef87) chore: e2e case for tcp proxy is unstable\nclose #473 (#486)
* [`bc71e3e`](https://github.com/apache/apisix-ingress-controller/commit/bc71e3e25a8514548fddbf900318457ded3e3076) chore: add apisixconsumer data structures (#462)
* [`269cf07`](https://github.com/apache/apisix-ingress-controller/commit/269cf07020cac239aac5e7d7334bc63305e740fb) test: add basic headless service e2e test (#466)
* [`1ffa862`](https://github.com/apache/apisix-ingress-controller/commit/1ffa862b788f003a07a259da4b9b10f018a87698) fix: event record scheme error (#469)
* [`456fbd2`](https://github.com/apache/apisix-ingress-controller/commit/456fbd2f776845d92c2899bb5fef61d688f49244) fix: remove upstream which is ref by multi-routes cause retry (#472)
* [`a7e187b`](https://github.com/apache/apisix-ingress-controller/commit/a7e187bd3a11218c0e24bd61974bad22becccc95) minor: optimize log message when the endpoint does not have a corresponding service (#458)
* [`63ae709`](https://github.com/apache/apisix-ingress-controller/commit/63ae709d064e28a565f80176aa82a3ff7b69b293) chore: fix broken links (#467)
* [`0bdd24b`](https://github.com/apache/apisix-ingress-controller/commit/0bdd24b86ca109948e786f7f13f84bc1bd0fbc39) chore: change the required PR approving number to 2 (#463)
* [`015940c`](https://github.com/apache/apisix-ingress-controller/commit/015940cedfe6951fe2ec8d2d56f11f3f484716b8) docs: fix APISIX helm installation (#459)
* [`4a55307`](https://github.com/apache/apisix-ingress-controller/commit/4a55307b6a34ba1145e131f85b5f05f909e8d244) fix: add status subresource permission in clusterRole (#452)
* [`5d479ae`](https://github.com/apache/apisix-ingress-controller/commit/5d479ae148d2acdb51082bb0f129548fdfa146b4) feat: blocklist-source-range annotation (#446)
* [`8824bbd`](https://github.com/apache/apisix-ingress-controller/commit/8824bbdf113bbf72649ccd5dc43af3a66773bf5b) chore: refactor the process of annotations (#443)
* [`9d0e0b8`](https://github.com/apache/apisix-ingress-controller/commit/9d0e0b856c3ebe0d6bb10ee4711ea266685fb866) fix: wait for the default cluster ready continuously (#450)
* [`fb11efc`](https://github.com/apache/apisix-ingress-controller/commit/fb11efc00a914e1992a8a730cf5443a3ea38e8be) chore: refactor the structures of kube clients, shared index informer factories (#431)
* [`f199cdb`](https://github.com/apache/apisix-ingress-controller/commit/f199cdb5f5bfe3cb5acb19dc1903b1f5f426a353) test: add e2e test cases for server-info plugin (#406)
* [`b0a6f3e`](https://github.com/apache/apisix-ingress-controller/commit/b0a6f3edba8a80e10e831ceaf408e43f89632adb) fix: typo in apisix_route_v2alpha1.md (#438)
* [`d269a01`](https://github.com/apache/apisix-ingress-controller/commit/d269a01fe69c287cf13a3574d8ce6566c18a306c) ci: introduce skywalking-eyes (#430)

</p>
</details>

### Dependency Changes

* **golang.org/x/net**       6772e930b67b -> 3d97a244fca7
* **gopkg.in/yaml.v2**       v2.3.0 -> v2.4.0
* **k8s.io/code-generator**  v0.21.1 **_new_**

Previous release can be found at [0.6.0](https://github.com/apache/apisix-ingress-controller/releases/tag/0.6.0)

# 0.6.0

We have added some new features, fixed some bugs, and made some optimizations to the internal code.

**Note: The CRDs should be re-applied because of some new features**

## Core

* Support TCP definition [#115](https://github.com/apache/apisix-ingress-controller/issues/115)
* Add labels to mark resources are pushed by ingress controller [#242](https://github.com/apache/apisix-ingress-controller/issues/242)
* Add jsonschema validate for ApisixUpstream and ApisixTls resource [#371](https://github.com/apache/apisix-ingress-controller/issues/371) [#372](https://github.com/apache/apisix-ingress-controller/issues/372)
* Support to record kubernetes events for resources processing [#394](https://github.com/apache/apisix-ingress-controller/issues/394)
* Support to report resources status [#395](https://github.com/apache/apisix-ingress-controller/issues/395)
* Support global_rules for cluster scoped plugins [#402](https://github.com/apache/apisix-ingress-controller/issues/402)

## Fix

* Remove upstream caching correctly [#421](https://github.com/apache/apisix-ingress-controller/issues/421)
* Avoid retrying caused by 404 when deleting cache [#424](https://github.com/apache/apisix-ingress-controller/pull/424)
* Handle cookie exprs correctly [#425](https://github.com/apache/apisix-ingress-controller/pull/425)

# 0.5.0

A lot of important features are supported in this release, it makes apisix-ingress-controller more powerful and flexible.
Also, several bugs are fixed so the robustness is also enhanced.

We recommend you to use [Apache APISIX 2.5](https://github.com/apache/apisix/releases/tag/2.5) with this release. Note since CRDs are updated, when
you upgrade your old release, **manual steps are required to apply the new ApisixRoute**. Please see the instruction `7` in [FAQ](https://github.com/apache/apisix-ingress-controller/blob/master/docs/en/latest/FAQ.md) for more details.

## Core

* Support traffic split feature ([#308](https://github.com/apache/apisix-ingress-controller/pull/308))
* Support route match exprs ([#304](https://github.com/apache/apisix-ingress-controller/pull/304), [#306](https://github.com/apache/apisix-ingress-controller/pull/306))
* Support to configure [Ingress](https://kubernetes.io/docs/concepts/services-networking/ingress/) resources in version `extensions/v1beta1` ([#296](https://github.com/apache/apisix-ingress-controller/pull/296), [#315](https://github.com/apache/apisix-ingress-controller/pull/315))
* Add name fields when generating APISIX Routes and Upstreams ([#333](https://github.com/apache/apisix-ingress-controller/pull/333))
* Support to use remote addrs as route match conditions ([#347](https://github.com/apache/apisix-ingress-controller/pull/347))
* Schema for ApisixRoute CRD ([#345](https://github.com/apache/apisix-ingress-controller/pull/345))

## Fix

* Sometimes ApisixRoute update is ineffective ([#319](https://github.com/apache/apisix-ingress-controller/pull/319))
* Priority field is not passed to APISIX ([#329](https://github.com/apache/apisix-ingress-controller/pull/329))
* Route rule name in ApisixRoute can be duplicated ([#330](https://github.com/apache/apisix-ingress-controller/pull/330))
* Use `PUT` instead of `PATCH` method when updating resources ([#353](https://github.com/apache/apisix-ingress-controller/pull/353))
* Secrets controller doesn't push the newest cert and priv key to APISIX ([#337](https://github.com/apache/apisix-ingress-controller/pull/337))

## Test

* Use [Kind](https://kind.sigs.k8s.io/) to run e2e suites ([#331](https://github.com/apache/apisix-ingress-controller/pull/331))
* Add e2e test cases for plugins redirect, uri-blocker, fault-injection, request-id, limit-count, echo, cors, response-rewrite, proxy-rewrite ([#320](https://github.com/apache/apisix-ingress-controller/pull/320), [#327](https://github.com/apache/apisix-ingress-controller/pull/327), [#328](https://github.com/apache/apisix-ingress-controller/pull/328), [#334](https://github.com/apache/apisix-ingress-controller/pull/334), [#336](https://github.com/apache/apisix-ingress-controller/pull/336), [#342](https://github.com/apache/apisix-ingress-controller/pull/342), [#341](https://github.com/apache/apisix-ingress-controller/pull/341))

# 0.4.0

This release mainly improves the program robustness and adds some features.

## Core

- Support Kubernetes Ingress resources [#250](https://github.com/apache/apisix-ingress-controller/pull/250)
- Support ApisixRoute v2alpha1 [#262](https://github.com/apache/apisix-ingress-controller/pull/262)
- Support healthchecks definition [#117](https://github.com/apache/apisix-ingress-controller/issues/117)
- Support secret controller [#284](https://github.com/apache/apisix-ingress-controller/pull/284)
- Project optimization [#92](https://github.com/apache/apisix-ingress-controller/issues/92)

## Test

- Add test cases for pkg/kube [#99](https://github.com/apache/apisix-ingress-controller/issues/99)

# 0.3.0

This release mainly improves the program robustness and adds some features.

## Core

- Support Leader election to let only the leader process resources [#173](https://github.com/apache/apisix-ingress-controller/pull/173);
- Let Controller itself generates resource ids instead of relying on APISIX [#199](https://github.com/apache/apisix-ingress-controller/pull/199);
- Change go module name from `github.com/api7/ingress-controller` to `github.com/apache/apisix-ingress-controller` [#220](https://github.com/apache/apisix-ingress-controller/pull/220);
- Re draw the design diagram [#214](https://github.com/apache/apisix-ingress-controller/pull/214);
- Support gRPC scheme in ApisixUpstream [#225](https://github.com/apache/apisix-ingress-controller/pull/225);
- SSL resource cache optimization [#203](https://github.com/apache/apisix-ingress-controller/pull/203);

## Deploy

- Complete the compatibility tests on Amazon EKS, Google Cloud GKE, Ali Cloud ACK and etc [#177](https://github.com/apache/apisix-ingress-controller/pull/177), [#180](https://github.com/apache/apisix-ingress-controller/pull/180), [#183](https://github.com/apache/apisix-ingress-controller/pull/183);
- Refactor the helm charts, merging ingress-apisix and ingress-apisix-base into apisix-ingress-controller [#213](https://github.com/apache/apisix-ingress-controller/pull/213);

## Test

- Now CI runs e2e test suites in parallel [#172](https://github.com/apache/apisix-ingress-controller/pull/172);

# 0.2.0

This release mainly improve basic features, bugfix and adds test cases.

## Core

- Enhanced documentation, easier to read and execute [#129](https://github.com/apache/apisix-ingress-controller/pull/129)
- API specification for CRDs [#151](https://github.com/apache/apisix-ingress-controller/pull/151)
- Support Canary plugin (Base on the [feature](https://github.com/apache/apisix/pull/2935) in Apache APISIX) [#13](https://github.com/apache/apisix-ingress-controller/issues/13)
- Support prometheus metrics [#143](https://github.com/apache/apisix-ingress-controller/pull/143)
- Support install ingress controller by Helm Chart [#153](https://github.com/apache/apisix-ingress-controller/pull/153)
- Support reconcile loop. [#149](https://github.com/apache/apisix-ingress-controller/pull/149) [#157](https://github.com/apache/apisix-ingress-controller/pull/157) [#163](https://github.com/apache/apisix-ingress-controller/pull/163)
- Support namespaces filtering. [#162](https://github.com/apache/apisix-ingress-controller/pull/162)
- Some Refactor. [#147](https://github.com/apache/apisix-ingress-controller/pull/147) [#155](https://github.com/apache/apisix-ingress-controller/pull/155) [#134](https://github.com/apache/apisix-ingress-controller/pull/134)

## Test case

more e2e case [#156](https://github.com/apache/apisix-ingress-controller/pull/156) [#142](https://github.com/apache/apisix-ingress-controller/pull/142)

[Back to TOC](#table-of-contents)

# 0.1.0

This release mainly improve basic features, bugfix and adds test cases.

## Core

- Enriched documentation.
- CI Integration. [#75](https://github.com/apache/apisix-ingress-controller/pull/75) [#80](https://github.com/apache/apisix-ingress-controller/pull/80) [#84](https://github.com/apache/apisix-ingress-controller/pull/84) [#87](https://github.com/apache/apisix-ingress-controller/pull/87) [#89](https://github.com/apache/apisix-ingress-controller/pull/89) [#97](https://github.com/apache/apisix-ingress-controller/pull/97)
- Support retry when sync failed. [#103](https://github.com/apache/apisix-ingress-controller/pull/103)
- Support using kustomize install all resources. [#72](https://github.com/apache/apisix-ingress-controller/pull/72)
- Support command line configuration. [#61](https://github.com/apache/apisix-ingress-controller/pull/61)
- Support to define SSL by CRD. [#95](https://github.com/apache/apisix-ingress-controller/pull/95)

## Test case

- Add E2E test environment. [#101](https://github.com/apache/apisix-ingress-controller/pull/101)

## Bugfix

- invalid memory address or nil pointer dereference. [#9](https://github.com/api7/seven/pull/9)

[Back to TOC](#table-of-contents)
