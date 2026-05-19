# lnk

**Git-native machine setup. Profiles, files, scripts, no daemon.**

Track setup across machines with one command. Lnk keeps your setup repo in Git
(defaults to `~/.config/lnk`; override with `LNK_HOME` or `XDG_CONFIG_HOME`),
applies profile-owned files into `$HOME`, and stays out of your way.

```bash
lnk init                         # create a setup repo
lnk apply base mac work           # select profiles and apply files
lnk apply                         # re-apply this machine's profiles
lnk status                        # see repo and live-file state
lnk run setup                     # run explicit setup scripts
```

v2 is not a hidden configuration platform. It manages files and explicit
scripts under `$HOME`; it does not manage packages, services, secrets, sudo
workflows, or system paths.

## Install

Build v2 from source:

```bash
git clone https://github.com/yarlson/lnk.git
cd lnk
cargo build --release
```

The binary will be at `target/release/lnk`.

For development:

```bash
cargo run -- <command>
```

## How it works

```
Before: ~/.zshrc (regular file)
After:  ~/.zshrc -> ~/.config/lnk/profiles/base/files/.zshrc
```

Profiles live under `profiles/<name>/`. Normal files go in `files/` and are
symlinked by convention. Scripts go in `scripts/` and run only when you ask for
them.

```text
~/.config/lnk/
├── lnk.toml
└── profiles/
    ├── base/
    │   ├── files/
    │   │   ├── .zshrc
    │   │   └── .config/nvim/init.lua
    │   └── scripts/
    │       └── setup
    ├── mac/
    │   └── files/
    │       └── .config/karabiner/karabiner.json
    └── work/
        ├── profile.toml
        ├── files/
        │   └── .gitconfig
        └── scripts/
            └── setup
```

`profile.toml` is optional. Use it only when a file needs non-default behavior:
copy mode, target remapping, explicit overrides, or script metadata.

```toml
[files.".gitconfig"]
mode = "copy"

[files.".ssh/config"]
target = ".ssh/config"
overrides = "base"

[scripts.setup]
platforms = ["macos", "linux"]
```

Local machine state stays outside Git under XDG state. That state records this
machine's active profiles and applied targets, so the shared repo stays
reviewable.

## Usage

### Initialize

```bash
lnk init
```

This creates `lnk.toml`, `profiles/base/files/`, `profiles/base/scripts/`,
initializes Git if needed, and creates the first commit.

### Add files

Put files under the owning profile:

```text
profiles/base/files/.zshrc
profiles/base/files/.config/nvim/init.lua
profiles/mac/files/.config/karabiner/karabiner.json
```

The default target is the same path under `$HOME`.

### Apply profiles

```bash
lnk apply base mac work
lnk apply --dry-run base mac work
lnk apply
```

`lnk apply <profiles...>` validates the ordered profile list, stores it for this
machine, and applies the files. `lnk apply` without arguments uses the stored
profile list.

Preview uses the same command with `--dry-run`; no separate plan command is
needed.

### Conflicts and backups

Unmanaged live files block by default. If a real file already exists where lnk
would write a symlink or copy, apply stops instead of overwriting it.

```bash
lnk apply --backup base mac work
```

`--backup` renames conflicting files next to the original target:

```text
<path>.lnk-backup-YYYYMMDD-HHMMSS
```

Check for `.lnk-backup-*` files after running with `--backup` if you expect
them.

### Sync

Use Git for Git:

```bash
git -C ~/.config/lnk diff
git -C ~/.config/lnk commit -am "update setup"
git -C ~/.config/lnk pull
git -C ~/.config/lnk push
```

Then use lnk for machine state:

```bash
lnk apply
lnk status
```

`status` shows active profiles, applied targets, missing or wrong symlinks,
copy drift, and dirty repo state.

### Scripts

Scripts are explicit. Applying files never runs scripts.

```bash
lnk run setup            # run setup across active profiles
lnk run work:setup       # run work's setup script only
```

### Health checks

```bash
lnk doctor
lnk doctor --json
```

`doctor` is read-only. It audits repo manifests, profile metadata, target paths,
override declarations, local state, symlinks, and copy drift.

### JSON

Structured commands support `--json`:

```bash
lnk profiles --json
lnk apply --json
lnk status --json
lnk doctor --json
```

JSON output is for automation. Human output stays terse.

## New machine setup

```bash
git clone git@github.com:you/dotfiles.git ~/.config/lnk
lnk apply base mac work
lnk run setup
```

That's it. Profiles are stored for this machine, symlinks get restored, scripts
run only when requested, and you're working.

## Migration from v1

v2 keeps migration explicit and plan-first:

```bash
lnk migrate v1
lnk migrate v1 --write --yes
```

The read-only form prints the plan. The write form mutates the current Git
branch only after `--write --yes`.

The v1 mapping is:

- `.lnk` becomes profile `base`
- `.lnk.<host>` becomes profile `<host>`
- common files move to `profiles/base/files/`
- host files move to `profiles/<host>/files/`
- host/common path collisions become explicit host overrides
- `bootstrap.sh` becomes `profiles/base/scripts/setup`

Migration must refuse a dirty working tree, create a backup tag, preserve Git
history and remotes, make one migration commit, and never run scripts.

## Commands

| Command                                               | What it does                        |
| ----------------------------------------------------- | ----------------------------------- |
| `init`                                                | Create a v2 setup repo              |
| `profiles [--json]`                                   | Show available and active profiles  |
| `apply [--dry-run] [--backup] [--json] [profiles...]` | Apply profile files                 |
| `status [--json]`                                     | Show repo and local machine state   |
| `run <script>`                                        | Run a script across active profiles |
| `run <profile>:<script>`                              | Run one profile-owned script        |
| `doctor [--json]`                                     | Audit repo and local machine health |
| `migrate v1 [--write --yes]`                          | Reserved for v1 migration           |

## Repo locations

`lnk` resolves the setup repo in this order:

1. `LNK_HOME`
2. `XDG_CONFIG_HOME/lnk`
3. `~/.config/lnk`

Local state is stored at:

1. `$XDG_STATE_HOME/lnk/machine.toml`
2. `~/.local/state/lnk/machine.toml`

## Why lnk over alternatives

|                | lnk v2                 | chezmoi          | yadm            | stow                     |
| -------------- | ---------------------- | ---------------- | --------------- | ------------------------ |
| Config model   | Profiles + conventions | Templates + YAML | Git knowledge   | Stow directory structure |
| Machine state  | Local, outside Git     | Template data    | Manual branches | Manual                   |
| Scripts        | Explicit `lnk run`     | Hooks/scripts    | Bootstrap       | No                       |
| Conflicts      | Blocking by default    | Template-driven  | Git/manual      | Manual                   |
| Git model      | Normal Git repo        | Managed workflow | Normal Git repo | Normal Git repo          |
| Learning curve | Minutes                | Hours            | Medium          | Low                      |

## Contributing

```bash
git clone https://github.com/yarlson/lnk.git
cd lnk
make verify
```

## License

[MIT](LICENSE)
