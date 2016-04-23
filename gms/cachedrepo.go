package gms

import "path/filepath"

// CachedRepo wraps over RemoteRepo to represent a local accessible repository
type CachedRepo struct {
	// Name of this local cache
	Name string
	// Remote is the remote repository
	Remote RemoteRepo
	// LocalDir is local path to clone of remote repository
	LocalDir string
}

// BasePath implements Repository
func (r *CachedRepo) BasePath() string {
	return filepath.Join(r.LocalDir, r.Remote.BasePath())
}

// Persist passthrough to remote repo
func (r *CachedRepo) Persist() PersistentHandle {
	return r.Remote.Persist()
}

// Sync explicitly updates the local cache
func (r *CachedRepo) Sync() error {
	return r.Remote.Sync(r.LocalDir)
}
