{{- define "ssi.imagePullSecrets" -}}
{{- if .Values.secretsStoreIntegration.registry }}
  {{- with .Values.secretsStoreIntegration.registry.dockercfg }}
imagePullSecrets:
- name: module-registry
  {{- end }}
{{- else }}
imagePullSecrets:
- name: deckhouse-registry
{{- end }}
{{- end }}
