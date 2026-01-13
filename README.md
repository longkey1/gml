# gml

Gmail CLI client - A command-line tool for interacting with Gmail.

## Installation

### Using Go

```bash
go install github.com/longkey1/gml@latest
```

### Using Homebrew (coming soon)

```bash
brew install longkey1/tap/gml
```

## Setup

### 1. Create Google Cloud Project and OAuth Credentials

1. Go to [Google Cloud Console](https://console.cloud.google.com/)
2. Create a new project or select an existing one
3. Enable the Gmail API
4. Create OAuth 2.0 credentials (Desktop application type)
5. Download the credentials JSON file

### 2. Create Configuration File

Create `~/.config/gml/config.toml`:

```toml
auth_type = "oauth"
application_credentials = "/path/to/credentials.json"
user_credentials = "/path/to/token.json"
```

### 3. Authenticate

```bash
gml auth
```

This will open your browser for Google OAuth authentication.

## Usage

### List Messages

```bash
# List recent messages
gml list

# List unread messages
gml list -u

# Search messages
gml list -q "from:example@gmail.com"

# Limit results
gml list -n 20

# Specify fields to include (available: id,from,to,subject,date,labels,snippet,body)
gml list -f id,from,subject,body

# Output as JSON
gml list --format json
```

### Get Message

```bash
# Get message by ID with full body
gml get <message-id>

# Output as JSON
gml get <message-id> --format json
```

### Version

```bash
gml version
```

## Configuration Options

| Option | Description |
|--------|-------------|
| `auth_type` | Authentication type: `oauth` or `service_account` |
| `application_credentials` | Path to OAuth client credentials JSON file |
| `user_credentials` | Path to store OAuth user token (for OAuth auth type) |

## License

Apache License 2.0
