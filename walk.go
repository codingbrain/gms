package gms

import (
	"io"
	"os"
	"path/filepath"
)

// WalkingItem is the current item being visited
type WalkingItem struct {
	// RepoName is name of the repository being visited
	RepoName string
	// Repo is the repository being visited
	Repo Repository
	// Path is the relative path inside repo without filename
	Path string
	// Name is the name of the item, file/dir name
	Name string
	// FileInfo is obtained using os.Lstat
	FileInfo os.FileInfo
}

// RepoWalkerFn is the function visits all objects inside the repository
type RepoWalkerFn func(item WalkingItem) error

// RepoWalkerFilter is filtering function before WalkerFn is called
type RepoWalkerFilter func(item *WalkingItem) (bool, error)

// RepoWalker helps visiting the whole repository
type RepoWalker struct {
	// WalkerFn is the walker function
	WalkerFn RepoWalkerFn
	// PathPrefix appends prefix to paths
	PathPrefix string
	// Filters are filters called before walkerFn
	Filters []RepoWalkerFilter
	// BreadthFirst visits in breadth first order, otherwise depth first
	BreadthFirst bool
}

// Visit walks over every entry inside the repo
func (w *RepoWalker) Visit(name string, repo Repository) error {
	return w.visit(repo.BasePath(), name, repo)
}

func (w *RepoWalker) visit(basePath, name string, repo Repository) error {
	fullPath := basePath
	if w.PathPrefix != "" {
		fullPath = w.PathPrefix + fullPath
	}
	f, err := os.Open(fullPath)
	if err != nil {
		return err
	}
	defer f.Close()
	var dirs []string
	for {
		var fi os.FileInfo
		if fis, e := f.Readdir(1); e == io.EOF {
			break
		} else if e != nil {
			return e
		} else {
			fi = fis[0]
		}

		item := &WalkingItem{
			RepoName: name,
			Repo:     repo,
			Path:     basePath,
			Name:     fi.Name(),
			FileInfo: fi,
		}
		skip := false
		for _, filter := range w.Filters {
			accepted, e := filter(item)
			if e != nil {
				return e
			}
			if !accepted {
				skip = true
				break
			}
		}
		if skip {
			continue
		}

		if err = w.WalkerFn(*item); err != nil {
			return err
		}
		if fi.IsDir() {
			if w.BreadthFirst {
				dirs = append(dirs, fi.Name())
			} else {
				err = w.visit(filepath.Join(basePath, fi.Name()), name, repo)
				if err != nil {
					return err
				}
			}
		}
	}
	for _, dir := range dirs {
		if err = w.visit(filepath.Join(basePath, dir), name, repo); err != nil {
			return err
		}
	}
	return nil
}

// Use registers walker filters
func (w *RepoWalker) Use(filters ...RepoWalkerFilter) *RepoWalker {
	w.Filters = append(w.Filters, filters...)
	return w
}
