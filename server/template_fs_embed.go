//go:build embed_templates
// +build embed_templates

package main

import "embed"

//go:embed templates/* static/*
var templateFS embed.FS
