# Amp Orb Runbook

NahCloud's Amp project is `jenicola/nahcloud` (`b1165141-ad8a-40d4-87a8-284957dcedfc`). A project orb clones `https://github.com/hypertf/nahcloud`, runs the repository lifecycle hooks, and can expose the local console through an authenticated Amp portal.

The local path has no production dependency: it runs one Go process with disposable SQLite state. No cloud credential is required to bootstrap, test, run, or take a screenshot.

## Fresh orb

Amp runs the executable `.agents/setup` hook on first boot. To reproduce the same steps manually:

```bash
./.agents/setup
./scripts/check
amp orb service ensure
```

`./scripts/bootstrap` installs Go 1.24.3 from an official archive after checking its SHA-256 digest, downloads Go modules from `go.sum`, and installs web packages with `npm ci` from `web/package-lock.json`. Runtime files stay in ignored `.amp/` directories.

The declared `nahcloud` service runs `./scripts/dev-server`, receives its port from Amp, checks `/buildz`, and publishes an authenticated portal. Inspect it with:

```bash
amp orb service status nahcloud
amp orb service logs nahcloud
```

Save visual evidence under `.amp/in/artifacts/`, for example:

```bash
mkdir -p .amp/in/artifacts
agent-browser screenshot --help
```

Use the browser tool available in the orb to open the service portal and write the final PNG into that directory. Do not use the production site for local acceptance.

## Local development

The same scripts work outside Amp:

```bash
./scripts/bootstrap
./scripts/check
PORT=8080 ./scripts/dev-server
```

Then open `http://127.0.0.1:8080/`. Override `NAH_SQLITE_DSN` only for another disposable local database. The defaults are listed, without secret values, in `.amp/environment.yaml`.

## Secrets and provider access

Local development uses no secrets. `FORGE_API_TOKEN` and `DIGITALOCEAN_ACCESS_TOKEN` are optional operator credentials and must be stored as Amp project secrets, never as environment entries, prompt text, command arguments, committed files, or portal manifests.

Load a secret from a protected file or hidden terminal prompt:

```bash
./scripts/amp-secret-set FORGE_API_TOKEN /secure/path/forge-token
./scripts/amp-secret-set DIGITALOCEAN_ACCESS_TOKEN
amp secrets list --project jenicola/nahcloud
```

The script sends values to `amp secrets set --secret --data-file`; it never echoes them. Prefer a current, expiring, read-only token. DigitalOcean needs only `droplet:read`. Forge needs only read scopes for the organization, server, site, deployments, and background processes. Do not reuse an old broad Forge token.

Amp orbs can mint short-lived workload identity with `amp orb id-token --audience <service>`. Prefer OIDC if Forge or DigitalOcean is later configured to exchange Amp identity for scoped access. Until that trust exists, OIDC is not a substitute for either provider token and no access-policy change should be made as part of routine project work.

List names and history without exposing values:

```bash
amp secrets list --project jenicola/nahcloud --json
amp secrets history --project jenicola/nahcloud --json
```

## Current production topology

Inventory on 2026-08-16 confirmed this path:

1. `nahcloud.com` and `www.nahcloud.com` resolve to `147.182.135.139`.
2. The address is DigitalOcean droplet `516040592`, named `us1`, in `nyc1`. It is an active Forge Ubuntu 24.04 host with 2 vCPUs, 4 GiB RAM, and an 80 GiB disk.
3. Laravel Forge server `955080` (`us1`) owns site `2999262` (`nahcloud.on-forge.com`), whose `nahcloud.com` alias has HTTPS enabled.
4. Forge checks out `https://github.com/hypertf/nahcloud` branch `main` under `/home/forge/nahcloud.on-forge.com` and has quick deploy enabled.
5. GitHub Actions builds the embedded web assets and a Linux `amd64` Go binary, then replaces the prerelease asset named `latest`.
6. The Forge deployment script pulls `main`, waits for the matching release, downloads the `latest` binary, and restarts supervisor program `daemon-656808`.
7. Nginx terminates TLS and proxies to the supervisor-managed Go process. The Go process persists application data in host-local SQLite. There is no separate frontend, database server, cache, queue, or object-store service for NahCloud.

The older personal `aops` map still points NahCloud at `canada1` (`167.99.189.241`); live DNS, DigitalOcean, Forge, and `/buildz` evidence supersede that stale entry for this runbook.

## Safe operator commands

These commands are read-only:

```bash
./scripts/ops health
./scripts/ops forge-inventory
./scripts/ops digitalocean-inventory
```

The provider commands return deliberately filtered fields, excluding Forge deployment hooks, environment contents, and other credential-bearing responses. If a credential is absent or insufficiently scoped, stop and report that narrow blocker.

Production mutations require explicit approval in the current Amp thread. This includes pushing or merging `main`, running `forge deploy`, calling a Forge deployment hook, restarting production services, changing site environment or Nginx configuration, changing DNS, rotating credentials, or changing provider access policy. A safe change flow is a feature branch plus pull request; do not select Amp's direct `Ship` behavior for this project because a `main` push releases production.

## Authoritative references

- [Amp Orbs manual](https://ampcode.com/manual/orbs)
- [Laravel Forge API introduction](https://forge.laravel.com/docs/api-reference/introduction)
- [Laravel Forge CLI](https://forge.laravel.com/docs/cli)
- [DigitalOcean Droplet API](https://docs.digitalocean.com/products/droplets/reference/api/droplets/)
