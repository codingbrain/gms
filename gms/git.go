package gms

import (
	"bytes"
	"encoding/json"
	"errors"
	"os"
	"os/exec"
	"strings"
)

const (
	// DefaultGitCmd is the default command of git
	DefaultGitCmd = "git"
	// GitRepoType is the type of the repository
	GitRepoType = "git"
)

var (
	// DefaultGitClient uses GitCmd as implementation
	DefaultGitClient = &GitCmd{Program: DefaultGitCmd}

	// ErrInvalidGitURL indicates no git respository is detected with the URL
	ErrInvalidGitURL = errors.New("invalid git url")
)

// GitError represents the error of git client
type GitError struct {
	// Output is optionally the stderr of git command
	Output string
	// Generic error object
	Err error
}

func (e *GitError) Error() string {
	return e.Err.Error() + ":\n" + e.Output
}

// GitClient is abstaction of functions from git
type GitClient interface {
	Exec(args ...string) (string, *GitError)
}

// GitCmd implements GitClient using git command
type GitCmd struct {
	// Program is path to git command, default is "git"
	Program string
}

// Exec implements GitClient
func (g *GitCmd) Exec(args ...string) (string, *GitError) {
	cmd := exec.Command(g.Program, args...)
	cmd.Env = append([]string{}, os.Environ()...)
	var errout bytes.Buffer
	cmd.Stderr = &errout
	out, err := cmd.Output()
	if err != nil {
		return string(out), &GitError{Output: errout.String(), Err: err}
	}
	return string(out), nil
}

// GitWorkTree wraps GitClient with working tree and git dir
type GitWorkTree struct {
	Client  GitClient
	WorkDir string
	GitDir  string
}

// Exec implements GitClient
func (g *GitWorkTree) Exec(args ...string) (string, *GitError) {
	if g.WorkDir == "" {
		panic("WorkDir is required")
	}
	argv := []string{}
	if g.GitDir != "" {
		argv = append(argv, "--work-tree="+g.WorkDir, "--git-dir="+g.GitDir)
	} else {
		argv = append(argv, "-C", g.WorkDir)
	}
	return g.Client.Exec(append(argv, args...)...)
}

// LatestCommit gets the latest commit Id in the working tree
func (g *GitWorkTree) LatestCommit() (string, error) {
	return g.Exec("log", "-1", "--format=%H")
}

// Pull fetches changes from remote and apply to current working tree
func (g *GitWorkTree) Pull() error {
	_, err := g.Exec("pull")
	return err
}

// PullAndVerify first pulls and verify by querying latest commit
func (g *GitWorkTree) PullAndVerify() (string, error) {
	if err := g.Pull(); err != nil {
		return "", err
	}
	return g.LatestCommit()
}

// Clone clones a remote repository
func (g *GitWorkTree) Clone(remote string, args ...string) error {
	_, err := g.Exec("clone", remote, g.WorkDir)
	return err
}

// GitRepo is a remote git repository
type GitRepo struct {
	// URL is full url of remote git repository
	URL string `json:"url"`

	// The following fields can be calculated from URL

	// Protocol used to talk to remote repository
	Protocol string `json:"protocol"`
	// RepoName is repository name portion in the URL
	RepoName string `json:"name"`
	// Remote is full URL to the respository (protocol+host+reponame)
	Remote string `json:"remote"`
	// Path is prefix in the repository
	Path string `json:"path"`

	// Client is git client
	Client GitClient `json:"-"`
}

// Detect parse the URL and find out the right information about the repository
func (r *GitRepo) Detect() (err error) {
	if r.URL == "" {
		panic("URL is required")
	}

	slashPos := strings.Index(r.URL, "/")
	colonPos := strings.Index(r.URL, ":")
	atPos := strings.Index(r.URL, "@")

	// user@host:repo/path
	if atPos > 0 && atPos < colonPos && (slashPos < 0 || colonPos < slashPos) {
		r.Protocol = "ssh"
		return r.detectPrefixed(r.URL[0:colonPos+1], r.URL[colonPos+1:])
	}

	// protocol://host/repo/path
	if colonPos > 0 && colonPos < slashPos &&
		strings.HasPrefix(r.URL[colonPos+1:], "//") {
		r.Protocol = r.URL[0:colonPos]
		return r.detectPrefixed(r.URL[0:colonPos+3], r.URL[colonPos+3:])
	}

	// ./path, ../path, /path
	if strings.HasPrefix(r.URL, "./") ||
		strings.HasPrefix(r.URL, "../") ||
		strings.HasPrefix(r.URL, "/") {
		r.Protocol = "file"
		return r.detectPrefixed(r.Protocol+"://", r.URL)
	}

	// host/repo/path
	if err := r.detectPrefixed("http://", r.URL); err == nil {
		r.Protocol = "http"
	} else if err := r.detectPrefixed("https://", r.URL); err == nil {
		r.Protocol = "https"
	} else if err := r.detectPrefixed("file://", r.URL); err == nil {
		r.Protocol = "file"
	} else {
		return ErrInvalidGitURL
	}

	return nil
}

func (r *GitRepo) detectPrefixed(prefix, path string) error {
	base := ""
	for path != "" {
		pos := strings.Index(path, "/")
		if pos > 0 {
			base += path[0:pos]
			path = path[pos:]
		} else if pos == 0 {
			base += "/"
			path = path[1:]
			continue
		} else {
			base += path
			path = ""
		}
		_, err := r.Client.Exec("ls-remote", prefix+base)
		if err == nil {
			r.RepoName = base
			r.Path = path
			r.Remote = prefix + base
			return nil
		}
	}
	return ErrInvalidGitURL
}

// BasePath implements Repository
func (r *GitRepo) BasePath() string {
	return r.Path
}

// Persist implements Repository
func (r *GitRepo) Persist() PersistentHandle {
	encoded, _ := json.Marshal(r)
	return PersistentHandle{Type: GitRepoType, Opaque: string(encoded)}
}

// Sync implements RemoteRepo
func (r *GitRepo) Sync(dir string) (err error) {
	git := &GitWorkTree{Client: r.Client, WorkDir: dir}
	_, err = git.LatestCommit()
	if err == nil {
		_, err = git.PullAndVerify()
	}
	if err != nil {
		os.RemoveAll(git.WorkDir)
		err = git.Clone(r.Remote)
	}
	return
}

// GitRepoFactory is the factory to restore a git repo
func GitRepoFactory(h PersistentHandle) (Repository, error) {
	if h.Type != GitRepoType {
		return nil, nil
	}
	r := &GitRepo{}
	return r, json.Unmarshal([]byte(h.Opaque), r)
}
