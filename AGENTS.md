# NahCloud Agent Notes

- `main` is the production branch.
- Pushes to `main` trigger the GitHub Actions release workflow that produces the `latest` binary used by production.
- Never push or merge to `main`, trigger Forge quick deploy, or run another production mutation without explicit approval in the current thread.
- Bootstrap a fresh checkout with `./.agents/setup`; it installs the pinned Go toolchain and locked web dependencies without relying on a Mac or personal dotfiles.
- Run the full local verification with `./scripts/check`.
- Run locally with `PORT=8080 ./scripts/dev-server`.
- In an Amp orb, start or reconcile the portal-backed service with `amp orb service ensure`. Save screenshots and other acceptance artifacts under `.amp/in/artifacts/`.
- Local/orb data is disposable SQLite state under `.amp/runtime/`. It must never reuse or download production data.
- Read `docs/amp-orbs.md` before any infrastructure work. `./scripts/ops health` is public and read-only; provider inventory needs separately injected, read-only credentials.
