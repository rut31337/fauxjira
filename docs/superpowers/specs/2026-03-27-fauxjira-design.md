# fauxjira — Jira API Simulator

A lightweight, single-binary Jira API simulator for training and demos. Mimics Jira's REST API and visual style so mixed audiences (developers and non-technical stakeholders) get a realistic experience without needing real Jira accounts.

## Architecture

Single Go binary embedding all static assets (htmx, CSS, HTML templates, logo) via `embed.FS`. Serves both the Jira-compatible REST API and the web frontend on the same port (default 8080).

```
fauxjira (single binary)
├── REST API  (/rest/api/2/...)   ← Jira-compatible, external-facing
├── Web UI    (/)                 ← htmx + Go templates, Jira-styled
├── Admin API (/admin/...)        ← reset, user management
└── SQLite    (fauxjira.db)       ← file-based persistence
```

**Key technology choices:**

- **Go** — single binary, zero runtime dependencies, excellent HTTP stdlib, `embed.FS` for static assets
- **SQLite via `modernc.org/sqlite`** — pure Go (no CGO), easy cross-compilation for container images
- **htmx** — single embeddable JS file, no build step, dynamic UI via HTML attributes
- **Go `html/template`** — server-side rendering, htmx swaps HTML fragments for interactivity
- **Custom CSS** — Atlassian/Jira-inspired design language (blue navbar, avatar circles, status badges)

## Data Model

### Users

| Field | Type | Notes |
|-------|------|-------|
| username | string, unique | Login identifier |
| display_name | string | e.g. "Alice Chen" |
| password_hash | string | bcrypt hashed |
| role | string | "admin" or "user" |

### Tickets

| Field | Type | Notes |
|-------|------|-------|
| id | int, auto-increment | Internal ID |
| key | string, unique | "FJ-1", "FJ-2", auto-generated |
| summary | string | Ticket title |
| description | text | Ticket body |
| status | string | "To Do", "In Progress", "In Review", "Done" |
| assignee | string | References Users.username |
| reporter | string | References Users.username (creator) |
| labels | JSON array | e.g. ["bug", "infra"] |
| created_at | timestamp | Auto-set on creation |
| updated_at | timestamp | Auto-updated on edit |

**Statuses (fixed set):** To Do, In Progress, In Review, Done

**Ticket keys:** Auto-incrementing with project prefix "FJ" — `FJ-1`, `FJ-2`, etc.

## Jira-Compatible REST API

All endpoints require HTTP Basic Auth. Request/response JSON matches Jira's structure so existing Jira clients get expected field shapes.

### Endpoints

| Endpoint | Method | Purpose |
|----------|--------|---------|
| `/rest/api/2/issue` | POST | Create ticket |
| `/rest/api/2/issue/{key}` | GET | Get ticket |
| `/rest/api/2/issue/{key}` | PUT | Edit ticket |
| `/rest/api/2/issue/{key}` | DELETE | Delete ticket |
| `/rest/api/2/search?jql=...` | GET | Search with basic JQL |
| `/rest/api/2/user?username=...` | GET | User lookup |
| `/rest/api/2/serverInfo` | GET | Server info |

### Request/Response Examples

**Create ticket:**
```json
POST /rest/api/2/issue
{
  "fields": {
    "summary": "Setup monitoring",
    "description": "Configure alerting for production",
    "assignee": {"name": "alice"},
    "labels": ["infra"]
  }
}
```

**Response:**
```json
{
  "id": "1",
  "key": "FJ-1",
  "self": "http://fauxjira.example.com/rest/api/2/issue/FJ-1"
}
```

**Get ticket:**
```json
GET /rest/api/2/issue/FJ-1

{
  "id": "1",
  "key": "FJ-1",
  "fields": {
    "summary": "Setup monitoring",
    "description": "Configure alerting for production",
    "status": {"name": "To Do"},
    "assignee": {"name": "alice", "displayName": "Alice Chen"},
    "reporter": {"name": "admin", "displayName": "Admin"},
    "labels": ["infra"],
    "created": "2026-03-27T10:00:00.000+0000",
    "updated": "2026-03-27T10:00:00.000+0000"
  }
}
```

**Search:**
```
GET /rest/api/2/search?jql=status%20%3D%20%22To%20Do%22%20AND%20assignee%20%3D%20%22alice%22
```

### JQL Support

Basic field matching only:

- `status = "To Do"`
- `assignee = "alice"`
- `labels = "bug"`
- `reporter = "admin"`
- `AND` / `OR` connectors
- String equality comparisons only
- No functions, sub-queries, or `ORDER BY`

## Web Frontend

Jira-inspired visual design using the Atlassian color palette and layout conventions.

### Pages

**Login page:**
- Large fauxjira dinosaur logo (centered, prominent)
- "FAUXJIRA — Project Mismanagement Tool" branding
- Username/password form
- Clean, minimal layout

**Ticket list (main view):**
- Blue Atlassian-style navbar with small logo, nav links, user avatar circle
- Table view: key (linked), summary, status badge (colored, uppercase), assignee with avatar circle, label pills
- "Create" button in header
- Clickable rows open ticket detail

**Ticket detail:**
- Accessible by clicking a row or navigating to `/issue/{key}`
- Editable fields via htmx: summary, description (inline edit), status (dropdown swap), assignee (dropdown swap), labels (tag editor)
- Reporter shown but not editable
- Created/updated timestamps

**Create ticket:**
- Modal dialog (overlay on ticket list, Jira-style)
- Form fields: summary, description, assignee dropdown, labels input, status defaults to "To Do"

### Interactivity (htmx)

- Status dropdown: clicking the badge shows a dropdown, selecting a status posts to the API and swaps the badge HTML
- Assignee dropdown: same pattern
- Inline editing: click summary/description to edit in place
- Ticket list: htmx-powered filtering/search without full page reloads

## Logo

The fauxjira mascot (a dinosaur holding a "BUG" ticket with a "WONTFIX" tag) is stored as `assets/fauxjira-logo-raw.png`. During build:

1. Crop to the green rectangle area
2. Make the green background transparent (ImageMagick: `convert -fuzz 20% -transparent '#00ff00'`)
3. Embed the processed PNG as `assets/fauxjira-logo.png`

Placement:
- **Login page:** large, centered above the login form
- **Navbar:** small (24-32px height) in the left corner

## Authentication

HTTP Basic Auth on all API endpoints and session-based auth for the web UI.

### Accounts

- **Admin account:** username `admin`, password provided via `FAUXJIRA_ADMIN_PASSWORD` env var. If not set, a random password is generated and printed to stdout on first run.
- **Demo users:** 1-2 users created on first run. All demo users share a single password, provided via `FAUXJIRA_USER_PASSWORD` env var. If not set, a random shared password is generated and printed to stdout.

All credentials are exported as variables for demo consumption (see Deployment section).

### Admin Operations

| Endpoint | Method | Purpose |
|----------|--------|---------|
| `/admin/reset` | POST | Wipe all tickets, regenerate demo users with new passwords |
| `/admin/users` | GET | List all users and their roles |

Admin endpoints require admin role authentication.

## Deployment

### Container Image

Multi-stage Dockerfile:

```dockerfile
# Build stage
FROM golang:1.22 AS build
WORKDIR /src
COPY . .
RUN go build -o fauxjira .

# Runtime stage
FROM scratch
COPY --from=build /src/fauxjira /fauxjira
EXPOSE 8080
ENTRYPOINT ["/fauxjira"]
```

### Environment Variables

| Variable | Default | Purpose |
|----------|---------|---------|
| `FAUXJIRA_ADMIN_PASSWORD` | (random) | Admin account password |
| `FAUXJIRA_USER_PASSWORD` | (random) | Shared password for all demo users |
| `FAUXJIRA_DB_PATH` | `/data/fauxjira.db` | SQLite database location |
| `FAUXJIRA_PORT` | `8080` | Listen port |

### Ansible Playbook

An Ansible playbook/role deploys fauxjira to OpenShift:

1. Creates namespace/project (or uses existing one)
2. Builds or pulls the container image
3. Creates Deployment with a PVC mounted at `/data` for SQLite persistence
4. Creates Service (port 8080) + Route for external HTTPS access
5. Generates random passwords for admin + 2 demo users if not provided as variables
6. Seeds demo users into fauxjira via the REST API after pod is ready
7. Registers/exports the following variables for demo consumption:
   - `fauxjira_url` — the Route URL
   - `fauxjira_admin_password`
   - `fauxjira_user_password` — shared password for all demo users
   - `fauxjira_user1` / `fauxjira_user2` — demo usernames

### Cleanup

- `POST /admin/reset` wipes tickets and regenerates demo users
- Ansible playbook can call this endpoint or fully tear down the OpenShift resources

## Out of Scope

The following are explicitly not included in the initial build but could be added later:

- Sprints / board view (kanban)
- Comments on tickets
- Priority field
- Ticket types (Bug, Story, Task)
- Attachments / file uploads
- Notifications / webhooks
- Multi-project support (single "FJ" project only)
- Watchers
- Transitions / workflow engine
