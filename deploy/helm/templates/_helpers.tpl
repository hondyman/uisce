{{/*
Uisce Semantic OS — Helper Templates
*/}}

{{/*
Expand the uisce full name, with optional suffix
*/}}
{{- define "uisce.fullname" -}}
{{- $name := default .Chart.Name .Values.nameOverride -}}
{{- if contains $name .Release.Name -}}
{{- .Release.Name | trunc 63 | trimSuffix "-" -}}
{{- else -}}
{{- printf "%s-%s" .Release.Name $name | trunc 63 | trimSuffix "-" -}}
{{- end -}}
{{- end -}}

{{/*
Uisce labels
*/}}
{{- define "uisce.labels" -}}
app.kubernetes.io/name: {{ include "uisce.fullname" . }}
app.kubernetes.io/instance: {{ .Release.Name }}
app.kubernetes.io/version: {{ .Chart.AppVersion | quote }}
app.kubernetes.io/part-of: uisce
app.kubernetes.io/managed-by: {{ .Release.Service }}
    helm.sh/chart: {{ printf "%s-%s" .Chart.Name .Chart.Version }}
{{- end -}}

{{/*
Raft peer FQDN for a given node
*/}}
{{- define "uisce.raftPeer" -}}
{{- printf "%s.%s.svc.cluster.local:7946" .nodeID .Release.Namespace -}}
{{- end -}}
