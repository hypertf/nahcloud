# NahCloud Repository Map

A mock cloud provider for testing Terraform configurations and infrastructure tooling.

---

## Architecture Overview

```
┌─────────────────────────────────────────────────────────────────────┐
│                         cmd/server/                                 │
│                    (Entry Point & Config)                           │
└─────────────────────────────────────────────────────────────────────┘
                                 │
                                 ▼
┌─────────────────────────────────────────────────────────────────────┐
│                            api/                                     │
│                  (HTTP Handlers & Router)                           │
└─────────────────────────────────────────────────────────────────────┘
                                 │
                                 ▼
┌─────────────────────────────────────────────────────────────────────┐
│                          service/                                   │
│                      (Business Logic)                               │
└─────────────────────────────────────────────────────────────────────┘
                                 │
                                 ▼
┌─────────────────────────────────────────────────────────────────────┐
│                          domain/                                    │
│                  (Models & Error Definitions)                       │
└─────────────────────────────────────────────────────────────────────┘
                                 │
                                 ▼
┌─────────────────────────────────────────────────────────────────────┐
│                       storage/sqlite/                               │
│                  (Repository Implementations)                       │
└─────────────────────────────────────────────────────────────────────┘
```

---

## Root Files

| File | Description |
|------|-------------|
| [go.mod](./go.mod) | Go module definition with dependencies (UUID, Gorilla Mux, SQLite3, Testify) |
| [go.sum](./go.sum) | Go dependency lock file with checksums |
| [Makefile](./Makefile) | Build automation: server build, testing, code quality, database ops, Docker |
| [README.md](./README.md) | Project documentation: features, API overview, configuration |

---

## `cmd/server/` - Application Entry Point

| File | Description |
|------|-------------|
| [main.go](./cmd/server/main.go) | Entry point: initializes DB, repositories, service layer, HTTP server with graceful shutdown |
| [config.go](./cmd/server/config.go) | Configuration via Spf13 Viper: flags, env vars, config files; priority order: flags > env > config file > defaults |

---

## `domain/` - Domain Models & Errors

| File | Description |
|------|-------------|
| [models.go](./domain/models.go) | Core domain structs: Organization, Project, Instance, Metadata, Bucket, Object, TFStateLock; request/response types |
| [errors.go](./domain/errors.go) | Structured error handling: NahError type, error codes (NOT_FOUND, ALREADY_EXISTS, INVALID_INPUT, etc.) |
| [errors_test.go](./domain/errors_test.go) | Unit tests for domain error functionality |

---

## `api/` - HTTP API Layer

| File | Description |
|------|-------------|
| [router.go](./api/router.go) | Gorilla Mux router setup; REST endpoints for all resources; CORS and logging middleware |
| [handlers.go](./api/handlers.go) | HTTP handlers for CRUD operations; authentication, error handling, JSON responses, URL param resolution |
| [tfstate_handlers.go](./api/tfstate_handlers.go) | Terraform state backend protocol: GET/POST/DELETE state, LOCK/UNLOCK operations |

---

## `service/` - Business Logic Layer

| File | Description |
|------|-------------|
| [service.go](./service/service.go) | Core service with repository interfaces; ID generation; interfaces for all repositories |
| [tfstate.go](./service/tfstate.go) | Terraform state management: GetTFState, SetTFState, DeleteTFState, lock operations |

---

## `storage/sqlite/` - Data Persistence Layer

| File | Description |
|------|-------------|
| [db.go](./storage/sqlite/db.go) | SQLite initialization, connection pooling, schema creation; enables foreign keys and WAL mode |
| [organization_repository.go](./storage/sqlite/organization_repository.go) | CRUD for organizations with constraint validation |
| [project_repository.go](./storage/sqlite/project_repository.go) | CRUD for projects; org_id scoping, unique constraint handling |
| [instance_repository.go](./storage/sqlite/instance_repository.go) | CRUD for compute instances |
| [metadata_repository.go](./storage/sqlite/metadata_repository.go) | CRUD for key-value metadata; prefix filtering; org-scoped Terraform state storage |
| [bucket_repository.go](./storage/sqlite/bucket_repository.go) | CRUD for storage buckets; project-scoped with name uniqueness |
| [object_repository.go](./storage/sqlite/object_repository.go) | CRUD for objects within buckets |
| [metadata_repository_test.go](./storage/sqlite/metadata_repository_test.go) | Unit tests for metadata repository |
| [testutil_test.go](./storage/sqlite/testutil_test.go) | Test utilities for database testing |

---

## `web/` - Web Console Frontend

| File | Description |
|------|-------------|
| [handlers.go](./web/handlers.go) | Web console HTTP handlers for BREAD operations (Browse, Read, Edit, Add, Delete); HTMX integration |
| [README.md](./web/README.md) | Web console documentation: BREAD operations, features, technology stack |
| [tailwind.config.js](./web/tailwind.config.js) | Tailwind CSS configuration |
| [package.json](./web/package.json) | Node.js config with Tailwind build/watch scripts |
| [package-lock.json](./web/package-lock.json) | Node.js dependency lock file |

### `web/static/` - Static Assets

| File | Description |
|------|-------------|
| [embed.go](./web/static/embed.go) | Go embed directives to include CSS and logo in binary |
| [input.css](./web/static/input.css) | Tailwind CSS source file |
| [output.css](./web/static/output.css) | Generated Tailwind CSS (minified, runtime) |
| [logo.png](./web/static/logo.png) | NahCloud logo image |

---

## `pkg/client/` - Go SDK

| File | Description |
|------|-------------|
| [client.go](./pkg/client/client.go) | Go SDK for NahCloud API: retry logic, configurable timeouts, bearer token auth |

---

## Resource Hierarchy

```
Organization
    │
    ├── Project
    │       │
    │       ├── Instance (compute)
    │       │
    │       └── Bucket
    │               │
    │               └── Object
    │
    └── Metadata (key-value, used for Terraform state)
```

---

## Key Features

- **Multi-tenant hierarchy**: Organizations → Projects → Instances/Buckets
- **Terraform HTTP state backend**: Full protocol implementation with locking
- **Web console**: HTML-based UI with HTMX for dynamic interactions
- **SQLite persistence**: Schema with foreign keys and WAL mode
- **Go SDK**: Client library with retry logic and authentication
