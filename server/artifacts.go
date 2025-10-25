package main

import (
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"
	"strings"
)

var artifactStorePath string

func initArtifactStore(uri string) {
	if strings.HasPrefix(uri, "file://") {
		artifactStorePath = strings.TrimPrefix(uri, "file://")
	} else {
		log.Fatalf("Invalid artifacts store URI format. Expected file:///path/to/store, got: %s", uri)
	}

	err := os.MkdirAll(artifactStorePath, os.ModePerm)
	if err != nil {
		log.Fatalf("Could not create artifact store: %v", err)
	}

	log.Printf("Artifact store initialized at: %s", artifactStorePath)
}

// storeArtifact saves a file to the artifact store and returns its URI
func storeArtifact(runUUID string, artifactPath string, fileData io.Reader) (string, error) {
	// Create directory structure: {artifactStorePath}/{runUUID}/{dir-of-artifactPath}
	fullDir := filepath.Join(artifactStorePath, runUUID, filepath.Dir(artifactPath))
	err := os.MkdirAll(fullDir, os.ModePerm)
	if err != nil {
		return "", fmt.Errorf("failed to create artifact directory: %v", err)
	}

	// Full file path
	fullPath := filepath.Join(artifactStorePath, runUUID, artifactPath)

	// Create file
	file, err := os.Create(fullPath)
	if err != nil {
		return "", fmt.Errorf("failed to create artifact file: %v", err)
	}
	defer file.Close()

	// Copy data to file
	_, err = io.Copy(file, fileData)
	if err != nil {
		return "", fmt.Errorf("failed to write artifact data: %v", err)
	}

	// Return URI in the format file:///path
	uri := fmt.Sprintf("file://%s", fullPath)
	return uri, nil
}
