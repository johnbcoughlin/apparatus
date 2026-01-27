package main

import (
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
)

func TestHandleServeArtifactBlob(t *testing.T) {
	// Create a temporary directory for the artifact store
	tempDir, err := os.MkdirTemp("", "artifact-store-test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// Set the global artifact store path
	artifactStorePath = tempDir

	// Create a test file in the artifact store
	testContent := []byte("test artifact content")
	testFilePath := filepath.Join(tempDir, "run123", "artifact.txt")
	err = os.MkdirAll(filepath.Dir(testFilePath), 0755)
	if err != nil {
		t.Fatalf("Failed to create test dir: %v", err)
	}
	err = os.WriteFile(testFilePath, testContent, 0644)
	if err != nil {
		t.Fatalf("Failed to write test file: %v", err)
	}

	tests := []struct {
		name           string
		path           string
		expectedStatus int
		expectContent  bool
	}{
		{
			name:           "valid relative path in artifact store",
			path:           "file://run123/artifact.txt",
			expectedStatus: http.StatusOK,
			expectContent:  true,
		},
		{
			name:           "path traversal attempt with ..",
			path:           "file://run123/../../../etc/passwd",
			expectedStatus: http.StatusForbidden,
			expectContent:  false,
		},
		{
			name:           "absolute path rejected",
			path:           "file:///etc/passwd",
			expectedStatus: http.StatusForbidden,
			expectContent:  false,
		},
		{
			name:           "path traversal at start",
			path:           "file://../etc/passwd",
			expectedStatus: http.StatusForbidden,
			expectContent:  false,
		},
		{
			name:           "path traversal at end",
			path:           "file://run123/../../../..",
			expectedStatus: http.StatusForbidden,
			expectContent:  false,
		},
		{
			name:           "missing file:// prefix",
			path:           "run123/artifact.txt",
			expectedStatus: http.StatusBadRequest,
			expectContent:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest("GET", "/artifacts/blob?uri="+tt.path, nil)
			w := httptest.NewRecorder()

			handleServeArtifactBlob(w, req)

			if w.Code != tt.expectedStatus {
				t.Errorf("expected status %d, got %d", tt.expectedStatus, w.Code)
			}

			if tt.expectContent && w.Body.String() != string(testContent) {
				t.Errorf("expected content %q, got %q", string(testContent), w.Body.String())
			}
		})
	}
}
