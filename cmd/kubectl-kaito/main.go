package main

import (
	"os"
	"path/filepath"
	"strings"

	"github.com/kaito-project/kubectl-kaito/pkg"
	"k8s.io/cli-runtime/pkg/genericclioptions"

	// Import auth plugins
	_ "k8s.io/client-go/plugin/pkg/client/auth"
)

func main() {
	// Determine if running as kubectl plugin
	isPlugin := strings.HasPrefix(filepath.Base(os.Args[0]), "kubectl-")

	// Create ConfigFlags to handle standard kubectl options
	configFlags := genericclioptions.NewConfigFlags(true)

	// Create and execute root command
	rootCmd := cmd.NewRootCmd(configFlags, isPlugin)
	if err := rootCmd.Execute(); err != nil {
		os.Exit(1)
	}
}
