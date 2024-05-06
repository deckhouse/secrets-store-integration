{{/*
Expand the name of the chart.
*/}}
{{- define "ssi.name" -}}
{{- default .Chart.Name .Values.secretsStoreIntegration.nameOverride | trunc 63 | trimSuffix "-" }}
{{- end }}


{{/*
Namespace used by all resources.
*/}}
{{- define "ssi.namespace" -}}
d8-secrets-store-integration
{{- end }}

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
