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

# Search messages
gml list -q "from:example@gmail.com"

# Set page size (automatically fetches all pages)
gml list -n 100

# Filter by label
gml list -l INBOX
gml list -l INBOX -l UNREAD
gml list -l "My Project"       # Custom labels resolved by name

# Specify fields to include (available: id,from,to,subject,date,labels,snippet,body)
gml list -f id,from,subject,body

# Output as JSON
gml list --format json
```

Common labels: `INBOX`, `SENT`, `DRAFT`, `SPAM`, `TRASH`, `STARRED`, `UNREAD`, `IMPORTANT`, `CATEGORY_PERSONAL`, `CATEGORY_SOCIAL`, `CATEGORY_PROMOTIONS`, `CATEGORY_UPDATES`, `CATEGORY_FORUMS`

Note: The list command automatically fetches all matching messages using pagination. The `-n` option sets the page size per API request (default: 10, max: 500).

### Get Message

```bash
# Get message by ID with full body
gml get <message-id>

# Labels in output are shown by name (system and custom labels)

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
