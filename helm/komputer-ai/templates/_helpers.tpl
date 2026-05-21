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
{{/*
Agent ServiceAccount name — resolves the four create/name combinations:
  create=true,  name=""       → "<release>-agent"   (chart creates SA with derived name)
  create=true,  name="foo"    → "foo"               (chart creates SA named "foo")
  create=false, name="foo"    → "foo"               (caller provides existing SA)
  create=false, name=""       → ""                  (no override; pod uses default SA)
The empty-string sentinel lets callers conditionally omit `serviceAccountName:` on the podSpec.
*/}}
{{- define "komputer.agent.serviceAccountName" -}}
{{- $sa := .Values.agent.serviceAccount -}}
{{- if $sa.create -}}
{{- default (printf "%s-agent" .Release.Name) $sa.name -}}
{{- else -}}
{{- default "" $sa.name -}}
{{- end -}}
{{- end }}

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
