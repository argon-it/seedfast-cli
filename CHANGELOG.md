# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

## [1.1.20] - 2025-10-23

### Added
- Display database connection info (masked DSN) at start of `seed` command
- Show database name and connection string with masked credentials
- Improved scope acceptance prompt with clear options (yes/no/feedback)

### Changed
- Enhanced user prompt for seeding scope with color-coded options
- Users can now see exactly which database they're seeding before starting
- Better guidance for accepting, rejecting, or providing feedback on scope

### Removed
- Removed Go test files from the project

## [1.1.6] - 2025-10-19

### Fixed
- **Critical**: Fix token corruption - never decode keys containing "token"
- Tokens that start with letter bytes (like 0x62='b') no longer incorrectly decoded
- `whoami` command now works correctly with proper access token

### Changed
- Key-name based decoding strategy: skip decode if key contains "token"
- Only decode non-token hex strings that start with JSON characters ({, [, ")
- More robust distinction between tokens and hex-encoded JSON data

## [1.1.5] - 2025-10-19

### Fixed
- **Critical**: Fix access tokens being corrupted by hex decoding on macOS 26.0
- Smart hex decoding - only decode if result starts with readable character (JSON, text)
- Access tokens and other hex strings now preserved as-is
- `whoami` command now correctly fetches user email from backend

### Changed
- Hex decoding only applies to JSON objects/arrays and text strings
- Tokens remain in their original hex string format

## [1.1.4] - 2025-10-19

### Fixed
- **Critical**: Fix auth state persistence on macOS 26.0 - decode hex-encoded keychain data
- macOS 26.0's `security find-generic-password -w` returns hex instead of plain text
- Auth state now correctly loads after login on macOS 26.0

### Added
- Automatic hex decoding for keychain data on macOS
- `isHexString()` helper to detect hex-encoded strings
- Verbose logging shows both hex and decoded values

## [1.1.3] - 2025-10-19

### Fixed
- **Critical**: Fix verbose mode not working - now checks SEEDFAST_VERBOSE dynamically
- Show raw keychain output to diagnose "invalid character 'b' after top-level value" error
- Verbose logging now actually appears when --verbose flag is used

### Added
- Display raw keychain data (first 100 chars) in verbose mode
- Show both raw and trimmed output from security command
- Hex dump of failed JSON data for debugging

## [1.1.2] - 2025-10-19

### Added
- Comprehensive verbose logging for macOS keychain debugging
- `--verbose` flag for `whoami`, `connect`, and `seed` commands
- Debug output shows auth state load/save flow step-by-step
- Environment variable `SEEDFAST_VERBOSE=1` enables verbose mode globally

### Changed
- Security backend (macOS) now logs all Set/Get/Delete operations in verbose mode
- Auth storage module logs all Load/Save operations with detailed state info
- Commands show auth.Load() errors when `--verbose` flag is used

## [1.1.1] - 2025-10-19

### Fixed
- Add error checking when saving auth state after login
- Show detailed error message if auth state save fails
- Add verbose logging for debugging auth state persistence issues
- Improve error messages in security_darwin.go to show which key failed

### Changed
- Login command now stops with error if auth.Save() fails (was silently ignored)
- Verbose mode (`--verbose`) shows auth state save progress for debugging

## [1.1.0] - 2025-10-19

### Fixed
- **Critical**: Fix auth state not persisting after login on macOS 26.0
- Handle "key not found" errors gracefully in LoadAuthState (treat as empty, not error)
- Fix `whoami` and `connect` commands showing "not logged in" after successful login
- Properly distinguish between "key doesn't exist yet" vs "real keychain error"

### Changed
- LoadAuthState returns nil (no data) instead of error when key doesn't exist
- Better error handling for first-run scenarios

## [1.0.9] - 2025-10-19

### Fixed
- **Critical**: Fix keychain on macOS 26.0 by using native `security` command directly
- Bypass keyring library which doesn't work on macOS 26.0
- Use macOS built-in security command for all keychain operations

### Changed
- macOS now uses `security` command directly (100% compatible with all macOS versions)
- No external dependencies required - works out of the box
- Windows continues using keyring library with WinCredBackend

## [1.0.8] - 2025-10-19

### Fixed
- Add PassBackend support for macOS 26.0 where native Keychain API is unavailable
- Provide clear error message with installation instructions for 'pass' utility
- Fix "No directory provided for file keyring" error on macOS 26.0

### Added
- Support for 'pass' (password store) as secure alternative on macOS 26.0+
- Helpful error message: install pass with `brew install pass gnupg`

### Changed
- macOS backend priority: Keychain first, then pass (both are secure)
- No file-based storage fallback - only encrypted backends

## [1.0.7] - 2025-10-19

### Fixed
- Fix "Specified keyring backend not available" on macOS 26.0 and newer
- Remove hardcoded AllowedBackends restriction to support future macOS versions
- Let keyring library automatically select best available backend

### Changed
- Keyring now adapts to platform-specific backends automatically
- Better compatibility with macOS updates

## [1.0.6] - 2025-10-19

### Fixed
- Fix infinite polling loop when authentication errors occur during login
- Stop silently ignoring errors in `PollLogin` during device authorization
- Display authentication errors immediately instead of continuing to poll

### Added
- Add `--verbose` flag to `login` command for debugging authentication issues
- Show device ID, poll interval, and polling status in verbose mode
- Better error messages when authentication fails

### Changed
- Login command now fails fast when errors occur instead of infinite polling
- Improved debugging experience for authentication troubleshooting

## [1.0.5] - 2025-10-19

### Fixed
- **Critical**: Replace all `MustGetManager()` calls with graceful `GetManager()` error handling
- Fix panic "Specified keyring backend not available" on unsupported platforms (e.g., Linux, MSYS)
- Add proper error handling in all 17 keychain access points across 8 files:
  - `cmd/login.go`: Handle keychain errors during login timeout cleanup
  - `cmd/connect.go`: Display user-friendly message when keychain unavailable
  - `cmd/logout.go`: Gracefully handle keychain errors during logout
  - `cmd/seed.go`: Handle keychain errors when loading DSN and access token
  - `internal/auth/storage.go`: Add error handling for Load/Save/Clear operations
  - `internal/auth/service.go`: Fix 11 keychain calls in auth service methods
- Improve error messages for unsupported platforms (Linux, MSYS/Git Bash on Windows)
- Preserve `MustGetManager()` function for backward compatibility (no longer used internally)

### Changed
- All keychain operations now return errors instead of panicking
- Better UX: informative error messages instead of crashes on unsupported systems

## [1.0.4] - 2025-10-19

### Fixed
- Fix nil pointer dereference in keychain manager after failed initialization
- Replace sync.Once with mutex to allow retry on initialization failure
- Properly handle keychain initialization errors during login flow

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

[Unreleased]: https://github.com/argon-it/seedfast-cli/compare/v1.1.1...HEAD
[1.1.1]: https://github.com/argon-it/seedfast-cli/compare/v1.1.0...v1.1.1
[1.1.0]: https://github.com/argon-it/seedfast-cli/compare/v1.0.9...v1.1.0
[1.0.9]: https://github.com/argon-it/seedfast-cli/compare/v1.0.8...v1.0.9
[1.0.8]: https://github.com/argon-it/seedfast-cli/compare/v1.0.7...v1.0.8
[1.0.7]: https://github.com/argon-it/seedfast-cli/compare/v1.0.6...v1.0.7
[1.0.6]: https://github.com/argon-it/seedfast-cli/compare/v1.0.5...v1.0.6
[1.0.5]: https://github.com/argon-it/seedfast-cli/compare/v1.0.4...v1.0.5
[1.0.4]: https://github.com/argon-it/seedfast-cli/compare/v1.0.3...v1.0.4
[1.0.3]: https://github.com/argon-it/seedfast-cli/compare/v1.0.2...v1.0.3
[1.0.2]: https://github.com/argon-it/seedfast-cli/compare/v1.0.1...v1.0.2
[1.0.1]: https://github.com/argon-it/seedfast-cli/compare/v1.0.0...v1.0.1
[1.0.0]: https://github.com/argon-it/seedfast-cli/releases/tag/v1.0.0
