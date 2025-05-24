# Release Process

This document describes how to create releases for the lnk project using GoReleaser.

## Prerequisites

- Push access to the main repository
- Git tags pushed to GitHub trigger releases automatically
- GoReleaser is configured in `.goreleaser.yml`
- GitHub Actions will handle the release process
- Access to the [homebrew-lnk](https://github.com/yarlson/homebrew-lnk) tap repository
- **Personal Access Token** set up as `HOMEBREW_TAP_TOKEN` secret (see Setup section)

## Setup (One-time)

### GitHub Personal Access Token

For GoReleaser to update the Homebrew formula, you need a Personal Access Token:

1. Go to GitHub Settings → Developer settings → Personal access tokens → Tokens (classic)
2. Click "Generate new token" → "Generate new token (classic)"
3. Name: "GoReleaser Homebrew Access"
4. Scopes: Select `repo` (Full control of private repositories)
5. Generate and copy the token
6. In your `yarlson/lnk` repository:
   - Go to Settings → Secrets and variables → Actions
   - Add new repository secret: `HOMEBREW_TAP_TOKEN`
   - Paste the token as the value

This allows GoReleaser to automatically update the Homebrew formula in [homebrew-lnk](https://github.com/yarlson/homebrew-lnk).

## Creating a Release

### 1. Ensure everything is ready

```bash
# Run all quality checks
make check

# Test GoReleaser configuration
make goreleaser-check

# Test build process
make goreleaser-snapshot
```

### 2. Create and push a version tag

```bash
# Create a new tag (replace x.y.z with actual version)
git tag -a v1.0.0 -m "Release v1.0.0"

# Push the tag to trigger the release
git push origin v1.0.0
```

### 3. Monitor the release

- GitHub Actions will automatically build and release when the tag is pushed
- Check the [Actions tab](https://github.com/yarlson/lnk/actions) for build status
- The release will appear in [GitHub Releases](https://github.com/yarlson/lnk/releases)
- The Homebrew formula will be automatically updated in [homebrew-lnk](https://github.com/yarlson/homebrew-lnk)

## What GoReleaser Does

1. **Builds binaries** for multiple platforms:
   - Linux (amd64, arm64)
   - macOS (amd64, arm64)
   - Windows (amd64)

2. **Creates archives** with consistent naming:
   - `lnk_Linux_x86_64.tar.gz`
   - `lnk_Darwin_arm64.tar.gz`
   - etc.

3. **Generates checksums** for verification

4. **Creates GitHub release** with:
   - Automatic changelog from conventional commits
   - Installation instructions
   - Download links for all platforms

5. **Updates Homebrew formula** automatically in the [homebrew-lnk](https://github.com/yarlson/homebrew-lnk) tap

## Manual Release (if needed)

If you need to create a release manually:

```bash
# Export GitHub token
export GITHUB_TOKEN="your_token_here"

# Create release (requires a git tag)
goreleaser release --clean
```

## Testing Releases Locally

```bash
# Test the build process without releasing
make goreleaser-snapshot

# Built artifacts will be in dist/
ls -la dist/

# Test a binary
./dist/lnk_<platform>/lnk --version
```

## Installation Methods

After a release is published, users can install lnk using multiple methods:

### 1. Shell Script (Recommended)
```bash
curl -sSL https://raw.githubusercontent.com/yarlson/lnk/main/install.sh | bash
```

### 2. Homebrew (macOS/Linux)
```bash
brew tap yarlson/lnk
brew install lnk
```

### 3. Manual Download
```bash
# Download from GitHub releases
wget https://github.com/yarlson/lnk/releases/latest/download/lnk_Linux_x86_64.tar.gz
tar -xzf lnk_Linux_x86_64.tar.gz
sudo mv lnk /usr/local/bin/
```

## Version Numbering

We use [Semantic Versioning](https://semver.org/):

- `v1.0.0` - Major release (breaking changes)
- `v1.1.0` - Minor release (new features, backward compatible)
- `v1.1.1` - Patch release (bug fixes)

## Changelog

GoReleaser automatically generates changelogs from git commits using conventional commit format:

- `feat:` - New features
- `fix:` - Bug fixes
- `docs:` - Documentation changes (excluded from changelog)
- `test:` - Test changes (excluded from changelog)
- `ci:` - CI changes (excluded from changelog)

## Homebrew Tap

The Homebrew formula is automatically maintained in the [homebrew-lnk](https://github.com/yarlson/homebrew-lnk) repository. When a new release is created:

1. GoReleaser automatically creates/updates the formula
2. The formula is committed to the tap repository
3. Users can immediately install the new version via `brew install yarlson/lnk/lnk`

## Troubleshooting

### Release failed to create

1. Check that the tag follows the format `vX.Y.Z`
2. Ensure GitHub Actions has proper permissions
3. Check the Actions log for detailed error messages

### Missing binaries in release

1. Verify GoReleaser configuration: `make goreleaser-check`
2. Test build locally: `make goreleaser-snapshot`
3. Check the build matrix in `.goreleaser.yml`

### Changelog is empty

1. Ensure commits follow conventional commit format
2. Check that there are commits since the last tag
3. Verify changelog configuration in `.goreleaser.yml`

### Homebrew formula not updated

1. Check that the GITHUB_TOKEN has access to the homebrew-lnk repository
2. Verify the repository name and owner in `.goreleaser.yml`
3. Check the release workflow logs for Homebrew-related errors
4. Ensure the homebrew-lnk repository exists and is accessible 