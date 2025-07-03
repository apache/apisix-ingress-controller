{*
 Licensed to the Apache Software Foundation (ASF) under one
 or more contributor license agreements.  See the NOTICE file
 distributed with this work for additional information
 regarding copyright ownership.  The ASF licenses this file
 to you under the Apache License, Version 2.0 (the
 "License"); you may not use this file except in compliance
 with the License.  You may obtain a copy of the License at

   http://www.apache.org/licenses/LICENSE-2.0

 Unless required by applicable law or agreed to in writing,
 software distributed under the License is distributed on an
 "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY
 KIND, either express or implied.  See the License for the
 specific language governing permissions and limitations
 under the License.
*}

{{- define "gvList" -}}
{{- $groupVersions := . -}}

---
title: Custom Resource Definitions API Reference
slug: /reference/apisix-ingress-controller/crd-reference
description: Explore detailed reference documentation for the custom resource definitions (CRDs) supported by the APISIX Ingress Controller.
---

This document provides the API resource description the APISIX Ingress Controller custom resource definitions (CRDs).

## Packages
{{- range $groupVersions }}
- {{ markdownRenderGVLink . }}
{{- end }}

{{ range $groupVersions }}
{{ template "gvDetails" . }}
{{ end }}

{{- end -}}
