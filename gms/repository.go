package gms

// PersistentHandle is opaque data which is used to persist/restore an object
type PersistentHandle struct {
	// Type indicate the object type
	Type string
	// Opaque is the object-specific opaque data
	Opaque string
}

// Repository defines a repository can be accessed using a local path
// The local path can be a relative path within repository
type Repository interface {
	BasePath() string
	Persist() PersistentHandle
}

// RemoteRepo is a remote repository which must sync before direct access
type RemoteRepo interface {
	Repository
	Sync(dir string) error
}

// RepoFactory is used to restore a repository from persistent handle
type RepoFactory func(PersistentHandle) (Repository, error)

var (
	// RepoFactories is the registry of repo factories
	RepoFactories = map[string]RepoFactory{
		GitRepoType:   GitRepoFactory,
		LocalRepoType: LocalRepoFactory,
	}
)
