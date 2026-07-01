{{- define "module_container_security_context_readonly_fs_flexible" -}}
{{- $uid  := 64535 -}}
{{- if hasKey . "uid" -}}
  {{- $uid = .uid -}}
{{- end -}}
{{- $privileged  := false -}}
{{- if hasKey . "privileged" -}}
  {{- $privileged = .privileged -}}
{{- end -}}
securityContext:
  readOnlyRootFilesystem: true
  allowPrivilegeEscalation: {{ $privileged }}
  privileged: {{ $privileged }}
  capabilities:
    drop:
    - ALL
  runAsUser: {{ $uid }}
  runAsGroup: {{ $uid }}
  runAsNonRoot: {{ ternary false true  ( eq $uid 0 ) }}
  seccompProfile:
    type: RuntimeDefault
{{- end }}
