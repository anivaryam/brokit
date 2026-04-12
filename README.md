# brokit

A package manager for [anivaryam](https://github.com/anivaryam)'s dev tools. Install, update, and manage all tools with a single command.

## Available Tools

| Tool | Description |
|------|-------------|
| [env-vault](https://github.com/anivaryam/env-vault) | Encrypted .env file manager powered by random-universe-cipher |
| [tunnel](https://github.com/anivaryam/tunnel) | Expose local services through a public tunnel |
| [merge-port](https://github.com/anivaryam/merge-port) | Local reverse proxy that merges multiple ports into one |
| [proc-compose](https://github.com/anivaryam/proc-compose) | Process runner and manager with daemon support |
| [proxy-relay](https://github.com/anivaryam/proxy-relay) | Authenticated SOCKS5/HTTP proxy client for routing traffic through a remote server |

## Install

### Linux / macOS

```sh
curl -sSfL https://raw.githubusercontent.com/anivaryam/brokit/main/install.sh | sh
```

By default this installs to `/usr/local/bin`. To change the install location:

```sh
INSTALL_DIR=~/.local/bin curl -sSfL https://raw.githubusercontent.com/anivaryam/brokit/main/install.sh | sh
```

### Windows (PowerShell)

```powershell
irm https://raw.githubusercontent.com/anivaryam/brokit/main/install.ps1 | iex
```

This installs to `%LOCALAPPDATA%\brokit\bin` and automatically adds it to your PATH.

### From Source

Requires [Go](https://go.dev/dl/) 1.22+.

```sh
git clone https://github.com/anivaryam/brokit.git
cd brokit
make install
```

## Usage

### Install tools

```sh
# Install a single tool
brokit install tunnel

# Install multiple tools
brokit install tunnel merge-port proc-compose env-vault

# Install all available tools
brokit install --all
```

### Update tools

```sh
# Update a specific tool
brokit update tunnel

# Update all installed tools
brokit update --all
```

### Force reinstall

```sh
brokit install --force tunnel
```

Reinstalls even if the tool is already installed. Useful when a binary is corrupted.

### Remove tools

```sh
brokit remove tunnel
```

### List tools

```sh
brokit list
```

```
TOOL          DESCRIPTION                                                STATUS         VERSION
env-vault     Encrypted .env file manager powered by random-universe-c…  installed      v0.1.0
merge-port    Local reverse proxy that merges multiple ports into one    installed      v0.2.1
proc-compose  Process runner and manager with daemon support             not installed  -
tunnel        Expose local services through a public tunnel              installed      v0.3.13
```

### Update brokit itself

```sh
brokit self-update
```

Downloads and replaces the running `brokit` binary with the latest release.

### Verbose / quiet mode

```sh
brokit install -v tunnel    # verbose: shows download URLs and HTTP status
brokit install -q --all     # quiet: only shows errors
```

### Short aliases

Every command has a short alias for convenience:

| Command   | Alias           |
|-----------|-----------------|
| `install` | `i`             |
| `update`  | `u`, `up`       |
| `remove`  | `rm`, `uninstall` |
| `list`    | `ls`            |

```sh
brokit i tunnel          # install
brokit u --all           # update all
brokit rm merge-port     # remove
brokit ls                # list
```

## How It Works

brokit is a lightweight CLI that manages tool binaries from GitHub Releases.

When you run `brokit install tunnel`, it:

1. Looks up the tool in the built-in registry
2. Queries the GitHub API for the latest release of [anivaryam/tunnel](https://github.com/anivaryam/tunnel)
3. Detects your OS and architecture
4. Downloads the correct archive (`.tar.gz` on Linux/macOS, `.zip` on Windows)
5. Extracts the binary and places it in the install directory
6. Records the installed version in a local state file

### File locations

| | Linux / macOS | Windows |
|---|---|---|
| Binaries | `~/.local/bin/` | `%LOCALAPPDATA%\brokit\bin\` |
| State file | `~/.local/share/brokit/state.json` | `%LOCALAPPDATA%\brokit\state.json` |

You can override the binary install directory with the `BROKIT_BIN` environment variable on any platform.

### Configuration

brokit supports optional TOML configuration for custom tools. Create `~/.config/brokit/tools.toml`:

```toml
[[tools]]
name = "my-tool"
repo = "username/my-tool"
description = "My custom tool"
```

### GitHub API rate limits

Unauthenticated requests to the GitHub API are limited to 60 per hour. If you hit the rate limit, set the `GITHUB_TOKEN` environment variable to increase the limit to 5,000 requests per hour:

```sh
export GITHUB_TOKEN=ghp_your_token_here
```

### Architecture

brokit is organized into modular packages:

| Package | Purpose |
|---------|---------|
| `downloader` | Fetches tool archives from GitHub Releases |
| `extractor` | Extracts archives (`.tar.gz`, `.zip`) |
| `registry` | Built-in tool registry and custom tool config |
| `state` | Tracks installed tools and versions |

### Supported platforms

| OS | Architecture |
|----|-------------|
| Linux | amd64, arm64 |
| macOS | amd64 (Intel), arm64 (Apple Silicon) |
| Windows | amd64 |

## License

MIT
