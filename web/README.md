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

## Frozen cloud graph contract

The `/cloud` console depends on the narrow `GraphConsole` interface in `cloud_graph.go`,
not directly on graph persistence or API handlers. `GraphScope` always carries the
authenticated organization ID and the project ID resolved from the route slug. The current
`main` branch has no corrected graph service contract, so the production adapter returns an
explicit unavailable state. It never creates process-local placeholder resources.

The adapter and templates follow these frozen semantics:

- **Networks** are project-scoped and own subnets. Network deletion is restricted while
  subnets exist. Subnet deletion is restricted while instances or load balancers reference it.
- **Disks** are project-scoped and have at most one attachment edge to a same-project,
  same-region instance. Disk deletion is restricted while attached; deleting an attachment
  removes only the edge.
- **Policies** are organization-scoped. Bindings connect organization or API-key principals
  to targets in the same organization. Deleting a policy cascades its bindings.
- **Load balancers** are project- and subnet-scoped and own backend edges. Backend health is
  computed as `enabled && instance.status == running`; no active probe is implied. Deleting a
  load balancer cascades its backends.
- **Instances** expose subnet, attached disks, and load-balancer memberships. Deleting an
  instance cascades attachment edges, backend edges, and bindings targeting it; disks survive.
- Every persisted node and edge has a stable, copyable opaque ID. Cross-tenant or wrong-parent
  references return not found. Empty and adapter-error states never invent data.
- **Fault Lab is read-only documentation/status.** Fault mutations remain API-key-only while
  organization permalinks grant writable browser sessions.

After PR #4's corrected service methods and domain types land, implement only
`serviceGraphConsole` mapping. The handlers and templates should continue consuming these
web view types so backend route or storage details do not leak into the console.
