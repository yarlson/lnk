// Package lnk implements the business logic for lnk.
package lnk

import (
	"fmt"
	"io"
	"os"
	"path/filepath"

	"github.com/yarlson/lnk/internal/bootstrapper"
	"github.com/yarlson/lnk/internal/doctor"
	"github.com/yarlson/lnk/internal/filemanager"
	"github.com/yarlson/lnk/internal/fs"
	"github.com/yarlson/lnk/internal/git"
	"github.com/yarlson/lnk/internal/initializer"
	"github.com/yarlson/lnk/internal/lnkerror"
	"github.com/yarlson/lnk/internal/syncer"
	"github.com/yarlson/lnk/internal/tracker"
)

// Sentinel errors re-exported from lnkerror for backwards compatibility.
var (
	ErrManagedFilesExist = lnkerror.ErrManagedFilesExist
	ErrGitRepoExists     = lnkerror.ErrGitRepoExists
	ErrAlreadyManaged    = lnkerror.ErrAlreadyManaged
	ErrNotManaged        = lnkerror.ErrNotManaged
	ErrNotInitialized    = lnkerror.ErrNotInitialized
	ErrBootstrapNotFound = lnkerror.ErrBootstrapNotFound
	ErrBootstrapFailed   = lnkerror.ErrBootstrapFailed
	ErrBootstrapPerms    = lnkerror.ErrBootstrapPerms
)

// ProgressCallback defines the signature for progress reporting callbacks.
type ProgressCallback = filemanager.ProgressCallback

// StatusInfo contains repository sync status information.
type StatusInfo = syncer.StatusInfo

// DoctorResult contains the results of a doctor scan or execution.
type DoctorResult = doctor.Result

// Lnk is the facade that composes focused collaborators for dotfile management.
type Lnk struct {
	repoPath string
	host     string
	tracker  *tracker.Tracker
	files    *filemanager.Manager
	syncer   *syncer.Syncer
	init     *initializer.Service
	boot     *bootstrapper.Runner
	health   *doctor.Checker
}

// Option configures a Lnk instance.
type Option func(*Lnk)

// WithHost sets the host for host-specific configuration.
func WithHost(host string) Option {
	return func(l *Lnk) {
		l.host = host
	}
}

// NewLnk creates a new Lnk instance with optional configuration.
func NewLnk(opts ...Option) *Lnk {
	repoPath := GetRepoPath()
	l := &Lnk{
		repoPath: repoPath,
		host:     "",
	}

	for _, opt := range opts {
		opt(l)
	}

	// Wire collaborators after options are applied (host may change).
	g := git.New(repoPath)
	f := fs.New()
	t := tracker.New(repoPath, l.host)

	l.tracker = t
	l.files = filemanager.New(repoPath, l.host, g, f, t)
	l.syncer = syncer.New(repoPath, l.host, g, f, t)
	l.init = initializer.New(repoPath, g, t)
	l.boot = bootstrapper.New(repoPath, g)
	l.health = doctor.New(repoPath, l.host, g, t, l.syncer)

	return l
}

// --- Initialization delegates ---

func (l *Lnk) Init() error                           { return l.init.Init() }
func (l *Lnk) InitWithRemote(remoteURL string) error { return l.init.InitWithRemote(remoteURL) }
func (l *Lnk) InitWithRemoteForce(remoteURL string, force bool) error {
	return l.init.InitWithRemoteForce(remoteURL, force)
}
func (l *Lnk) Clone(url string) error           { return l.init.Clone(url) }
func (l *Lnk) AddRemote(name, url string) error { return l.init.AddRemote(name, url) }
func (l *Lnk) HasUserContent() bool             { return l.init.HasUserContent() }

// --- File management delegates ---

func (l *Lnk) Add(filePath string) error        { return l.files.Add(filePath) }
func (l *Lnk) AddMultiple(paths []string) error { return l.files.AddMultiple(paths, nil) }
func (l *Lnk) AddRecursive(paths []string) error {
	return l.files.AddRecursiveWithProgress(paths, nil)
}
func (l *Lnk) AddRecursiveWithProgress(paths []string, progress ProgressCallback) error {
	return l.files.AddRecursiveWithProgress(paths, progress)
}
func (l *Lnk) PreviewAdd(paths []string, recursive bool) ([]string, error) {
	return l.files.PreviewAdd(paths, recursive)
}
func (l *Lnk) Remove(filePath string) error      { return l.files.Remove(filePath) }
func (l *Lnk) RemoveForce(filePath string) error { return l.files.RemoveForce(filePath) }

// --- Sync delegates ---

func (l *Lnk) Status() (*StatusInfo, error)      { return l.syncer.Status() }
func (l *Lnk) Diff(color bool) (string, error)   { return l.syncer.Diff(color) }
func (l *Lnk) Push(message string) error          { return l.syncer.Push(message) }
func (l *Lnk) Pull() ([]string, error)            { return l.syncer.Pull() }
func (l *Lnk) List() ([]string, error)            { return l.syncer.List() }
func (l *Lnk) GetCommits() ([]string, error)      { return l.syncer.GetCommits() }
func (l *Lnk) RestoreSymlinks() ([]string, error) { return l.syncer.RestoreSymlinks() }

// --- Bootstrap delegates ---

func (l *Lnk) FindBootstrapScript() (string, error) { return l.boot.FindScript() }
func (l *Lnk) RunBootstrapScript(scriptName string, stdout, stderr io.Writer, stdin io.Reader) error {
	return l.boot.RunScript(scriptName, stdout, stderr, stdin)
}

// --- Doctor delegates ---

func (l *Lnk) PreviewDoctor() (*DoctorResult, error) { return l.health.Preview() }
func (l *Lnk) Doctor() (*DoctorResult, error)         { return l.health.Fix() }

// --- Package-level helpers ---

// GetCurrentHostname returns the current system hostname.
func GetCurrentHostname() (string, error) {
	hostname, err := os.Hostname()
	if err != nil {
		return "", fmt.Errorf("failed to get hostname: %w", err)
	}
	return hostname, nil
}

// GetRepoPath returns the path to the lnk repository directory.
// It respects XDG_CONFIG_HOME if set, otherwise defaults to ~/.config/lnk.
func GetRepoPath() string {
	xdgConfig := os.Getenv("XDG_CONFIG_HOME")
	if xdgConfig == "" {
		homeDir, err := os.UserHomeDir()
		if err != nil {
			xdgConfig = "."
		} else {
			xdgConfig = filepath.Join(homeDir, ".config")
		}
	}
	return filepath.Join(xdgConfig, "lnk")
}
