# Seedfast CLI

> AI-powered PostgreSQL database seeding tool

Seedfast CLI is a command-line tool that generates realistic test data for PostgreSQL databases. It connects to a backend service that uses AI to understand your database schema and create meaningful, contextually appropriate seed data.

## Features

- **AI-Powered Planning**: Automatically analyzes your database schema and generates an intelligent seeding plan
- **Realistic Data**: Creates contextually appropriate test data based on table relationships and constraints
- **Interactive UI**: Rich terminal interface with real-time progress tracking
- **Concurrent Execution**: Multi-worker architecture for fast data generation
- **Secure Authentication**: OAuth-style device flow with OS-level credential storage
- **Schema-Aware**: Respects foreign keys, constraints, and relationships between tables

## Installation

### Homebrew (macOS and Linux)

```bash
brew install argon-it/tap/seedfast
```

### Download Binary

Download the latest release for your platform from [GitHub Releases](https://github.com/argon-it/seedfast-cli/releases/latest).

### From Source

Requires Go 1.24+:

```bash
git clone https://github.com/argon-it/seedfast-cli.git
cd seedfast-cli
go build -o seedfast
```

## Quick Start

### 1. Authenticate

```bash
seedfast login
```

This opens your browser for authentication. The CLI polls the backend until you complete the login.

### 2. Connect to Database

```bash
seedfast connect
```

Enter your PostgreSQL connection string (DSN):
```
postgres://user:password@localhost:5432/dbname?sslmode=disable
```

### 3. Seed Your Database

```bash
seedfast seed
```

The CLI will:
1. Analyze your database schema
2. Present a seeding plan for your approval
3. Generate and insert realistic test data
4. Show real-time progress for each table

## Commands

```
seedfast login      # Authenticate with the backend service
seedfast connect    # Configure database connection
seedfast seed       # Start the seeding process
seedfast whoami     # Check authentication status
seedfast logout     # Clear stored credentials
seedfast version    # Show version information
```

## Configuration

### Environment Variables

- `SEEDFAST_DSN` - PostgreSQL connection string (overrides stored DSN from keychain)
- `DATABASE_URL` - Alternative PostgreSQL connection string (fallback if SEEDFAST_DSN not set)


## How It Works

1. **Authentication**: The CLI uses an OAuth-style device flow to securely authenticate with the backend
2. **Schema Analysis**: The backend analyzes your PostgreSQL schema to understand tables, relationships, and constraints
3. **AI Planning**: An AI planner determines the optimal seeding strategy and generates realistic data
4. **Execution**: The CLI receives SQL tasks via gRPC and executes them locally against your database
5. **Progress Tracking**: Real-time UI shows progress for each table being seeded


## Requirements

- Go 1.25.1+ (for building from source)
- Active internet connection for backend communication

## Troubleshooting

### Authentication Issues

```bash
# Check current auth status
seedfast whoami

# Re-authenticate
seedfast logout
seedfast login
```

### Database Connection

```bash
# Test connection
seedfast connect
```


## Contributing

Contributions are welcome! Please open an issue or submit a pull request.

## License

See [LICENSE](LICENSE) file for details.


## Acknowledgments

Built with:
- [Cobra](https://github.com/spf13/cobra) - CLI framework
- [pterm](https://github.com/pterm/pterm) - Terminal UI
- [pgx](https://github.com/jackc/pgx) - PostgreSQL driver
- [gRPC](https://grpc.io/) - RPC framework
