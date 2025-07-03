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

{{- define "gvDetails" -}}
{{- $gv := . -}}

## {{ $gv.GroupVersionString }}

{{ $gv.Doc }}

{{- if $gv.Kinds  }}
{{- range $gv.SortedKinds }}
- {{ $gv.TypeForKind . | markdownRenderTypeLink }}
{{- end }}
{{ end }}

{{- /* Display exported Kinds first */ -}}
{{- range $gv.SortedKinds -}}
{{- $typ := $gv.TypeForKind . }}
{{- $isKind := true -}}
{{ template "type" (dict "type" $typ "isKind" $isKind) }}
{{ end -}}

### Types

In this section you will find types that the CRDs rely on.

{{- /* Display Types that are not exported Kinds */ -}}
{{- range $typ := $gv.SortedTypes -}}
{{- $isKind := false -}}
{{- range $kind := $gv.SortedKinds -}}
{{- if eq $typ.Name $kind -}}
{{- $isKind = true -}}
{{- end -}}
{{- end -}}
{{- if not $isKind }}
{{ template "type" (dict "type" $typ "isKind" $isKind) }}
{{ end -}}
{{- end }}

{{- end -}}
