# NahCloud

A fake cloud API for testing Terraform tooling without provisioning real infrastructure.

## Why

You want to test your Terraform providers, CI/CD pipelines, or automation tooling. You don't want to:
- Pay for real cloud resources
- Wait for slow API calls
- Deal with rate limits and quotas
- Clean up orphaned resources

NahCloud gives you a local cloud-shaped API that accepts your requests, stores state in SQLite, and lets you move on with your life.

## How it's different from LocalStack

**LocalStack** mocks AWS. It's great if you're testing AWS-specific Terraform configs.

**NahCloud** is a generic fake cloud with simple, predictable resources. It's for testing:
- Terraform provider development
- CI/CD pipeline logic
- State backend behavior

## Features

### Simple Resources
- **Projects** - top-level containers
- **Instances** - compute resources with CPU, memory, image, status
- **Metadata** - key-value storage with path-based hierarchy
- **Buckets & Objects** - blob storage

### Terraform State Backend
NahCloud implements the Terraform HTTP state backend protocol:
- `GET/POST/DELETE /v1/tfstate/{id}` - state operations
- `LOCK/UNLOCK /v1/tfstate/{id}` - state locking

### Web Console
Browse and manage resources at `http://localhost:8080/web/`

## Quick Start

```bash
# Run the server
make run-server
```

The API is available at `http://localhost:8080/v1/`

## Configuration

| Variable | Default | Description |
|----------|---------|-------------|
| `NAH_HTTP_ADDR` | `:8080` | Server listen address |
| `NAH_SQLITE_DSN` | `file:nah.db?...` | SQLite connection string |

## Authentication

NahCloud uses **API key authentication**. Each organization gets an API key when created, and you can create additional keys.

### Create an org (public endpoint)
```bash
curl -X POST http://localhost:8080/v1/orgs \
  -H "Content-Type: application/json" \
  -d '{"slug": "my-org", "name": "My Organization"}'
```

Response includes the org and an initial API key:
```json
{
  "id": "abc123...",
  "slug": "my-org",
  "name": "My Organization",
  "api_key": {
    "id": "def456...",
    "name": "default",
    "token": "nah_api_xxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxx"
  }
}
```

**Save this token!** It's only shown once.

### Use the API key
All other API calls require an API key:
```bash
curl http://localhost:8080/v1/orgs/my-org/projects \
  -H "Authorization: Bearer nah_api_xxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxx"
```

The API key only grants access to its own org's resources.

### Manage API keys
```bash
# Create additional key
curl -X POST http://localhost:8080/v1/orgs/my-org/api-keys \
  -H "Authorization: Bearer nah_api_xxx" \
  -H "Content-Type: application/json" \
  -d '{"name": "ci-pipeline"}'

# List keys
curl http://localhost:8080/v1/orgs/my-org/api-keys \
  -H "Authorization: Bearer nah_api_xxx"

# Delete a key
curl -X DELETE http://localhost:8080/v1/orgs/my-org/api-keys/{key_id} \
  -H "Authorization: Bearer nah_api_xxx"
```

## API Overview

```
# Projects
POST   /v1/projects
GET    /v1/projects
GET    /v1/projects/{id}
PATCH  /v1/projects/{id}
DELETE /v1/projects/{id}

# Instances
POST   /v1/instances
GET    /v1/instances
GET    /v1/instances/{id}
PATCH  /v1/instances/{id}
DELETE /v1/instances/{id}

# Metadata
POST   /v1/metadata
GET    /v1/metadata?prefix=...
GET    /v1/metadata/{id}
PATCH  /v1/metadata/{id}
DELETE /v1/metadata/{id}

# Buckets
POST   /v1/buckets
GET    /v1/buckets
GET    /v1/buckets/{id}
PATCH  /v1/buckets/{id}
DELETE /v1/buckets/{id}

# Objects
POST   /v1/bucket/{bucket_id}/objects
GET    /v1/bucket/{bucket_id}/objects?prefix=...
GET    /v1/bucket/{bucket_id}/objects/{id}
PATCH  /v1/bucket/{bucket_id}/objects/{id}
DELETE /v1/bucket/{bucket_id}/objects/{id}

# Terraform State
GET    /v1/tfstate/{id}
POST   /v1/tfstate/{id}
DELETE /v1/tfstate/{id}
LOCK   /v1/tfstate/{id}
UNLOCK /v1/tfstate/{id}
```

## License

MIT
