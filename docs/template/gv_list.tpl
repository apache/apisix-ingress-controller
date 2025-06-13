{{- define "gvList" -}}
{{- $groupVersions := . -}}

---
title: Custom Resource Definitions API Reference
slug: /reference/apisix-ingress-controller/crd-reference
description: Explore detailed reference documentation for the custom resource definitions (CRDs) supported by the APISIX Ingress Controller.
---

This document provides the API resource description the API7 Ingress Controller custom resource definitions (CRDs).

## Packages
{{- range $groupVersions }}
- {{ markdownRenderGVLink . }}
{{- end }}

{{ range $groupVersions }}
{{ template "gvDetails" . }}
{{ end }}

{{- end -}}
