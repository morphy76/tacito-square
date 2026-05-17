{{- /*
Helper: resolve image registry. Per-component override > global.
*/ -}}
{{- define "tacito-square.imageRegistry" -}}
{{- $registry := .componentRegistry | default .global.imageRegistry | default "localhost:5000" -}}
{{- $registry -}}
{{- end -}}

{{- /*
Helper: full image reference with registry/name:tag.
*/ -}}
{{- define "tacito-square.image" -}}
{{- $registry := include "tacito-square.imageRegistry" (dict "componentRegistry" .registry "global" .global) -}}
{{- printf "%s/%s:%s" $registry .name .tag -}}
{{- end -}}
