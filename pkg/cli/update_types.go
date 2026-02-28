package cli

import "strconv"

// workflowWithSource represents a workflow with its source information
type workflowWithSource struct {
	Name       string
	Path       string
	SourceSpec string // e.g., "owner/repo/path@ref"
}

// releaseCache stores resolved refs and release lists to avoid redundant
// GitHub API calls within a single update command invocation.
type releaseCache struct {
	// releases maps "repo|currentRef|allowMajor" → resolved latest tag
	releases map[string]string
	// branchSHAs maps "repo|branch" → latest commit SHA
	branchSHAs map[string]string
	// defaultBranches maps "repo" → default branch name
	defaultBranches map[string]string
}

// newReleaseCache creates a new empty release cache.
func newReleaseCache() *releaseCache {
	return &releaseCache{
		releases:        make(map[string]string),
		branchSHAs:      make(map[string]string),
		defaultBranches: make(map[string]string),
	}
}

// makeReleaseCacheKey builds the cache key for resolveLatestRelease results.
func makeReleaseCacheKey(repo, currentRef string, allowMajor bool) string {
	return repo + "|" + currentRef + "|" + strconv.FormatBool(allowMajor)
}

// makeBranchSHACacheKey builds the cache key for getLatestBranchCommitSHA results.
func makeBranchSHACacheKey(repo, branch string) string {
	return repo + "|" + branch
}

// updateFailure represents a failed workflow update
type updateFailure struct {
	Name  string
	Error string
}

// actionsLockEntry represents a single action pin entry
type actionsLockEntry struct {
	Repo    string `json:"repo"`
	Version string `json:"version"`
	SHA     string `json:"sha"`
}

// actionsLockFile represents the structure of actions-lock.json
type actionsLockFile struct {
	Entries map[string]actionsLockEntry `json:"entries"`
}
