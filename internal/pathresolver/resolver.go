package pathresolver

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// Resolver implements the models.PathResolver interface
type Resolver struct{}

// New creates a new PathResolver instance
func New() *Resolver {
	return &Resolver{}
}

// GetRepoStoragePath returns the base path where lnk repositories are stored
// This is based on XDG Base Directory specification
func (r *Resolver) GetRepoStoragePath() (string, error) {
	xdgConfig := os.Getenv("XDG_CONFIG_HOME")
	if xdgConfig == "" {
		homeDir, err := os.UserHomeDir()
		if err != nil {
			return "", fmt.Errorf("failed to get home directory: %w", err)
		}
		xdgConfig = filepath.Join(homeDir, ".config")
	}
	return filepath.Join(xdgConfig, "lnk"), nil
}

// GetFileStoragePathInRepo returns the path where a file should be stored in the repository
func (r *Resolver) GetFileStoragePathInRepo(repoPath, host, relativePath string) (string, error) {
	hostPath, err := r.GetHostStoragePath(repoPath, host)
	if err != nil {
		return "", err
	}
	return filepath.Join(hostPath, relativePath), nil
}

// GetTrackingFilePath returns the path to the tracking file for a host
func (r *Resolver) GetTrackingFilePath(repoPath, host string) (string, error) {
	var fileName string
	if host == "" {
		// Common configuration
		fileName = ".lnk"
	} else {
		// Host-specific configuration
		fileName = ".lnk." + host
	}
	return filepath.Join(repoPath, fileName), nil
}

// GetHomePath returns the user's home directory path
func (r *Resolver) GetHomePath() (string, error) {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("failed to get home directory: %w", err)
	}
	return homeDir, nil
}

// GetRelativePathFromHome converts an absolute path to relative from home directory
// This is migrated from the original getRelativePath function
func (r *Resolver) GetRelativePathFromHome(absPath string) (string, error) {
	homeDir, err := r.GetHomePath()
	if err != nil {
		return "", err
	}

	// Check if the file is under home directory
	relPath, err := filepath.Rel(homeDir, absPath)
	if err != nil {
		return "", fmt.Errorf("failed to get relative path: %w", err)
	}

	// If the relative path starts with "..", the file is outside home directory
	// In this case, use the absolute path as relative (without the leading slash)
	if strings.HasPrefix(relPath, "..") {
		// Use absolute path but remove leading slash and drive letter (for cross-platform)
		cleanPath := strings.TrimPrefix(absPath, "/")
		return cleanPath, nil
	}

	return relPath, nil
}

// GetAbsolutePathInHome converts a relative path to absolute within home directory
func (r *Resolver) GetAbsolutePathInHome(relPath string) (string, error) {
	homeDir, err := r.GetHomePath()
	if err != nil {
		return "", err
	}

	// If the relative path looks like an absolute path (starts with / or drive letter),
	// it's probably a file outside home directory
	if filepath.IsAbs(relPath) {
		return relPath, nil
	}

	// If it starts with a drive letter on Windows or looks like an absolute path,
	// treat it as absolute
	if len(relPath) > 0 && !strings.HasPrefix(relPath, ".") {
		// Check if it looks like an absolute path stored without leading slash
		// This handles paths like "etc/hosts" which should become "/etc/hosts"
		if strings.HasPrefix(relPath, "etc/") ||
			strings.HasPrefix(relPath, "usr/") ||
			strings.HasPrefix(relPath, "var/") ||
			strings.HasPrefix(relPath, "opt/") ||
			strings.HasPrefix(relPath, "tmp/") {
			// Reconstruct the absolute path
			return "/" + relPath, nil
		}
		// Windows drive patterns like "C:" or contains drive separator
		if strings.Contains(relPath, ":") {
			return relPath, nil
		}
	}

	return filepath.Join(homeDir, relPath), nil
}

// GetHostStoragePath returns the directory where files for a host are stored
// This is migrated from the original getHostStoragePath method
func (r *Resolver) GetHostStoragePath(repoPath, host string) (string, error) {
	if host == "" {
		// Common configuration - store in root of repo
		return repoPath, nil
	}
	// Host-specific configuration - store in host subdirectory
	return filepath.Join(repoPath, host+".lnk"), nil
}

// IsUnderHome checks if a path is under the home directory
func (r *Resolver) IsUnderHome(path string) (bool, error) {
	homeDir, err := r.GetHomePath()
	if err != nil {
		return false, err
	}

	// Clean both paths to handle relative components like .. and .
	cleanPath := filepath.Clean(path)
	cleanHome := filepath.Clean(homeDir)

	// Get relative path
	relPath, err := filepath.Rel(cleanHome, cleanPath)
	if err != nil {
		return false, nil // If we can't get relative path, assume not under home
	}

	// If relative path starts with "..", it's outside home directory
	return !strings.HasPrefix(relPath, ".."), nil
}
