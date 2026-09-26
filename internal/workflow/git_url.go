package workflow

import (
	"fmt"
	"regexp"
	"strings"
)

// normalizeGitURL normalizes git remote URLs to a standard form for comparison
// Treats SCP-style git@github.com:o/r.git as equal to ssh://git@github.com/o/r.git
// Returns host/path without .git suffix
func normalizeGitURL(url string) string {
	// Remove any trailing whitespace
	url = strings.TrimSpace(url)

	// Handle SCP-style URLs: git@github.com:owner/repo.git
	// Match: git@host:owner/repo.git or git@host:owner/repo
	scpRegex := regexp.MustCompile(`^git@([^:]+):(.+?)(?:\.git)?$`)
	if matches := scpRegex.FindStringSubmatch(url); matches != nil {
		host := matches[1]
		path := matches[2]
		// Remove .git if present
		path = strings.TrimSuffix(path, ".git")
		return fmt.Sprintf("%s/%s", host, path)
	}

	// Handle SSH-style URLs: ssh://git@github.com/owner/repo.git
	// Match: ssh://git@host/owner/repo.git or ssh://git@host/owner/repo
	sshRegex := regexp.MustCompile(`^ssh://git@([^/]+)/(.+?)(?:\.git)?$`)
	if matches := sshRegex.FindStringSubmatch(url); matches != nil {
		host := matches[1]
		path := matches[2]
		path = strings.TrimSuffix(path, ".git")
		return fmt.Sprintf("%s/%s", host, path)
	}

	// Handle HTTPS URLs: https://github.com/owner/repo.git
	// Match: https://host/owner/repo.git or https://host/owner/repo
	httpsRegex := regexp.MustCompile(`^https?://([^/]+)/(.+?)(?:\.git)?$`)
	if matches := httpsRegex.FindStringSubmatch(url); matches != nil {
		host := matches[1]
		path := matches[2]
		path = strings.TrimSuffix(path, ".git")
		return fmt.Sprintf("%s/%s", host, path)
	}

	// Handle Git protocol URLs: git://github.com/owner/repo.git
	gitRegex := regexp.MustCompile(`^git://([^/]+)/(.+?)(?:\.git)?$`)
	if matches := gitRegex.FindStringSubmatch(url); matches != nil {
		host := matches[1]
		path := matches[2]
		path = strings.TrimSuffix(path, ".git")
		return fmt.Sprintf("%s/%s", host, path)
	}

	// If no match, return as-is (trimmed)
	return strings.TrimSpace(url)
}

// URLsAreEqual compares two git URLs after normalization
func URLsAreEqual(url1, url2 string) bool {
	return normalizeGitURL(url1) == normalizeGitURL(url2)
}

// GetGitHubRawURL converts a normalized git URL to a raw GitHub URL
// Input: "github.com/owner/repo" -> "https://raw.githubusercontent.com/owner/repo"
func GetGitHubRawURL(normalized string) string {
	// If already a full URL, return it
	if strings.HasPrefix(normalized, "http") {
		return normalized
	}
	// Remove any leading slashes
	normalized = strings.TrimPrefix(normalized, "/")
	return fmt.Sprintf("https://raw.githubusercontent.com/%s", normalized)
}
