package main

import (
	"testing"
)

func TestAssembleArtifactsTree(t *testing.T) {
	artifacts := []Artifact{
		{"file1.txt", "abc1", "text"},
		{"plots/1.png", "abc2", "image"},
		{"plots/2.png", "abc3", "image"},
		{"plots/barcharts/G.png", "abc4", "image"},
		{"plots/barcharts/H.png", "abc5", "image"},
	}
	result := assembleArtifactsTree("foo-uuid", artifacts)
	if *(*result.Children["file1.txt"]).ArtifactURI != "abc1" {
		t.Error("Failure 1")
	}
	if *(*result.Children["plots"].Children["1.png"]).ArtifactURI != "abc2" {
		t.Error("Failure 2")
	}
	if *(*result.Children["plots"].Children["2.png"]).ArtifactURI != "abc3" {
		t.Error("Failure 3")
	}
	if *(*result.Children["plots"].Children["barcharts"].Children["G.png"]).ArtifactURI != "abc4" {
		t.Error("Failure 4")
	}
	if *(*result.Children["plots"].Children["barcharts"].Children["H.png"]).ArtifactURI != "abc5" {
		t.Error("Failure 5")
	}

	if *(*result.Children["plots"].Children["1.png"]).RunUUID != "foo-uuid" {
		t.Error("Failure  ")
	}
}

func TestIsValidArtifactPath(t *testing.T) {
	tests := []struct {
		name    string
		path    string
		wantErr bool
	}{
		// Valid paths
		{"simple file", "artifact.txt", false},
		{"nested path", "subdir/artifact.txt", false},
		{"with hyphens", "my-artifact.txt", false},
		{"with underscores", "my_artifact.txt", false},
		{"with dots", "file.tar.gz", false},
		{"alphanumeric", "run123/data456.json", false},

		// Invalid: path traversal
		{"path traversal dotdot", "../etc/passwd", true},
		{"path traversal middle", "foo/../bar", true},
		{"path traversal end", "foo/..", true},

		// Invalid: absolute paths
		{"absolute path", "/etc/passwd", true},

		// Invalid: empty
		{"empty path", "", true},

		// Invalid: XSS/injection characters
		{"html brackets", "foo<script>.txt", true},
		{"double quotes", "foo\"bar.txt", true},
		{"single quotes", "foo'bar.txt", true},
		{"ampersand", "foo&bar.txt", true},
		{"backslash", "foo\\bar.txt", true},
		{"space", "foo bar.txt", true},
		{"newline", "foo\nbar.txt", true},
		{"null byte", "foo\x00bar.txt", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := isValidArtifactPath(tt.path)
			if (err != nil) != tt.wantErr {
				t.Errorf("isValidArtifactPath(%q) error = %v, wantErr %v", tt.path, err, tt.wantErr)
			}
		})
	}
}
