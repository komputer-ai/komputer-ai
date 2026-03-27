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
{{/*
Redis address — full service endpoint for KomputerConfig and API.
- redis.enabled=true: simple single-node Redis deployed by this chart
- redis-ha.enabled=true: HA Redis via dandydeveloper/redis-ha subchart
- both false: external Redis from externalRedis.address
*/}}
{{- define "komputer.redisAddress" -}}
{{- $redisHa := (index .Values "redis-ha") -}}
{{- if .Values.redis.enabled -}}
{{ .Release.Name }}-redis.{{ .Release.Namespace }}.svc.cluster.local:6379
{{- else if $redisHa.enabled -}}
{{- $redisHaContext := dict "Chart" (dict "Name" "redis-ha") "Release" .Release "Values" $redisHa -}}
{{- if $redisHa.haproxy.enabled -}}
{{ printf "%s-haproxy" (include "redis-ha.fullname" $redisHaContext) }}.{{ .Release.Namespace }}.svc.cluster.local:6379
{{- else -}}
{{ include "redis-ha.fullname" $redisHaContext }}.{{ .Release.Namespace }}.svc.cluster.local:6379
{{- end -}}
{{- else -}}
{{ .Values.externalRedis.address }}
{{- end -}}
{{- end }}
