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
* Xin Rong
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
