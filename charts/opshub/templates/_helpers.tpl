{{/*
Expand the name of the chart.
*/}}
{{- define "iom.name" -}}
{{- default .Chart.Name .Values.nameOverride | trunc 63 | trimSuffix "-" }}
{{- end }}

{{/*
Create a default fully qualified app name.
*/}}
{{- define "iom.fullname" -}}
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
{{- define "iom.chart" -}}
{{- printf "%s-%s" .Chart.Name .Chart.Version | replace "+" "_" | trunc 63 | trimSuffix "-" }}
{{- end }}

{{/*
Common labels
*/}}
{{- define "iom.labels" -}}
helm.sh/chart: {{ include "iom.chart" . }}
{{ include "iom.selectorLabels" . }}
{{- if .Chart.AppVersion }}
app.kubernetes.io/version: {{ .Chart.AppVersion | quote }}
{{- end }}
app.kubernetes.io/managed-by: {{ .Release.Service }}
{{- end }}

{{/*
Selector labels
*/}}
{{- define "iom.selectorLabels" -}}
app.kubernetes.io/name: {{ include "iom.name" . }}
app.kubernetes.io/instance: {{ .Release.Name }}
{{- end }}

{{/*
Backend labels
*/}}
{{- define "iom.backend.labels" -}}
{{ include "iom.labels" . }}
app.kubernetes.io/component: backend
{{- end }}

{{- define "iom.backend.selectorLabels" -}}
{{ include "iom.selectorLabels" . }}
app.kubernetes.io/component: backend
{{- end }}

{{/*
Frontend labels
*/}}
{{- define "iom.frontend.labels" -}}
{{ include "iom.labels" . }}
app.kubernetes.io/component: frontend
{{- end }}

{{- define "iom.frontend.selectorLabels" -}}
{{ include "iom.selectorLabels" . }}
app.kubernetes.io/component: frontend
{{- end }}

{{/*
MySQL labels
*/}}
{{- define "iom.mysql.labels" -}}
{{ include "iom.labels" . }}
app.kubernetes.io/component: mysql
{{- end }}

{{- define "iom.mysql.selectorLabels" -}}
{{ include "iom.selectorLabels" . }}
app.kubernetes.io/component: mysql
{{- end }}

{{/*
Redis labels
*/}}
{{- define "iom.redis.labels" -}}
{{ include "iom.labels" . }}
app.kubernetes.io/component: redis
{{- end }}

{{- define "iom.redis.selectorLabels" -}}
{{ include "iom.selectorLabels" . }}
app.kubernetes.io/component: redis
{{- end }}

{{/*
MySQL host
*/}}
{{- define "iom.mysql.host" -}}
{{- if .Values.mysql.enabled }}
{{- printf "%s-mysql" (include "iom.fullname" .) }}
{{- else }}
{{- .Values.externalDatabase.host }}
{{- end }}
{{- end }}

{{/*
MySQL port
*/}}
{{- define "iom.mysql.port" -}}
{{- if .Values.mysql.enabled }}
{{- printf "3306" }}
{{- else }}
{{- .Values.externalDatabase.port | toString }}
{{- end }}
{{- end }}

{{/*
MySQL database
*/}}
{{- define "iom.mysql.database" -}}
{{- if .Values.mysql.enabled }}
{{- .Values.mysql.auth.database }}
{{- else }}
{{- .Values.externalDatabase.database }}
{{- end }}
{{- end }}

{{/*
Redis host
*/}}
{{- define "iom.redis.host" -}}
{{- if .Values.redis.enabled }}
{{- printf "%s-redis" (include "iom.fullname" .) }}
{{- else }}
{{- .Values.externalRedis.host }}
{{- end }}
{{- end }}

{{/*
Redis port
*/}}
{{- define "iom.redis.port" -}}
{{- if .Values.redis.enabled }}
{{- printf "6379" }}
{{- else }}
{{- .Values.externalRedis.port | toString }}
{{- end }}
{{- end }}

{{/*
Create the name of the service account to use
*/}}
{{- define "iom.serviceAccountName" -}}
{{- if .Values.serviceAccount.create }}
{{- default (include "iom.fullname" .) .Values.serviceAccount.name }}
{{- else }}
{{- default "default" .Values.serviceAccount.name }}
{{- end }}
{{- end }}

{{/*
Return the proper image name
*/}}
{{- define "iom.image" -}}
{{- $registryName := .imageRoot.registry -}}
{{- $repositoryName := .imageRoot.repository -}}
{{- $tag := .imageRoot.tag | toString -}}
{{- if $registryName }}
{{- printf "%s/%s:%s" $registryName $repositoryName $tag -}}
{{- else }}
{{- printf "%s:%s" $repositoryName $tag -}}
{{- end }}
{{- end }}
