// Package transport provides HTTP transport implementation for Git protocol communication.
//
// This package implements Git's HTTP transport protocol for communicating with
// remote repositories, particularly GitHub. It supports:
//
//   - Git HTTP protocol (info/refs and upload-pack endpoints)
//   - GitHub API integration with token authentication
//   - URL parsing for various Git URL formats (SSH, HTTPS, shorthand)
//   - Ref discovery and pack file negotiation
//
// Example usage:
//
//	// Create HTTP transport
//	transport := transport.NewHTTPTransport("https://github.com/user/repo")
//	
//	// Discover remote refs
//	ctx := context.Background()
//	discovery, err := transport.DiscoverRefs(ctx, "git-upload-pack")
//	if err != nil {
//	    log.Fatal(err)
//	}
//	
//	// For GitHub with authentication
//	githubTransport, err := transport.NewGitHubTransport("git@github.com:user/repo.git", "token")
//	if err != nil {
//	    log.Fatal(err)
//	}
//
// The transport layer handles the low-level Git protocol details, allowing
// higher-level commands like fetch, push, and pull to work with remote repositories.
package transport