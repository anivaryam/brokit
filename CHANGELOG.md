# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

### Added
- `make lint` target for running golangci-lint
- Binary checksum verification (SHA256) for downloaded releases
- Shared `internal/errors` package for common error handling

### Fixed
- Unchecked error returns from `state.Set()` and `state.Remove()`
- Duplicate `wrapNetworkError` implementations
- Unused mock types in installer tests

### Security
- Added security note to README about `GITHUB_TOKEN` handling

## [0.3.3] - 2024-04-13

### Fixed
- Set `StatePath` on `Installer` after creation

## [0.3.2] - 2024-04-10

### Fixed
- Resolve nil pointer in `State.Save` calls

## [0.3.1] - 2024-04-08

### Added
- Test coverage tracking with `covermode=atomic`

## [0.3.0] - 2024-04-08

### Added
- Self-update command (`brokit self-update`)
- Progress indicator for downloads
- Proper timeouts for HTTP clients
- `GITHUB_TOKEN` support for higher API rate limits
- Windows architecture detection improvements

### Fixed
- "text file busy" error when installing over a running binary

## [0.2.2] - 2024-04-08

### Fixed
- Remove command silently succeeding without removing binaries

## [0.2.1] - 2024-04-08

### Added
- `proxy-relay` to the tool registry

## [0.2.0] - 2024-04-08

### Added
- `env-vault` to the tool registry

## [0.1.3] - 2024-04-07

### Fixed
- Fallback to `PROCESSOR_ARCHITECTURE` when `OSArchitecture` is empty

## [0.1.2] - 2024-04-07

### Added
- golangci-lint configuration

## [0.1.1] - 2024-04-07

### Fixed
- Windows architecture detection with 32-bit support

## 0.1.0 - 2024-04-07

Initial release of brokit — package manager for anivaryam's dev tools.

### Added
- Install, update, remove, and list commands
- Support for Linux (amd64, arm64), macOS (amd64, arm64), Windows (amd64)
- TOML configuration for custom tools
- Verbose and quiet output modes
