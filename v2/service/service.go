// Package service implements the v2 lnk command semantics.
package service

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"slices"
	"sort"
	"strings"

	bootpkg "github.com/yarlson/lnk/internal/bootstrapper"
	filemgr "github.com/yarlson/lnk/internal/filemanager"
	fspkg "github.com/yarlson/lnk/internal/fs"
	gitpkg "github.com/yarlson/lnk/internal/git"
	"github.com/yarlson/lnk/internal/initializer"
	"github.com/yarlson/lnk/internal/lnk"
	"github.com/yarlson/lnk/internal/tracker"
)

const (
	repoMarkerFile    = ".lnkrepo"
	repoMarkerVersion = "version=2\n"
)

// Service owns the v2 CLI semantics while reusing the existing low-level git
// and filesystem collaborators.
type Service struct {
	repoPath string
}

// ScopeList describes tracked items for one storage scope.
type ScopeList struct {
	Name  string
	Items []string
}

// ListResult contains tracked items grouped by storage scope.
type ListResult struct {
	Scopes []ScopeList
}

// RestoreInfo reports machine-state changes made or planned by restore/update.
type RestoreInfo struct {
	Restored []string
	BackedUp []string
}

// ScopeDoctorResult captures doctor findings for one profile or storage scope.
type ScopeDoctorResult struct {
	Name           string
	InvalidEntries []string
	BrokenSymlinks []string
}

// OwnershipCollision describes a path claimed by more than one scope.
type OwnershipCollision struct {
	Path   string
	Scopes []string
}

// DoctorReport captures read-only or fix-mode doctor results.
type DoctorReport struct {
	Mode                    string
	ScopeResults            []ScopeDoctorResult
	Collisions              []OwnershipCollision
	MarkerMissing           bool
	MarkerFixed             bool
	BrokenSymlinkFixSkipped bool
}

// HasIssues reports whether the doctor found actionable issues.
func (r DoctorReport) HasIssues() bool {
	if r.MarkerMissing || len(r.Collisions) > 0 {
		return true
	}
	for _, result := range r.ScopeResults {
		if len(result.InvalidEntries) > 0 || len(result.BrokenSymlinks) > 0 {
			return true
		}
	}
	return false
}

// New returns a v2 service for the resolved repo path.
func New(repoPath string) *Service {
	return &Service{repoPath: repoPath}
}

// ResolveRepoPath resolves the repo path from explicit flag or environment.
func ResolveRepoPath(explicit string) string {
	if explicit != "" {
		return explicit
	}
	if env := os.Getenv("LNK_REPO"); env != "" {
		return env
	}
	return lnk.GetRepoPath()
}

// RepoPath returns the configured repo path.
func (s *Service) RepoPath() string {
	return s.repoPath
}

// Init creates a new local repo when needed.
func (s *Service) Init(ctx context.Context) error {
	g := gitpkg.New(s.repoPath)
	if g.IsGitRepository() {
		initSvc := initializer.New(s.repoPath, g, tracker.New(s.repoPath, ""))
		if err := initSvc.Init(); err != nil {
			return err
		}
		return nil
	}

	initSvc := initializer.New(s.repoPath, g, tracker.New(s.repoPath, ""))
	if err := initSvc.Init(); err != nil {
		return err
	}
	if err := s.writeMarkerFile(); err != nil {
		return err
	}
	if err := s.stagePaths(ctx, repoMarkerFile); err != nil {
		return err
	}
	if err := g.Commit("lnk: initialize repository"); err != nil {
		return err
	}
	return nil
}

// Clone clones a remote repo and optionally runs bootstrap.
func (s *Service) Clone(ctx context.Context, url string, runBootstrap bool, stdout, stderr io.Writer, stdin io.Reader) (bool, error) {
	g := gitpkg.New(s.repoPath)
	if err := g.Clone(url); err != nil {
		return false, err
	}

	if !runBootstrap {
		return false, nil
	}

	runner := bootpkg.New(s.repoPath, g)
	script, err := runner.FindScript()
	if err != nil {
		return false, err
	}
	if script == "" {
		return false, nil
	}

	if err := runner.RunScript(script, stdout, stderr, stdin); err != nil {
		return true, err
	}
	return true, nil
}

// Add tracks one or more paths in common or one host scope.
func (s *Service) Add(ctx context.Context, host string, paths []string) error {
	if err := s.requireGitRepo(); err != nil {
		return err
	}
	if len(paths) == 0 {
		return fmt.Errorf("no paths provided")
	}

	seen := make(map[string]struct{}, len(paths))
	for _, input := range paths {
		relativePath, _, err := homeRelativePath(input)
		if err != nil {
			return err
		}
		if _, ok := seen[relativePath]; ok {
			return fmt.Errorf("duplicate path in one add invocation: %s", relativePath)
		}
		seen[relativePath] = struct{}{}

		owner, err := s.findOwner(relativePath)
		if err != nil {
			return err
		}
		if owner != nil {
			return fmt.Errorf("path already managed in scope %s: %s", scopeName(owner.Host), relativePath)
		}
	}

	manager := s.fileManager(host)
	if len(paths) == 1 {
		return manager.Add(paths[0])
	}
	return manager.AddMultiple(paths, nil)
}

// Move transfers ownership of a tracked path between scopes.
func (s *Service) Move(ctx context.Context, input string, toHost string, toCommon bool) error {
	if err := s.requireGitRepo(); err != nil {
		return err
	}
	relativePath, _, err := homeRelativePath(input)
	if err != nil {
		return err
	}
	if (toCommon && toHost != "") || (!toCommon && toHost == "") {
		return fmt.Errorf("exactly one of --to-common or --to-host must be set")
	}

	owner, err := s.findOwner(relativePath)
	if err != nil {
		return err
	}
	if owner == nil {
		return fmt.Errorf("path is not managed: %s", relativePath)
	}

	targetHost := toHost
	if toCommon {
		targetHost = ""
	}
	if owner.Host == targetHost {
		return fmt.Errorf("path is already managed in scope %s: %s", scopeName(owner.Host), relativePath)
	}

	if otherOwner, err := s.findOwnerInScope(relativePath, targetHost); err != nil {
		return err
	} else if otherOwner != nil {
		return fmt.Errorf("target scope already owns path %s", relativePath)
	}

	sourceTracker := tracker.New(s.repoPath, owner.Host)
	targetTracker := tracker.New(s.repoPath, targetHost)
	sourcePath := filepath.Join(sourceTracker.HostStoragePath(), relativePath)
	targetPath := filepath.Join(targetTracker.HostStoragePath(), relativePath)

	info, err := os.Stat(sourcePath)
	if err != nil {
		return fmt.Errorf("stat source storage path: %w", err)
	}
	if err := os.MkdirAll(filepath.Dir(targetPath), 0o755); err != nil {
		return fmt.Errorf("create target storage directory: %w", err)
	}

	fs := fspkg.New()
	if err := fs.Move(sourcePath, targetPath, info); err != nil {
		return err
	}

	if err := sourceTracker.RemoveManagedItem(relativePath); err != nil {
		_ = fs.Move(targetPath, sourcePath, info)
		return fmt.Errorf("remove source tracked item: %w", err)
	}
	if err := targetTracker.AddManagedItem(relativePath); err != nil {
		_ = sourceTracker.AddManagedItem(relativePath)
		_ = fs.Move(targetPath, sourcePath, info)
		return fmt.Errorf("add target tracked item: %w", err)
	}

	if err := s.repointManagedSymlink(relativePath, targetPath); err != nil {
		return err
	}

	if err := s.stagePaths(ctx, relativePath, sourceTracker.LnkFileName(), targetTracker.LnkFileName(), hostGitPath(targetHost, relativePath)); err != nil {
		return err
	}
	return gitpkg.New(s.repoPath).Commit(fmt.Sprintf("lnk: moved %s to %s", filepath.Base(relativePath), scopeName(targetHost)))
}

// Remove stops managing a path and restores it to the current machine.
func (s *Service) Remove(ctx context.Context, host, input string) error {
	if err := s.requireGitRepo(); err != nil {
		return err
	}
	resolvedHost, err := s.resolveRemovalScope(input, host)
	if err != nil {
		return err
	}
	return s.fileManager(resolvedHost).Remove(input)
}

// Forget stops managing a path but preserves the stored repo copy.
func (s *Service) Forget(ctx context.Context, host, input string) error {
	if err := s.requireGitRepo(); err != nil {
		return err
	}
	resolvedHost, err := s.resolveRemovalScope(input, host)
	if err != nil {
		return err
	}
	relativePath, absPath, err := homeRelativePath(input)
	if err != nil {
		return err
	}

	tr := tracker.New(s.repoPath, resolvedHost)
	items, err := tr.GetManagedItems()
	if err != nil {
		return fmt.Errorf("read tracked items: %w", err)
	}
	if !slices.Contains(items, relativePath) {
		return fmt.Errorf("path is not managed in scope %s: %s", scopeName(resolvedHost), relativePath)
	}

	repoItem := filepath.Join(tr.HostStoragePath(), relativePath)
	if isManagedSymlink(absPath, repoItem) {
		if err := os.Remove(absPath); err != nil {
			return fmt.Errorf("remove managed symlink: %w", err)
		}
	}

	if err := tr.RemoveManagedItem(relativePath); err != nil {
		return fmt.Errorf("update tracked items: %w", err)
	}
	if err := s.stagePaths(ctx, tr.LnkFileName()); err != nil {
		return err
	}
	return gitpkg.New(s.repoPath).Commit(fmt.Sprintf("lnk: forgot %s", filepath.Base(relativePath)))
}

// List returns tracked items for common, one host, or all storage scopes.
func (s *Service) List(ctx context.Context, host string, all bool) (ListResult, error) {
	if err := s.requireGitRepo(); err != nil {
		return ListResult{}, err
	}
	if all && host != "" {
		return ListResult{}, fmt.Errorf("--host and --all cannot be combined")
	}

	var scopes []string
	if all {
		hosts, err := s.hosts()
		if err != nil {
			return ListResult{}, err
		}
		scopes = append(scopes, "")
		scopes = append(scopes, hosts...)
	} else {
		scopes = []string{host}
	}

	result := ListResult{Scopes: make([]ScopeList, 0, len(scopes))}
	for _, scope := range scopes {
		items, err := tracker.New(s.repoPath, scope).GetManagedItems()
		if err != nil {
			return ListResult{}, err
		}
		result.Scopes = append(result.Scopes, ScopeList{
			Name:  scopeName(scope),
			Items: items,
		})
	}
	return result, nil
}

// Status returns repo status information.
func (s *Service) Status(ctx context.Context) (*gitpkg.StatusInfo, error) {
	if err := s.requireGitRepo(); err != nil {
		return nil, err
	}
	return gitpkg.New(s.repoPath).GetStatus()
}

// Diff returns the uncommitted repo diff.
func (s *Service) Diff(ctx context.Context, color bool) (string, error) {
	if err := s.requireGitRepo(); err != nil {
		return "", err
	}
	return gitpkg.New(s.repoPath).Diff(color)
}

// Commit stages all repo changes and creates a commit.
func (s *Service) Commit(ctx context.Context, message string) error {
	if err := s.requireGitRepo(); err != nil {
		return err
	}
	g := gitpkg.New(s.repoPath)
	hasChanges, err := g.HasChanges()
	if err != nil {
		return err
	}
	if !hasChanges {
		return fmt.Errorf("no changes to commit")
	}
	if err := g.AddAll(); err != nil {
		return err
	}
	return g.Commit(message)
}

// Push pushes existing commits only.
func (s *Service) Push(ctx context.Context) error {
	if err := s.requireGitRepo(); err != nil {
		return err
	}
	g := gitpkg.New(s.repoPath)
	hasChanges, err := g.HasChanges()
	if err != nil {
		return err
	}
	if hasChanges {
		return fmt.Errorf("working tree is dirty; run 'lnk commit' or commit manually before push")
	}
	return g.Push()
}

// Pull updates the repo only.
func (s *Service) Pull(ctx context.Context) error {
	if err := s.requireGitRepo(); err != nil {
		return err
	}
	return gitpkg.New(s.repoPath).Pull()
}

// Restore applies the effective machine profile: common only, or common + host.
func (s *Service) Restore(ctx context.Context, host string, dryRun bool) (RestoreInfo, error) {
	if err := s.requireGitRepo(); err != nil {
		return RestoreInfo{}, err
	}
	collisions, err := s.scanCollisions()
	if err != nil {
		return RestoreInfo{}, err
	}
	if len(collisions) > 0 {
		return RestoreInfo{}, fmt.Errorf("restore blocked by duplicate ownership; run 'lnk doctor' first")
	}

	items, err := s.profileItems(host)
	if err != nil {
		return RestoreInfo{}, err
	}

	info := RestoreInfo{}
	fs := fspkg.New()
	for _, item := range items {
		if _, err := os.Stat(item.RepoPath); os.IsNotExist(err) {
			continue
		}
		if isManagedSymlink(item.LivePath, item.RepoPath) {
			continue
		}

		if _, err := os.Lstat(item.LivePath); err == nil {
			currentInfo, statErr := os.Lstat(item.LivePath)
			if statErr != nil {
				return RestoreInfo{}, fmt.Errorf("stat live path %s: %w", item.LivePath, statErr)
			}
			if currentInfo.Mode()&os.ModeSymlink == 0 {
				info.BackedUp = append(info.BackedUp, item.RelativePath)
				if !dryRun {
					backupPath := item.LivePath + ".lnk-backup"
					if _, err := os.Lstat(backupPath); err == nil {
						return RestoreInfo{}, fmt.Errorf("backup path already exists: %s", backupPath)
					}
					if err := os.Rename(item.LivePath, backupPath); err != nil {
						return RestoreInfo{}, fmt.Errorf("backup existing file %s: %w", item.LivePath, err)
					}
				}
			} else if !dryRun {
				if err := os.Remove(item.LivePath); err != nil {
					return RestoreInfo{}, fmt.Errorf("remove stale symlink %s: %w", item.LivePath, err)
				}
			}
		}

		info.Restored = append(info.Restored, item.RelativePath)
		if dryRun {
			continue
		}
		if err := os.MkdirAll(filepath.Dir(item.LivePath), 0o755); err != nil {
			return RestoreInfo{}, fmt.Errorf("create live parent directory: %w", err)
		}
		if err := fs.CreateSymlink(item.RepoPath, item.LivePath); err != nil {
			return RestoreInfo{}, err
		}
	}
	return info, nil
}

// Update pulls repo changes and then restores the effective machine profile.
func (s *Service) Update(ctx context.Context, host string) (RestoreInfo, error) {
	if err := s.Pull(ctx); err != nil {
		return RestoreInfo{}, err
	}
	return s.Restore(ctx, host, false)
}

// Doctor scans or repairs repo and profile state.
func (s *Service) Doctor(ctx context.Context, host string, all, fix bool) (DoctorReport, error) {
	if err := s.requireGitRepo(); err != nil {
		return DoctorReport{}, err
	}
	if all && host != "" {
		return DoctorReport{}, fmt.Errorf("--host and --all cannot be combined")
	}

	report := DoctorReport{}
	if all {
		report.Mode = "all"
	} else if host != "" {
		report.Mode = "profile:" + host
	} else {
		report.Mode = "profile:common"
	}

	markerPath := filepath.Join(s.repoPath, repoMarkerFile)
	if _, err := os.Stat(markerPath); os.IsNotExist(err) {
		report.MarkerMissing = true
	}

	collisions, err := s.scanCollisions()
	if err != nil {
		return DoctorReport{}, err
	}
	report.Collisions = collisions

	scopes := []string{""}
	if all {
		hosts, hostErr := s.hosts()
		if hostErr != nil {
			return DoctorReport{}, hostErr
		}
		scopes = append(scopes, hosts...)
	} else if host != "" {
		scopes = append(scopes, host)
	}

	for _, scope := range scopes {
		result, resErr := s.scanInvalidEntries(scope)
		if resErr != nil {
			return DoctorReport{}, resErr
		}
		report.ScopeResults = append(report.ScopeResults, ScopeDoctorResult{
			Name:           scopeName(scope),
			InvalidEntries: result,
		})
	}

	if !all {
		broken, brokenErr := s.scanBrokenSymlinks(host)
		if brokenErr != nil {
			return DoctorReport{}, brokenErr
		}
		report.ScopeResults = append(report.ScopeResults, ScopeDoctorResult{
			Name:           profileName(host),
			BrokenSymlinks: broken,
		})
	}

	if !fix {
		return report, nil
	}

	if report.MarkerMissing {
		if err := s.writeMarkerFile(); err != nil {
			return DoctorReport{}, err
		}
		if err := s.stagePaths(ctx, repoMarkerFile); err != nil {
			return DoctorReport{}, err
		}
		report.MarkerFixed = true
	}

	stageSet := make(map[string]struct{})
	for _, result := range report.ScopeResults {
		scope := scopeFromName(result.Name)
		if len(result.InvalidEntries) == 0 || strings.HasPrefix(result.Name, "profile:") {
			continue
		}
		tr := tracker.New(s.repoPath, scope)
		items, err := tr.GetManagedItems()
		if err != nil {
			return DoctorReport{}, err
		}
		filtered := make([]string, 0, len(items))
		for _, item := range items {
			if !slices.Contains(result.InvalidEntries, item) {
				filtered = append(filtered, item)
			}
		}
		if err := tr.WriteManagedItems(filtered); err != nil {
			return DoctorReport{}, err
		}
		stageSet[tr.LnkFileName()] = struct{}{}
	}

	if !all {
		broken, err := s.scanBrokenSymlinks(host)
		if err != nil {
			return DoctorReport{}, err
		}
		if len(broken) > 0 {
			if _, err := s.Restore(ctx, host, false); err != nil {
				return DoctorReport{}, err
			}
		}
	} else {
		report.BrokenSymlinkFixSkipped = true
	}

	if len(stageSet) == 0 && !report.MarkerFixed {
		return report, nil
	}

	stagePaths := make([]string, 0, len(stageSet))
	for path := range stageSet {
		stagePaths = append(stagePaths, path)
	}
	if report.MarkerFixed {
		stagePaths = append(stagePaths, repoMarkerFile)
	}
	sort.Strings(stagePaths)
	if err := s.stagePaths(ctx, stagePaths...); err != nil {
		return DoctorReport{}, err
	}
	return report, gitpkg.New(s.repoPath).Commit("lnk: doctor fixes")
}

// Bootstrap runs bootstrap.sh explicitly.
func (s *Service) Bootstrap(ctx context.Context, stdout, stderr io.Writer, stdin io.Reader) (bool, error) {
	if err := s.requireGitRepo(); err != nil {
		return false, err
	}
	g := gitpkg.New(s.repoPath)
	runner := bootpkg.New(s.repoPath, g)
	script, err := runner.FindScript()
	if err != nil {
		return false, err
	}
	if script == "" {
		return false, nil
	}
	if err := runner.RunScript(script, stdout, stderr, stdin); err != nil {
		return true, err
	}
	return true, nil
}

type profileItem struct {
	RelativePath string
	RepoPath     string
	LivePath     string
}

type owner struct {
	Host string
}

func (s *Service) requireGitRepo() error {
	if !gitpkg.New(s.repoPath).IsGitRepository() {
		return fmt.Errorf("lnk repository not initialized: run 'lnk init' first")
	}
	return nil
}

func (s *Service) fileManager(host string) *filemgr.Manager {
	g := gitpkg.New(s.repoPath)
	fs := fspkg.New()
	tr := tracker.New(s.repoPath, host)
	return filemgr.New(s.repoPath, host, g, fs, tr)
}

func (s *Service) hosts() ([]string, error) {
	entries, err := os.ReadDir(s.repoPath)
	if err != nil {
		return nil, err
	}
	hosts := make([]string, 0)
	for _, entry := range entries {
		name := entry.Name()
		if strings.HasPrefix(name, ".lnk.") {
			hosts = append(hosts, strings.TrimPrefix(name, ".lnk."))
		}
	}
	sort.Strings(hosts)
	return hosts, nil
}

func (s *Service) findOwner(relativePath string) (*owner, error) {
	if owner, err := s.findOwnerInScope(relativePath, ""); err != nil {
		return nil, err
	} else if owner != nil {
		return owner, nil
	}
	hosts, err := s.hosts()
	if err != nil {
		return nil, err
	}
	for _, host := range hosts {
		if owner, ownerErr := s.findOwnerInScope(relativePath, host); ownerErr != nil {
			return nil, ownerErr
		} else if owner != nil {
			return owner, nil
		}
	}
	return nil, nil
}

func (s *Service) findOwnerInScope(relativePath, host string) (*owner, error) {
	items, err := tracker.New(s.repoPath, host).GetManagedItems()
	if err != nil {
		return nil, err
	}
	if slices.Contains(items, relativePath) {
		return &owner{Host: host}, nil
	}
	return nil, nil
}

func (s *Service) resolveRemovalScope(input, host string) (string, error) {
	relativePath, _, err := homeRelativePath(input)
	if err != nil {
		return "", err
	}
	if host != "" {
		owner, ownerErr := s.findOwnerInScope(relativePath, host)
		if ownerErr != nil {
			return "", ownerErr
		}
		if owner == nil {
			return "", fmt.Errorf("path is not managed in scope %s: %s", scopeName(host), relativePath)
		}
		return host, nil
	}

	owner, ownerErr := s.findOwner(relativePath)
	if ownerErr != nil {
		return "", ownerErr
	}
	if owner == nil {
		return "", fmt.Errorf("path is not managed: %s", relativePath)
	}
	if owner.Host != "" {
		return "", fmt.Errorf("path is managed in host scope %s; use --host %s", owner.Host, owner.Host)
	}
	return "", nil
}

func (s *Service) profileItems(host string) ([]profileItem, error) {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return nil, fmt.Errorf("resolve home directory: %w", err)
	}

	scopes := []string{""}
	if host != "" {
		scopes = append(scopes, host)
	}

	items := make([]profileItem, 0)
	seen := make(map[string]struct{})
	for _, scope := range scopes {
		tr := tracker.New(s.repoPath, scope)
		managedItems, trackErr := tr.GetManagedItems()
		if trackErr != nil {
			return nil, trackErr
		}
		for _, relativePath := range managedItems {
			if _, ok := seen[relativePath]; ok {
				return nil, fmt.Errorf("profile contains duplicate ownership for %s", relativePath)
			}
			seen[relativePath] = struct{}{}
			items = append(items, profileItem{
				RelativePath: relativePath,
				RepoPath:     filepath.Join(tr.HostStoragePath(), relativePath),
				LivePath:     filepath.Join(homeDir, relativePath),
			})
		}
	}

	sort.Slice(items, func(i, j int) bool {
		return items[i].RelativePath < items[j].RelativePath
	})
	return items, nil
}

func (s *Service) scanInvalidEntries(host string) ([]string, error) {
	tr := tracker.New(s.repoPath, host)
	items, err := tr.GetManagedItems()
	if err != nil {
		return nil, err
	}
	invalid := make([]string, 0)
	for _, item := range items {
		cleaned := filepath.Clean(item)
		if cleaned == "." || strings.HasPrefix(cleaned, "..") || filepath.IsAbs(cleaned) {
			invalid = append(invalid, item)
			continue
		}
		if _, err := os.Stat(filepath.Join(tr.HostStoragePath(), cleaned)); os.IsNotExist(err) {
			invalid = append(invalid, item)
		}
	}
	sort.Strings(invalid)
	return invalid, nil
}

func (s *Service) scanBrokenSymlinks(host string) ([]string, error) {
	items, err := s.profileItems(host)
	if err != nil {
		return nil, err
	}
	broken := make([]string, 0)
	for _, item := range items {
		if _, err := os.Stat(item.RepoPath); os.IsNotExist(err) {
			continue
		}
		if !isManagedSymlink(item.LivePath, item.RepoPath) {
			broken = append(broken, item.RelativePath)
		}
	}
	sort.Strings(broken)
	return broken, nil
}

func (s *Service) scanCollisions() ([]OwnershipCollision, error) {
	paths := make(map[string][]string)
	commonItems, err := tracker.New(s.repoPath, "").GetManagedItems()
	if err != nil {
		return nil, err
	}
	for _, item := range commonItems {
		paths[item] = append(paths[item], "common")
	}
	hosts, err := s.hosts()
	if err != nil {
		return nil, err
	}
	for _, host := range hosts {
		items, itemErr := tracker.New(s.repoPath, host).GetManagedItems()
		if itemErr != nil {
			return nil, itemErr
		}
		for _, item := range items {
			paths[item] = append(paths[item], scopeName(host))
		}
	}
	collisions := make([]OwnershipCollision, 0)
	for path, scopes := range paths {
		if len(scopes) > 1 {
			sort.Strings(scopes)
			collisions = append(collisions, OwnershipCollision{Path: path, Scopes: scopes})
		}
	}
	sort.Slice(collisions, func(i, j int) bool {
		return collisions[i].Path < collisions[j].Path
	})
	return collisions, nil
}

func (s *Service) repointManagedSymlink(relativePath, targetPath string) error {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return fmt.Errorf("resolve home directory: %w", err)
	}
	livePath := filepath.Join(homeDir, relativePath)
	if _, err := os.Lstat(livePath); errors.Is(err, os.ErrNotExist) {
		return nil
	}
	if !isManagedSymlink(livePath, filepath.Clean(filepath.Join(s.repoPath, relativePath))) &&
		!isManagedSymlink(livePath, filepath.Clean(filepath.Join(s.repoPath, filepath.Base(filepath.Dir(targetPath))))) {
		if info, err := os.Lstat(livePath); err == nil && info.Mode()&os.ModeSymlink == 0 {
			return nil
		}
	}
	if err := os.Remove(livePath); err != nil {
		return fmt.Errorf("remove existing symlink: %w", err)
	}
	return fspkg.New().CreateSymlink(targetPath, livePath)
}

func (s *Service) writeMarkerFile() error {
	return os.WriteFile(filepath.Join(s.repoPath, repoMarkerFile), []byte(repoMarkerVersion), 0o644)
}

func (s *Service) stagePaths(ctx context.Context, paths ...string) error {
	if len(paths) == 0 {
		return nil
	}
	args := []string{"add", "-A"}
	args = append(args, paths...)
	return s.execGit(ctx, args...)
}

func (s *Service) execGit(ctx context.Context, args ...string) error {
	cmd := exec.CommandContext(ctx, "git", args...)
	cmd.Dir = s.repoPath
	output, err := cmd.CombinedOutput()
	if err != nil {
		if len(output) > 0 {
			return fmt.Errorf("git %s: %s", strings.Join(args, " "), strings.TrimSpace(string(output)))
		}
		return fmt.Errorf("git %s: %w", strings.Join(args, " "), err)
	}
	return nil
}

func homeRelativePath(input string) (string, string, error) {
	absPath, err := filepath.Abs(input)
	if err != nil {
		return "", "", fmt.Errorf("resolve path %s: %w", input, err)
	}
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return "", "", fmt.Errorf("resolve home directory: %w", err)
	}
	relativePath, err := filepath.Rel(homeDir, absPath)
	if err != nil {
		return "", "", fmt.Errorf("resolve relative path %s: %w", input, err)
	}
	cleaned := filepath.Clean(relativePath)
	if cleaned == "." || strings.HasPrefix(cleaned, "..") || filepath.IsAbs(cleaned) {
		return "", "", fmt.Errorf("path must be inside $HOME: %s", input)
	}
	return cleaned, absPath, nil
}

func isManagedSymlink(livePath, expectedTarget string) bool {
	info, err := os.Lstat(livePath)
	if err != nil {
		return false
	}
	if info.Mode()&os.ModeSymlink == 0 {
		return false
	}
	target, err := os.Readlink(livePath)
	if err != nil {
		return false
	}
	if !filepath.IsAbs(target) {
		target = filepath.Join(filepath.Dir(livePath), target)
	}
	targetAbs, err := filepath.Abs(target)
	if err != nil {
		return false
	}
	expectedAbs, err := filepath.Abs(expectedTarget)
	if err != nil {
		return false
	}
	return targetAbs == expectedAbs
}

func hostGitPath(host, relativePath string) string {
	if host == "" {
		return relativePath
	}
	return filepath.Join(host+".lnk", relativePath)
}

func scopeName(host string) string {
	if host == "" {
		return "common"
	}
	return host
}

func profileName(host string) string {
	if host == "" {
		return "profile:common"
	}
	return "profile:" + host
}

func scopeFromName(name string) string {
	if name == "common" {
		return ""
	}
	return name
}
