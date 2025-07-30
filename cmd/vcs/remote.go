package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"

	"github.com/fenilsonani/vcs/pkg/vcs"
)

func newRemoteCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "remote",
		Short: "Manage set of tracked repositories",
		Long:  `Manage the set of repositories ("remotes") whose branches you track.`,
	}

	cmd.AddCommand(
		newRemoteAddCommand(),
		newRemoteRemoveCommand(),
		newRemoteListCommand(),
		newRemoteShowCommand(),
	)

	return cmd
}

func newRemoteAddCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "add <name> <url>",
		Short: "Add a remote repository",
		Args:  cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			repo, err := findRepository()
			if err != nil {
				return err
			}

			vcsRepo, err := vcs.Open(repo)
			if err != nil {
				return err
			}

			return addRemote(vcsRepo, args[0], args[1])
		},
	}
}

func newRemoteRemoveCommand() *cobra.Command {
	return &cobra.Command{
		Use:     "remove <name>",
		Aliases: []string{"rm"},
		Short:   "Remove a remote repository",
		Args:    cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			repo, err := findRepository()
			if err != nil {
				return err
			}

			vcsRepo, err := vcs.Open(repo)
			if err != nil {
				return err
			}

			return removeRemote(vcsRepo, args[0])
		},
	}
}

func newRemoteListCommand() *cobra.Command {
	var verbose bool

	cmd := &cobra.Command{
		Use:     "list",
		Aliases: []string{"ls"},
		Short:   "List remote repositories",
		RunE: func(cmd *cobra.Command, args []string) error {
			repo, err := findRepository()
			if err != nil {
				return err
			}

			vcsRepo, err := vcs.Open(repo)
			if err != nil {
				return err
			}

			return listRemotes(vcsRepo, verbose)
		},
	}

	cmd.Flags().BoolVarP(&verbose, "verbose", "v", false, "Show URLs")
	return cmd
}

func newRemoteShowCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "show <name>",
		Short: "Show information about a remote",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			repo, err := findRepository()
			if err != nil {
				return err
			}

			vcsRepo, err := vcs.Open(repo)
			if err != nil {
				return err
			}

			return showRemote(vcsRepo, args[0])
		},
	}
}

func addRemote(repo *vcs.Repository, name, url string) error {
	if err := validateRemoteName(name); err != nil {
		return err
	}

	if err := validateRemoteURL(url); err != nil {
		return err
	}

	// Check if remote already exists
	if remoteExists(repo, name) {
		return fmt.Errorf("remote %s already exists", name)
	}

	// Add remote to config
	if err := writeRemoteConfig(repo, name, url); err != nil {
		return fmt.Errorf("failed to add remote: %w", err)
	}

	fmt.Printf("Added remote '%s' -> '%s'\n", name, url)
	return nil
}

func removeRemote(repo *vcs.Repository, name string) error {
	if !remoteExists(repo, name) {
		return fmt.Errorf("remote '%s' does not exist", name)
	}

	if err := removeRemoteConfig(repo, name); err != nil {
		return fmt.Errorf("failed to remove remote: %w", err)
	}

	fmt.Printf("Removed remote '%s'\n", name)
	return nil
}

func listRemotes(repo *vcs.Repository, verbose bool) error {
	remotes, err := getRemotes(repo)
	if err != nil {
		return fmt.Errorf("failed to list remotes: %w", err)
	}

	for name, url := range remotes {
		if verbose {
			fmt.Printf("%s\t%s (fetch)\n", name, url)
			fmt.Printf("%s\t%s (push)\n", name, url)
		} else {
			fmt.Println(name)
		}
	}

	return nil
}

func showRemote(repo *vcs.Repository, name string) error {
	if !remoteExists(repo, name) {
		return fmt.Errorf("remote '%s' does not exist", name)
	}

	remotes, err := getRemotes(repo)
	if err != nil {
		return fmt.Errorf("failed to get remote info: %w", err)
	}

	url := remotes[name]
	fmt.Printf("* remote %s\n", name)
	fmt.Printf("  Fetch URL: %s\n", url)
	fmt.Printf("  Push  URL: %s\n", url)
	fmt.Printf("  HEAD branch: (unknown)\n")

	return nil
}

func validateRemoteName(name string) error {
	if name == "" {
		return fmt.Errorf("remote name cannot be empty")
	}

	if strings.Contains(name, " ") {
		return fmt.Errorf("remote name cannot contain spaces")
	}

	// Reserve some special names
	reserved := []string{"HEAD", "refs", "objects", "config", "hooks"}
	for _, r := range reserved {
		if name == r {
			return fmt.Errorf("'%s' is a reserved name", name)
		}
	}

	return nil
}

func validateRemoteURL(url string) error {
	if url == "" {
		return fmt.Errorf("remote URL cannot be empty")
	}

	// Basic URL validation - in practice would be more sophisticated
	if !strings.HasPrefix(url, "http://") && 
		!strings.HasPrefix(url, "https://") && 
		!strings.HasPrefix(url, "git://") && 
		!strings.HasPrefix(url, "ssh://") &&
		!strings.Contains(url, "@") { // git@github.com:user/repo.git format
		return fmt.Errorf("invalid URL format")
	}

	return nil
}

func remoteExists(repo *vcs.Repository, name string) bool {
	remotes, err := getRemotes(repo)
	if err != nil {
		return false
	}
	_, exists := remotes[name]
	return exists
}

func getRemotes(repo *vcs.Repository) (map[string]string, error) {
	configPath := filepath.Join(repo.GitDir(), "config")
	content, err := os.ReadFile(configPath)
	if err != nil {
		return make(map[string]string), nil // No config file yet
	}

	return parseRemotesFromConfig(string(content)), nil
}

func parseRemotesFromConfig(content string) map[string]string {
	remotes := make(map[string]string)
	lines := strings.Split(content, "\n")
	
	var currentRemote string
	for _, line := range lines {
		line = strings.TrimSpace(line)
		
		// Look for [remote "name"] sections
		if strings.HasPrefix(line, "[remote \"") && strings.HasSuffix(line, "\"]") {
			start := strings.Index(line, "\"") + 1
			end := strings.LastIndex(line, "\"")
			if start < end {
				currentRemote = line[start:end]
			}
		} else if currentRemote != "" && strings.HasPrefix(line, "url = ") {
			url := strings.TrimPrefix(line, "url = ")
			remotes[currentRemote] = url
			currentRemote = "" // Reset after finding URL
		}
	}

	return remotes
}

func writeRemoteConfig(repo *vcs.Repository, name, url string) error {
	configPath := filepath.Join(repo.GitDir(), "config")
	
	// Read existing config
	var content string
	if data, err := os.ReadFile(configPath); err == nil {
		content = string(data)
	}

	// Append remote configuration
	remoteConfig := fmt.Sprintf("\n[remote \"%s\"]\n\turl = %s\n", name, url)
	content += remoteConfig

	return os.WriteFile(configPath, []byte(content), 0644)
}

func removeRemoteConfig(repo *vcs.Repository, name string) error {
	configPath := filepath.Join(repo.GitDir(), "config")
	
	content, err := os.ReadFile(configPath)
	if err != nil {
		return err
	}

	lines := strings.Split(string(content), "\n")
	var newLines []string
	
	inRemoteSection := false
	targetSection := fmt.Sprintf("[remote \"%s\"]", name)
	
	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		
		if trimmed == targetSection {
			inRemoteSection = true
			continue // Skip this line
		}
		
		if inRemoteSection && strings.HasPrefix(trimmed, "[") && trimmed != targetSection {
			inRemoteSection = false // Entered a new section
		}
		
		if !inRemoteSection {
			newLines = append(newLines, line)
		}
	}

	return os.WriteFile(configPath, []byte(strings.Join(newLines, "\n")), 0644)
}