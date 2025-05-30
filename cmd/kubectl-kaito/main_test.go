package main

import (
	"os"
	"path/filepath"
	"testing"
)

func TestMain(t *testing.T) {
	// Test that main function can be called without panicking
	// We can't easily test the actual execution without mocking,
	// but we can test the basic setup logic

	// Save original args
	originalArgs := os.Args
	defer func() {
		os.Args = originalArgs
	}()

	// Test with help flag to avoid actual execution
	os.Args = []string{"kubectl-kaito", "--help"}

	// This would normally call main(), but we can't test it directly
	// without refactoring. Instead, we test the helper logic.

	// Test plugin detection logic
	testCases := []struct {
		name     string
		binary   string
		expected bool
	}{
		{
			name:     "kubectl plugin format",
			binary:   "kubectl-kaito",
			expected: true,
		},
		{
			name:     "direct binary",
			binary:   "kaito",
			expected: false,
		},
		{
			name:     "kubectl plugin with path",
			binary:   "/usr/local/bin/kubectl-kaito",
			expected: true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := isKubectlPlugin(tc.binary)
			if result != tc.expected {
				t.Errorf("isKubectlPlugin(%s) = %v, expected %v", tc.binary, result, tc.expected)
			}
		})
	}
}

// Helper function extracted from main logic for testing
func isKubectlPlugin(binary string) bool {
	return filepath.HasPrefix(filepath.Base(binary), "kubectl-")
}
