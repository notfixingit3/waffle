package main

import "embed"

//go:embed static/css/output.css static/js/sw.js static/js/offline-handler.js static/js/spot-selection.js static/js/websocket-client.js static/js/reports.js static/js/admin-spot-actions.js static/img/* static/offline.html static/manifest.json static/favicon.svg
var staticFS embed.FS
