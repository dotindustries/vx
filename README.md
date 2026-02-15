# vx

Vault-backed secret manager for monorepos. Resolves secrets from HashiCorp Vault and injects them as environment variables into any command.

## Install

```bash
brew install dotindustries/tap/vx
```

## Quick start

```bash
# Authenticate with Vault via OIDC
vx login

# Run a command with secrets injected
vx exec -- your-command --flag

# List resolved secrets for a workspace
vx list -w api
```

## Configuration

Root `vx.toml` — defines Vault connection, environments, shared secrets, and workspaces:

```toml
workspaces = [
  "packages/api/vx.toml",
  "web/vx.toml",
]

[vault]
address = "https://vault.example.com"
auth_method = "oidc"
auth_role = "admin"
base_path = "secret"

[environments]
default = "dev"
available = ["dev", "staging", "production"]

[secrets]
DATABASE_URL = "${env}/database/url"
OPENAI_API_KEY = "shared/openai/api_key"

[defaults]
NODE_ENV = "development"

[defaults.production]
NODE_ENV = "production"
```

Workspace `vx.toml` — adds workspace-specific secrets:

```toml
[secrets]
TURSO_PLATFORM_TOKEN = "${env}/database/platform_token"

[defaults]
SOME_KEY = "value"
```

## Features

- Workspace-scoped secret loading via `vx.toml`
- Parallel Vault reads
- Automatic token renewal daemon
- Environment-aware (dev, staging, prod)

## License

MIT
