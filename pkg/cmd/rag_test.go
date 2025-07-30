package cmd

import (
	"testing"

	"github.com/spf13/cobra"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"k8s.io/cli-runtime/pkg/genericclioptions"
)

func TestNewRagCmd(t *testing.T) {
	configFlags := genericclioptions.NewConfigFlags(true)
	cmd := NewRagCmd(configFlags)

	t.Run("Command structure", func(t *testing.T) {
		assert.Equal(t, "rag", cmd.Use)
		assert.Contains(t, cmd.Short, "RAG")
		assert.NotEmpty(t, cmd.Long)
		assert.NotEmpty(t, cmd.Example)
	})

	t.Run("Subcommands", func(t *testing.T) {
		subcommands := cmd.Commands()
		assert.Len(t, subcommands, 2)

		subcommandNames := make([]string, len(subcommands))
		for i, subcmd := range subcommands {
			subcommandNames[i] = subcmd.Name()
		}

		assert.Contains(t, subcommandNames, "deploy")
		assert.Contains(t, subcommandNames, "query")
	})
}

func TestNewRagDeployCmd(t *testing.T) {
	configFlags := genericclioptions.NewConfigFlags(true)
	ragCmd := NewRagCmd(configFlags)
	
	var deployCmd *cobra.Command
	for _, cmd := range ragCmd.Commands() {
		if cmd.Name() == "deploy" {
			deployCmd = cmd
			break
		}
	}

	require.NotNil(t, deployCmd, "deploy subcommand should exist")

	t.Run("Command structure", func(t *testing.T) {
		assert.Equal(t, "deploy", deployCmd.Use)
		assert.Contains(t, deployCmd.Short, "Deploy a RAG")
		assert.NotEmpty(t, deployCmd.Long)
		assert.NotEmpty(t, deployCmd.Example)
		assert.NotNil(t, deployCmd.RunE)
	})

	t.Run("Required flags", func(t *testing.T) {
		flags := deployCmd.Flags()

		requiredFlags := []string{
			"name",
		}

		for _, flagName := range requiredFlags {
			flag := flags.Lookup(flagName)
			assert.NotNil(t, flag, "Required flag %s should be present", flagName)
		}
	})

	t.Run("Optional flags", func(t *testing.T) {
		flags := deployCmd.Flags()

		optionalFlags := []string{
			"vector-db",
			"index-service",
			"embedding-model",
			"data-source",
			"chunk-size",
			"chunk-overlap",
			"access-mode",
			"access-secret",
			"storage-size",
			"storage-class",
			"dry-run",
		}

		for _, flagName := range optionalFlags {
			flag := flags.Lookup(flagName)
			assert.NotNil(t, flag, "Optional flag %s should be present", flagName)
		}
	})
}

func TestNewRagQueryCmd(t *testing.T) {
	configFlags := genericclioptions.NewConfigFlags(true)
	ragCmd := NewRagCmd(configFlags)
	
	var queryCmd *cobra.Command
	for _, cmd := range ragCmd.Commands() {
		if cmd.Name() == "query" {
			queryCmd = cmd
			break
		}
	}

	require.NotNil(t, queryCmd, "query subcommand should exist")

	t.Run("Command structure", func(t *testing.T) {
		assert.Equal(t, "query", queryCmd.Use)
		assert.Contains(t, queryCmd.Short, "Query a deployed RAG")
		assert.NotEmpty(t, queryCmd.Long)
		assert.NotEmpty(t, queryCmd.Example)
		assert.NotNil(t, queryCmd.RunE)
	})

	t.Run("Required flags", func(t *testing.T) {
		flags := queryCmd.Flags()

		requiredFlags := []string{
			"name",
		}

		for _, flagName := range requiredFlags {
			flag := flags.Lookup(flagName)
			assert.NotNil(t, flag, "Required flag %s should be present", flagName)
		}
	})

	t.Run("Optional flags", func(t *testing.T) {
		flags := queryCmd.Flags()

		optionalFlags := []string{
			"question",
			"interactive",
			"temperature",
		}

		for _, flagName := range optionalFlags {
			flag := flags.Lookup(flagName)
			assert.NotNil(t, flag, "Optional flag %s should be present", flagName)
		}
	})
}

func TestValidateRagDeployOptions(t *testing.T) {
	tests := []struct {
		name         string
		ragName      string
		vectorDB     string
		indexService string
		expectError  bool
		errorMsg     string
	}{
		{
			name:         "Valid options",
			ragName:      "test-rag",
			vectorDB:     "faiss",
			indexService: "llamaindex",
			expectError:  false,
		},
		{
			name:         "Missing name",
			ragName:      "",
			vectorDB:     "faiss",
			indexService: "llamaindex",
			expectError:  true,
			errorMsg:     "RAG engine name is required",
		},
		{
			name:         "Invalid vector database",
			ragName:      "test-rag",
			vectorDB:     "invalid-db",
			indexService: "llamaindex",
			expectError:  true,
			errorMsg:     "invalid vector database",
		},
		{
			name:         "Invalid index service",
			ragName:      "test-rag",
			vectorDB:     "faiss",
			indexService: "invalid-service",
			expectError:  true,
			errorMsg:     "invalid index service",
		},
		{
			name:         "Valid chroma database",
			ragName:      "test-rag",
			vectorDB:     "chroma",
			indexService: "langchain",
			expectError:  false,
		},
		{
			name:         "Valid qdrant database",
			ragName:      "test-rag",
			vectorDB:     "qdrant",
			indexService: "llamaindex",
			expectError:  false,
		},
		{
			name:         "Valid pinecone database",
			ragName:      "test-rag",
			vectorDB:     "pinecone",
			indexService: "langchain",
			expectError:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateRagDeployOptions(tt.ragName, tt.vectorDB, tt.indexService)

			if tt.expectError {
				assert.Error(t, err)
				if tt.errorMsg != "" {
					assert.Contains(t, err.Error(), tt.errorMsg)
				}
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestValidateRagQueryOptions(t *testing.T) {
	tests := []struct {
		name        string
		ragName     string
		question    string
		interactive bool
		expectError bool
		errorMsg    string
	}{
		{
			name:        "Valid non-interactive with question",
			ragName:     "test-rag",
			question:    "What is AI?",
			interactive: false,
			expectError: false,
		},
		{
			name:        "Valid interactive mode",
			ragName:     "test-rag",
			question:    "",
			interactive: true,
			expectError: false,
		},
		{
			name:        "Missing RAG name",
			ragName:     "",
			question:    "What is AI?",
			interactive: false,
			expectError: true,
			errorMsg:    "RAG engine name is required",
		},
		{
			name:        "Non-interactive without question",
			ragName:     "test-rag",
			question:    "",
			interactive: false,
			expectError: true,
			errorMsg:    "question is required in non-interactive mode",
		},
		{
			name:        "Interactive with question (allowed)",
			ragName:     "test-rag",
			question:    "What is AI?",
			interactive: true,
			expectError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateRagQueryOptions(tt.ragName, tt.question, tt.interactive)

			if tt.expectError {
				assert.Error(t, err)
				if tt.errorMsg != "" {
					assert.Contains(t, err.Error(), tt.errorMsg)
				}
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestSimpleBuildRAGEngine(t *testing.T) {
	t.Run("Basic RAG engine creation", func(t *testing.T) {
		ragEngine := buildRAGEngine(
			"test-rag",
			"default",
			"faiss",
			"llamaindex",
			"all-minilm-l6-v2",
			"",
			512,
			50,
			"public",
			"",
			"5Gi",
			"",
		)

		// Check basic structure without complex nested access
		assert.Equal(t, "kaito.sh/v1beta1", ragEngine.GetAPIVersion())
		assert.Equal(t, "RAGEngine", ragEngine.GetKind())
		assert.Equal(t, "test-rag", ragEngine.GetName())
		assert.Equal(t, "default", ragEngine.GetNamespace())
		assert.NotNil(t, ragEngine.Object["spec"])
	})
}

func TestShowRagDeployDryRun(t *testing.T) {
	tests := []struct {
		name         string
		ragName      string
		namespace    string
		vectorDB     string
		indexService string
		contains     []string
	}{
		{
			name:         "Basic dry run",
			ragName:      "test-rag",
			namespace:    "default",
			vectorDB:     "faiss",
			indexService: "llamaindex",
			contains: []string{
				"Dry-run mode",
				"RAG Engine Configuration",
				"Name: test-rag",
				"Namespace: default",
				"Vector Database: faiss",
				"Index Service: llamaindex",
			},
		},
		{
			name:         "Detailed dry run",
			ragName:      "complex-rag",
			namespace:    "production",
			vectorDB:     "chroma",
			indexService: "langchain",
			contains: []string{
				"Name: complex-rag",
				"Namespace: production",
				"Vector Database: chroma",
				"Index Service: langchain",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := showRagDeployDryRun(
				tt.ragName,
				tt.namespace,
				tt.vectorDB,
				tt.indexService,
				"all-minilm-l6-v2",
				"",
				512,
				50,
				"public",
				"5Gi",
				"",
			)
			assert.NoError(t, err)

			// Note: Since this function uses klog.Info, we can't easily capture the output
			// in unit tests. The test mainly verifies that the function doesn't panic
			// and returns no error. The actual output testing is covered in e2e tests.
		})
	}
}

func TestContainsHelper(t *testing.T) {
	tests := []struct {
		name     string
		slice    []string
		item     string
		expected bool
	}{
		{
			name:     "Item found",
			slice:    []string{"apple", "banana", "cherry"},
			item:     "banana",
			expected: true,
		},
		{
			name:     "Item not found",
			slice:    []string{"apple", "banana", "cherry"},
			item:     "grape",
			expected: false,
		},
		{
			name:     "Empty slice",
			slice:    []string{},
			item:     "apple",
			expected: false,
		},
		{
			name:     "Empty item",
			slice:    []string{"apple", "banana", "cherry"},
			item:     "",
			expected: false,
		},
		{
			name:     "Exact match",
			slice:    []string{"test", "TEST", "Test"},
			item:     "test",
			expected: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := contains(tt.slice, tt.item)
			assert.Equal(t, tt.expected, result)
		})
	}
}

