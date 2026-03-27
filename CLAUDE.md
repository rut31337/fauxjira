# fauxjira

Single-binary Jira API simulator with web UI. Go + SQLite + htmx.

## Build & Test

```bash
make check        # fmt + lint + test + build
make test         # go test -v -race ./...
make lint         # golangci-lint run ./...
go build -o fauxjira .
```

## Run

```bash
FAUXJIRA_ADMIN_PASSWORD=admin123 FAUXJIRA_USER_PASSWORD=user123 ./fauxjira
# Listens on :6778 by default (FAUXJIRA_PORT to override)
```

## Architecture

- Single Go package (`package main`), no internal packages
- `config.go` - env var config
- `db.go` / `models.go` - SQLite layer, all CRUD
- `auth.go` - HTTP Basic Auth middleware, cookie sessions
- `api_issues.go` / `api_search.go` / `api_users.go` / `api_admin.go` - REST API
- `jql.go` - JQL parser (field = "value" with AND/OR)
- `web.go` - Web UI handlers, embedded templates/static/logo via `embed.FS`
- `templates/` - Go html/template files
- `static/` - CSS, htmx

## Key Patterns

- All routes registered via `Register*Routes(mux, db)` functions in `main.go`
- API auth: HTTP Basic Auth. Web auth: cookie `fauxjira_user`
- Ticket keys: `FJ-1`, `FJ-2`, etc. (sequential via `ticket_seq` table)
- Test helper `testDB(t)` in `db_test.go` creates temp DB with seeded users
- Test helper `testServer(t)` in `api_test.go` creates mux with routes

## Deploy

```bash
export KUBECONFIG=~/secrets/ocpvdev01.dal13.infra.demo.redhat.com.kubeconfig
ansible-playbook ansible/deploy-fauxjira.yml
```

Builds on-cluster via OpenShift BuildConfig from GitHub repo.
