{{/*
Expand the name of the chart.
*/}}
{{- define "telemetry-gen-and-send.name" -}}
{{- default .Chart.Name .Values.nameOverride | trunc 63 | trimSuffix "-" }}
{{- end }}

{{/*
Create a default fully qualified app name.
*/}}
{{- define "telemetry-gen-and-send.fullname" -}}
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
Create chart name and version as used by the chart label.
*/}}
{{- define "telemetry-gen-and-send.chart" -}}
{{- printf "%s-%s" .Chart.Name .Chart.Version | replace "+" "_" | trunc 63 | trimSuffix "-" }}
{{- end }}

{{/*
Common labels
*/}}
{{- define "telemetry-gen-and-send.labels" -}}
helm.sh/chart: {{ include "telemetry-gen-and-send.chart" . }}
{{ include "telemetry-gen-and-send.selectorLabels" . }}
{{- if .Chart.AppVersion }}
app.kubernetes.io/version: {{ .Chart.AppVersion | quote }}
{{- end }}
app.kubernetes.io/managed-by: {{ .Release.Service }}
{{- end }}

{{/*
Selector labels
*/}}
{{- define "telemetry-gen-and-send.selectorLabels" -}}
app.kubernetes.io/name: {{ include "telemetry-gen-and-send.name" . }}
app.kubernetes.io/instance: {{ .Release.Name }}
{{- end }}

{{/*
Create the name of the service account to use
*/}}
{{- define "telemetry-gen-and-send.serviceAccountName" -}}
{{- if .Values.serviceAccount.create }}
{{- default (include "telemetry-gen-and-send.fullname" .) .Values.serviceAccount.name }}
{{- else }}
{{- default "default" .Values.serviceAccount.name }}
{{- end }}
{{- end }}

{{/*
Get the secret name for Honeycomb API key
*/}}
{{- define "telemetry-gen-and-send.secretName" -}}
{{- if .Values.honeycomb.existingSecret }}
{{- .Values.honeycomb.existingSecret }}
{{- else }}
{{- include "telemetry-gen-and-send.fullname" . }}
{{- end }}
{{- end }}

{{/*
Container image
*/}}
{{- define "telemetry-gen-and-send.image" -}}
{{- $tag := .Values.image.tag | default .Chart.AppVersion }}
{{- printf "%s:%s" .Values.image.repository $tag }}
{{- end }}
