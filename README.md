> **Warning**
> This project is complete create by Claude, not a single line of code is written by my hand. Please use it with caution.

# GTA (Grant Temporary Access)

GTA is a command-line tool for managing temporary IAM roles across cloud providers. It currently supports Google Cloud Platform (GCP) and allows you to grant temporary permissions that are automatically revoked when the program exits.

## Features

- Grant temporary IAM roles in GCP
- Support for multiple roles in a single command
- Automatic role revocation on program exit or interrupt
- Configurable time-to-live (TTL) for permissions
- Support for both user accounts and service accounts
- Automatic current user detection
- List and track temporary permissions
- Unique identifiers for each temporary binding for easy cleanup
- Multiple verbosity levels
- Flexible output formats (plain text and JSON)
- Comprehensive logging with source location tracking

## Installation

```bash
go install github.com/yckao/gta@latest
```

## Prerequisites

- Go 1.16 or later
- GCP credentials configured (either through gcloud or service account)
- Appropriate permissions to manage IAM policies in your GCP project

## Usage

### Global Options

The following options are available for all commands:

- `--verbosity, -v`: Set the logging level (default: info)
  - `debug`: Show all messages including debug information
  - `info`: Show informational messages and above
  - `warn`: Show warning messages and above
  - `error`: Show only error messages
- `--format`: Set the output format (default: plain)
  - `plain`: Human-readable text format with timestamps
  - `json`: JSON format for machine processing
- `--quiet, -q`: Quiet mode, only show errors
- `--config`: Config file path (default: $HOME/.gta.yaml)

### Grant Temporary Access

Grant temporary roles to a user:

```bash
# Grant a single role to yourself (plain text output)
gta grant roles/viewer \
    --provider=gcp \
    --project=my-project-id \
    --ttl=1h
```

Options:
- `--provider, -c`: Cloud provider (currently supports: gcp)
- `--project, -p`: Project ID (required)
- `--user, -u`: User or service account to grant the role to (defaults to current user)
- `--ttl, -t`: Time-to-live for the granted permission (default: 1h)

The permissions will be automatically revoked when:
1. The specified TTL expires
2. The program receives an interrupt signal (Ctrl+C)
3. The program exits

### List Temporary Bindings

List all temporary role bindings:

```bash
# List bindings in plain text format
gta list --provider=gcp --project=my-project-id
```

This is useful for:
- Tracking active temporary permissions
- Finding permissions that weren't properly cleaned up
- Auditing temporary access grants
- Integration with other tools (using JSON output)

## Configuration

GTA supports configuration through:
1. Command line flags
2. Environment variables
3. Configuration file (`$HOME/.gta.yaml`)

Example configuration file:
```yaml
project: default-project-id
verbosity: debug  # Set default verbosity level
format: json     # Set default output format
```

## License

MIT 