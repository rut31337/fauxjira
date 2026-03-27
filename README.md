# fauxjira

A lightweight Jira API simulator with a Jira-styled web UI, built for training and demos. Single Go binary, SQLite persistence, deployable to OpenShift.

![fauxjira logo](assets/fauxjira-logo.png)

## Features

- **Jira-compatible REST API** (`/rest/api/2/...`) for issue CRUD and search
- **JQL support** for filtering issues by status, assignee, reporter, and labels
- **Web UI** with login, ticket board, ticket detail, inline status/assignee editing
- **htmx-powered** frontend with no JavaScript framework
- **Basic Auth** for API access, cookie sessions for web UI
- **Admin endpoints** for data reset and user listing
- **Single binary** with embedded static assets and templates

## Quick Start

```bash
# Build
go build -o fauxjira .

# Run (generates random passwords if not set)
./fauxjira

# Or set passwords explicitly
FAUXJIRA_ADMIN_PASSWORD=admin123 FAUXJIRA_USER_PASSWORD=user123 ./fauxjira
```

The server starts on port `6778` by default. Open `http://localhost:6778/login` for the web UI.

### Demo Users

| User  | Display Name | Role  |
|-------|-------------|-------|
| admin | Admin       | admin |
| alice | Alice Chen  | user  |
| bob   | Bob Park    | user  |

## Configuration

| Environment Variable       | Default        | Description          |
|---------------------------|----------------|----------------------|
| `FAUXJIRA_PORT`           | `6778`         | HTTP listen port     |
| `FAUXJIRA_DB_PATH`        | `fauxjira.db`  | SQLite database path |
| `FAUXJIRA_ADMIN_PASSWORD` | *(generated)*  | Admin user password  |
| `FAUXJIRA_USER_PASSWORD`  | *(generated)*  | Demo user password   |

## API Examples

```bash
# Server info (no auth required)
curl http://localhost:6778/rest/api/2/serverInfo

# Create an issue
curl -u alice:user123 -X POST -H "Content-Type: application/json" \
  -d '{"fields":{"summary":"Bug report","assignee":{"name":"bob"},"labels":["bug"]}}' \
  http://localhost:6778/rest/api/2/issue

# Search with JQL
curl -u alice:user123 'http://localhost:6778/rest/api/2/search?jql=assignee+%3D+%22bob%22'

# Get issue
curl -u alice:user123 http://localhost:6778/rest/api/2/issue/FJ-1

# Update issue
curl -u alice:user123 -X PUT -H "Content-Type: application/json" \
  -d '{"fields":{"status":{"name":"In Progress"}}}' \
  http://localhost:6778/rest/api/2/issue/FJ-1

# Delete issue
curl -u admin:admin123 -X DELETE http://localhost:6778/rest/api/2/issue/FJ-1

# Admin reset (wipes all data, re-seeds users)
curl -u admin:admin123 -X POST http://localhost:6778/admin/reset
```

## Container

```bash
# Build
docker build -t fauxjira .

# Run
docker run -p 6778:6778 \
  -e FAUXJIRA_ADMIN_PASSWORD=admin123 \
  -e FAUXJIRA_USER_PASSWORD=user123 \
  fauxjira
```

## Deploy to OpenShift

The Ansible playbook builds the image on-cluster from the GitHub repo using a BuildConfig. It only triggers a new build when the git source has changed.

### Prerequisites

- `KUBECONFIG` env var pointing to a cluster with the internal image registry enabled
- `ansible` and `kubernetes.core` collection installed (`make venv` sets this up)

### Quick deploy

```bash
export KUBECONFIG=/path/to/your/kubeconfig
ansible-playbook ansible/deploy-fauxjira.yml
```

### Local overrides

Copy the example vars file and customize:

```bash
cp ansible/vars.local.yml.example ansible/vars.local.yml
```

Available overrides in `vars.local.yml`:

```yaml
fauxjira_route_host: fauxjira.apps.mycluster.example.com
fauxjira_admin_password: my-admin-pass
fauxjira_user_password: my-user-pass
fauxjira_namespace: fauxjira
fauxjira_port: "6778"
fauxjira_storage_size: 1Gi
```

Override via extra vars:

```bash
ansible-playbook ansible/deploy-fauxjira.yml \
  -e fauxjira_namespace=my-fauxjira \
  -e fauxjira_admin_password=secretpass
```

### Notes

- Uses `Recreate` deployment strategy (required for RWO PVC)
- Build is skipped if the latest git commit already has a successful build
- Passwords are stable defaults — override in `vars.local.yml` for your environment
- `vars.local.yml` is gitignored

## Development

```bash
# Set up Python venv for ansible-lint
make venv

# Run tests
make test

# Lint (Go + Ansible)
make lint
make ansible-lint

# Format code
make fmt

# Full check (fmt + lint + test + build + ansible-lint)
make check

# Install pre-commit hooks
make pre-commit-install
```

## Tech Stack

- Go 1.25+
- SQLite via `modernc.org/sqlite` (pure Go, no CGO)
- htmx for dynamic UI
- Go `html/template` for server-side rendering
- `embed.FS` for embedded static assets
- Ansible + `kubernetes.core` for OpenShift deployment
