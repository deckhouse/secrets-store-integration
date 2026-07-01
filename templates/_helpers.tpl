{{- define "ssi.imagePullSecrets" -}}
{{- $secretName := include "ssi.registrySecretName" . | trim -}}
imagePullSecrets:
- name: {{ $secretName }}
{{- end }}
{{- define "ssi.registrySecretName" -}}
{{- $moduleDockercfg := dig "secretsStoreIntegration" "registry" "dockercfg" "" .Values -}}
{{- $deckhouseDockercfg := dig "global" "modulesImages" "registry" "dockercfg" "" .Values -}}
{{- if $moduleDockercfg }}
module-registry
{{- else if $deckhouseDockercfg }}
deckhouse-registry
{{- else }}
{{- fail "neither global nor module registry configuration is set, cannot render imagePullSecrets" }}
{{- end }}
{{- end }}
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
