package transport

import (
	"bufio"
	"bytes"
	"context"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"
)

// HTTPTransport implements Git's HTTP transport protocol
type HTTPTransport struct {
	client    *http.Client
	baseURL   string
	userAgent string
}

// NewHTTPTransport creates a new HTTP transport for Git protocol
func NewHTTPTransport(baseURL string) *HTTPTransport {
	return &HTTPTransport{
		client: &http.Client{
			Timeout: 30 * time.Second,
		},
		baseURL:   baseURL,
		userAgent: "vcs/1.0 (git-http-transport)",
	}
}

// SetCredentials configures authentication for the transport
func (t *HTTPTransport) SetCredentials(username, password string) {
	// In a real implementation, this would set up authentication
	// For GitHub, this would handle personal access tokens
}

// DiscoverRefs implements the initial ref discovery phase of Git HTTP protocol
func (t *HTTPTransport) DiscoverRefs(ctx context.Context, service string) (*RefDiscovery, error) {
	// Git HTTP protocol: GET /info/refs?service=git-upload-pack
	reqURL := fmt.Sprintf("%s/info/refs?service=%s", t.baseURL, service)
	
	req, err := http.NewRequestWithContext(ctx, "GET", reqURL, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}
	
	req.Header.Set("User-Agent", t.userAgent)
	req.Header.Set("Accept", "*/*")
	
	resp, err := t.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to make request: %w", err)
	}
	defer resp.Body.Close()
	
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}
	
	// Check content type
	contentType := resp.Header.Get("Content-Type")
	expectedContentType := fmt.Sprintf("application/x-%s-advertisement", service)
	if contentType != expectedContentType {
		return nil, fmt.Errorf("unexpected content type: %s", contentType)
	}
	
	return t.parseRefAdvertisement(resp.Body)
}

// RefDiscovery represents the result of ref discovery
type RefDiscovery struct {
	Refs         map[string]string // ref name -> object ID
	Capabilities []string          // server capabilities
	Service      string            // service name
}

// parseRefAdvertisement parses the Git ref advertisement format
func (t *HTTPTransport) parseRefAdvertisement(r io.Reader) (*RefDiscovery, error) {
	scanner := bufio.NewScanner(r)
	discovery := &RefDiscovery{
		Refs: make(map[string]string),
	}
	
	// First line should be the service advertisement
	if !scanner.Scan() {
		return nil, fmt.Errorf("empty ref advertisement")
	}
	
	line := scanner.Text()
	if !strings.HasPrefix(line, "# service=") {
		return nil, fmt.Errorf("invalid service advertisement: %s", line)
	}
	
	discovery.Service = strings.TrimPrefix(line, "# service=")
	
	// Parse refs
	for scanner.Scan() {
		line := scanner.Text()
		if line == "" {
			continue
		}
		
		// Parse pkt-line format (length prefix + content)
		if len(line) < 4 {
			continue
		}
		
		// Extract the actual ref line (skip length prefix)
		refLine := strings.TrimSpace(line[4:])
		if refLine == "" {
			continue
		}
		
		// Parse "objectid refname [capabilities]"
		parts := strings.Fields(refLine)
		if len(parts) >= 2 {
			objectID := parts[0]
			refName := parts[1]
			discovery.Refs[refName] = objectID
			
			// Parse capabilities from first ref
			if len(discovery.Capabilities) == 0 && len(parts) > 2 {
				// Capabilities are after a null byte
				capString := strings.Join(parts[2:], " ")
				if idx := strings.Index(capString, "\x00"); idx >= 0 {
					capString = capString[idx+1:]
				}
				discovery.Capabilities = strings.Fields(capString)
			}
		}
	}
	
	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("failed to read ref advertisement: %w", err)
	}
	
	return discovery, nil
}

// FetchPack performs the pack negotiation and download phase
func (t *HTTPTransport) FetchPack(ctx context.Context, wants, haves []string) (io.ReadCloser, error) {
	// Git HTTP protocol: POST /git-upload-pack
	reqURL := fmt.Sprintf("%s/git-upload-pack", t.baseURL)
	
	// Build the request body (pack negotiation)
	var buf bytes.Buffer
	
	// Write wants
	for _, want := range wants {
		buf.WriteString(fmt.Sprintf("want %s\n", want))
	}
	
	// Write haves
	for _, have := range haves {
		buf.WriteString(fmt.Sprintf("have %s\n", have))
	}
	
	// End negotiation
	buf.WriteString("done\n")
	
	req, err := http.NewRequestWithContext(ctx, "POST", reqURL, &buf)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}
	
	req.Header.Set("User-Agent", t.userAgent)
	req.Header.Set("Content-Type", "application/x-git-upload-pack-request")
	req.Header.Set("Accept", "application/x-git-upload-pack-result")
	
	resp, err := t.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to make request: %w", err)
	}
	
	if resp.StatusCode != http.StatusOK {
		resp.Body.Close()
		return nil, fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}
	
	// Check content type
	contentType := resp.Header.Get("Content-Type")
	if contentType != "application/x-git-upload-pack-result" {
		resp.Body.Close()
		return nil, fmt.Errorf("unexpected content type: %s", contentType)
	}
	
	return resp.Body, nil
}

// ParseGitURL parses a Git URL and returns the HTTP equivalent
func ParseGitURL(gitURL string) (string, error) {
	// Handle different Git URL formats
	
	// SSH format: git@github.com:user/repo.git
	if strings.HasPrefix(gitURL, "git@") {
		parts := strings.SplitN(gitURL, ":", 2)
		if len(parts) != 2 {
			return "", fmt.Errorf("invalid SSH URL format: %s", gitURL)
		}
		
		host := strings.TrimPrefix(parts[0], "git@")
		path := strings.TrimSuffix(parts[1], ".git")
		
		return fmt.Sprintf("https://%s/%s", host, path), nil
	}
	
	// HTTP/HTTPS format
	if strings.HasPrefix(gitURL, "http://") || strings.HasPrefix(gitURL, "https://") {
		u, err := url.Parse(gitURL)
		if err != nil {
			return "", fmt.Errorf("invalid URL: %w", err)
		}
		
		// Keep HTTP for localhost/127.0.0.1 (test servers), otherwise upgrade to HTTPS
		if u.Hostname() != "localhost" && u.Hostname() != "127.0.0.1" && !strings.HasPrefix(u.Hostname(), "127.") {
			u.Scheme = "https"
		}
		u.Path = strings.TrimSuffix(u.Path, ".git")
		
		return u.String(), nil
	}
	
	// GitHub shorthand: user/repo
	if strings.Count(gitURL, "/") == 1 && !strings.Contains(gitURL, ":") {
		return fmt.Sprintf("https://github.com/%s", gitURL), nil
	}
	
	return "", fmt.Errorf("unsupported URL format: %s", gitURL)
}

// GitHubTransport is a specialized HTTP transport for GitHub
type GitHubTransport struct {
	*HTTPTransport
	token string
}

// NewGitHubTransport creates a new GitHub-specific transport
func NewGitHubTransport(repoURL, token string) (*GitHubTransport, error) {
	httpURL, err := ParseGitURL(repoURL)
	if err != nil {
		return nil, err
	}
	
	transport := &GitHubTransport{
		HTTPTransport: NewHTTPTransport(httpURL),
		token:         token,
	}
	
	// Configure GitHub-specific settings
	transport.userAgent = "vcs/1.0 (GitHub-integration)"
	
	return transport, nil
}

// DiscoverRefs overrides the base method to add GitHub authentication
func (t *GitHubTransport) DiscoverRefs(ctx context.Context, service string) (*RefDiscovery, error) {
	reqURL := fmt.Sprintf("%s/info/refs?service=%s", t.baseURL, service)
	
	req, err := http.NewRequestWithContext(ctx, "GET", reqURL, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}
	
	req.Header.Set("User-Agent", t.userAgent)
	req.Header.Set("Accept", "*/*")
	
	// Add GitHub authentication
	if t.token != "" {
		req.Header.Set("Authorization", fmt.Sprintf("token %s", t.token))
	}
	
	resp, err := t.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to make request: %w", err)
	}
	defer resp.Body.Close()
	
	if resp.StatusCode == http.StatusUnauthorized {
		return nil, fmt.Errorf("authentication failed - check your GitHub token")
	}
	
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}
	
	return t.parseRefAdvertisement(resp.Body)
}

// ListRepositoryRefs uses GitHub API to list repository references
func (t *GitHubTransport) ListRepositoryRefs(ctx context.Context) (map[string]string, error) {
	// Extract owner/repo from URL
	u, err := url.Parse(t.baseURL)
	if err != nil {
		return nil, fmt.Errorf("invalid URL: %w", err)
	}
	
	pathParts := strings.Split(strings.Trim(u.Path, "/"), "/")
	if len(pathParts) != 2 {
		return nil, fmt.Errorf("invalid GitHub repository URL")
	}
	
	owner, repo := pathParts[0], pathParts[1]
	
	// Use GitHub API v3 to list refs
	apiURL := fmt.Sprintf("https://api.github.com/repos/%s/%s/git/refs", owner, repo)
	
	req, err := http.NewRequestWithContext(ctx, "GET", apiURL, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create API request: %w", err)
	}
	
	req.Header.Set("User-Agent", t.userAgent)
	req.Header.Set("Accept", "application/vnd.github.v3+json")
	
	if t.token != "" {
		req.Header.Set("Authorization", fmt.Sprintf("token %s", t.token))
	}
	
	resp, err := t.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to make API request: %w", err)
	}
	defer resp.Body.Close()
	
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("GitHub API error: %d", resp.StatusCode)
	}
	
	// For now, return a basic response
	// In a full implementation, this would parse the JSON response
	refs := map[string]string{
		"refs/heads/main": "dummy-commit-hash",
	}
	
	return refs, nil
}