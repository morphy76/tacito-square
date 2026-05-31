{{/*
Expand the name of the chart.
*/}}
{{- define "tacito-agent.name" -}}
{{- default .Chart.Name .Values.nameOverride | trunc 63 | trimSuffix "-" }}
{{- end }}

{{/*
Create a default fully qualified app name.
*/}}
{{- define "tacito-agent.fullname" -}}
{{- if .Values.fullnameOverride }}
{{- .Values.fullnameOverride | trunc 63 | trimSuffix "-" }}
{{- else }}
{{- $name := default .Chart.Name .Values.nameOverride }}
{{- if contains $name .Release.Name }}
{{- .Release.Name | trunc 63 | trimSuffix "-" }}
{{- else }}
{{- printf "%s-%s" .Release.Name $name | trunc 63 | trimSuffix "-" }}
{{- end }}
{{- end }}
{{- end }}

{{/*
Common labels.
*/}}
{{- define "tacito-agent.labels" -}}
helm.sh/chart: {{ include "tacito-agent.name" . }}
app.kubernetes.io/name: {{ include "tacito-agent.name" . }}
app.kubernetes.io/instance: {{ .Release.Name }}
app.kubernetes.io/managed-by: {{ .Release.Service }}
app.kubernetes.io/part-of: tacito-square
{{- end }}

{{/*
Selector labels.
*/}}
{{- define "tacito-agent.selectorLabels" -}}
app.kubernetes.io/name: {{ include "tacito-agent.name" . }}
app.kubernetes.io/instance: {{ .Release.Name }}
{{- end }}
