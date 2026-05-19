# Context Map

For v2 work, start with the root [v2 design draft](../../DESIGN.md). The other
context docs describe the current Go/v1 implementation unless the v2 draft
explicitly overrides them.

## Root

- [v2 design draft](../../DESIGN.md) — current Rust v2 product and architecture direction
- [summary](summary.md) — what lnk is and how it is built
- [intent](intent.md) — high-level product intent, inferred goals, and open questions
- [terminology](terminology.md) — stable terms (managed item, host scope, .lnk file)
- [practices](practices.md) — enforced conventions and invariants
- [architecture](architecture.md) — package layering and collaborator responsibilities
- [repo-layout](repo-layout.md) — on-disk shape, index format, host scoping
- [platform](platform.md) — build, release, CI, distribution

## Flows

- [init](flows/init.md) — empty init vs. clone, bootstrap, repo adoption
- [add-remove](flows/add-remove.md) — atomic add/multi/recursive, dry-run, remove, force-remove
- [sync](flows/sync.md) — status, diff, push, pull, restore symlinks, list
- [doctor](flows/doctor.md) — invalid entries, broken symlinks, dry-run vs fix
- [bootstrap](flows/bootstrap.md) — discovery and execution of bootstrap.sh
