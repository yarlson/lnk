# Context Map

## Root

- [summary](summary.md) — what lnk is and how it is built
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
