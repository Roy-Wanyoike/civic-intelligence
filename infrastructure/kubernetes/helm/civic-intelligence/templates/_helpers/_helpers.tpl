{{/*
Common helpers for the civic-intelligence Helm chart.
*/}}

{{- define "civic.name" -}}
{{- default .Chart.Name .Values.nameOverride | trunc 63 | trimSuffix "-" -}}
{{- end -}}

{{- define "civic.fullname" -}}
{{- $name := include "civic.name" . -}}
{{- printf "%s-%s" .Release.Name $name | trunc 63 | trimSuffix "-" -}}
{{- end -}}

{{- define "civic.chart" -}}
{{- printf "%s-%s" .Chart.Name .Chart.Version | replace "+" "_" | trunc 63 | trimSuffix "-" -}}
{{- end -}}

{{/*
Standard labels applied to every resource.
*/}}
{{- define "civic.labels" -}}
helm.sh/chart: {{ include "civic.chart" . }}
app.kubernetes.io/managed-by: {{ .Release.Service }}
app.kubernetes.io/instance: {{ .Release.Name }}
app.kubernetes.io/part-of: civic-intelligence
{{- end -}}

{{/*
Per-service labels — call with (dict "root" $ "service" $svc "name" $name).
*/}}
{{- define "civic.serviceLabels" -}}
{{- $root := .root -}}
{{- $svc  := .svc -}}
{{- $name := .name -}}
{{ include "civic.labels" $root }}
app.kubernetes.io/name: {{ $name }}
app.kubernetes.io/component: {{ $name }}
app.kubernetes.io/version: {{ $svc.image | default "latest" | toString | trunc 63 }}
{{- end -}}

{{/*
Resolve a service image ref: <registry>/<image>:<tag>.
*/}}
{{- define "civic.image" -}}
{{- $svc := .svc -}}
{{- $root := .root -}}
{{- $registry := $root.Values.global.imageRegistry -}}
{{- $tag := $root.Values.global.imageTag | default "latest" -}}
{{- printf "%s/%s:%s" $registry $svc.image $tag -}}
{{- end -}}

{{/*
Resolve a backing-service DNS name: <release>-<name>.<namespace>.svc.cluster.local
Used by NetworkPolicy toService entries.
*/}}
{{- define "civic.serviceDNS" -}}
{{- $root := .root -}}
{{- $svc := .svc -}}
{{- printf "%s-%s.%s.svc.cluster.local" $root.Release.Name $svc $root.Release.Namespace -}}
{{- end -}}
