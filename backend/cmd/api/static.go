package main

import "embed"

//go:embed static/css/output.css static/js/* static/img/* static/manifest.json static/favicon.svg
var staticFS embed.FS
