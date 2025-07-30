package main

import (
	"fmt"
	"path/filepath"

	"github.com/fenilsonani/vcs/pkg/vcs"
	"github.com/spf13/cobra"
)

func newInitCommand() *cobra.Command {
	var bare bool
	
	cmd := &cobra.Command{
		Use:   "init [path]",
		Short: "Initialize a new repository",
		Long:  "Create an empty VCS repository or reinitialize an existing one",
		Args:  cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			path := "."
			if len(args) > 0 {
				path = args[0]
			}
			
			// Get absolute path
			absPath, err := filepath.Abs(path)
			if err != nil {
				return fmt.Errorf("failed to get absolute path: %w", err)
			}
			
			// Initialize repository
			repo, err := vcs.Init(absPath)
			if err != nil {
				return fmt.Errorf("failed to initialize repository: %w", err)
			}
			
			// Print success message
			if bare {
				fmt.Printf("Initialized empty VCS repository in %s\n", repo.GitDir())
			} else {
				fmt.Printf("Initialized empty VCS repository in %s\n", filepath.Join(repo.GitDir()))
			}
			
			return nil
		},
	}
	
	cmd.Flags().BoolVar(&bare, "bare", false, "Create a bare repository")
	
	return cmd
}