/*
Copyright (c) 2024 Kaito Project

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package cmd

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"

	"github.com/spf13/cobra"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/cli-runtime/pkg/genericclioptions"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/kubernetes"
	"k8s.io/klog/v2"
)

// NewRagCmd creates the rag command with subcommands
func NewRagCmd(configFlags *genericclioptions.ConfigFlags) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "rag",
		Short: "Manage RAG (Retrieval Augmented Generation) engines",
		Long: `Deploy and query RAG engines for enhanced AI capabilities.

RAG engines combine retrieval and generation to provide more accurate and context-aware
responses by retrieving relevant information from knowledge bases.`,
		Example: `  # Deploy a RAG engine
  kubectl kaito rag deploy --name my-rag --vector-db faiss --index-service llamaindex

  # Query a deployed RAG engine
  kubectl kaito rag query --name my-rag --question "What is machine learning?"

  # Deploy RAG with custom parameters
  kubectl kaito rag deploy --name my-rag --vector-db chroma --embedding-model sentence-transformers/all-MiniLM-L6-v2`,
		RunE: func(cmd *cobra.Command, args []string) error {
			klog.Info("Use 'kubectl kaito rag deploy' or 'kubectl kaito rag query' for RAG operations")
			return cmd.Help()
		},
	}

	// Add subcommands
	cmd.AddCommand(newRagDeployCmd(configFlags))
	cmd.AddCommand(newRagQueryCmd(configFlags))

	return cmd
}

// RAG Deploy Command
func newRagDeployCmd(configFlags *genericclioptions.ConfigFlags) *cobra.Command {
	var (
		ragName        string
		namespace      string
		vectorDB       string
		indexService   string
		embeddingModel string
		dataSource     string
		chunkSize      int
		chunkOverlap   int
		accessMode     string
		accessSecret   string
		storageSize    string
		storageClass   string
		dryRun         bool
	)

	cmd := &cobra.Command{
		Use:   "deploy",
		Short: "Deploy a RAG engine",
		Long: `Deploy a RAG (Retrieval Augmented Generation) engine.

This creates a RAGEngine resource that sets up the vector database, indexing service,
and necessary components for document retrieval and generation.`,
		Example: `  # Deploy basic RAG engine
  kubectl kaito rag deploy --name my-rag --vector-db faiss --index-service llamaindex

  # Deploy with custom embedding model
  kubectl kaito rag deploy --name my-rag --vector-db chroma --embedding-model sentence-transformers/all-MiniLM-L6-v2

  # Deploy with persistent storage
  kubectl kaito rag deploy --name my-rag --vector-db qdrant --storage-size 10Gi --storage-class fast-ssd

  # Deploy with data source
  kubectl kaito rag deploy --name my-rag --vector-db faiss --data-source "s3://my-bucket/documents/"`,
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := validateRagDeployOptions(ragName, vectorDB, indexService); err != nil {
				klog.Errorf("Validation failed: %v", err)
				return fmt.Errorf("validation failed: %w", err)
			}
			return runRagDeploy(configFlags, ragName, namespace, vectorDB, indexService,
				embeddingModel, dataSource, chunkSize, chunkOverlap, accessMode, accessSecret,
				storageSize, storageClass, dryRun)
		},
	}

	cmd.Flags().StringVar(&ragName, "name", "", "Name of the RAG engine (required)")
	cmd.Flags().StringVarP(&namespace, "namespace", "n", "", "Kubernetes namespace")
	cmd.Flags().StringVar(&vectorDB, "vector-db", "faiss", "Vector database type (faiss, chroma, qdrant, pinecone)")
	cmd.Flags().StringVar(&indexService, "index-service", "llamaindex", "Indexing service (llamaindex, langchain)")
	cmd.Flags().StringVar(&embeddingModel, "embedding-model", "all-minilm-l6-v2", "Embedding model for text vectorization")
	cmd.Flags().StringVar(&dataSource, "data-source", "", "Data source URI (s3://, gs://, etc.)")
	cmd.Flags().IntVar(&chunkSize, "chunk-size", 512, "Document chunk size")
	cmd.Flags().IntVar(&chunkOverlap, "chunk-overlap", 50, "Chunk overlap size")
	cmd.Flags().StringVar(&accessMode, "access-mode", "public", "Access mode (public, private)")
	cmd.Flags().StringVar(&accessSecret, "access-secret", "", "Secret for private access")
	cmd.Flags().StringVar(&storageSize, "storage-size", "5Gi", "Persistent storage size")
	cmd.Flags().StringVar(&storageClass, "storage-class", "", "Storage class for persistent volumes")
	cmd.Flags().BoolVar(&dryRun, "dry-run", false, "Show what would be created without actually creating")

	if err := cmd.MarkFlagRequired("name"); err != nil {
		klog.Errorf("Failed to mark name flag as required: %v", err)
	}

	return cmd
}

// RAG Query Command
func newRagQueryCmd(configFlags *genericclioptions.ConfigFlags) *cobra.Command {
	var (
		ragName     string
		namespace   string
		question    string
		topK        int
		temperature float64
		format      string
		interactive bool
	)

	cmd := &cobra.Command{
		Use:   "query",
		Short: "Query a deployed RAG engine",
		Long: `Query a deployed RAG engine with questions.

This command sends queries to the RAG engine and returns augmented responses
based on the indexed knowledge base.`,
		Example: `  # Ask a question
  kubectl kaito rag query --name my-rag --question "What is machine learning?"

  # Interactive query mode
  kubectl kaito rag query --name my-rag --interactive

  # Query with custom parameters
  kubectl kaito rag query --name my-rag --question "Explain neural networks" --top-k 5 --temperature 0.3

  # JSON output format
  kubectl kaito rag query --name my-rag --question "What is AI?" --format json`,
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := validateRagQueryOptions(ragName, question, interactive); err != nil {
				klog.Errorf("Validation failed: %v", err)
				return fmt.Errorf("validation failed: %w", err)
			}
			return runRagQuery(configFlags, ragName, namespace, question, topK, temperature, format, interactive)
		},
	}

	cmd.Flags().StringVar(&ragName, "name", "", "Name of the RAG engine (required)")
	cmd.Flags().StringVarP(&namespace, "namespace", "n", "", "Kubernetes namespace")
	cmd.Flags().StringVarP(&question, "question", "q", "", "Question to ask the RAG engine")
	cmd.Flags().IntVar(&topK, "top-k", 3, "Number of top documents to retrieve")
	cmd.Flags().Float64Var(&temperature, "temperature", 0.7, "Temperature for generation")
	cmd.Flags().StringVar(&format, "format", "text", "Output format (text, json)")
	cmd.Flags().BoolVarP(&interactive, "interactive", "i", false, "Interactive query mode")

	if err := cmd.MarkFlagRequired("name"); err != nil {
		klog.Errorf("Failed to mark name flag as required: %v", err)
	}

	return cmd
}

func validateRagDeployOptions(ragName, vectorDB, indexService string) error {
	klog.V(4).Info("Validating RAG deploy options")

	if ragName == "" {
		return fmt.Errorf("RAG engine name is required")
	}

	validVectorDBs := []string{"faiss", "chroma", "qdrant", "pinecone"}
	if !contains(validVectorDBs, vectorDB) {
		return fmt.Errorf("invalid vector database '%s', must be one of: %v", vectorDB, validVectorDBs)
	}

	validIndexServices := []string{"llamaindex", "langchain"}
	if !contains(validIndexServices, indexService) {
		return fmt.Errorf("invalid index service '%s', must be one of: %v", indexService, validIndexServices)
	}

	klog.V(4).Info("RAG deploy validation completed successfully")
	return nil
}

func validateRagQueryOptions(ragName, question string, interactive bool) error {
	klog.V(4).Info("Validating RAG query options")

	if ragName == "" {
		return fmt.Errorf("RAG engine name is required")
	}

	if !interactive && question == "" {
		return fmt.Errorf("question is required in non-interactive mode")
	}

	klog.V(4).Info("RAG query validation completed successfully")
	return nil
}

func runRagDeploy(configFlags *genericclioptions.ConfigFlags, ragName, namespace, vectorDB, indexService,
	embeddingModel, dataSource string, chunkSize, chunkOverlap int, accessMode, accessSecret,
	storageSize, storageClass string, dryRun bool) error {
	klog.V(2).Infof("Deploying RAG engine: %s", ragName)

	// Get namespace
	if namespace == "" {
		if ns, _, err := configFlags.ToRawKubeConfigLoader().Namespace(); err == nil {
			namespace = ns
		} else {
			klog.V(4).Info("No namespace specified, using 'default'")
			namespace = "default"
		}
	}

	if dryRun {
		return showRagDeployDryRun(ragName, namespace, vectorDB, indexService, embeddingModel, dataSource,
			chunkSize, chunkOverlap, accessMode, storageSize, storageClass)
	}

	// Get REST config
	config, err := configFlags.ToRESTConfig()
	if err != nil {
		klog.Errorf("Failed to get REST config: %v", err)
		return fmt.Errorf("failed to get REST config: %w", err)
	}

	// Create dynamic client
	dynamicClient, err := dynamic.NewForConfig(config)
	if err != nil {
		klog.Errorf("Failed to create dynamic client: %v", err)
		return fmt.Errorf("failed to create dynamic client: %w", err)
	}

	// Create RAGEngine resource
	ragEngine := buildRAGEngine(ragName, namespace, vectorDB, indexService, embeddingModel, dataSource,
		chunkSize, chunkOverlap, accessMode, accessSecret, storageSize, storageClass)

	gvr := schema.GroupVersionResource{
		Group:    "kaito.sh",
		Version:  "v1beta1",
		Resource: "ragengines",
	}

	klog.V(3).Infof("Creating RAGEngine resource: %s", ragName)

	_, err = dynamicClient.Resource(gvr).Namespace(namespace).Create(
		context.TODO(),
		ragEngine,
		metav1.CreateOptions{},
	)

	if err != nil {
		klog.Errorf("Failed to create RAGEngine: %v", err)
		return fmt.Errorf("failed to create RAGEngine: %w", err)
	}

	klog.Infof("✓ RAG engine %s deployed successfully", ragName)
	klog.Infof("ℹ️  Use 'kubectl kaito status' to check the deployment status")
	return nil
}

func runRagQuery(configFlags *genericclioptions.ConfigFlags, ragName, namespace, question string,
	topK int, temperature float64, format string, interactive bool) error {
	klog.V(2).Infof("Querying RAG engine: %s", ragName)

	// Get namespace
	if namespace == "" {
		if ns, _, err := configFlags.ToRawKubeConfigLoader().Namespace(); err == nil {
			namespace = ns
		} else {
			klog.V(4).Info("No namespace specified, using 'default'")
			namespace = "default"
		}
	}

	// Get REST config
	config, err := configFlags.ToRESTConfig()
	if err != nil {
		klog.Errorf("Failed to get REST config: %v", err)
		return fmt.Errorf("failed to get REST config: %w", err)
	}

	// Create clients
	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		klog.Errorf("Failed to create kubernetes client: %v", err)
		return fmt.Errorf("failed to create kubernetes client: %w", err)
	}

	// Get RAG service endpoint
	endpoint, err := getRagEndpoint(clientset, ragName, namespace)
	if err != nil {
		klog.Errorf("Failed to get RAG endpoint: %v", err)
		return fmt.Errorf("failed to get RAG endpoint: %w", err)
	}

	klog.V(3).Infof("Using RAG endpoint: %s", endpoint)

	if interactive {
		return startRagInteractiveSession(endpoint, topK, temperature)
	}

	// Single query mode
	response, err := sendRagQuery(endpoint, question, topK, temperature)
	if err != nil {
		klog.Errorf("Failed to send query: %v", err)
		return fmt.Errorf("failed to send query: %w", err)
	}

	if format == "json" {
		jsonOutput, err := json.MarshalIndent(response, "", "  ")
		if err != nil {
			klog.Errorf("Failed to marshal JSON response: %v", err)
			return fmt.Errorf("failed to marshal JSON response: %w", err)
		}
		fmt.Println(string(jsonOutput))
	} else {
		if answer, ok := response["answer"].(string); ok {
			fmt.Println(answer)
		} else {
			return fmt.Errorf("invalid response format")
		}
	}

	return nil
}

func buildRAGEngine(ragName, namespace, vectorDB, indexService, embeddingModel, dataSource string,
	chunkSize, chunkOverlap int, accessMode, accessSecret, storageSize, storageClass string) *unstructured.Unstructured {
	klog.V(4).Info("Building RAGEngine configuration")

	spec := map[string]interface{}{
		"compute": map[string]interface{}{
			"inference": map[string]interface{}{
				"preset": map[string]interface{}{
					"name": "text-embedding-ada-002", // Default embedding model
				},
			},
		},
		"ragSpec": map[string]interface{}{
			"vectorDB": map[string]interface{}{
				"name": vectorDB,
			},
			"indexService": map[string]interface{}{
				"name": indexService,
			},
			"embeddingModel": embeddingModel,
			"chunkSize":      chunkSize,
			"chunkOverlap":   chunkOverlap,
		},
	}

	// Add data source if specified
	if dataSource != "" {
		spec["ragSpec"].(map[string]interface{})["dataSource"] = map[string]interface{}{
			"name": dataSource,
		}
		klog.V(4).Infof("Added data source: %s", dataSource)
	}

	// Add access configuration if private
	if accessMode == "private" && accessSecret != "" {
		spec["ragSpec"].(map[string]interface{})["accessMode"] = "private"
		spec["ragSpec"].(map[string]interface{})["secretName"] = accessSecret
		klog.V(4).Info("Added private access configuration")
	}

	// Add storage configuration
	if storageSize != "" {
		storage := map[string]interface{}{
			"size": storageSize,
		}
		if storageClass != "" {
			storage["storageClass"] = storageClass
		}
		spec["ragSpec"].(map[string]interface{})["storage"] = storage
		klog.V(4).Infof("Added storage configuration: %s", storageSize)
	}

	ragEngine := &unstructured.Unstructured{
		Object: map[string]interface{}{
			"apiVersion": "kaito.sh/v1beta1",
			"kind":       "RAGEngine",
			"metadata": map[string]interface{}{
				"name":      ragName,
				"namespace": namespace,
			},
			"spec": spec,
		},
	}

	return ragEngine
}

func showRagDeployDryRun(ragName, namespace, vectorDB, indexService, embeddingModel, dataSource string,
	chunkSize, chunkOverlap int, accessMode, storageSize, storageClass string) error {
	klog.V(2).Info("Running RAG deploy in dry-run mode")

	klog.Info("🔍 Dry-run mode: Showing what would be created")
	klog.Info("")
	klog.Info("RAG Engine Configuration:")
	klog.Info("========================")
	klog.Infof("Name: %s", ragName)
	klog.Infof("Namespace: %s", namespace)
	klog.Infof("Vector Database: %s", vectorDB)
	klog.Infof("Index Service: %s", indexService)
	klog.Infof("Embedding Model: %s", embeddingModel)
	klog.Infof("Chunk Size: %d", chunkSize)
	klog.Infof("Chunk Overlap: %d", chunkOverlap)

	if dataSource != "" {
		klog.Infof("Data Source: %s", dataSource)
	}

	if accessMode == "private" {
		klog.Infof("Access Mode: %s", accessMode)
	}

	if storageSize != "" {
		klog.Infof("Storage Size: %s", storageSize)
		if storageClass != "" {
			klog.Infof("Storage Class: %s", storageClass)
		}
	}

	klog.Info("")
	klog.Info("✓ RAG engine definition is valid")
	klog.Info("ℹ️  Run without --dry-run to create the RAG engine")

	return nil
}

func getRagEndpoint(clientset kubernetes.Interface, ragName, namespace string) (string, error) {
	klog.V(3).Infof("Getting RAG endpoint for: %s", ragName)

	// Get the service for the RAG engine (assuming service name equals RAG name)
	svc, err := clientset.CoreV1().Services(namespace).Get(context.TODO(), ragName, metav1.GetOptions{})
	if err != nil {
		klog.Errorf("Failed to get service for RAG engine %s: %v", ragName, err)
		return "", fmt.Errorf("failed to get service for RAG engine %s: %v", ragName, err)
	}

	if svc.Spec.ClusterIP == "" || svc.Spec.ClusterIP == "None" {
		return "", fmt.Errorf("service %s has no cluster IP", ragName)
	}

	endpoint := fmt.Sprintf("http://%s.%s.svc.cluster.local:80/query", ragName, namespace)
	klog.V(3).Infof("RAG endpoint: %s", endpoint)
	return endpoint, nil
}

func startRagInteractiveSession(endpoint string, topK int, temperature float64) error {
	klog.V(2).Info("Starting interactive RAG session")

	klog.Info("RAG Interactive Mode")
	klog.Info("===================")
	klog.Info("Type your questions below. Use '/quit' to exit.")
	klog.Info("")

	// This would implement interactive querying similar to chat
	// For now, just show placeholder
	klog.Info("Interactive RAG querying not fully implemented in this version")
	klog.Info("Use single query mode: kubectl kaito rag query --name <name> --question \"your question\"")

	return nil
}

func sendRagQuery(endpoint, question string, topK int, temperature float64) (map[string]interface{}, error) {
	klog.V(4).Infof("Sending RAG query to endpoint: %s", endpoint)

	payload := map[string]interface{}{
		"question":    question,
		"top_k":       topK,
		"temperature": temperature,
	}

	jsonData, err := json.Marshal(payload)
	if err != nil {
		klog.Errorf("Failed to marshal query payload: %v", err)
		return nil, fmt.Errorf("failed to marshal query payload: %w", err)
	}

	resp, err := http.Post(endpoint, "application/json", bytes.NewBuffer(jsonData))
	if err != nil {
		klog.Errorf("Failed to send RAG query: %v", err)
		return nil, fmt.Errorf("failed to send RAG query: %w", err)
	}
	defer resp.Body.Close()

	// Read response
	body, err := io.ReadAll(resp.Body)
	if err == nil && resp.StatusCode != http.StatusOK {
		klog.Errorf("RAG query failed with status %d: %s", resp.StatusCode, string(body))
		return nil, fmt.Errorf("RAG query failed with status %d: %s", resp.StatusCode, string(body))
	}

	if err != nil {
		klog.Errorf("Failed to read RAG response: %v", err)
		return nil, fmt.Errorf("failed to read RAG response: %w", err)
	}

	var response map[string]interface{}
	if err := json.Unmarshal(body, &response); err != nil {
		klog.Errorf("Failed to parse RAG response: %v", err)
		return nil, fmt.Errorf("failed to parse RAG response: %w", err)
	}

	return response, nil
}

// Helper function to check if slice contains a string
func contains(slice []string, item string) bool {
	for _, s := range slice {
		if s == item {
			return true
		}
	}
	return false
}
