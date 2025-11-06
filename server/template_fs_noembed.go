//go:build !embed_templates
// +build !embed_templates

package main

import "os"

var templateFS = os.DirFS(".")
