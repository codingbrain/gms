package gms

import (
	"encoding/json"
	"errors"
	"path/filepath"
	"sort"

	"github.com/codingbrain/clix.go/clix"
	"github.com/codingbrain/clix.go/conf"
)

const (
	// CacheConfFile is the filename of cache configuration file
	CacheConfFile = "repos.conf"
	// CacheReposDir is the name of sub-directory containing cached repos
	CacheReposDir = "repos"
)

var (
	// ErrRepoAlreadyExists indicates repository with the name already exists
	ErrRepoAlreadyExists = errors.New("repository already exists")
)

// CacheConfig is the format of cache config file
type CacheConfig struct {
	Repos map[string]PersistentHandle
}

// RepoCache is a cache of multiple remote repositories
type RepoCache struct {
	// BaseDir is root directory of cache
	BaseDir string

	repos map[string]*CachedRepo
}

// Load loads cached repository from file system
func (c *RepoCache) Load() error {
	fs := conf.NewFileStore(filepath.Join(c.BaseDir, CacheConfFile))
	rd, err := fs.Read()
	if err != nil {
		return err
	}
	defer rd.Close()
	var cfg CacheConfig
	if err = json.NewDecoder(rd).Decode(&cfg); err != nil {
		return err
	}

	if c.repos == nil {
		c.repos = make(map[string]*CachedRepo)
	}

	if cfg.Repos == nil {
		return nil
	}

	var errs clix.AggregatedError
	for name, h := range cfg.Repos {
		f := RepoFactories[h.Type]
		if f == nil {
			continue
		}
		repo, err := f(h)
		if errs.Add(err) || repo == nil {
			continue
		}
		if remote, ok := repo.(RemoteRepo); !ok {
			continue
		} else {
			c.repos[name] = &CachedRepo{
				Name:     name,
				Remote:   remote,
				LocalDir: filepath.Join(c.BaseDir, CacheReposDir, name),
			}
		}
	}
	return errs.Aggregate()
}

// Save flushes in memory changes to file system
func (c *RepoCache) Save() error {
	cfg := &CacheConfig{
		Repos: make(map[string]PersistentHandle),
	}
	for name, repo := range c.repos {
		cfg.Repos[name] = repo.Persist()
	}
	encoded, err := json.Marshal(cfg)
	if err != nil {
		return err
	}
	fs := conf.NewFileStore(filepath.Join(c.BaseDir, CacheConfFile))
	if w, err := fs.Write(); err == nil {
		defer w.Close()
		if _, err = w.Write(encoded); err != nil {
			return err
		}
		w.Commit(true)
	} else {
		return err
	}
	return nil
}

// Add adds a remote repo as a new cached repo
func (c *RepoCache) Add(name string, repo RemoteRepo) (*CachedRepo, error) {
	if r, exists := c.repos[name]; exists {
		return r, ErrRepoAlreadyExists
	}
	cachedRepo := &CachedRepo{
		Name:     name,
		Remote:   repo,
		LocalDir: filepath.Join(c.BaseDir, CacheReposDir, name),
	}
	c.repos[name] = cachedRepo
	if err := c.Save(); err != nil {
		delete(c.repos, name)
		return nil, err
	}
	return cachedRepo, nil
}

// Remove deletes a cached repo
func (c *RepoCache) Remove(name string) error {
	if r, exists := c.repos[name]; exists {
		delete(c.repos, name)
		if err := c.Save(); err != nil {
			c.repos[name] = r
			return err
		}
	}
	return nil
}

// RepoNames returns names of cached repos
func (c *RepoCache) RepoNames() []string {
	names := make([]string, 0, len(c.repos))
	for name := range c.repos {
		names = append(names, name)
	}
	sort.Strings(names)
	return names
}

// Find returns a cached repo by name
func (c *RepoCache) Find(name string) *CachedRepo {
	return c.repos[name]
}
