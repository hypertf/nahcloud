# NahCloud Agent Notes

- `main` is the production branch.
- Pushes to `main` trigger the GitHub Actions release workflow that produces the `latest` binary used by production.
- For production-facing code changes, verify locally, then deploy as part of the task unless the user explicitly says not to deploy yet.
