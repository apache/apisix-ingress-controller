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
