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

{{- define "type" -}}
{{- $type := $.type -}}
{{- $isKind := $.isKind -}}
{{- if markdownShouldRenderType $type -}}

{{- if $isKind -}}
### {{ $type.Name }}
{{ else -}}
#### {{ $type.Name }}
{{ end -}}

{{ if $type.IsAlias }}_Base type:_ `{{ markdownRenderTypeLink $type.UnderlyingType  }}`{{ end }}

{{ $type.Doc | replace "\n\n" "<br /><br />" }}

{{ if $type.GVK -}}
<!-- {{ $type.Name }} resource -->
{{- end }}

{{ if $type.Members -}}
| Field | Description |
| --- | --- |
{{ if $type.GVK -}}
| `apiVersion` _string_ | `{{ $type.GVK.Group }}/{{ $type.GVK.Version }}`
| `kind` _string_ | `{{ $type.GVK.Kind }}`
{{ end -}}

{{ range $type.Members -}}
| `{{ .Name  }}` _{{ markdownRenderType .Type }}_ | {{ template "type_members" . }} |
{{ end -}}

{{ end }}

{{ if $type.References -}}
_Appears in:_
{{- range $type.SortedReferences }}
- {{ markdownRenderTypeLink . }}
{{- end }}
{{- end }}

{{- end -}}
{{- end -}}
