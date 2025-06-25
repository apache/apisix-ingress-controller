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

{{- define "type_members" -}}
{{- $field := . -}}
{{- if eq $field.Name "metadata" -}}
Please refer to the Kubernetes API documentation for details on the `metadata` field.
{{- else -}}
{{- /* First replace makes paragraphs separated, second merges lines in paragraphs. */ -}}
{{ $field.Doc | replace "\n\n" "<br /><br />" |  replace "\n" " " | replace " *" "<br /> •" | replace "<br /><br /><br />" "<br /><br />" }}
{{- end -}}
{{- end -}}
