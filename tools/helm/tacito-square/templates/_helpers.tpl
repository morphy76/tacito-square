{{/*
Expand the name of the chart.
*/}}
{{- define "tacito-square.name" -}}
{{- default .Chart.Name .Values.nameOverride | trunc 63 | trimSuffix "-" }}
{{- end }}

{{/*
Create a default fully qualified app name.
*/}}
{{- define "tacito-square.fullname" -}}
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
Resolve image registry: per-component override > global.
*/}}
{{- define "tacito-square.imageRegistry" -}}
{{- $registry := .componentRegistry | default .global.imageRegistry | default "" -}}
{{- $registry -}}
{{- end -}}

{{/*
Full image reference: registry/name:tag.
*/}}
{{- define "tacito-square.image" -}}
{{- $registry := include "tacito-square.imageRegistry" (dict "componentRegistry" .registry "global" .global) -}}
{{- if $registry -}}
{{- printf "%s/%s:%s" $registry .name .tag -}}
{{- else -}}
{{- printf "%s:%s" .name .tag -}}
{{- end -}}
{{- end -}}

{{/*
Common labels for a component.
Usage: include "tacito-square.labels" (dict "component" "keeper" "context" .)
*/}}
{{- define "tacito-square.labels" -}}
helm.sh/chart: {{ include "tacito-square.name" .context }}
app.kubernetes.io/name: {{ include "tacito-square.name" .context }}
app.kubernetes.io/instance: {{ .context.Release.Name }}
app.kubernetes.io/component: {{ .component }}
app.kubernetes.io/managed-by: {{ .context.Release.Service }}
app.kubernetes.io/part-of: tacito-square
{{- end }}

{{/*
Selector labels for a component.
Usage: include "tacito-square.selectorLabels" (dict "component" "keeper" "context" .)
*/}}
{{- define "tacito-square.selectorLabels" -}}
app.kubernetes.io/name: {{ include "tacito-square.name" .context }}
app.kubernetes.io/instance: {{ .context.Release.Name }}
app.kubernetes.io/component: {{ .component }}
{{- end }}
