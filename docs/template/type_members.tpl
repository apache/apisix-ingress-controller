{{- define "type_members" -}}
{{- $field := . -}}
{{- if eq $field.Name "metadata" -}}
Please refer to the Kubernetes API documentation for details on the `metadata` field.
{{- else -}}
{{- /* First replace makes paragraphs separated, second merges lines in paragraphs. */ -}}
{{ $field.Doc | replace "\n\n" "<br /><br />" |  replace "\n" " " | replace " *" "<br /> •" | replace "<br /><br /><br />" "<br /><br />" }}
{{- end -}}
{{- end -}}
