{{- define "gvList" -}}
{{- $groupVersions := . -}}

---
title: "API Reference"
description: "Generated Octorun API Reference"
lead: ""
date: 2022-03-23T23:27:26+07:00
lastmod: 2022-03-23T23:27:26+07:00
draft: false
images: []
menu:
  docs:
    parent: "reference"
weight: 500
toc: true
---

## Packages
{{- range $groupVersions }}
- {{ markdownRenderGVLink . }}
{{- end }}

{{ range $groupVersions }}
{{ template "gvDetails" . }}
{{ end }}

{{- end -}}
