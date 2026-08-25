package httpui

import "embed"

//go:embed web/index.html web/app.css web/workflow.css web/app.js
var assets embed.FS
