# Installation Guide

## Homebrew (macOS and Linux)

The easiest way to install Seedfast CLI is via Homebrew:

```bash
# Install (automatically taps the repository)
brew install argon-it/tap/seedfast

# Verify installation
seedfast version
```

Or if you prefer to tap first:

```bash
brew tap argon-it/tap
brew install seedfast
```

### Updating

```bash
brew update
brew upgrade seedfast
```

### Uninstalling

```bash
brew uninstall seedfast
brew untap argon-it/tap
```

## Pre-built Binaries

Download the latest release for your platform:

**Latest Release:** https://github.com/argon-it/seedfast-cli/releases/latest

### macOS

```bash
# Intel
curl -L -o seedfast.tar.gz https://github.com/argon-it/seedfast-cli/releases/download/v0.1.0/seedfast-cli_Darwin_x86_64.tar.gz
tar -xzf seedfast.tar.gz
sudo mv seedfast /usr/local/bin/

# Apple Silicon (M1/M2/M3)
curl -L -o seedfast.tar.gz https://github.com/argon-it/seedfast-cli/releases/download/v0.1.0/seedfast-cli_Darwin_arm64.tar.gz
tar -xzf seedfast.tar.gz
sudo mv seedfast /usr/local/bin/
```

### Linux

```bash
# AMD64
curl -L -o seedfast.tar.gz https://github.com/argon-it/seedfast-cli/releases/download/v0.1.0/seedfast-cli_Linux_x86_64.tar.gz
tar -xzf seedfast.tar.gz
sudo mv seedfast /usr/local/bin/

# ARM64
curl -L -o seedfast.tar.gz https://github.com/argon-it/seedfast-cli/releases/download/v0.1.0/seedfast-cli_Linux_arm64.tar.gz
tar -xzf seedfast.tar.gz
sudo mv seedfast /usr/local/bin/
```

### Windows

1. Download [seedfast-cli_Windows_x86_64.zip](https://github.com/argon-it/seedfast-cli/releases/download/v0.1.0/seedfast-cli_Windows_x86_64.zip)
2. Extract `seedfast.exe`
3. Add to PATH or move to a directory in PATH

## From Source

Requirements:
- Go 1.24 or later
- Git

```bash
# Clone repository
git clone https://github.com/argon-it/seedfast-cli.git
cd seedfast-cli

# Build
go build -o seedfast

# Install (optional)
go install
```

## Docker

Currently not available. Follow [#1](https://github.com/argon-it/seedfast-cli/issues) for updates.

## Verification

After installation, verify it works:

```bash
seedfast version
```

You should see output like:
```
seedfast 0.1.0
backend unknown
```

## Next Steps

1. Authenticate: `seedfast login`
2. Connect to database: `seedfast connect`
3. Seed database: `seedfast seed`

See [README.md](README.md) for full documentation.
