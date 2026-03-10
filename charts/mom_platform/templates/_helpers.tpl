{{/*
Expand the name of the chart.
*/}}
{{- define "mom.name" -}}
{{- default .Chart.Name .Values.nameOverride | trunc 63 | trimSuffix "-" }}
{{- end }}

{{/*
Create a default fully qualified app name.
*/}}
{{- define "mom.fullname" -}}
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
{{- define "mom.chart" -}}
{{- printf "%s-%s" .Chart.Name .Chart.Version | replace "+" "_" | trunc 63 | trimSuffix "-" }}
{{- end }}

{{/*
Common labels
*/}}
{{- define "mom.labels" -}}
helm.sh/chart: {{ include "mom.chart" . }}
{{ include "mom.selectorLabels" . }}
{{- if .Chart.AppVersion }}
app.kubernetes.io/version: {{ .Chart.AppVersion | quote }}
{{- end }}
app.kubernetes.io/managed-by: {{ .Release.Service }}
{{- end }}

{{/*
Selector labels
*/}}
{{- define "mom.selectorLabels" -}}
app.kubernetes.io/name: {{ include "mom.name" . }}
app.kubernetes.io/instance: {{ .Release.Name }}
{{- end }}

{{/*
Backend labels
*/}}
{{- define "mom.backend.labels" -}}
{{ include "mom.labels" . }}
app.kubernetes.io/component: backend
{{- end }}

{{- define "mom.backend.selectorLabels" -}}
{{ include "mom.selectorLabels" . }}
app.kubernetes.io/component: backend
{{- end }}

{{/*
Frontend labels
*/}}
{{- define "mom.frontend.labels" -}}
{{ include "mom.labels" . }}
app.kubernetes.io/component: frontend
{{- end }}

{{- define "mom.frontend.selectorLabels" -}}
{{ include "mom.selectorLabels" . }}
app.kubernetes.io/component: frontend
{{- end }}

{{/*
MySQL labels
*/}}
{{- define "mom.mysql.labels" -}}
{{ include "mom.labels" . }}
app.kubernetes.io/component: mysql
{{- end }}

{{- define "mom.mysql.selectorLabels" -}}
{{ include "mom.selectorLabels" . }}
app.kubernetes.io/component: mysql
{{- end }}

{{/*
Redis labels
*/}}
{{- define "mom.redis.labels" -}}
{{ include "mom.labels" . }}
app.kubernetes.io/component: redis
{{- end }}

{{- define "mom.redis.selectorLabels" -}}
{{ include "mom.selectorLabels" . }}
app.kubernetes.io/component: redis
{{- end }}

{{/*
MySQL host
*/}}
{{- define "mom.mysql.host" -}}
{{- if .Values.mysql.enabled }}
{{- printf "%s-mysql" (include "mom.fullname" .) }}
{{- else }}
{{- .Values.externalDatabase.host }}
{{- end }}
{{- end }}

{{/*
MySQL port
*/}}
{{- define "mom.mysql.port" -}}
{{- if .Values.mysql.enabled }}
{{- printf "3306" }}
{{- else }}
{{- .Values.externalDatabase.port | toString }}
{{- end }}
{{- end }}

{{/*
MySQL database
*/}}
{{- define "mom.mysql.database" -}}
{{- if .Values.mysql.enabled }}
{{- .Values.mysql.auth.database }}
{{- else }}
{{- .Values.externalDatabase.database }}
{{- end }}
{{- end }}

{{/*
Redis host
*/}}
{{- define "mom.redis.host" -}}
{{- if .Values.redis.enabled }}
{{- printf "%s-redis" (include "mom.fullname" .) }}
{{- else }}
{{- .Values.externalRedis.host }}
{{- end }}
{{- end }}

{{/*
Redis port
*/}}
{{- define "mom.redis.port" -}}
{{- if .Values.redis.enabled }}
{{- printf "6379" }}
{{- else }}
{{- .Values.externalRedis.port | toString }}
{{- end }}
{{- end }}

{{/*
Create the name of the service account to use
*/}}
{{- define "mom.serviceAccountName" -}}
{{- if .Values.serviceAccount.create }}
{{- default (include "mom.fullname" .) .Values.serviceAccount.name }}
{{- else }}
{{- default "default" .Values.serviceAccount.name }}
{{- end }}
{{- end }}

{{/*
Return the proper image name
*/}}
{{- define "mom.image" -}}
{{- $registryName := .imageRoot.registry -}}
{{- $repositoryName := .imageRoot.repository -}}
{{- $tag := .imageRoot.tag | toString -}}
{{- if $registryName }}
{{- printf "%s/%s:%s" $registryName $repositoryName $tag -}}
{{- else }}
{{- printf "%s:%s" $repositoryName $tag -}}
{{- end }}
{{- end }}
