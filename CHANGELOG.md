# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

## [1.0.3] - 2025-10-19

### Fixed
- Fix keychain initialization panic on first run
- Replace `MustGetManager()` with `GetManager()` for graceful error handling
- Properly initialize keychain before saving tokens during login

## [1.0.2] - 2025-10-19

### Changed
- ⚠️ **Deprecated**: This version introduced an insecure encrypted file storage fallback
- Please upgrade to v1.0.3 which properly fixes the keychain initialization issue

## [1.0.1] - 2025-10-19

### Fixed
- Add `-trimpath` flag to remove absolute build paths from binaries
- Fixes panic with keychain on macOS when using bottles from GitHub Releases

## [1.0.0] - 2025-10-19

### Added
- Initial release of Seedfast CLI
- `seedfast login` command for OAuth-style device flow authentication
- `seedfast connect` command for PostgreSQL database connection configuration
- `seedfast seed` command for AI-powered database seeding
- `seedfast whoami` command to check current authentication status
- `seedfast logout` command to clear stored credentials
- `seedfast version` command to display CLI and backend versions
- OS-level keychain integration for secure credential storage (macOS Keychain, Windows Credential Manager)
- gRPC bidirectional streaming for real-time backend communication
- Interactive terminal UI with spinners and real-time progress tracking
- Concurrent SQL execution with configurable worker pool (4 workers)
- Automatic schema detection and relationship handling
- Support for macOS (Intel & Apple Silicon), Linux, and Windows
- Homebrew tap for easy installation on macOS
- MIT License
- Comprehensive README documentation
- GoReleaser configuration for automated releases

### Technical
- Static binary compilation (`CGO_ENABLED=0`) - no C dependencies required
- Pre-compiled bottles for Homebrew to avoid Xcode/CLT requirement
- pgx v5 PostgreSQL driver for efficient database operations
- Cobra framework for CLI structure
- pterm for rich terminal output
- Secure token storage using OS native keychains

[Unreleased]: https://github.com/argon-it/seedfast-cli/compare/v1.0.3...HEAD
[1.0.3]: https://github.com/argon-it/seedfast-cli/compare/v1.0.2...v1.0.3
[1.0.2]: https://github.com/argon-it/seedfast-cli/compare/v1.0.1...v1.0.2
[1.0.1]: https://github.com/argon-it/seedfast-cli/compare/v1.0.0...v1.0.1
[1.0.0]: https://github.com/argon-it/seedfast-cli/releases/tag/v1.0.0
