package cmd

import (
	"fmt"
	"runtime"

	"github.com/spf13/cobra"
	"k8s.io/cli-runtime/pkg/genericclioptions"
)

var (
	// These will be set by goreleaser or build scripts
	version = "dev"
	commit  = "unknown"
	date    = "unknown"
)

type VersionOptions struct {
	configFlags *genericclioptions.ConfigFlags
	Short       bool
}

func NewVersionCmd(configFlags *genericclioptions.ConfigFlags) *cobra.Command {
	o := &VersionOptions{
		configFlags: configFlags,
	}

	cmd := &cobra.Command{
		Use:   "version",
		Short: "Display version information",
		Long: `Display version information for kubectl-kaito plugin.

Shows the plugin version, build commit, build date, and Go runtime information.`,
		Example: `  # Show full version information
  kubectl kaito version
  
  # Show short version only
  kubectl kaito version --short`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return o.Run()
		},
	}

	cmd.Flags().BoolVar(&o.Short, "short", false, "Show only the version number")

	return cmd
}

func (o *VersionOptions) Run() error {
	if o.Short {
		fmt.Println(version)
		return nil
	}

	fmt.Printf("kubectl-kaito version: %s\n", version)
	fmt.Printf("Git commit: %s\n", commit)
	fmt.Printf("Build date: %s\n", date)
	fmt.Printf("Go version: %s\n", runtime.Version())
	fmt.Printf("Go compiler: %s\n", runtime.Compiler)
	fmt.Printf("Platform: %s/%s\n", runtime.GOOS, runtime.GOARCH)

	return nil
}

// SetVersionInfo allows setting version information at build time
func SetVersionInfo(v, c, d string) {
	version = v
	commit = c
	date = d
} 