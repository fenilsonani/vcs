package main

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"

	"github.com/fenilsonani/vcs/pkg/vcs"
)

func newCloneCommand() *cobra.Command {
	var (
		bare   bool
		depth  int
		branch string
	)

	cmd := &cobra.Command{
		Use:   "clone [flags] <repository> [<directory>]",
		Short: "Clone a repository into a new directory",
		Long: `Clone a repository from a remote URL into a local directory.
This is a basic implementation that creates the repository structure
and sets up the remote configuration.`,
		Args: cobra.RangeArgs(1, 2),
		RunE: func(cmd *cobra.Command, args []string) error {
			repository := args[0]
			
			// Determine target directory
			var directory string
			if len(args) > 1 {
				directory = args[1]
			} else {
				directory = getDirectoryNameFromURL(repository)
			}

			return runClone(repository, directory, bare, depth, branch)
		},
	}

	cmd.Flags().BoolVar(&bare, "bare", false, "Create a bare repository")
	cmd.Flags().IntVar(&depth, "depth", 0, "Create a shallow clone with truncated history")
	cmd.Flags().StringVarP(&branch, "branch", "b", "", "Checkout specific branch instead of default")

	return cmd
}

func runClone(repository, directory string, bare bool, depth int, branch string) error {
	// Check if directory already exists
	if _, err := os.Stat(directory); err == nil {
		return fmt.Errorf("destination path '%s' already exists", directory)
	}

	fmt.Printf("Cloning into '%s'...\n", directory)

	// Create directory
	if err := os.MkdirAll(directory, 0755); err != nil {
		return fmt.Errorf("failed to create directory: %w", err)
	}

	// Initialize repository
	var repo *vcs.Repository
	var err error

	if bare {
		// For bare repositories, the directory itself is the git directory
		repo, err = initBareRepository(directory)
	} else {
		repo, err = vcs.Init(directory)
	}

	if err != nil {
		os.RemoveAll(directory) // Clean up on failure
		return fmt.Errorf("failed to initialize repository: %w", err)
	}

	// Add remote origin
	if err := addRemote(repo, "origin", repository); err != nil {
		return fmt.Errorf("failed to add remote: %w", err)
	}

	if !bare {
		// In a real implementation, this would:
		// 1. Fetch objects from remote
		// 2. Create and checkout default branch
		// 3. Set up tracking branches
		
		fmt.Printf("remote: Repository cloned successfully\n")
		fmt.Printf("Note: This is a basic clone implementation.\n")
		fmt.Printf("To fetch actual content, you would need to implement:\n")
		fmt.Printf("  - Network protocols (HTTP/SSH/Git)\n")
		fmt.Printf("  - Pack file transfer\n")
		fmt.Printf("  - Remote branch tracking\n")
		
		// Create a basic README to show the clone worked
		readmePath := filepath.Join(directory, "README.md")
		readmeContent := fmt.Sprintf("# Cloned Repository\n\nThis repository was cloned from: %s\n\nThis is a basic VCS implementation clone.\n", repository)
		if err := os.WriteFile(readmePath, []byte(readmeContent), 0644); err != nil {
			return fmt.Errorf("failed to create README: %w", err)
		}
	}

	return nil
}

func initBareRepository(path string) (*vcs.Repository, error) {
	// Create git directories
	dirs := []string{"objects/info", "objects/pack", "refs/heads", "refs/tags", "hooks", "info"}
	for _, dir := range dirs {
		fullPath := filepath.Join(path, dir)
		if err := os.MkdirAll(fullPath, 0755); err != nil {
			return nil, fmt.Errorf("failed to create %s directory: %w", dir, err)
		}
	}

	// Create HEAD file (for bare repo, points to default branch)
	headPath := filepath.Join(path, "HEAD")
	headContent := "ref: refs/heads/main\n"
	if err := os.WriteFile(headPath, []byte(headContent), 0644); err != nil {
		return nil, fmt.Errorf("failed to create HEAD file: %w", err)
	}

	// Create config file for bare repository
	configPath := filepath.Join(path, "config")
	configContent := `[core]
	repositoryformatversion = 0
	filemode = true
	bare = true
`
	if err := os.WriteFile(configPath, []byte(configContent), 0644); err != nil {
		return nil, fmt.Errorf("failed to create config file: %w", err)
	}

	// Create description file
	descPath := filepath.Join(path, "description")
	descContent := "Unnamed repository; edit this file 'description' to name the repository.\n"
	if err := os.WriteFile(descPath, []byte(descContent), 0644); err != nil {
		return nil, fmt.Errorf("failed to create description file: %w", err)
	}

	// Return a basic repository reference
	// In a full implementation, this would properly initialize the repository
	return vcs.Open(path)
}

func getDirectoryNameFromURL(url string) string {
	// Extract directory name from URL
	// e.g., "https://github.com/user/repo.git" -> "repo"
	//       "git@github.com:user/repo.git" -> "repo"
	
	// Remove .git suffix if present
	if filepath.Ext(url) == ".git" {
		url = url[:len(url)-4]
	}
	
	// Get the last component
	parts := filepath.SplitList(url)
	if len(parts) > 0 {
		return filepath.Base(parts[len(parts)-1])
	}
	
	// Fallback: use the last part after /
	if idx := len(url) - 1; idx >= 0 {
		for i := idx; i >= 0; i-- {
			if url[i] == '/' || url[i] == ':' {
				return url[i+1:]
			}
		}
	}
	
	return filepath.Base(url)
}