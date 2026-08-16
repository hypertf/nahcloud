# NahCloud

NahCloud is the fake cloud API running in production at `https://nahcloud.com`. It exists to test Terraform providers, CI/CD flows, and state operations without provisioning real infrastructure.

## Production

- Console: `https://nahcloud.com/`
- API: `https://nahcloud.com/v1/`
- Build info: `https://nahcloud.com/buildz`
- Apex is canonical: `https://www.nahcloud.com/` redirects to `https://nahcloud.com/`

## Deployment

NahCloud is deployed directly to production on a DigitalOcean droplet managed by Laravel Forge.

The current production path is:

1. `main` is the production branch.
2. GitHub Actions builds a Linux `amd64` `nahcloud` binary from `./cmd/server`.
3. The workflow publishes that binary to the GitHub release named `latest`.
4. Forge quick deploy pulls `main`, downloads the `latest` release asset, and restarts the supervisor-managed NahCloud daemon.
5. `nginx` terminates TLS and proxies traffic to the Go server.
6. Data is persisted in SQLite on the production host.

## Working Model

- Production still deploys from `main`, but changes are developed and verified on branches before any approved production push.
- A reproducible local and Amp orb workflow is documented in [`docs/amp-orbs.md`](docs/amp-orbs.md).
- Local/orb runs use disposable SQLite data and never require production credentials or data.

## Operational Notes

- TLS is managed by Forge and Let's Encrypt through `nginx`.
- Split/custom Forge sites need a name-based server snippet bridge:
  `/etc/nginx/forge-conf/<site-name>/server -> /etc/nginx/forge-conf/<site-id>/server`
- Without that bridge, Forge certificate renewals can fail to place ACME challenge config even when the site itself is otherwise healthy.

## Features

### Core resources

- **Projects**: top-level containers
- **Instances**: compute resources with CPU, memory, image, and status
- **Metadata**: key-value storage with path-based hierarchy
- **Buckets and objects**: blob storage

### Terraform state backend

NahCloud implements the Terraform HTTP backend protocol:

- `GET /v1/tfstate/{id}`
- `POST /v1/tfstate/{id}`
- `DELETE /v1/tfstate/{id}`
- `LOCK /v1/tfstate/{id}`
- `UNLOCK /v1/tfstate/{id}`

## Authentication

NahCloud uses API key authentication. Creating an organization returns the initial token once.

### Create an org

```bash
curl -X POST https://nahcloud.com/v1/orgs \
  -H "Content-Type: application/json" \
  -d '{"slug":"my-org","name":"My Organization"}'
```

Response includes the org and the initial API key:

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

Save the token. It is only returned at creation time.

### Use the API key

Authenticated API routes are scoped by the bearer token's organization.

```bash
curl https://nahcloud.com/v1/org \
  -H "Authorization: Bearer nah_api_xxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxx"
```

```bash
curl https://nahcloud.com/v1/projects \
  -H "Authorization: Bearer nah_api_xxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxx"
```

### Manage API keys

```bash
# Create an additional key
curl -X POST https://nahcloud.com/v1/api-keys \
  -H "Authorization: Bearer nah_api_xxx" \
  -H "Content-Type: application/json" \
  -d '{"name":"ci-pipeline"}'

# List keys
curl https://nahcloud.com/v1/api-keys \
  -H "Authorization: Bearer nah_api_xxx"

# Delete a key
curl -X DELETE https://nahcloud.com/v1/api-keys/{key_id} \
  -H "Authorization: Bearer nah_api_xxx"
```

## Web Console

The browser console is served by the same production Go binary as the API.

Useful routes:

- `/`
- `/o/{org-slug}`
- `/projects`
- `/projects/{project}/instances`
- `/projects/{project}/storage`
- `/metadata`
- `/settings`

First browser visit creates a blank org and browser session without creating any resources.
`/` is the org home, `/o/{org-slug}` is the shareable permalink, and `/settings` exposes sharing and reset controls.

## API Overview

```text
# Build info
GET    /buildz

# Organization and API keys
POST   /v1/orgs
GET    /v1/org
POST   /v1/api-keys
GET    /v1/api-keys
DELETE /v1/api-keys/{key_id}

# Projects
POST   /v1/projects
GET    /v1/projects
GET    /v1/projects/{project}
PATCH  /v1/projects/{project}
DELETE /v1/projects/{project}

# Instances
POST   /v1/projects/{project}/instances
GET    /v1/projects/{project}/instances
GET    /v1/projects/{project}/instances/{id}
PATCH  /v1/projects/{project}/instances/{id}
DELETE /v1/projects/{project}/instances/{id}

# Buckets
POST   /v1/projects/{project}/buckets
GET    /v1/projects/{project}/buckets
GET    /v1/projects/{project}/buckets/{bucket}
PATCH  /v1/projects/{project}/buckets/{bucket}
DELETE /v1/projects/{project}/buckets/{bucket}

# Objects
POST   /v1/projects/{project}/buckets/{bucket}/objects
GET    /v1/projects/{project}/buckets/{bucket}/objects
GET    /v1/projects/{project}/buckets/{bucket}/objects/{id}
PATCH  /v1/projects/{project}/buckets/{bucket}/objects/{id}
DELETE /v1/projects/{project}/buckets/{bucket}/objects/{id}

# Metadata
POST   /v1/metadata
GET    /v1/metadata
GET    /v1/metadata/{id}
PATCH  /v1/metadata/{id}
DELETE /v1/metadata/{id}

# Terraform state
GET    /v1/tfstate/{id}
POST   /v1/tfstate/{id}
DELETE /v1/tfstate/{id}
LOCK   /v1/tfstate/{id}
UNLOCK /v1/tfstate/{id}
```

## License

MIT
