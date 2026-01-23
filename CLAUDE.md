# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

`gml` is a Gmail CLI client built with Go that provides command-line access to Gmail using the Gmail API. It supports OAuth2 and Service Account authentication methods.

## Build and Development Commands

### Building

```bash
# Build the binary to ./bin/gml
make build

# Install development tools (goreleaser)
make tools
```

### Release Management

The project uses GitHub Actions with GoReleaser for automated releases:

```bash
# Show what release would create (dry run)
make release type=patch dryrun=true

# Create and push a new version tag
make release type=patch dryrun=false   # v1.2.3 -> v1.2.4
make release type=minor dryrun=false   # v1.2.3 -> v1.3.0
make release type=major dryrun=false   # v1.2.3 -> v2.0.0

# Re-release an existing tag (deletes and recreates)
make re-release tag=v1.2.3 dryrun=false
make re-release dryrun=false  # Uses most recent tag
```

When a tag is pushed, GitHub Actions automatically builds binaries for multiple platforms using GoReleaser.

### Testing the CLI

```bash
# Run the built binary
./bin/gml --help

# Test authentication
./bin/gml auth

# List messages
./bin/gml list
./bin/gml list -l INBOX -n 50

# Get a specific message
./bin/gml get <message-id>
```

## Architecture

### Package Structure

```
gml/
├── main.go                 # Entry point, sets version info
├── cmd/                    # Cobra CLI commands (thin layer)
│   ├── root.go            # Root command, config loading, error handling
│   ├── auth.go            # OAuth authentication command
│   ├── list.go            # List messages command (delegates to internal/gml)
│   ├── get.go             # Get message command (delegates to internal/gml)
│   └── version.go         # Version command
├── internal/
│   ├── gml/               # Core application logic
│   │   ├── config.go      # Config file handling (TOML)
│   │   ├── service.go     # Main service orchestration
│   │   ├── labels.go      # Label operations (fetch, resolve, map)
│   │   ├── messages.go    # Message operations (list, get, parse)
│   │   └── format.go      # Output formatting (JSON, table)
│   ├── google/            # Google API integration
│   │   ├── auth.go        # OAuth and Service Account auth
│   │   └── gmail.go       # Gmail API service wrapper
│   └── version/           # Version information
│       └── version.go
```

### Authentication Flow

The application supports two authentication methods:

1. **OAuth2** (default): Interactive browser-based authentication
   - Runs a local HTTP server on a random port to receive the OAuth callback
   - Stores token in `user_credentials` path (default: `~/.config/gml/token.json`)
   - Uses `gmail.GmailReadonlyScope` (read-only access)

2. **Service Account**: For server-side or automated use
   - Sets `GOOGLE_APPLICATION_CREDENTIALS` environment variable
   - Uses Application Default Credentials

Authentication is abstracted through the `Authenticator` interface in `internal/google/auth.go`, with concrete implementations:
- `OAuthAuthenticator`: Browser-based OAuth flow
- `ServiceAccountAuthenticator`: Service account credentials

### Configuration

Configuration is loaded via Viper from `~/.config/gml/config.toml`:

```toml
auth_type = "oauth"  # or "service_account"
application_credentials = "/path/to/credentials.json"
user_credentials = "/path/to/token.json"
```

The `cmd/root.go` `initConfig()` function handles loading, and configuration is optional for commands like `version`.

### Service Initialization

The `gml.Service` struct (in `internal/gml/service.go`) is the main orchestrator:
1. Takes a `Config` and selects the appropriate `Authenticator`
2. Creates a `google.GmailService` wrapper around the Gmail API client
3. Commands use this service to interact with Gmail

### Cobra Best Practices

The codebase follows Cobra best practices:
- **RunE over Run**: All commands use `RunE` to return errors instead of `log.Fatalf`
- **cmd.Context()**: Commands use `cmd.Context()` instead of `context.Background()` for proper cancellation
- **Business logic separation**: cmd/ package is thin; business logic lives in internal/gml/
- **No package-level variables**: Flags are retrieved locally in command functions
- **Testable output**: Commands use `cmd.OutOrStdout()` for testable output
- **Error handling**: Root command uses `SilenceErrors` and `SilenceUsage` for clean error display

### Message Handling

Business logic is separated into focused modules in `internal/gml/`:

- **messages.go**:
  - `ListMessages()`: Fetches messages with pagination, supports query, labels, field filtering
  - `GetMessage()`: Retrieves a single message by ID with full details
  - `ParseFields()`: Parses comma-separated field strings
  - Body extraction with MIME type handling (text/plain, text/html)

- **labels.go**:
  - `LabelIndex`: Fast lookup structure for label names/IDs
  - `FetchLabelIndex()`: Fetches all labels and builds index
  - `ResolveLabelIDs()`: Converts label names to IDs (supports system and custom labels)
  - `MapLabelIDsToNames()`: Converts IDs to human-readable names

- **format.go**:
  - `FormatMessageList()`: Outputs messages as JSON or table
  - `FormatMessageDetail()`: Outputs single message as JSON or text
  - Table formatting with configurable field display

### Version Information

Version info is injected at build time via `-ldflags` in GoReleaser:
- `version.Version`: Git tag
- `version.CommitSHA`: Git commit hash
- `version.BuildTime`: Build timestamp

In development builds, defaults to "dev" / "unknown".

## Key Dependencies

- `spf13/cobra`: CLI framework
- `spf13/viper`: Configuration management
- `google.golang.org/api/gmail/v1`: Gmail API client
- `golang.org/x/oauth2`: OAuth2 authentication
- `olekukonko/tablewriter`: Table formatting for list output

## Development Notes

- The application uses read-only Gmail scope (`GmailReadonlyScope`)
- OAuth callback uses a dynamically allocated port to avoid conflicts
- Cross-platform browser launching is handled in `openBrowser()` (Darwin, Linux, Windows)
- All API interactions are context-aware for proper cancellation and timeouts
