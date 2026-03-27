{{/*
Common labels
*/}}
{{- define "komputer.labels" -}}
app.kubernetes.io/name: komputer
app.kubernetes.io/instance: {{ .Release.Name }}
app.kubernetes.io/managed-by: {{ .Release.Service }}
helm.sh/chart: {{ .Chart.Name }}-{{ .Chart.Version }}
{{- end }}

{{/*
API internal URL (used by KomputerConfig and manager agents)
*/}}
{{- define "komputer.apiURL" -}}
{{- if .Values.config.apiURL -}}
{{ .Values.config.apiURL }}
{{- else -}}
http://{{ .Release.Name }}-api.{{ .Release.Namespace }}.svc.cluster.local:{{ .Values.api.service.port }}
{{- end -}}
{{- end }}

{{/*
Redis address
*/}}
{{- define "komputer.redisAddress" -}}
{{- if .Values.redis.enabled -}}
{{- if (index .Values "redis-ha" "haproxy" "enabled") -}}
{{ .Release.Name }}-redis-ha-haproxy.{{ .Release.Namespace }}.svc.cluster.local:6379
{{- else -}}
{{ .Release.Name }}-redis-ha-announce-0.{{ .Release.Namespace }}.svc.cluster.local:6379
{{- end -}}
{{- else -}}
{{ .Values.externalRedis.address }}
{{- end -}}
{{- end }}
