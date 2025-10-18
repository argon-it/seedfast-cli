# Release Guide

## Prerequisites

- GoReleaser installed (v2.12.5 or later)
- GitHub Personal Access Token with `repo` scope
- Clean git working directory

## Release Process

### 1. Update Version

Update version in relevant files if needed (currently pulled from git tags).

### 2. Create and Push Tag

```bash
# Create tag
git tag -a v0.1.0 -m "Release v0.1.0"

# Push tag to remote
git push origin v0.1.0
```

### 3. Create Release

For GitHub release:

```bash
# Set GitHub token
export GITHUB_TOKEN="your_github_token"

# Create release
goreleaser release --clean
```

For local build only (no GitHub release):

```bash
# Build all platforms without publishing
goreleaser release --snapshot --clean --skip=publish
```

### 4. Test Release

Artifacts will be in `dist/` directory:
- `seedfast-cli_Darwin_arm64.tar.gz` - macOS ARM64
- `seedfast-cli_Darwin_x86_64.tar.gz` - macOS Intel
- `seedfast-cli_Linux_arm64.tar.gz` - Linux ARM64
- `seedfast-cli_Linux_x86_64.tar.gz` - Linux AMD64
- `seedfast-cli_Windows_x86_64.zip` - Windows AMD64

## Configuration

GoReleaser config is in [.goreleaser.yaml](.goreleaser.yaml).

Key settings:
- Builds for: Linux, macOS, Windows
- Architectures: amd64, arm64 (except Windows ARM64)
- Archives: tar.gz (Linux/macOS), zip (Windows)
- CGO disabled for static binaries

## Troubleshooting

### "git doesn't contain any tags"
```bash
git tag v0.0.1
```

### "dirty git state"
```bash
git status
git add .
git commit -m "commit message"
```

### Test without publishing
```bash
goreleaser release --snapshot --clean --skip=publish
```
