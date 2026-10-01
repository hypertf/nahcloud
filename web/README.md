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

## Proposed cloud graph contract

The `/cloud` console intentionally depends on the narrow `GraphConsole` interface in
`cloud_graph.go`, not on backend graph domain, service, API, or storage packages. Until
those backend types exist, `NewHandler` uses a process-local preview adapter with seeded
data. Preview mutations are non-persistent and are clearly labelled in the UI.

The adapter assumes every operation receives an already-authorized, stable project ID:

- **Networks** own zero or more subnets. A subnet has one network parent; CIDRs, regions,
  and zones are display strings validated by the eventual backend.
- **Disks** belong to one project and optionally attach to one project instance at a mount
  path. The console expects resolved instance name/ID pairs in snapshots.
- **Policies** own bindings. Each binding connects a principal and role to one target in
  the same project (`network`, `disk`, `load-balancer`, or `instance`).
- **Load balancers** own backends. Each backend references one project instance and
  exposes port and health state.
- **Fault scenarios** are explicit project/tenant-scoped testing operations. The assumed
  kinds are `latency`, `packet-loss`, and `backend-outage`; enable/disable is idempotent.
- Snapshot failures return an error without a partial graph. Deletes return not-found when
  the resource is absent from the authorized project. Parent and target references must
  also resolve inside that project.

Backend integration should replace `newPreviewGraphConsole()` with an adapter over the
graph service while preserving this interface. The web layer must continue resolving the
organization and project slug before passing only the authorized project ID to the adapter.
