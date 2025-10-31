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
