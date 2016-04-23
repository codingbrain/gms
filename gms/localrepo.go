package gms

import (
	"encoding/json"
	"path/filepath"
)

const (
	// LocalRepoType is the type name of local repo
	LocalRepoType = "local"
)

// LocalRepo is a local file system based repository
type LocalRepo struct {
	// BaseDir is local root directory of the repo
	BaseDir string `json:"base"`
	// Path is relative path inside the repo
	Path string `json:"path"`
}

// BasePath implements Repository
func (r *LocalRepo) BasePath() string {
	return filepath.Join(r.BaseDir, r.Path)
}

// Persist implements Repository
func (r *LocalRepo) Persist() PersistentHandle {
	encoded, _ := json.Marshal(r)
	return PersistentHandle{Type: LocalRepoType, Opaque: string(encoded)}
}

// LocalRepoFactory is repo factory
func LocalRepoFactory(h PersistentHandle) (Repository, error) {
	if h.Type != LocalRepoType {
		return nil, nil
	}
	r := &LocalRepo{}
	return r, json.Unmarshal([]byte(h.Opaque), r)
}
