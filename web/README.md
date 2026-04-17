# NahCloud Web Console

The NahCloud web console is served from the same production Go binary as the API. It is not a separately deployed frontend.

## Production Routing

- Dashboard: `https://nahcloud.com/`
- Org permalink: `https://nahcloud.com/o/{org-slug}`
- Projects: `https://nahcloud.com/projects`
- Instances: `https://nahcloud.com/projects/{project}/instances`
- Storage: `https://nahcloud.com/projects/{project}/storage`
- Metadata: `https://nahcloud.com/metadata`
- Settings: `https://nahcloud.com/settings`

Despite older handler comments, the console is mounted at the site root, not under `/web`.

## Authentication

- Browser access uses session-based auth middleware.
- API access lives under `/v1/` and uses bearer API keys.

## Deployment

- Deployed directly to production on the Forge-managed DigitalOcean host.
- Served by `nginx`, which proxies to the NahCloud Go process.
- Static assets, templates, and handlers ship inside the same deployed binary.
- Changes flow through `main`, the GitHub Actions `latest` release, and the Forge quick-deploy script.

## Features

The console provides BREAD operations for NahCloud resources:

- **Projects**: browse, create, edit, and delete
- **Instances**: browse, create, edit, and delete within a project
- **Metadata**: browse, create, edit, and delete with prefix filtering
- **Storage**: manage buckets and inspect project-scoped objects

Other behavior:

- First browser visit auto-creates a blank organization and session, without creating any projects or other resources.
- HTMX-driven partial updates
- Confirmation flows for destructive actions
- Stable org permalinks that create a session for the shared org
- Settings page for permalink sharing and org reset
- Validation that respects API constraints
- Responsive layout for routine browser-based operations
